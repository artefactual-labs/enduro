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

	"github.com/artefactual-sdps/temporal-activities/archive"
	"github.com/go-logr/logr"
	"github.com/oklog/run"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.artefactual.dev/tools/log"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
	temporalsdk_activity "go.temporal.io/sdk/activity"
	temporalsdk_client "go.temporal.io/sdk/client"
	temporalsdk_worker "go.temporal.io/sdk/worker"
	temporalsdk_workflow "go.temporal.io/sdk/workflow"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/artefactual-labs/enduro/internal/api"
	"github.com/artefactual-labs/enduro/internal/batch"
	"github.com/artefactual-labs/enduro/internal/collection"
	"github.com/artefactual-labs/enduro/internal/db"
	"github.com/artefactual-labs/enduro/internal/metadata"
	nha_activities "github.com/artefactual-labs/enduro/internal/nha/activities"
	"github.com/artefactual-labs/enduro/internal/pipeline"
	"github.com/artefactual-labs/enduro/internal/temporal"
	"github.com/artefactual-labs/enduro/internal/validation"
	"github.com/artefactual-labs/enduro/internal/watcher"
	"github.com/artefactual-labs/enduro/internal/workflow"
	"github.com/artefactual-labs/enduro/internal/workflow/activities"
	"github.com/artefactual-labs/enduro/internal/workflow/hooks"
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
	logger := log.New(os.Stderr,
		log.WithName(appName),
		log.WithDebug(config.Debug),
		log.WithLevel(config.Verbosity),
	)
	defer log.Sync(logger)

	logger.Info("Starting...", "version", version, "pid", os.Getpid())

	if configFileFound {
		logger.Info("Configuration file loaded.", "path", v.ConfigFileUsed())
	} else {
		logger.Info("Configuration file not found.")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up tracing.
	tp, shutdown, err := initTracerProvider(ctx, logger, config.Telemetry)
	if err != nil {
		logger.Error(err, "Error creating tracer provider.")
		os.Exit(1)
	}
	defer func() { _ = shutdown(ctx) }()
	tracer := tp.Tracer("enduro")

	database, err := db.Connect(config.Database.DSN)
	if err != nil {
		logger.Error(err, "Database configuration failed.")
		os.Exit(1)
	}
	_, span := tracer.Start(ctx, "db-ping")
	span.SetAttributes(attribute.String("db.driver", "mysql"))
	if err := database.Ping(); err != nil {
		span.SetStatus(codes.Error, "ping failed")
		span.RecordError(err)
	}
	span.AddEvent("Connected!")
	span.End()

	temporalClient, err := temporalsdk_client.Dial(temporalsdk_client.Options{
		Namespace: config.Temporal.Namespace,
		HostPort:  config.Temporal.Address,
		Logger:    temporal.Logger(logger.WithName("temporal-client")),
	})
	if err != nil {
		logger.Error(err, "Error creating Temporal client.")
		os.Exit(1)
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
		batchsvc = batch.NewService(logger.WithName("batch"), temporalClient, config.Temporal.TaskQueue, config.Watcher.CompletedDirs())
	}

	// Set up the collection service.
	var colsvc collection.Service
	{
		colsvc = collection.NewService(logger.WithName("collection"), database, temporalClient, config.Temporal.TaskQueue, pipelineRegistry)
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
				srv = api.HTTPServer(logger, tp, &config.API, pipesvc, batchsvc, colsvc)
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
			w := w
			done := make(chan struct{})
			g.Add(
				func() error {
					for {
						select {
						case <-done:
							return nil
						default:
							event, err := w.Watch(ctx)
							if err != nil {
								if !errors.Is(err, watcher.ErrWatchTimeout) {
									logger.Error(err, "Error monitoring watcher interface.", "watcher", w)
								}
								continue
							}
							ctx, span := tracer.Start(ctx, "Watcher")
							span.SetAttributes(
								attribute.String("watcher", event.WatcherName),
								attribute.String("bucket", event.Bucket),
								attribute.String("key", event.Key),
								attribute.Bool("dir", event.IsDir),
							)
							logger.V(1).Info("Starting new workflow", "watcher", event.WatcherName, "bucket", event.Bucket, "key", event.Key, "dir", event.IsDir)
							req := collection.ProcessingWorkflowRequest{
								WatcherName:        event.WatcherName,
								PipelineNames:      event.PipelineName,
								RetentionPeriod:    event.RetentionPeriod,
								CompletedDir:       event.CompletedDir,
								StripTopLevelDir:   event.StripTopLevelDir,
								RejectDuplicates:   event.RejectDuplicates,
								ExcludeHiddenFiles: event.ExcludeHiddenFiles,
								TransferType:       event.TransferType,
								Key:                event.Key,
								IsDir:              event.IsDir,
								ValidationConfig:   config.Validation,
								MetadataConfig:     config.Metadata,
							}
							if err := collection.InitProcessingWorkflow(ctx, tracer, temporalClient, config.Temporal.TaskQueue, &req); err != nil {
								logger.Error(err, "Error initializing processing workflow.")
							}
							span.End()
						}
					}
				},
				func(err error) {
					close(done)
				},
			)
		}
	}

	// Workflow and activity worker.
	{
		h := hooks.NewHooks(config.Hooks)

		done := make(chan struct{})
		w := temporalsdk_worker.New(temporalClient, config.Temporal.TaskQueue, temporalsdk_worker.Options{
			EnableSessionWorker:               true,
			MaxConcurrentSessionExecutionSize: 5000,
		})
		if err != nil {
			logger.Error(err, "Error creating Temporal worker.")
			os.Exit(1)
		}

		w.RegisterWorkflowWithOptions(workflow.NewProcessingWorkflow(h, colsvc, pipelineRegistry, logger).Execute, temporalsdk_workflow.RegisterOptions{Name: collection.ProcessingWorkflowName})
		w.RegisterActivityWithOptions(activities.NewAcquirePipelineActivity(pipelineRegistry).Execute, temporalsdk_activity.RegisterOptions{Name: activities.AcquirePipelineActivityName})
		w.RegisterActivityWithOptions(activities.NewDownloadActivity(h, pipelineRegistry, wsvc).Execute, temporalsdk_activity.RegisterOptions{Name: activities.DownloadActivityName})
		w.RegisterActivityWithOptions(archive.NewExtractActivity(config.ExtractActivity).Execute, temporalsdk_activity.RegisterOptions{Name: archive.ExtractActivityName})
		w.RegisterActivityWithOptions(activities.NewBundleActivity().Execute, temporalsdk_activity.RegisterOptions{Name: activities.BundleActivityName})
		w.RegisterActivityWithOptions(activities.NewValidateTransferActivity().Execute, temporalsdk_activity.RegisterOptions{Name: activities.ValidateTransferActivityName})
		w.RegisterActivityWithOptions(activities.NewTransferActivity(pipelineRegistry).Execute, temporalsdk_activity.RegisterOptions{Name: activities.TransferActivityName})
		w.RegisterActivityWithOptions(activities.NewPollTransferActivity(pipelineRegistry).Execute, temporalsdk_activity.RegisterOptions{Name: activities.PollTransferActivityName})
		w.RegisterActivityWithOptions(activities.NewPollIngestActivity(pipelineRegistry).Execute, temporalsdk_activity.RegisterOptions{Name: activities.PollIngestActivityName})
		w.RegisterActivityWithOptions(activities.NewCleanUpActivity().Execute, temporalsdk_activity.RegisterOptions{Name: activities.CleanUpActivityName})
		w.RegisterActivityWithOptions(activities.NewHidePackageActivity(pipelineRegistry).Execute, temporalsdk_activity.RegisterOptions{Name: activities.HidePackageActivityName})
		w.RegisterActivityWithOptions(activities.NewDeleteOriginalActivity(wsvc).Execute, temporalsdk_activity.RegisterOptions{Name: activities.DeleteOriginalActivityName})
		w.RegisterActivityWithOptions(activities.NewDisposeOriginalActivity(wsvc).Execute, temporalsdk_activity.RegisterOptions{Name: activities.DisposeOriginalActivityName})
		w.RegisterActivityWithOptions(activities.NewPopulateMetadataActivity(pipelineRegistry).Execute, temporalsdk_activity.RegisterOptions{Name: activities.PopulateMetadataActivityName})

		w.RegisterActivityWithOptions(workflow.NewAsyncCompletionActivity(colsvc).Execute, temporalsdk_activity.RegisterOptions{Name: workflow.AsyncCompletionActivityName})
		w.RegisterActivityWithOptions(nha_activities.NewUpdateHARIActivity(h).Execute, temporalsdk_activity.RegisterOptions{Name: nha_activities.UpdateHARIActivityName})
		w.RegisterActivityWithOptions(nha_activities.NewUpdateProductionSystemActivity(h).Execute, temporalsdk_activity.RegisterOptions{Name: nha_activities.UpdateProductionSystemActivityName})

		w.RegisterWorkflowWithOptions(collection.BulkWorkflow, temporalsdk_workflow.RegisterOptions{Name: collection.BulkWorkflowName})
		w.RegisterActivityWithOptions(collection.NewBulkActivity(colsvc).Execute, temporalsdk_activity.RegisterOptions{Name: collection.BulkActivityName})

		w.RegisterWorkflowWithOptions(batch.BatchWorkflow, temporalsdk_workflow.RegisterOptions{Name: batch.BatchWorkflowName})
		w.RegisterActivityWithOptions(batch.NewBatchActivity(batchsvc).Execute, temporalsdk_activity.RegisterOptions{Name: batch.BatchActivityName})

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

			srv := &http.Server{
				ReadTimeout:  5 * time.Second,
				WriteTimeout: 5 * time.Second,
				Handler:      mux,
			}
			return srv.Serve(ln)
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

type configuration struct {
	Verbosity       int
	Debug           bool
	DebugListen     string
	API             api.Config
	ExtractActivity archive.Config
	Database        db.Config
	Temporal        temporal.Config
	Watcher         watcher.Config
	Pipeline        []pipeline.Config
	Validation      validation.Config
	Telemetry       TelemetryConfig
	Metadata        metadata.Config

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
	v.Set("api.appVersion", version)

	temporal.SetDefaults(v)
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
		return found, fmt.Errorf("failed to read configuration file: %w", err)
	}

	err = v.Unmarshal(config)
	if err != nil {
		return found, fmt.Errorf("failed to unmarshal configuration: %w", err)
	}

	if err := config.Validate(); err != nil {
		return found, fmt.Errorf("failed to validate the provided config: %w", err)
	}

	return found, nil
}

