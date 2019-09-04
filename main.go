package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/artefactual-labs/enduro/internal/api"
	"github.com/artefactual-labs/enduro/internal/cadence"
	"github.com/artefactual-labs/enduro/internal/collection"
	"github.com/artefactual-labs/enduro/internal/db"
	"github.com/artefactual-labs/enduro/internal/pipeline"
	"github.com/artefactual-labs/enduro/internal/watcher"
	"github.com/artefactual-labs/enduro/internal/workflow"

	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"github.com/oklog/run"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.uber.org/cadence/client"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const appName = "enduro"

var (
	version   = "(dev-version)"
	gitCommit = "(dev-commit)"
	buildTime = "(dev-buildtime)"
	goVersion = runtime.Version()
)

func main() {
	var (
		v = viper.New()
		p = pflag.NewFlagSet(appName, pflag.ExitOnError)
	)

	configureViper(v)

	p.String("config", "", "Configuration file")
	p.Bool("version", false, "Show version information")
	_ = p.Parse(os.Args[1:])

	if v, _ := p.GetBool("version"); v {
		fmt.Printf(
			"%s version %s (commit=%s) built on %s using %s\n",
			appName, version, gitCommit, buildTime, goVersion)
		os.Exit(0)
	}

	var config configuration
	configFile, _ := p.GetString("config")
	configFileFound, err := readConfig(v, &config, configFile)
	if err != nil {
		fmt.Printf("Failed to read configuration: %v\n", err)
		os.Exit(1)
	}

	// Logging configuration.
	var logger logr.Logger
	var zlogger *zap.Logger
	{
		var zconfig zap.Config
		if config.Debug {
			encoderConfig := zap.NewDevelopmentEncoderConfig()
			encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
			zconfig = zap.NewDevelopmentConfig()
			zconfig.EncoderConfig = encoderConfig
		} else {
			zconfig = zap.NewProductionConfig()
		}

		zlogger, err = zconfig.Build(zap.AddCallerSkip(1))
		zlogger = zlogger.Named(appName)
		defer func() { _ = zlogger.Sync() }()
		if err != nil {
			fmt.Printf("Failed to set up logger %v", err)
			os.Exit(1)
		}

		logger = zapr.NewLogger(zlogger)
		logger.Info("Starting...", "version", version)
	}

	if configFileFound {
		logger.Info("Configuration file not found.")
	} else {
		logger.Info("Configuration file loaded.", "path", v.ConfigFileUsed())
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	database, err := db.Connect(config.Database.DSN)
	if err != nil {
		logger.Error(err, "Database configuration failed.")
		os.Exit(1)
	}
	_ = database.Ping()

	var workflowClient client.Client
	{
		workflowClient, err = cadence.NewWorkflowClient(zlogger.Named("cadence-client"), appName, config.Cadence)
		if err != nil {
			logger.Error(err, "Cadence workflow client creation failed.")
			os.Exit(1)
		}
	}

	// Set up the collection service.
	var colsvc collection.Service
	{
		colsvc = collection.NewService(database, workflowClient)
	}

	// Set up the watcher service.
	var wsvc watcher.Service
	{
		wsvc, err = watcher.New(ctx, &config.Watcher)
		if err != nil {
			logger.Error(err, "Error setting up watchers.")
			os.Exit(1)
		}
	}

	var g run.Group

	// API server.
	{
		var srv *http.Server

		g.Add(
			func() error {
				srv = api.HTTPServer(logger, &config.API, colsvc.Goa())
				return srv.ListenAndServe()
			},
			func(err error) {
				ctx, cancel := context.WithTimeout(ctx, time.Second*5)
				defer cancel()
				_ = srv.Shutdown(ctx)
			},
		)
	}

	// Watchers, where each watcher is a group actor.
	{
		for _, w := range wsvc.Watchers() {
			done := make(chan struct{})
			cur := w
			g.Add(
				func() error {
					for {
						select {
						case <-done:
							return nil
						default:
							event, err := cur.Watch(ctx)
							if err != nil {
								if errors.Is(err, watcher.ErrWatchTimeout) {
									continue
								}
								logger.Error(err, "Error monitoring watcher interface.")
							}
							go func() {
								if err := collection.InitProcessingWorkflow(ctx, workflowClient, event); err != nil {
									logger.Error(err, "Error initializing processing workflow.")
								}
							}()
						}
					}
				},
				func(err error) {
					close(done)
				},
			)
		}
	}

	// Create workflow workers which manage workflow and activity executions.
	// This section could be executed as a different process and have replicas.
	{
		// TODO: this is a temporary workaround for dependency injection until we
		// figure out what's the depdencency tree is going to look like after POC.
		// The share-everything pattern should be avoided.
		m := workflow.NewManager(logger, colsvc, wsvc, pipeline.NewPipelineRegistry(config.Pipeline), config.Hooks)

		cadence.RegisterWorkflow(workflow.NewProcessingWorkflow(m).Execute, collection.ProcessingWorkflowName)
		cadence.RegisterActivity(workflow.NewDownloadActivity(m).Execute, workflow.DownloadActivityName)
		cadence.RegisterActivity(workflow.NewTransferActivity(m).Execute, workflow.TransferActivityName)
		cadence.RegisterActivity(workflow.NewPollTransferActivity(m).Execute, workflow.PollTransferActivityName)
		cadence.RegisterActivity(workflow.NewPollIngestActivity(m).Execute, workflow.PollIngestActivityName)
		cadence.RegisterActivity(workflow.NewUpdateHARIActivity(m).Execute, workflow.UpdateHARIActivityName)
		cadence.RegisterActivity(workflow.NewUpdateProductionSystemActivity(m).Execute, workflow.UpdateProductionSystemActivityName)
		cadence.RegisterActivity(workflow.NewCleanUpActivity(m).Execute, workflow.CleanUpActivityName)

		done := make(chan struct{})
		w, err := cadence.NewWorker(zlogger.Named("cadence-worker"), appName, config.Cadence)
		if err != nil {
			logger.Error(err, "Error creating Cadence worker.")
			os.Exit(1)
		}

		g.Add(
			func() error {
				if err := w.Start(); err != nil {
					return err
				}
				<-done
				return nil
			},
			func(err error) {
				w.Stop()
				close(done)
			},
		)
	}

	{
		ln, err := net.Listen("tcp", config.DebugListen)
		if err != nil {
			logger.Error(err, "Error setting up the debug interface.")
			os.Exit(1)
		}

		g.Add(func() error {
			mux := http.NewServeMux()

			// Health check.
			mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				fmt.Fprintln(w, "OK")
			})

			// Prometheus metrics.
			mux.Handle("/metrics", promhttp.Handler())

			// Profiling data.
			mux.HandleFunc("/debug/pprof/", pprof.Index)
			mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
			mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
			mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
			mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
			mux.Handle("/debug/pprof/block", pprof.Handler("block"))
			mux.Handle("/debug/pprof/goroutine", pprof.Handler("goroutine"))
			mux.Handle("/debug/pprof/heap", pprof.Handler("heap"))
			mux.Handle("/debug/pprof/threadcreate", pprof.Handler("threadcreate"))

			return http.Serve(ln, mux)
		}, func(error) {
			ln.Close()
		})
	}

	// Signal handler.
	{
		var (
			cancelInterrupt = make(chan struct{})
			ch              = make(chan os.Signal, 2)
		)
		defer close(ch)

		g.Add(
			func() error {
				signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)

				select {
				case <-ch:
				case <-cancelInterrupt:
				}

				return nil
			}, func(err error) {
				logger.Info("Quitting...")
				close(cancelInterrupt)
				cancel()
				signal.Stop(ch)
			},
		)
	}

	logger.Error(g.Run(), "Bye!")
}

