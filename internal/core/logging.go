package core

import (
	"fmt"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// buildLoggerConfig returns a logger config writing to standard error.
func buildLoggerConfig() zap.Config {
	cfg := zap.NewProductionConfig()
	cfg.Sampling = nil
	cfg.DisableCaller = true
	cfg.DisableStacktrace = true
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	cfg.EncoderConfig.EncodeDuration = zapcore.StringDurationEncoder
	cfg.EncoderConfig.EncodeName = func(loggerName string, enc zapcore.PrimitiveArrayEncoder) {
		enc.AppendString("[" + loggerName + "]")
	}
	cfg.EncoderConfig.NewReflectedEncoder = nil
	cfg.Encoding = "console"

	return cfg
}

type LoggingSetup struct {
	logger *zap.Logger
	level  *zap.AtomicLevel
}

func NewLoggingSetup() (*LoggingSetup, error) {
	loggerConfig := buildLoggerConfig()

	logger, err := loggerConfig.Build()
	if err != nil {
		return nil, fmt.Errorf("building logger: %v", err)
	}

	return &LoggingSetup{
		logger: logger,
		level:  &loggerConfig.Level,
	}, nil
}

func (s *LoggingSetup) SetLevel(level zapcore.Level) {
	s.level.SetLevel(level)
}

func (s *LoggingSetup) Logger() *zap.Logger {
	return s.logger
}

// ReplaceGlobals replaces the global zap and standard loggers before returning
// a function to restore the original values.
func (s *LoggingSetup) ReplaceGlobals() func() {
	restoreGlobals := zap.ReplaceGlobals(s.logger)
	restoreStdLog := zap.RedirectStdLog(s.logger)

	return func() {
		restoreStdLog()
		restoreGlobals()
		s.logger.Sync()
	}
}
