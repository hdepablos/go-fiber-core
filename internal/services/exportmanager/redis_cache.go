package exportmanager

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type redisCache struct {
	client *redis.Client
}

func NewRedisCache(client *redis.Client) Cache {
	return &redisCache{client: client}
}

func (c *redisCache) Del(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}
	return c.client.Del(ctx, keys...).Err()
}

func (c *redisCache) Expire(ctx context.Context, key string, ttl time.Duration) error {
	return c.client.Expire(ctx, key, ttl).Err()
}

func (c *redisCache) GetBytes(ctx context.Context, key string) ([]byte, error) {
	return c.client.Get(ctx, key).Bytes()
}

func (c *redisCache) LIndex(ctx context.Context, key string, index int64) (string, error) {
	return c.client.LIndex(ctx, key, index).Result()
}

func (c *redisCache) LRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	return c.client.LRange(ctx, key, start, stop).Result()
}

func (c *redisCache) RPush(ctx context.Context, key string, values ...string) error {
	if len(values) == 0 {
		return nil
	}
	vals := make([]any, 0, len(values))
	for _, v := range values {
		vals = append(vals, v)
	}
	return c.client.RPush(ctx, key, vals...).Err()
}

func (c *redisCache) SetBytes(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return c.client.Set(ctx, key, value, ttl).Err()
}
