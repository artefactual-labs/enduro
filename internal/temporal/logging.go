package temporal

import (
	"context"
	"errors"

	"github.com/go-logr/logr"
	temporalsdk_activity "go.temporal.io/sdk/activity"
	temporalsdk_interceptor "go.temporal.io/sdk/interceptor"
	temporalsdk_log "go.temporal.io/sdk/log"
)

// logrWrapper implements the Temporal logger interface wrapping a logr.Logger.
type logrWrapper struct {
	logger logr.Logger
}

var _ temporalsdk_log.Logger = (*logrWrapper)(nil)

// Logger returns a logger for the Temporal Go SDK.
func Logger(logger logr.Logger) temporalsdk_log.Logger {
	return logrWrapper{logger.WithCallDepth(1)}
}

func (l logrWrapper) Debug(msg string, keyvals ...interface{}) {
	l.logger.V(1).WithValues("level", "debug").Info(msg, keyvals...)
}

func (l logrWrapper) Info(msg string, keyvals ...interface{}) {
	l.logger.WithValues("level", "info").Info(msg, keyvals...)
}

func (l logrWrapper) Warn(msg string, keyvals ...interface{}) {
	l.logger.WithValues("level", "warn").Info(msg, keyvals...)
}

func (l logrWrapper) Error(msg string, keyvals ...interface{}) {
	l.logger.Error(errors.New(msg), "error", keyvals...)
}

type workerInterceptor struct {
	temporalsdk_interceptor.WorkerInterceptorBase
	logger logr.Logger
}

var _ temporalsdk_interceptor.WorkerInterceptor = (*workerInterceptor)(nil)

// NewWorkerInterceptor returns an interceptor that makes the application logger
// available to activities via context.
func NewLoggerInterceptor(logger logr.Logger) *workerInterceptor {
	return &workerInterceptor{
		logger: logger,
	}
}

func (w *workerInterceptor) InterceptActivity(ctx context.Context, next temporalsdk_interceptor.ActivityInboundInterceptor) temporalsdk_interceptor.ActivityInboundInterceptor {
	i := &activityInboundInterceptor{
		root:   w,
		logger: w.logger,
	}
	i.Next = next
	return i
}

type activityInboundInterceptor struct {
	temporalsdk_interceptor.ActivityInboundInterceptorBase
	root   *workerInterceptor
	logger logr.Logger
}

type contextKey struct{}

var loggerContextKey = contextKey{}

func (a *activityInboundInterceptor) ExecuteActivity(ctx context.Context, in *temporalsdk_interceptor.ExecuteActivityInput) (interface{}, error) {
	ctx = context.WithValue(ctx, loggerContextKey, a.logger)
	return a.Next.ExecuteActivity(ctx, in)
}

func GetLogger(ctx context.Context) logr.Logger {
	v := ctx.Value(loggerContextKey)
	if v == nil {
		return logr.Discard()
	}

	logger := v.(logr.Logger)

	info := temporalsdk_activity.GetInfo(ctx)

	return logger.WithValues(
		"ActivityID", info.ActivityID,
		"ActivityType", info.ActivityType.Name,
	)
}