type configuration struct {
	Debug       bool
	DebugListen string
	API         api.Config
	Database    db.Config
	Cadence     cadence.Config
	Watcher     watcher.Config
	Pipeline    []pipeline.Config

	// This is a workaround for client-specific functionality.
	// Simple mechanism to support an arbitrary number of hooks and parameters.
	Hooks map[string]map[string]interface{}
}

func (c configuration) Validate() error {
	return nil
}

func configureViper(v *viper.Viper) {
	v.AddConfigPath(".")
	v.AddConfigPath("$HOME/.config/")
	v.AddConfigPath("/etc")
	v.SetConfigName(appName)
	v.SetDefault("debugListen", "127.0.0.1:9001")
	v.SetDefault("api.listen", "127.0.0.1:9000")
	v.SetDefault("cadence.address", ":7933")
	v.Set("api.appVersion", version)
}

func readConfig(v *viper.Viper, config *configuration, configFile string) (found bool, err error) {
	if configFile != "" {
		v.SetConfigFile(configFile)
	}

	err = v.ReadInConfig()
	_, found = err.(viper.ConfigFileNotFoundError)
	if err != nil && !found {
		return found, fmt.Errorf("Failed to read configuration file: %w", err)
	}

	err = v.Unmarshal(config)
	if err != nil {
		return found, fmt.Errorf("Failed to unmarshal configuration: %w", err)
	}

	if err := config.Validate(); err != nil {
		return found, fmt.Errorf("Failed to validate the provided config: %w", err)
	}

	return found, nil
}
