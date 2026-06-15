package observability

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/dim-pep/Market2/spotservice/config"
	promgrpc "github.com/grpc-ecosystem/go-grpc-middleware/providers/prometheus"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

const (
	healthCheckInterval = 15 * time.Second
	shutdownTimeout     = 30 * time.Second
)

type HealthChecker interface {
	CheckHealth() error
}

var (
	serverMetrics *promgrpc.ServerMetrics

	operationsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "app_operations_total",
			Help: "Total number of completed application operations.",
		},
		[]string{"service", "method", "status"},
	)

	operationDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "app_operation_duration_seconds",
			Help:    "Duration of application operations in seconds.",
			Buckets: prometheus.ExponentialBuckets(0.005, 2, 12),
		},
		[]string{"service", "method"},
	)

	activeRequests = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "app_active_requests",
			Help: "Number of requests currently being processed.",
		},
		[]string{"service", "method"},
	)

	criticalErrorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "app_critical_errors_total",
			Help: "Total number of critical application errors.",
		},
		[]string{"error_type"},
	)

	dependencyUp = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "app_dependency_up",
			Help: "Dependency availability: 1 means up, 0 means down.",
		},
		[]string{"service", "dependency"},
	)

	cacheMissesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "app_cache_misses_total",
			Help: "Total number of cache misses.",
		},
		[]string{"service", "cache"},
	)
)

func InitServerMetrics(opts ...promgrpc.ServerMetricsOption) *promgrpc.ServerMetrics {
	serverOpts := append(
		opts,
		promgrpc.WithServerHandlingTimeHistogram(
			promgrpc.WithHistogramBuckets([]float64{
				0.001, 0.005, 0.01, 0.025, 0.05,
				0.1, 0.25, 0.5, 1, 2.5, 5, 10,
			}),
		),
	)

	serverMetrics = promgrpc.NewServerMetrics(serverOpts...)
	prometheus.MustRegister(serverMetrics)

	return serverMetrics
}

func StartHTTPMetricsServer(ctx context.Context, cfg *config.Config, logger *zap.Logger) error {
	if logger == nil {
		logger = zap.NewNop()
	}

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprintln(w, "ok")
	})

	server := &http.Server{
		Addr:              fmt.Sprintf("%s:%d", cfg.Prometheus.Host, cfg.Prometheus.Port),
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      5 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	errCh := make(chan error, 1)

	logger.Info("starting metrics http server", zap.String("addr", server.Addr))

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("metrics http server failed", zap.Error(err))
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return err

	case <-ctx.Done():
		logger.Info("stopping metrics http server")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			logger.Warn("metrics graceful shutdown failed; forcing close", zap.Error(err))

			if closeErr := server.Close(); closeErr != nil {
				logger.Error("metrics force close failed", zap.Error(closeErr))
			}

			return fmt.Errorf("failed to shutdown metrics server: %w", err)
		}

		logger.Info("metrics http server stopped")
		return nil
	}
}

func UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	if serverMetrics == nil {
		InitServerMetrics()
	}

	return serverMetrics.UnaryServerInterceptor()
}

func StreamServerInterceptor() grpc.StreamServerInterceptor {
	if serverMetrics == nil {
		InitServerMetrics()
	}

	return serverMetrics.StreamServerInterceptor()
}

func ObserveOperationDuration(service, method string) func() {
	start := time.Now()

	return func() {
		operationDuration.WithLabelValues(service, method).Observe(time.Since(start).Seconds())
	}
}

func IncActiveRequest(service, method string) {
	activeRequests.WithLabelValues(service, method).Inc()
}

func DecActiveRequest(service, method string) {
	activeRequests.WithLabelValues(service, method).Dec()
}

func IncOperation(service, method, status string) {
	operationsTotal.WithLabelValues(service, method, status).Inc()
}

func IncCriticalError(errorType string) {
	criticalErrorsTotal.WithLabelValues(errorType).Inc()
}

func IncCacheMiss(service, cacheName string) {
	cacheMissesTotal.WithLabelValues(service, cacheName).Inc()
}

func RunDependencyHealthChecks(ctx context.Context, serviceName string, dependencies map[string]HealthChecker, logger *zap.Logger) {
	if logger == nil {
		logger = zap.NewNop()
	}

	ticker := time.NewTicker(healthCheckInterval)
	defer ticker.Stop()

	checkOnce := func() {
		for dependencyName, dependency := range dependencies {
			if dependency == nil {
				dependencyUp.WithLabelValues(serviceName, dependencyName).Set(0)
				continue
			}

			if err := dependency.CheckHealth(); err != nil {
				dependencyUp.WithLabelValues(serviceName, dependencyName).Set(0)
				logger.Warn(
					"dependency health check failed",
					zap.String("service", serviceName),
					zap.String("dependency", dependencyName),
					zap.Error(err),
				)
				continue
			}

			dependencyUp.WithLabelValues(serviceName, dependencyName).Set(1)
		}
	}

	checkOnce()

	for {
		select {
		case <-ctx.Done():
			logger.Info("stopping dependency health checks", zap.String("service", serviceName))
			return

		case <-ticker.C:
			checkOnce()
		}
	}
}
