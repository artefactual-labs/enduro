package temporal

import (
	"errors"

	"github.com/go-logr/logr"
	temporalsdk_log "go.temporal.io/sdk/log"
)

// logrWrapper implements temporalsdk_log.Logger.
type logrWrapper struct {
	logger logr.Logger
}

var _ temporalsdk_log.Logger = (*logrWrapper)(nil)

func (l logrWrapper) Debug(msg string, keyvals ...interface{}) {
	l.logger.WithValues("level", "debug").Info(msg, keyvals...)
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

func Logger(logger logr.Logger) temporalsdk_log.Logger {
	return logrWrapper{logger.WithCallDepth(1)}
}
