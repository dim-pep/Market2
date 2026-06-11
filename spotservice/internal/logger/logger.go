package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	modeDev  = "dev"
	modeProd = "prod"
)

func SetupLogger(mode string) (*zap.Logger, error) {
	switch mode {
	case modeDev:
		config := zap.NewDevelopmentConfig()
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		config.DisableStacktrace = true
		return config.Build(
			zap.AddCaller(),
			zap.Fields(zap.String("mode", mode)),
		)
	case modeProd:
		return zap.NewProduction()
	default:
		config := zap.NewDevelopmentConfig()
		return config.Build(
			zap.AddCaller(),
			zap.Fields(zap.String("mode", "unknow mode")),
		)
	}
}
