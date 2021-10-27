package logging

import (
	"context"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type contextKey string

const loggerKey = contextKey("zaplogger")

var (
	defaultLogger     logr.Logger //nolint:gochecknoglobals
	defaultLoggerOnce sync.Once   //nolint:gochecknoglobals
)

// NewLogger creates a new logger with the given configuration.
func NewLogger(verbosityLevel int, development bool) logr.Logger {
	var config *zap.Config
	if development {
		config = &zap.Config{
			Level:            zap.NewAtomicLevelAt(verbosityToZapLevel(verbosityLevel)),
			Development:      true,
			Encoding:         encodingConsole,
			EncoderConfig:    developmentEncoderConfig,
			OutputPaths:      outputStderr,
			ErrorOutputPaths: outputStderr,
		}
	} else {
		config = &zap.Config{
			Level:            zap.NewAtomicLevelAt(verbosityToZapLevel(verbosityLevel)),
			Encoding:         encodingJSON,
			EncoderConfig:    productionEncoderConfig,
			OutputPaths:      outputStderr,
			ErrorOutputPaths: outputStderr,
		}
	}

	logger, err := config.Build()
	if err != nil {
		logger = zap.NewExample()
	}

	return zapr.NewLogger(logger)
}

// NewLoggerFromEnv creates a new logger from the environment. It consumes
// LOG_LEVEL for determining the level and LOG_MODE for determining the output
// parameters.
func NewLoggerFromEnv() logr.Logger {
	verbosityLevel, err := strconv.Atoi(os.Getenv("LOG_VERBOSITY"))
	if err != nil {
		verbosityLevel = 1
	}

	development := strings.ToLower(strings.TrimSpace(os.Getenv("LOG_MODE"))) == "development"
	return NewLogger(verbosityLevel, development)
}

// DefaultLogger gives you a logger with the default settings replied.
func DefaultLogger() logr.Logger {
	defaultLoggerOnce.Do(func() {
		defaultLogger = NewLoggerFromEnv()
	})
	return defaultLogger
}

// WithLogger creates a new context with the given logger attached.
func WithLogger(ctx context.Context, logger logr.Logger) context.Context {
	return context.WithValue(ctx, loggerKey, logger)
}

// FromContext returns the logger stored within the context.
// If no logger is stored, the default logger is returned.
func FromContext(ctx context.Context) logr.Logger {
	if logger, ok := ctx.Value(loggerKey).(logr.Logger); ok {
		return logger
	}
	return DefaultLogger()
}

const (
	timestamp  = "timestamp"
	levelKey   = "verbosity"
	logger     = "logger"
	caller     = "caller"
	message    = "message"
	stacktrace = "stacktrace"

	encodingConsole = "console"
	encodingJSON    = "json"
)

var outputStderr = []string{"stderr"}

var productionEncoderConfig = zapcore.EncoderConfig{
	TimeKey:        timestamp,
	LevelKey:       levelKey,
	NameKey:        logger,
	CallerKey:      caller,
	MessageKey:     message,
	StacktraceKey:  stacktrace,
	LineEnding:     zapcore.DefaultLineEnding,
	EncodeLevel:    verbosityEncoder(),
	EncodeTime:     timeEncoder(),
	EncodeDuration: zapcore.SecondsDurationEncoder,
	EncodeCaller:   zapcore.ShortCallerEncoder,
}

var developmentEncoderConfig = zapcore.EncoderConfig{
	TimeKey:        "T",
	LevelKey:       "V",
	NameKey:        "N",
	CallerKey:      "C",
	FunctionKey:    zapcore.OmitKey,
	MessageKey:     "M",
	StacktraceKey:  "S",
	LineEnding:     zapcore.DefaultLineEnding,
	EncodeLevel:    verbosityEncoder(),
	EncodeTime:     zapcore.ISO8601TimeEncoder,
	EncodeDuration: zapcore.StringDurationEncoder,
	EncodeCaller:   zapcore.ShortCallerEncoder,
}

// verbosityToZapLevel converts the given logr verbosity level to the appropriate zap level
// value.
func verbosityToZapLevel(verbosity int) zapcore.Level {
	level := int8(-verbosity)
	return zapcore.Level(level)
}

func verbosityEncoder() zapcore.LevelEncoder {
	// This is the inverse of verbosityToZapLevel
	return func(level zapcore.Level, encoder zapcore.PrimitiveArrayEncoder) {
		inv := -level
		encoder.AppendInt8(int8(inv))
	}
}

// timeEncoder encodes the time as RFC3339 nano (UTC)
func timeEncoder() zapcore.TimeEncoder {
	return func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
		enc.AppendString(t.UTC().Format(time.RFC3339Nano))
	}
}
