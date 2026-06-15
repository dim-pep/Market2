package interceptors

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cespare/xxhash/v2"
	"github.com/dim-pep/Market2/spotservice/internal/cache"
	"go.uber.org/zap"
	"golang.org/x/sync/singleflight"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
)

type ResponseFactory func(method string) proto.Message
type CacheMissObserver func(service string, method string)

type CacheInterceptorConfig struct {
	KeyPrefix        string
	ServiceName      string
	CacheableMethods map[string]struct{}
	NewResponse      ResponseFactory
	OnCacheMiss      CacheMissObserver
}

func UnaryCacheInterceptor(cache cache.CacheRepo, cfg CacheInterceptorConfig, logger *zap.Logger) grpc.UnaryServerInterceptor {
	if logger == nil {
		logger = zap.NewNop()
	}

	if cfg.KeyPrefix == "" {
		cfg.KeyPrefix = "app:grpc_cache"
	}

	if cfg.ServiceName == "" {
		cfg.ServiceName = "unknown_service"
	}

	group := singleflight.Group{}

	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if _, ok := cfg.CacheableMethods[info.FullMethod]; !ok {
			return handler(ctx, req)
		}

		if cfg.NewResponse == nil {
			logger.Warn(
				"cache response factory is not configured; skipping cache",
				zap.String("grpc.method", info.FullMethod),
			)
			return handler(ctx, req)
		}

		cachedResponse := cfg.NewResponse(info.FullMethod)
		if cachedResponse == nil {
			logger.Warn(
				"cache response factory returned nil; skipping cache",
				zap.String("grpc.method", info.FullMethod),
			)
			return handler(ctx, req)
		}

		requestID := RequestIDFromContext(ctx)

		callLogger := logger.With(
			zap.String("request_id", requestID),
			zap.String("grpc.method", info.FullMethod),
		)

		key := GenerateCacheKey(ctx, cfg.KeyPrefix, info.FullMethod, req)

		cacheGetStart := time.Now()
		if err := cache.Get(ctx, key, cachedResponse); err == nil {
			callLogger.Debug(
				"cache hit",
				zap.String("cache_key", key),
				zap.Duration("cache_get_duration", time.Since(cacheGetStart)),
			)

			return cachedResponse, nil
		} else {
			callLogger.Debug(
				"cache miss",
				zap.String("cache_key", key),
				zap.Error(err),
			)

			if cfg.OnCacheMiss != nil {
				cfg.OnCacheMiss(cfg.ServiceName, info.FullMethod)
			}
		}

		value, err, shared := group.Do(key, func() (any, error) {
			originStart := time.Now()

			originResponse, callErr := handler(ctx, req)
			if callErr != nil {
				return nil, callErr
			}

			callLogger.Debug(
				"origin handler completed",
				zap.String("cache_key", key),
				zap.Duration("origin_duration", time.Since(originStart)),
			)

			return originResponse, nil
		})

		if err != nil {
			return nil, err
		}

		if shared {
			callLogger.Debug(
				"singleflight shared response",
				zap.String("cache_key", key),
			)
		}

		protoResponse, ok := value.(proto.Message)
		if !ok {
			callLogger.Warn(
				"handler response is not a protobuf message; skipping cache set",
				zap.String("cache_key", key),
			)
			return value, nil
		}

		cacheSetStart := time.Now()
		if err := cache.Set(ctx, key, protoResponse, cache.DefaultTTL()); err != nil {
			callLogger.Debug(
				"cache set failed",
				zap.String("cache_key", key),
				zap.Error(err),
			)

			return value, nil
		}

		callLogger.Debug(
			"response stored in cache",
			zap.String("cache_key", key),
			zap.Duration("cache_set_duration", time.Since(cacheSetStart)),
			zap.Duration("cache_ttl", cache.DefaultTTL()),
		)

		return value, nil
	}
}

func GenerateCacheKey(ctx context.Context, prefix string, method string, req any) string {
	hash := xxhash.New()

	_, _ = hash.Write([]byte(method))

	if roles, ok := UserRolesFromMetadata(ctx); ok && len(roles) > 0 {
		_, _ = hash.Write([]byte(strings.Join(roles, "|")))
	}

	if message, ok := req.(proto.Message); ok {
		data, err := proto.Marshal(message)
		if err != nil {
			_, _ = hash.WriteString(fmt.Sprintf("%#v", req))
		} else {
			_, _ = hash.Write(data)
		}
	} else {
		_, _ = hash.WriteString(fmt.Sprintf("%#v", req))
	}

	return fmt.Sprintf("%s:%x", prefix, hash.Sum64())
}
