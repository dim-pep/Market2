package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"github.com/dim-pep/Market2/spotservice/config"
	"github.com/dim-pep/Market2/spotservice/internal/app"
	"github.com/dim-pep/Market2/spotservice/internal/cache"
	redis_cache "github.com/dim-pep/Market2/spotservice/internal/cache/redis"
	handler "github.com/dim-pep/Market2/spotservice/internal/grpc"
	"github.com/dim-pep/Market2/spotservice/internal/logger"
	observability "github.com/dim-pep/Market2/spotservice/internal/metrics"
	"github.com/dim-pep/Market2/spotservice/internal/repository"
	"github.com/dim-pep/Market2/spotservice/internal/repository/postgres"
	"github.com/dim-pep/Market2/spotservice/internal/service"

	"github.com/joho/godotenv"
	"go.uber.org/zap"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Print("failed to load .env file")
	}

	cfg := config.MustLoad()

	appLogger, err := logger.SetupLoggerFromConfig(cfg)
	if err != nil {
		log.Fatalf("failed to setup logger: %v", err)
	}
	defer func() {
		_ = appLogger.Sync()
	}()

	appLogger.Info("starting application", zap.String("env", cfg.Env))

	observability.InitServerMetrics()
	appLogger.Info("metrics initialized")

	rootCtx, stop := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT,
		syscall.SIGTERM,
	)
	defer stop()

	var marketsRepo repository.MarketsRepo

	marketsRepo, err = postgres.NewPostgresMarketsRepo(cfg)
	if err != nil {
		observability.IncCriticalError("boot_fail")
		appLogger.Fatal("failed to initialize postgres repository", zap.Error(err))
	}

	appLogger.Info("postgres repository initialized")

	var marketCache cache.CacheRepo

	marketCache, err = redis_cache.NewRedisRepo(cfg)
	if err != nil {
		observability.IncCriticalError("cache_boot_fail")
		appLogger.Warn("failed to connect to redis; cache disabled", zap.Error(err))
		marketCache = nil
	} else {
		appLogger.Info("redis connection established")
	}

	marketService := service.NewMarketService(marketsRepo, appLogger, marketCache)
	marketHandler := handler.NewSportServerHandler(marketService)

	application := app.NewApplication(
		cfg,
		marketHandler,
		appLogger,
		marketService,
		marketsRepo,
		marketCache,
	)

	go func() {
		if err := observability.StartHTTPMetricsServer(rootCtx, cfg, appLogger); err != nil {
			observability.IncCriticalError("metrics_server_failed")
			appLogger.Warn("metrics server stopped", zap.Error(err))
			stop()
		}
	}()

	appLogger.Info("starting infrastructure health checks")
	go observability.RunDependencyHealthChecks(
		rootCtx,
		"spot_service",
		map[string]observability.HealthChecker{
			"postgres": marketsRepo,
			"redis":    marketCache,
		},
		appLogger,
	)

	if err := application.Run(rootCtx, stop); err != nil {
		observability.IncCriticalError("app_crashed")
		appLogger.Fatal("application crashed", zap.Error(err))
	}

	appLogger.Info("application stopped gracefully")
}
