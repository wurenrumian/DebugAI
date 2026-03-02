package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Logger *zap.Logger

func InitLogger(env string) {
	var config zap.Config

	if env == "production" {
		config = zap.NewProductionConfig()
		config.Encoding = "json"
	} else {
		config = zap.NewDevelopmentConfig()
		config.Encoding = "console"
	}

	config.DisableCaller = false
	config.DisableStacktrace = false
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	logger, err := config.Build()
	if err != nil {
		panic("Failed to initialize logger: " + err.Error())
	}

	Logger = logger
	defer logger.Sync()
}

func Info(msg string, fields ...zap.Field) {
	Logger.Info(msg, fields...)
}

func Error(msg string, fields ...zap.Field) {
	Logger.Error(msg, fields...)
}

func Debug(msg string, fields ...zap.Field) {
	Logger.Debug(msg, fields...)
}

func Warn(msg string, fields ...zap.Field) {
	Logger.Warn(msg, fields...)
}
