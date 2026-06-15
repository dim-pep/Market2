package logger

import (
	"github.com/dim-pep/Market2/spotservice/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	envLocal = "local"
	envDev   = "dev"
	envProd  = "prod"
)

func SetupLoggerFromConfig(cfg *config.Config) (*zap.Logger, error) {
	if cfg == nil {
		return SetupLogger(envLocal)
	}

	return SetupLogger(cfg.Env)
}

func SetupLogger(env string) (*zap.Logger, error) {
	switch env {
	case envLocal, envDev:
		cfg := zap.NewDevelopmentConfig()
		cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		cfg.DisableStacktrace = true

		return cfg.Build(
			zap.AddCaller(),
			zap.Fields(zap.String("env", env)),
		)

	case envProd:
		cfg := zap.NewProductionConfig()

		return cfg.Build(
			zap.AddCaller(),
			zap.Fields(zap.String("env", env)),
		)

	default:
		cfg := zap.NewDevelopmentConfig()
		cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		cfg.DisableStacktrace = true

		return cfg.Build(
			zap.AddCaller(),
			zap.Fields(zap.String("env", env)),
		)
	}
}
