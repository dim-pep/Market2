package cache

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/redis/go-redis/v9"
	"google.golang.org/protobuf/proto"
)

var ErrCacheMiss = errors.New("cache entry not found")

type RedisConfig struct {
	Host         string
	Port         int
	Password     string
	Database     int
	TTL          string
	DialTimeout  string
	ReadTimeout  string
	WriteTimeout string
	PoolTimeout  string
	PoolSize     int
	MinIdleConns int
}

type RedisMessageCache struct {
	client     *redis.Client
	defaultTTL time.Duration
}

func NewRedisMessageCache(cfg RedisConfig) (*RedisMessageCache, error) {
	var client *redis.Client

	connect := func() error {
		var err error
		client, err = openRedisClient(cfg)
		return err
	}

	retryPolicy := backoff.NewExponentialBackOff()
	retryPolicy.InitialInterval = 1 * time.Second
	retryPolicy.MaxInterval = 5 * time.Second
	retryPolicy.MaxElapsedTime = 10 * time.Second
	retryPolicy.RandomizationFactor = 0.5

	if err := backoff.Retry(connect, retryPolicy); err != nil {
		return nil, fmt.Errorf("connect to redis with retries: %w", err)
	}

	defaultTTL, err := time.ParseDuration(cfg.TTL)
	if err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("parse redis ttl %q: %w", cfg.TTL, err)
	}

	return &RedisMessageCache{
		client:     client,
		defaultTTL: defaultTTL,
	}, nil
}

func openRedisClient(cfg RedisConfig) (*redis.Client, error) {
	dialTimeout, err := parseDurationSetting("dial timeout", cfg.DialTimeout)
	if err != nil {
		return nil, err
	}

	readTimeout, err := parseDurationSetting("read timeout", cfg.ReadTimeout)
	if err != nil {
		return nil, err
	}

	writeTimeout, err := parseDurationSetting("write timeout", cfg.WriteTimeout)
	if err != nil {
		return nil, err
	}

	poolTimeout, err := parseDurationSetting("pool timeout", cfg.PoolTimeout)
	if err != nil {
		return nil, err
	}

	client := redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password:     cfg.Password,
		DB:           cfg.Database,
		DialTimeout:  dialTimeout,
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
		PoolSize:     cfg.PoolSize,
		PoolTimeout:  poolTimeout,
		MinIdleConns: cfg.MinIdleConns,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("ping redis: %w", err)
	}

	return client, nil
}

func parseDurationSetting(name string, value string) (time.Duration, error) {
	duration, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("parse redis %s %q: %w", name, value, err)
	}

	return duration, nil
}

func (c *RedisMessageCache) Get(ctx context.Context, key string, destination proto.Message) error {
	if c == nil || c.client == nil {
		return errors.New("redis cache is not initialized")
	}

	if err := ctx.Err(); err != nil {
		return err
	}

	data, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return fmt.Errorf("get cache key %q: %w", key, ErrCacheMiss)
		}

		return fmt.Errorf("get cache key %q: %w", key, err)
	}

	if err := proto.Unmarshal(data, destination); err != nil {
		return fmt.Errorf("unmarshal cache key %q: %w", key, err)
	}

	return nil
}

func (c *RedisMessageCache) Set(ctx context.Context, key string, value proto.Message, expiration time.Duration) error {
	if c == nil || c.client == nil {
		return errors.New("redis cache is not initialized")
	}

	if err := ctx.Err(); err != nil {
		return err
	}

	data, err := proto.Marshal(value)
	if err != nil {
		return fmt.Errorf("marshal cache key %q: %w", key, err)
	}

	if err := c.client.Set(ctx, key, data, expiration).Err(); err != nil {
		return fmt.Errorf("set cache key %q: %w", key, err)
	}

	return nil
}

func (c *RedisMessageCache) Delete(ctx context.Context, key string) error {
	if c == nil || c.client == nil {
		return errors.New("redis cache is not initialized")
	}

	if err := ctx.Err(); err != nil {
		return err
	}

	if err := c.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("delete cache key %q: %w", key, err)
	}

	return nil
}

func (c *RedisMessageCache) DefaultTTL() time.Duration {
	if c == nil {
		return 0
	}

	return c.defaultTTL
}

func (c *RedisMessageCache) HealthCheck(ctx context.Context) error {
	if c == nil || c.client == nil {
		return errors.New("redis cache is not initialized")
	}

	if err := c.client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("redis health check failed: %w", err)
	}

	return nil
}

func (c *RedisMessageCache) Close() error {
	if c == nil || c.client == nil {
		return nil
	}

	return c.client.Close()
}
