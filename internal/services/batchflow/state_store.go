package batchflow

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

type redisStateStore struct {
	cache Cache
}

func NewRedisStateStore(cache Cache) StateStore {
	return &redisStateStore{cache: cache}
}

func (s *redisStateStore) RuntimeValues(input Input, ttl time.Duration) RuntimeValues {
	return newRedisRuntimeValues(s.cache, input, ttl)
}

func (s *redisStateStore) Initialize(ctx context.Context, input Input, batches []Batch, summary Summary, metadata map[string]any, ttl time.Duration) (string, error) {
	batchesListKey := fmt.Sprintf("%s:batches", input.RedisKey)
	summaryKey := fmt.Sprintf("%s:summary", input.RedisKey)

	_ = s.cache.Del(ctx, batchesListKey, summaryKey)

	summaryPayload, err := json.Marshal(map[string]any{
		"summary":  summary,
		"metadata": metadata,
	})
	if err != nil {
		return "", err
	}
	if err := s.cache.SetBytes(ctx, summaryKey, summaryPayload, ttl); err != nil {
		return "", err
	}

	for i, batch := range batches {
		batchKey := fmt.Sprintf("%s:batch:%06d", input.RedisKey, i)
		payload, err := json.Marshal(batch)
		if err != nil {
			return "", err
		}
		if err := s.cache.SetBytes(ctx, batchKey, payload, ttl); err != nil {
			return "", err
		}
		if err := s.cache.RPush(ctx, batchesListKey, batchKey); err != nil {
			return "", err
		}
	}

	if len(batches) == 0 {
		batchKey := fmt.Sprintf("%s:batch:%06d", input.RedisKey, 0)
		payload, err := json.Marshal(Batch{Items: []json.RawMessage{}})
		if err != nil {
			return "", err
		}
		if err := s.cache.SetBytes(ctx, batchKey, payload, ttl); err != nil {
			return "", err
		}
		if err := s.cache.RPush(ctx, batchesListKey, batchKey); err != nil {
			return "", err
		}
	}

	_ = s.cache.Expire(ctx, batchesListKey, ttl)
	_ = s.cache.Expire(ctx, summaryKey, ttl)

	return batchesListKey, nil
}

func (s *redisStateStore) LoadSummary(ctx context.Context, input Input) (Summary, error) {
	summaryKey := fmt.Sprintf("%s:summary", input.RedisKey)
	payload, err := s.cache.GetBytes(ctx, summaryKey)
	if err != nil {
		return Summary{}, err
	}

	var raw struct {
		Summary Summary `json:"summary"`
	}
	if err := json.Unmarshal(payload, &raw); err != nil {
		return Summary{}, err
	}
	return raw.Summary, nil
}

func (s *redisStateStore) LoadBatch(ctx context.Context, batchesListKey string, batchIndex int) (Batch, error) {
	batchKey, err := s.cache.LIndex(ctx, batchesListKey, int64(batchIndex))
	if err != nil {
		return Batch{}, err
	}
	payload, err := s.cache.GetBytes(ctx, batchKey)
	if err != nil {
		return Batch{}, err
	}

	var batch Batch
	if err := json.Unmarshal(payload, &batch); err != nil {
		return Batch{}, err
	}
	return batch, nil
}

func (s *redisStateStore) Cleanup(ctx context.Context, input Input, batchesListKey string) error {
	keysToDelete := []string{
		fmt.Sprintf("%s:summary", input.RedisKey),
		fmt.Sprintf("%s:runtime_keys", input.RedisKey),
	}

	if batchesListKey != "" {
		batchKeys, err := s.cache.LRange(ctx, batchesListKey, 0, -1)
		if err != nil {
			return err
		}
		keysToDelete = append(keysToDelete, batchKeys...)
		keysToDelete = append(keysToDelete, batchesListKey)
	}

	runtimeKeys, err := s.cache.LRange(ctx, fmt.Sprintf("%s:runtime_keys", input.RedisKey), 0, -1)
	if err != nil {
		return err
	}
	keysToDelete = append(keysToDelete, runtimeKeys...)

	counterPrefix := fmt.Sprintf("%s:counter:", input.RedisKey)
	counterKeys, err := s.cache.LRange(ctx, fmt.Sprintf("%s:counter_keys", input.RedisKey), 0, -1)
	if err == nil {
		keysToDelete = append(keysToDelete, counterKeys...)
		keysToDelete = append(keysToDelete, fmt.Sprintf("%s:counter_keys", input.RedisKey))
	} else {
		_ = counterPrefix
	}

	return s.cache.Del(ctx, keysToDelete...)
}

func (s *redisStateStore) SetCounter(ctx context.Context, key string, value int64, ttl time.Duration) error {
	if err := s.cache.SetString(ctx, key, fmt.Sprintf("%d", value), ttl); err != nil {
		return err
	}
	runKey := counterRunRegistryKey(key)
	if runKey != "" {
		return s.cache.RPush(ctx, runKey, key)
	}
	return nil
}

func (s *redisStateStore) IncrCounter(ctx context.Context, key string, delta int64, ttl time.Duration) (int64, error) {
	value, err := s.cache.IncrBy(ctx, key, delta)
	if err != nil {
		return 0, err
	}
	if ttl > 0 {
		_ = s.cache.Expire(ctx, key, ttl)
	}
	runKey := counterRunRegistryKey(key)
	if runKey != "" {
		_ = s.cache.RPush(ctx, runKey, key)
	}
	return value, nil
}

func (s *redisStateStore) GetCounter(ctx context.Context, key string) (int64, error) {
	raw, err := s.cache.GetString(ctx, key)
	if err != nil {
		return 0, err
	}
	return parseInt64(raw)
}

func counterRunRegistryKey(key string) string {
	for i := 0; i < len(key); i++ {
		if key[i] != ':' {
			continue
		}
		return key[:i] + ":counter_keys"
	}
	return ""
}
