package batchflow

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

type RuntimeValuesProvider interface {
	RuntimeValues(input Input, ttl time.Duration) RuntimeValues
}

type redisRuntimeValues struct {
	cache Cache
	input Input
	ttl   time.Duration
}

func newRedisRuntimeValues(cache Cache, input Input, ttl time.Duration) RuntimeValues {
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	return &redisRuntimeValues{
		cache: cache,
		input: input,
		ttl:   ttl,
	}
}

func (r *redisRuntimeValues) Set(ctx context.Context, key string, value any) error {
	payload, err := json.Marshal(value)
	if err != nil {
		return err
	}
	composedKey := r.composeKey(key)
	if err := r.cache.SetBytes(ctx, composedKey, payload, r.ttl); err != nil {
		return err
	}
	return r.cache.RPush(ctx, r.registryKey(), composedKey)
}

func (r *redisRuntimeValues) Get(ctx context.Context, key string, dest any) error {
	payload, err := r.cache.GetBytes(ctx, r.composeKey(key))
	if err != nil {
		return err
	}
	return json.Unmarshal(payload, dest)
}

func (r *redisRuntimeValues) Delete(ctx context.Context, key string) error {
	return r.cache.Del(ctx, r.composeKey(key))
}

func (r *redisRuntimeValues) composeKey(key string) string {
	return fmt.Sprintf("%s:runtime:%s", r.input.RedisKey, key)
}

func (r *redisRuntimeValues) registryKey() string {
	return fmt.Sprintf("%s:runtime_keys", r.input.RedisKey)
}
