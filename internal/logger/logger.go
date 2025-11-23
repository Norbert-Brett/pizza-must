package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// New creates a new structured logger
func New(env string) (*zap.Logger, error) {
	var config zap.Config

	if env == "production" {
		config = zap.NewProductionConfig()
	} else {
		config = zap.NewDevelopmentConfig()
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	// Always log to stdout for container compatibility
	config.OutputPaths = []string{"stdout"}
	config.ErrorOutputPaths = []string{"stderr"}

	// Ensure structured JSON format in production
	if env == "production" {
		config.Encoding = "json"
	}

	logger, err := config.Build(
		zap.AddCaller(),
		zap.AddStacktrace(zapcore.ErrorLevel),
	)
	if err != nil {
		return nil, err
	}

	return logger, nil
}

// NewWithDefaults creates a logger with default settings
func NewWithDefaults() *zap.Logger {
	env := os.Getenv("SERVER_ENV")
	if env == "" {
		env = "development"
	}

	logger, err := New(env)
	if err != nil {
		// Fallback to basic logger
		logger, _ = zap.NewProduction()
	}

	return logger
}
