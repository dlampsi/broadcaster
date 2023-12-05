// Package logging sets up and configures logging.
package logging

import (
	"context"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// contextKey is a private string type to prevent collisions in the context map.
type contextKey string

const (
	// loggerKey points to the value in the context where the logger is stored.
	loggerKey = contextKey("logger")

	nameKey       = "logger"
	messageKey    = "msg"
	stacktraceKey = "stacktrace"
	levelKey      = "level"
	timeKey       = "logtime"
)

var (
	devEncoderConfig = zapcore.EncoderConfig{
		NameKey:        nameKey,
		MessageKey:     messageKey,
		StacktraceKey:  stacktraceKey,
		LevelKey:       levelKey,
		EncodeLevel:    zapcore.CapitalColorLevelEncoder,
		TimeKey:        timeKey,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
		LineEnding:     zapcore.DefaultLineEnding,
	}
	prodEncoderConfig = zapcore.EncoderConfig{
		NameKey:        nameKey,
		MessageKey:     messageKey,
		StacktraceKey:  stacktraceKey,
		LevelKey:       levelKey,
		EncodeLevel:    zapcore.CapitalLevelEncoder,
		TimeKey:        timeKey,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
		LineEnding:     zapcore.DefaultLineEnding,
	}
)

func NewLogger(level string, development bool) *zap.SugaredLogger {
	var config *zap.Config

	if development {
		config = &zap.Config{
			Level:            zap.NewAtomicLevelAt(toZapLevel(level)),
			Development:      true,
			Encoding:         "console",
			EncoderConfig:    devEncoderConfig,
			OutputPaths:      []string{"stdout"},
			ErrorOutputPaths: []string{"stderr"},
		}
	} else {
		config = &zap.Config{
			Level:            zap.NewAtomicLevelAt(toZapLevel(level)),
			Encoding:         "json",
			EncoderConfig:    prodEncoderConfig,
			OutputPaths:      []string{"stdout"},
			ErrorOutputPaths: []string{"stderr"},
		}
	}

	logger, err := config.Build()
	if err != nil {
		logger = zap.NewNop()
	}

	// Output stacktracess at the levels above the warning
	logger = logger.WithOptions(zap.AddStacktrace(zap.ErrorLevel))

	return logger.Sugar()
}

func toZapLevel(level string) zapcore.Level {
	switch strings.ToLower(level) {
	case "debug":
		return zapcore.DebugLevel
	case "info":
		return zapcore.InfoLevel
	case "warning":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	case "fatal":
		return zapcore.FatalLevel
	}
	return zapcore.InfoLevel
}

func DefaultLogger() *zap.SugaredLogger {
	return zap.NewNop().Sugar()
}

// Creates new logger from context.
// If no logger in context, returns DefaultLogger.
func FromContext(ctx context.Context) *zap.SugaredLogger {
	if logger, ok := ctx.Value(loggerKey).(*zap.SugaredLogger); ok {
		return logger
	}
	return DefaultLogger()
}

// Populates context by logger.
func ContextWithLogger(ctx context.Context, logger *zap.SugaredLogger) context.Context {
	return context.WithValue(ctx, loggerKey, logger)
}
