// Package observability provides Prometheus metrics and a small HTTP metrics server.
//
// Integration notes:
//  1. Call InitGRPCServerMetrics() before creating your gRPC server.
//  2. Add UnaryServerInterceptor() and StreamServerInterceptor() to your gRPC server options.
//  3. Start RunMetricsHTTPServer() in a goroutine or an errgroup.
//  4. Use the helper functions from your service handlers to record business-level metrics.
//  5. Replace or remove dependency health checks if your project does not use external services.
package observability

import (
	"context"
	"fmt"
	"net/http"
	"time"

	promgrpc "github.com/grpc-ecosystem/go-grpc-middleware/providers/prometheus"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

const (
	defaultHealthCheckInterval = 15 * time.Second
	defaultShutdownTimeout     = 30 * time.Second
)

// AppConfig contains only the settings required by this metrics package.
// Replace this with your own application config if needed.
type AppConfig struct {
	MetricsHost string
	MetricsPort int
}

// HealthChecker is implemented by dependencies that can report availability.
//
// Examples:
//   - database repository
//   - Redis cache
//   - message broker client
//   - external API client wrapper
type HealthChecker interface {
	HealthCheck(ctx context.Context) error
}

var (
	grpcServerMetrics *promgrpc.ServerMetrics

	appOperationsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "app_operations_total",
			Help: "Total number of completed application operations.",
		},
		[]string{"service", "method", "status"},
	)

	appOperationDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "app_operation_duration_seconds",
			Help:    "Duration of application operations in seconds.",
			Buckets: prometheus.ExponentialBuckets(0.005, 2, 12),
		},
		[]string{"service", "method"},
	)

	appActiveRequests = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "app_active_requests",
			Help: "Number of requests currently being processed.",
		},
		[]string{"service", "method"},
	)

	appCriticalErrorsTotal = promauto.NewCounterVec(
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

// InitGRPCServerMetrics creates and registers gRPC server metrics.
//
// Call this once during application startup before attaching interceptors
// to your gRPC server.
func InitGRPCServerMetrics(opts ...promgrpc.ServerMetricsOption) *promgrpc.ServerMetrics {
	defaultOptions := []promgrpc.ServerMetricsOption{
		promgrpc.WithServerHandlingTimeHistogram(
			promgrpc.WithHistogramBuckets([]float64{
				0.001,
				0.005,
				0.01,
				0.025,
				0.05,
				0.1,
				0.25,
				0.5,
				1,
				2.5,
				5,
				10,
			}),
		),
	}

	allOptions := append(opts, defaultOptions...)

	grpcServerMetrics = promgrpc.NewServerMetrics(allOptions...)

	prometheus.MustRegister(grpcServerMetrics)

	return grpcServerMetrics
}

// RunMetricsHTTPServer exposes Prometheus metrics and a basic health endpoint.
//
// Endpoints:
//   - GET /metrics exposes Prometheus metrics.
//   - GET /health returns 200 OK when the HTTP metrics server is alive.
func RunMetricsHTTPServer(ctx context.Context, cfg AppConfig, logger *zap.Logger) error {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprintln(w, "ok")
	})

	server := &http.Server{
		Addr:              fmt.Sprintf("%s:%d", cfg.MetricsHost, cfg.MetricsPort),
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
			logger.Error("metrics http server stopped unexpectedly", zap.Error(err))
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return err

	case <-ctx.Done():
		logger.Info("stopping metrics http server")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), defaultShutdownTimeout)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			logger.Warn("graceful metrics shutdown failed; forcing close", zap.Error(err))

			if closeErr := server.Close(); closeErr != nil {
				logger.Error("forced metrics server close failed", zap.Error(closeErr))
			}

			return fmt.Errorf("shutdown metrics http server: %w", err)
		}

		logger.Info("metrics http server stopped")
		return nil
	}
}

// UnaryServerInterceptor returns the Prometheus unary gRPC interceptor.
func UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	if grpcServerMetrics == nil {
		InitGRPCServerMetrics()
	}

	return grpcServerMetrics.UnaryServerInterceptor()
}

// StreamServerInterceptor returns the Prometheus streaming gRPC interceptor.
func StreamServerInterceptor() grpc.StreamServerInterceptor {
	if grpcServerMetrics == nil {
		InitGRPCServerMetrics()
	}

	return grpcServerMetrics.StreamServerInterceptor()
}

// ObserveOperationDuration returns a defer-friendly function that records
// operation duration.
//
// Example:
//
//	done := observability.ObserveOperationDuration("CustomService", "CreateItem")
//	defer done()
func ObserveOperationDuration(service string, method string) func() {
	start := time.Now()

	return func() {
		duration := time.Since(start).Seconds()
		appOperationDuration.WithLabelValues(service, method).Observe(duration)
	}
}

// IncActiveRequest increments the number of in-flight requests.
func IncActiveRequest(service string, method string) {
	appActiveRequests.WithLabelValues(service, method).Inc()
}

// DecActiveRequest decrements the number of in-flight requests.
func DecActiveRequest(service string, method string) {
	appActiveRequests.WithLabelValues(service, method).Dec()
}

// IncOperation records a completed application operation.
//
// Suggested status values:
//   - "success"
//   - "error"
//   - "not_found"
//   - "invalid_argument"
func IncOperation(service string, method string, status string) {
	appOperationsTotal.WithLabelValues(service, method, status).Inc()
}

// IncCriticalError records a high-priority application error.
func IncCriticalError(errorType string) {
	appCriticalErrorsTotal.WithLabelValues(errorType).Inc()
}

// IncCacheMiss records a cache miss.
func IncCacheMiss(service string, cacheName string) {
	cacheMissesTotal.WithLabelValues(service, cacheName).Inc()
}

// RunDependencyHealthChecks periodically checks dependency availability
// and exports the result as Prometheus gauges.
//
// The checks stop when ctx is canceled.
func RunDependencyHealthChecks(
	ctx context.Context,
	serviceName string,
	dependencies map[string]HealthChecker,
	logger *zap.Logger,
) {
	ticker := time.NewTicker(defaultHealthCheckInterval)
	defer ticker.Stop()

	checkOnce := func() {
		for dependencyName, dependency := range dependencies {
			checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			err := dependency.HealthCheck(checkCtx)
			cancel()

			if err != nil {
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
