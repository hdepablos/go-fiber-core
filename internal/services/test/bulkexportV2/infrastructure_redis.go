package bulkexportv2

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisCache struct {
	client *redis.Client
}

func NewRedisCache(client *redis.Client) *RedisCache {
	return &RedisCache{client: client}
}

func (c *RedisCache) Del(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}
	return c.client.Del(ctx, keys...).Err()
}

func (c *RedisCache) Expire(ctx context.Context, key string, ttl time.Duration) error {
	return c.client.Expire(ctx, key, ttl).Err()
}

func (c *RedisCache) GetBytes(ctx context.Context, key string) ([]byte, error) {
	return c.client.Get(ctx, key).Bytes()
}

func (c *RedisCache) LIndex(ctx context.Context, key string, index int64) (string, error) {
	return c.client.LIndex(ctx, key, index).Result()
}

func (c *RedisCache) LRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	return c.client.LRange(ctx, key, start, stop).Result()
}

func (c *RedisCache) RPush(ctx context.Context, key string, values ...string) error {
	if len(values) == 0 {
		return nil
	}
	vals := make([]any, 0, len(values))
	for _, v := range values {
		vals = append(vals, v)
	}
	return c.client.RPush(ctx, key, vals...).Err()
}

func (c *RedisCache) SetBytes(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return c.client.Set(ctx, key, value, ttl).Err()
}
