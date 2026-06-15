package redis_cache

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/dim-pep/Market2/spotservice/config"
	"github.com/dim-pep/Market2/spotservice/internal/errs"
	"github.com/redis/go-redis/v9"
	"google.golang.org/protobuf/proto"
)

type redisRepo struct {
	client *redis.Client
	ttl    time.Duration
}

func NewRedisRepo(cfg *config.Config) (*redisRepo, error) {
	var client *redis.Client

	connect := func() error {
		var err error
		client, err = openRedis(cfg)
		return err
	}

	bo := backoff.NewExponentialBackOff()
	bo.InitialInterval = 1 * time.Second
	bo.MaxInterval = 5 * time.Second
	bo.MaxElapsedTime = 10 * time.Second
	bo.RandomizationFactor = 0.5

	if err := backoff.Retry(connect, bo); err != nil {
		return nil, fmt.Errorf("failed to connect to redis after retries: %w", err)
	}

	ttl, err := time.ParseDuration(cfg.Redis.TTL)
	if err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("failed to parse redis ttl %q: %w", cfg.Redis.TTL, err)
	}

	return &redisRepo{
		client: client,
		ttl:    ttl,
	}, nil
}

func openRedis(cfg *config.Config) (*redis.Client, error) {
	dialTimeout, err := parseDuration("redis dial timeout", cfg.Redis.DialTimeout)
	if err != nil {
		return nil, err
	}

	readTimeout, err := parseDuration("redis read timeout", cfg.Redis.ReadTimeout)
	if err != nil {
		return nil, err
	}

	writeTimeout, err := parseDuration("redis write timeout", cfg.Redis.WriteTimeout)
	if err != nil {
		return nil, err
	}

	poolTimeout, err := parseDuration("redis pool timeout", cfg.Redis.PoolTimeout)
	if err != nil {
		return nil, err
	}

	client := redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port),
		Password:     cfg.Redis.Password,
		DB:           cfg.Redis.DbIndex,
		DialTimeout:  dialTimeout,
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
		PoolSize:     cfg.Redis.PoolSize,
		PoolTimeout:  poolTimeout,
		MinIdleConns: cfg.Redis.MinIdleConns,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("failed to ping redis: %w", err)
	}

	return client, nil
}

func parseDuration(name string, value string) (time.Duration, error) {
	duration, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("failed to parse %s %q: %w", name, value, err)
	}

	return duration, nil
}

func (rr *redisRepo) Get(ctx context.Context, key string, dst proto.Message) error {
	if rr == nil || rr.client == nil {
		return errs.ErrCacheNotConfigured
	}

	if err := ctx.Err(); err != nil {
		return err
	}

	data, err := rr.client.Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return fmt.Errorf("redis cache miss for key %q: %w", key, errs.ErrCacheMiss)
		}

		return fmt.Errorf("redis get failed for key %q: %w", key, err)
	}

	if err := proto.Unmarshal(data, dst); err != nil {
		return fmt.Errorf("failed to unmarshal cached value for key %q: %w", key, err)
	}

	return nil
}

func (rr *redisRepo) Set(ctx context.Context, key string, value proto.Message, ttl time.Duration) error {
	if rr == nil || rr.client == nil {
		return errs.ErrCacheNotConfigured
	}

	if err := ctx.Err(); err != nil {
		return err
	}

	data, err := proto.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal cache value for key %q: %w", key, err)
	}

	if err := rr.client.Set(ctx, key, data, ttl).Err(); err != nil {
		return fmt.Errorf("redis set failed for key %q: %w", key, err)
	}

	return nil
}

func (rr *redisRepo) Delete(ctx context.Context, key string) error {
	if rr == nil || rr.client == nil {
		return errs.ErrCacheNotConfigured
	}

	if err := ctx.Err(); err != nil {
		return err
	}

	if err := rr.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("redis delete failed for key %q: %w", key, err)
	}

	return nil
}

func (rr *redisRepo) DefaultTTL() time.Duration {
	if rr == nil {
		return 0
	}

	return rr.ttl
}

func (rr *redisRepo) CheckHealth() error {
	if rr == nil || rr.client == nil {
		return errs.ErrCacheNotConfigured
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return rr.client.Ping(ctx).Err()
}

func (rr *redisRepo) Close() error {
	if rr == nil || rr.client == nil {
		return nil
	}

	return rr.client.Close()
}
