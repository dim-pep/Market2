package app

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/dim-pep/Market2/proto/pb/spot_service"
	"github.com/dim-pep/Market2/spotservice/config"
	"github.com/dim-pep/Market2/spotservice/internal/cache"
	handler "github.com/dim-pep/Market2/spotservice/internal/grpc"
	"github.com/dim-pep/Market2/spotservice/internal/interceptors"
	observability "github.com/dim-pep/Market2/spotservice/internal/metrics"
	"github.com/dim-pep/Market2/spotservice/internal/repository"
	"github.com/dim-pep/Market2/spotservice/internal/service"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
)

const gracefulTimeout = 60 * time.Second

type Application struct {
	cfg           *config.Config
	grpcServer    *grpc.Server
	logger        *zap.Logger
	marketService *service.MarketService
	db            repository.MarketsRepo
	cache         cache.CacheRepo
}

func NewApplication(cfg *config.Config, handler *handler.SportServerHandler, logger *zap.Logger, marketService *service.MarketService, db repository.MarketsRepo, cacheRepo cache.CacheRepo) *Application {
	if logger == nil {
		logger = zap.NewNop()
	}

	observability.InitServerMetrics()

	cacheInterceptor := interceptors.UnaryCacheInterceptor(
		cacheRepo,
		interceptors.CacheInterceptorConfig{
			KeyPrefix:   "spot_service:grpc_cache",
			ServiceName: "spot_service",
			CacheableMethods: map[string]struct{}{
				"/spot_service.SportInstrumentService/ViewMarkets": {},
			},
			NewResponse: func(method string) proto.Message {
				switch method {
				case "/spot_service.SportInstrumentService/ViewMarkets":
					return &spot_service.ViewMarketsResponse{}
				default:
					return nil
				}
			},
			OnCacheMiss: func(serviceName string, method string) {
				observability.IncCacheMiss(serviceName, "redis")
			},
		},
		logger,
	)

	opts := []grpc.ServerOption{
		grpc.ChainUnaryInterceptor(
			observability.UnaryServerInterceptor(),
			interceptors.UnaryRequestIDInterceptor(),
			interceptors.UnaryLoggingInterceptor(logger),
			interceptors.UnaryPanicRecoveryInterceptor(logger),
			cacheInterceptor,
		),
		grpc.ChainStreamInterceptor(
			observability.StreamServerInterceptor(),
		),
	}

	grpcServer := grpc.NewServer(opts...)

	handler.Register(grpcServer)

	return &Application{
		cfg:           cfg,
		grpcServer:    grpcServer,
		logger:        logger,
		marketService: marketService,
		db:            db,
		cache:         cacheRepo,
	}
}

func (a *Application) Run(ctx context.Context, stop context.CancelFunc) error {
	if a == nil {
		return fmt.Errorf("application is not initialized")
	}

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", a.cfg.GRPC.Port))
	if err != nil {
		return fmt.Errorf("failed to listen on grpc port %d: %w", a.cfg.GRPC.Port, err)
	}

	a.logger.Info("gRPC server starting", zap.String("addr", listener.Addr().String()))

	go func() {
		if err := a.grpcServer.Serve(listener); err != nil && err != grpc.ErrServerStopped {
			a.logger.Error("gRPC server crashed", zap.Error(err))

			if stop != nil {
				stop()
			}
		}
	}()

	<-ctx.Done()

	a.logger.Info("stop signal received, starting graceful shutdown")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), gracefulTimeout)
	defer cancel()

	done := make(chan struct{})

	go func() {
		a.grpcServer.GracefulStop()
		close(done)
	}()

	select {
	case <-shutdownCtx.Done():
		a.logger.Warn("graceful shutdown timeout, forcing gRPC server stop", zap.Error(shutdownCtx.Err()))
		a.grpcServer.Stop()

	case <-done:
		a.logger.Info("gRPC server stopped gracefully")
	}

	if a.marketService != nil {
		if err := a.marketService.CloseDBConnection(); err != nil {
			a.logger.Warn("failed to close db connection pool", zap.Error(err))
		}

		if err := a.marketService.CloseCacheClient(); err != nil {
			a.logger.Warn("failed to close cache connection pool", zap.Error(err))
		}
	}

	return nil
}