type TelemetryConfig struct {
	Traces struct {
		Enabled       bool
		Address       string
		SamplingRatio *float64
	}
}

func initTracerProvider(ctx context.Context, logger logr.Logger, cfg TelemetryConfig) (trace.TracerProvider, func(context.Context) error, error) {
	if !cfg.Traces.Enabled || cfg.Traces.Address == "" {
		logger.V(1).Info("Tracing system is disabled.", "enabled", cfg.Traces.Enabled, "addr", cfg.Traces.Address)
		shutdown := func(context.Context) error { return nil }
		return noop.NewTracerProvider(), shutdown, nil
	}

	conn, err := grpc.DialContext(
		ctx,
		cfg.Traces.Address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("can't connect to telemetry data collector: %v", err)
	}

	exporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithGRPCConn(conn))
	if err != nil {
		return nil, nil, fmt.Errorf("can't create gRPC telemetry data exporter: %v", err)
	}

	resource, _ := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(appName),
			semconv.ServiceVersion(version),
		),
	)

	var ratio float64 = 1
	if cfg.Traces.SamplingRatio != nil {
		ratio = *cfg.Traces.SamplingRatio
	}
	sampler := sdktrace.ParentBased(sdktrace.TraceIDRatioBased(ratio))

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sampler),
		sdktrace.WithResource(resource),
		sdktrace.WithBatcher(exporter),
	)
	shutdown := func(context.Context) error { return tp.Shutdown(ctx) }

	logger.V(1).Info("Using OTel gRPC tracer provider.", "addr", cfg.Traces.Address, "ratio", ratio)

	return tp, shutdown, nil
}
