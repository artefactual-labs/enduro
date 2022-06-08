package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/oklog/run"
	"github.com/opensearch-project/opensearch-go"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/pflag"
	temporalsdk_activity "go.temporal.io/sdk/activity"
	temporalsdk_client "go.temporal.io/sdk/client"
	temporalsdk_worker "go.temporal.io/sdk/worker"

	"github.com/artefactual-labs/enduro/internal/a3m"
	"github.com/artefactual-labs/enduro/internal/collection"
	"github.com/artefactual-labs/enduro/internal/config"
	"github.com/artefactual-labs/enduro/internal/db"
	"github.com/artefactual-labs/enduro/internal/log"
	sdps_activities "github.com/artefactual-labs/enduro/internal/sdps/activities"
	"github.com/artefactual-labs/enduro/internal/temporal"
	"github.com/artefactual-labs/enduro/internal/version"
	"github.com/artefactual-labs/enduro/internal/watcher"
	"github.com/artefactual-labs/enduro/internal/workflow/activities"
)

const (
	appName = "enduro-a3m-worker"
)

func main() {
	p := pflag.NewFlagSet(appName, pflag.ExitOnError)

	p.String("config", "", "Configuration file")
	p.Bool("version", false, "Show version information")
	_ = p.Parse(os.Args[1:])

	if v, _ := p.GetBool("version"); v {
		fmt.Printf(
			"%s version %s (commit=%s) built on %s using %s\n",
			appName, version.Version, version.GitCommit, version.BuildTime, version.GoVersion)
		os.Exit(0)
	}

	var cfg config.Configuration
	configFile, _ := p.GetString("config")
	configFileFound, configFileUsed, err := config.Read(&cfg, configFile)
	if err != nil {
		fmt.Printf("Failed to read configuration: %v\n", err)
		os.Exit(1)
	}

	logger, err := log.Logger(appName, cfg.Debug)
	if err != nil {
		fmt.Printf("Failed to create logger: %v\n", err)
		os.Exit(1)
	}

	logger.Info("Starting...", "version", version.Version, "pid", os.Getpid())

	if configFileFound {
		logger.Info("Configuration file loaded.", "path", configFileUsed)
	} else {
		logger.Info("Configuration file not found.")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	database, err := db.Connect(cfg.Database.DSN)
	if err != nil {
		logger.Error(err, "Database configuration failed.")
		os.Exit(1)
	}
	_ = database.Ping()

	temporalClient, err := temporalsdk_client.NewClient(temporalsdk_client.Options{
		Namespace: cfg.Temporal.Namespace,
		HostPort:  cfg.Temporal.Address,
		Logger:    temporal.Logger(logger.WithName("temporal-client")),
	})
	if err != nil {
		logger.Error(err, "Error creating Temporal client.")
		os.Exit(1)
	}

	// Set up the collection service.
	var colsvc collection.Service
	{
		colsvc = collection.NewService(logger.WithName("collection"), database, temporalClient)
	}

	searchConfig := opensearch.Config{
		Addresses: cfg.Search.Addresses,
		Username:  cfg.Search.Username,
		Password:  cfg.Search.Password,
		Transport: &http.Transport{
			MaxIdleConnsPerHost:   10,
			ResponseHeaderTimeout: time.Second,
			DialContext:           (&net.Dialer{Timeout: time.Second}).DialContext,
			TLSClientConfig: &tls.Config{
				MinVersion: tls.VersionTLS11,
			},
		},
	}
	searchClient, err := opensearch.NewClient(searchConfig)
	if err != nil {
		logger.Error(err, "Search configuration failed.")
		os.Exit(1)
	}
	_, err = searchClient.API.Ping()
	if err != nil {
		logger.Info("Search server not available.", "msg", err, "retries", !searchConfig.DisableRetry)
	}
	logger.Info("Search client configured")

	// Set up the watcher service.
	var wsvc watcher.Service
	{
		wsvc, err = watcher.New(ctx, &cfg.Watcher)
		if err != nil {
			logger.Error(err, "Error setting up watchers.")
			os.Exit(1)
		}
	}

	var g run.Group

	// Activity worker.
	{
		done := make(chan struct{})
		workerOpts := temporalsdk_worker.Options{
			DisableWorkflowWorker:              true,
			EnableSessionWorker:                true,
			MaxConcurrentSessionExecutionSize:  1,
			MaxConcurrentActivityExecutionSize: 1,
		}
		w := temporalsdk_worker.New(temporalClient, temporal.A3mWorkerTaskQueue, workerOpts)
		if err != nil {
			logger.Error(err, "Error creating Temporal worker.")
			os.Exit(1)
		}

		w.RegisterActivityWithOptions(activities.NewDownloadActivity(wsvc).Execute, temporalsdk_activity.RegisterOptions{Name: activities.DownloadActivityName})
		w.RegisterActivityWithOptions(activities.NewBundleActivity(wsvc).Execute, temporalsdk_activity.RegisterOptions{Name: activities.BundleActivityName})
		w.RegisterActivityWithOptions(activities.NewValidateTransferActivity().Execute, temporalsdk_activity.RegisterOptions{Name: activities.ValidateTransferActivityName})
		w.RegisterActivityWithOptions(a3m.NewCreateAIPActivity(logger, &cfg.A3m, colsvc).Execute, temporalsdk_activity.RegisterOptions{Name: a3m.CreateAIPActivityName})
		w.RegisterActivityWithOptions(activities.NewCleanUpActivity().Execute, temporalsdk_activity.RegisterOptions{Name: activities.CleanUpActivityName})
		w.RegisterActivityWithOptions(activities.NewUploadActivity(cfg.AIPStore).Execute, temporalsdk_activity.RegisterOptions{Name: activities.UploadActivityName})

		w.RegisterActivityWithOptions(sdps_activities.NewValidatePackageActivity().Execute, temporalsdk_activity.RegisterOptions{Name: sdps_activities.ValidatePackageActivityName})
		w.RegisterActivityWithOptions(sdps_activities.NewIndexActivity(logger, searchClient).Execute, temporalsdk_activity.RegisterOptions{Name: sdps_activities.IndexActivityName})

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

	// Observability server.
	{
		ln, err := net.Listen("tcp", cfg.DebugListen)
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

	err = g.Run()
	if err != nil {
		logger.Error(err, "Application failure.")
		os.Exit(1)
	}
	logger.Info("Bye!")
}
