package api

import (
	"github.com/go-logr/logr"
	"goa.design/goa/v3/middleware"
)

// adapter is a thin wrapper around logr.Logger that implements Goa's Logger.
type adapter struct {
	logger logr.Logger
}

// loggerAdapter returns a new adapter.
func loggerAdapter(logger logr.Logger) middleware.Logger {
	return &adapter{
		logger: logger,
	}
}

// Log implements middleware.Logger.
func (a *adapter) Log(keyvals ...any) error {
	a.logger.Info("", keyvals...)
	return nil
}
