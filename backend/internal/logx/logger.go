package logx

import (
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func New(level, format string) (*zap.Logger, error) {
	var cfg zap.Config
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "development":
		cfg = zap.NewDevelopmentConfig()
	default:
		cfg = zap.NewProductionConfig()
	}
	cfg.Level = zap.NewAtomicLevelAt(parseLevel(level))
	return cfg.Build()
}

func parseLevel(level string) zapcore.Level {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		return zapcore.DebugLevel
	case "warn", "warning":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	default:
		return zapcore.InfoLevel
	}
}
