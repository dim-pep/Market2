package cache

import (
	"context"
	"time"

	"google.golang.org/protobuf/proto"
)

type CacheRepo interface {
	Get(ctx context.Context, key string, destination proto.Message) error
	Set(ctx context.Context, key string, value proto.Message, expiration time.Duration) error
	Delete(ctx context.Context, key string) error
	DefaultTTL() time.Duration
	HealthCheck() error
	Close() error
}
