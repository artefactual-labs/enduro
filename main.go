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

	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"github.com/oklog/run"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	cadencesdk_activity "go.uber.org/cadence/activity"
	cadencesdk_client "go.uber.org/cadence/client"
	cadencesdk_workflow "go.uber.org/cadence/workflow"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/artefactual-labs/enduro/internal/api"
	"github.com/artefactual-labs/enduro/internal/batch"
	"github.com/artefactual-labs/enduro/internal/cadence"
	"github.com/artefactual-labs/enduro/internal/collection"
	"github.com/artefactual-labs/enduro/internal/db"
	nha_activities "github.com/artefactual-labs/enduro/internal/nha/activities"
	"github.com/artefactual-labs/enduro/internal/pipeline"
	"github.com/artefactual-labs/enduro/internal/validation"
	"github.com/artefactual-labs/enduro/internal/watcher"
	"github.com/artefactual-labs/enduro/internal/workflow"
	"github.com/artefactual-labs/enduro/internal/workflow/activities"
	"github.com/artefactual-labs/enduro/internal/workflow/manager"
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
		logger.Info("Starting...", "version", version, "pid", os.Getpid())
	}

	if configFileFound {
		logger.Info("Configuration file loaded.", "path", v.ConfigFileUsed())
	} else {
		logger.Info("Configuration file not found.")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	database, err := db.Connect(config.Database.DSN)
	if err != nil {
		logger.Error(err, "Database configuration failed.")
		os.Exit(1)
	}
	_ = database.Ping()

	var workflowClient cadencesdk_client.Client
	{
		workflowClient, err = cadence.NewWorkflowClient(zlogger.Named("cadence-client"), appName, config.Cadence)
		if err != nil {
			logger.Error(err, "Cadence workflow client creation failed.")
			os.Exit(1)
		}
	}

	// Set up the pipeline registry.
	pipelineRegistry, err := pipeline.NewPipelineRegistry(logger.WithName("registry"), config.Pipeline)
	if err != nil {
		logger.Error(err, "Pipeline registry cannot be initialized.")
		os.Exit(1)
	}

	// Set up the pipeline service.
	var pipesvc pipeline.Service
	{
		pipesvc = pipeline.NewService(logger.WithName("pipeline"), pipelineRegistry)
	}

	// Set up the batch service.
	var batchsvc batch.Service
	{
		batchsvc = batch.NewService(logger.WithName("batch"), workflowClient, config.Watcher.CompletedDirs())
	}

	// Set up the collection service.
	var colsvc collection.Service
	{
		colsvc = collection.NewService(logger.WithName("collection"), database, workflowClient, pipelineRegistry)
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
				srv = api.HTTPServer(logger, &config.API, pipesvc, batchsvc, colsvc)
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
								if !errors.Is(err, watcher.ErrWatchTimeout) {
									logger.Error(err, "Error monitoring watcher interface.", "watcher", cur)
								}
								continue
							}
							logger.V(1).Info("Starting new workflow", "watcher", event.WatcherName, "bucket", event.Bucket, "key", event.Key, "dir", event.IsDir)
							go func() {
								req := collection.ProcessingWorkflowRequest{
									WatcherName:      event.WatcherName,
									PipelineNames:    event.PipelineName,
									RetentionPeriod:  event.RetentionPeriod,
									CompletedDir:     event.CompletedDir,
									StripTopLevelDir: event.StripTopLevelDir,
									Key:              event.Key,
									IsDir:            event.IsDir,
									ValidationConfig: config.Validation,
								}
								if err := collection.InitProcessingWorkflow(ctx, workflowClient, &req); err != nil {
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
		m := manager.NewManager(logger, colsvc, wsvc, pipelineRegistry, config.Hooks)

		done := make(chan struct{})
		w, err := cadence.NewWorker(zlogger.Named("cadence-worker"), appName, config.Cadence)
		if err != nil {
			logger.Error(err, "Error creating Cadence worker.")
			os.Exit(1)
		}

		w.RegisterWorkflowWithOptions(workflow.NewProcessingWorkflow(m).Execute, cadencesdk_workflow.RegisterOptions{Name: collection.ProcessingWorkflowName})
		w.RegisterActivityWithOptions(activities.NewAcquirePipelineActivity(m).Execute, cadencesdk_activity.RegisterOptions{Name: activities.AcquirePipelineActivityName})
		w.RegisterActivityWithOptions(activities.NewDownloadActivity(m).Execute, cadencesdk_activity.RegisterOptions{Name: activities.DownloadActivityName})
		w.RegisterActivityWithOptions(activities.NewBundleActivity(m).Execute, cadencesdk_activity.RegisterOptions{Name: activities.BundleActivityName})
		w.RegisterActivityWithOptions(activities.NewValidateTransferActivity().Execute, cadencesdk_activity.RegisterOptions{Name: activities.ValidateTransferActivityName})
		w.RegisterActivityWithOptions(activities.NewTransferActivity(m).Execute, cadencesdk_activity.RegisterOptions{Name: activities.TransferActivityName})
		w.RegisterActivityWithOptions(activities.NewPollTransferActivity(m).Execute, cadencesdk_activity.RegisterOptions{Name: activities.PollTransferActivityName})
		w.RegisterActivityWithOptions(activities.NewPollIngestActivity(m).Execute, cadencesdk_activity.RegisterOptions{Name: activities.PollIngestActivityName})
		w.RegisterActivityWithOptions(activities.NewCleanUpActivity(m).Execute, cadencesdk_activity.RegisterOptions{Name: activities.CleanUpActivityName})
		w.RegisterActivityWithOptions(activities.NewHidePackageActivity(m).Execute, cadencesdk_activity.RegisterOptions{Name: activities.HidePackageActivityName})
		w.RegisterActivityWithOptions(activities.NewDeleteOriginalActivity(m).Execute, cadencesdk_activity.RegisterOptions{Name: activities.DeleteOriginalActivityName})
		w.RegisterActivityWithOptions(activities.NewDisposeOriginalActivity(m).Execute, cadencesdk_activity.RegisterOptions{Name: activities.DisposeOriginalActivityName})

		w.RegisterActivityWithOptions(workflow.NewAsyncCompletionActivity(m).Execute, cadencesdk_activity.RegisterOptions{Name: workflow.AsyncCompletionActivityName})
		w.RegisterActivityWithOptions(nha_activities.NewUpdateHARIActivity(m).Execute, cadencesdk_activity.RegisterOptions{Name: nha_activities.UpdateHARIActivityName})
		w.RegisterActivityWithOptions(nha_activities.NewUpdateProductionSystemActivity(m).Execute, cadencesdk_activity.RegisterOptions{Name: nha_activities.UpdateProductionSystemActivityName})

		w.RegisterWorkflowWithOptions(collection.BulkWorkflow, cadencesdk_workflow.RegisterOptions{Name: collection.BulkWorkflowName})
		w.RegisterActivityWithOptions(collection.NewBulkActivity(colsvc).Execute, cadencesdk_activity.RegisterOptions{Name: collection.BulkActivityName})

		w.RegisterWorkflowWithOptions(batch.BatchWorkflow, cadencesdk_workflow.RegisterOptions{Name: batch.BatchWorkflowName})
		w.RegisterActivityWithOptions(batch.NewBatchActivity(batchsvc).Execute, cadencesdk_activity.RegisterOptions{Name: batch.BatchActivityName})

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
	Validation  validation.Config

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
	_, ok := err.(viper.ConfigFileNotFoundError)
	if !ok {
		found = true
	}
	if found && err != nil {
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
