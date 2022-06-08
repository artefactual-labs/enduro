package log

import (
	"fmt"

	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger returns a new application logger based on the Zap logger.
func Logger(appName string, debug bool) (logr.Logger, error) {
	var zconfig zap.Config
	if debug {
		encoderConfig := zap.NewDevelopmentEncoderConfig()
		encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		zconfig = zap.NewDevelopmentConfig()
		zconfig.EncoderConfig = encoderConfig
	} else {
		zconfig = zap.NewProductionConfig()
	}

	zlogger, err := zconfig.Build(zap.AddCallerSkip(1))
	zlogger = zlogger.Named(appName)
	defer func() { _ = zlogger.Sync() }()
	if err != nil {
		return logr.Discard(), fmt.Errorf("zap logger error: %v", err)
	}

	return zapr.NewLogger(zlogger), nil
}
