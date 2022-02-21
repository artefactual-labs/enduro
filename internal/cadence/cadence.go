/*
Package cadence provide some utilities to interact with Cadence.
*/
package cadence

import (
	"fmt"

	"github.com/uber-go/tally"
	cadencesdk_gen_workflowserviceclient "go.uber.org/cadence/.gen/go/cadence/workflowserviceclient"
	cadencesdk_client "go.uber.org/cadence/client"
	cadencesdk_worker "go.uber.org/cadence/worker"
	"go.uber.org/yarpc"
	"go.uber.org/yarpc/transport/tchannel"
	"go.uber.org/zap"
)

const (
	// This is the task list name we use to identify our global client worker.
	// It also identifies the group of workflow and activity implementations
	// that are hosted by a single worker process.
	GlobalTaskListName = "global"

	// Name of the Cadence service, used by YARPC.
	cadenceServiceName = "cadence-frontend"
)

type Config struct {
	Domain  string
	Address string
}

func NewWorker(logger *zap.Logger, appName string, config Config) (cadencesdk_worker.Worker, error) {
	svc, err := serviceClient(logger, appName, config.Address)
	if err != nil {
		return nil, err
	}
	opts := cadencesdk_worker.Options{
		MetricsScope:                      tally.NoopScope,
		Logger:                            logger,
		EnableSessionWorker:               true,
		MaxConcurrentSessionExecutionSize: 5000,
	}
	return cadencesdk_worker.New(svc, config.Domain, GlobalTaskListName, opts), nil
}

// NewWorkflowClient returns a new Cadence client.
func NewWorkflowClient(logger *zap.Logger, appName string, config Config) (cadencesdk_client.Client, error) {
	svc, err := serviceClient(logger, appName, config.Address)
	if err != nil {
		return nil, err
	}
	opts := &cadencesdk_client.Options{
		MetricsScope: tally.NoopScope,
	}
	return cadencesdk_client.NewClient(svc, config.Domain, opts), nil
}

// NewDomainClient returns a Cadence Domain client.
func NewDomainClient(logger *zap.Logger, appName string, config Config) (cadencesdk_client.DomainClient, error) {
	svc, err := serviceClient(logger, appName, config.Address)
	if err != nil {
		return nil, err
	}
	opts := &cadencesdk_client.Options{
		MetricsScope: tally.NoopScope,
	}
	return cadencesdk_client.NewDomainClient(svc, opts), nil
}

// serviceClient returns a new client for the WorkflowService service.
func serviceClient(logger *zap.Logger, appName, addr string) (cadencesdk_gen_workflowserviceclient.Interface, error) {
	ch, err := tchannel.NewChannelTransport(
		tchannel.ServiceName(appName),
		tchannel.Logger(logger),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to set up tchannel: %w", err)
	}

	dispatcher := yarpc.NewDispatcher(yarpc.Config{
		Name: appName,
		Outbounds: yarpc.Outbounds{
			cadenceServiceName: {
				Unary: ch.NewSingleOutbound(addr),
			},
		},
	})
	if err := dispatcher.Start(); err != nil {
		return nil, fmt.Errorf("failed to start yarpc dispatcher: %w", err)
	}

	return cadencesdk_gen_workflowserviceclient.New(dispatcher.ClientConfig(cadenceServiceName)), nil
}
