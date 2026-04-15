package exportmanager

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

type StateStore interface {
	Initialize(ctx context.Context, input Input, batches []Batch, summary Summary, metadata map[string]any, ttl time.Duration) (batchesListKey string, partsListKey string, err error)
	LoadSummary(ctx context.Context, input Input) (Summary, error)
	LoadBatch(ctx context.Context, batchesListKey string, batchIndex int) (Batch, error)
	AppendPartKey(ctx context.Context, partsListKey string, partKey string) error
	LoadPartKeys(ctx context.Context, partsListKey string) ([]string, error)
	Cleanup(ctx context.Context, input Input, batchesListKey string, partsListKey string) error
}

type redisStateStore struct {
	cache Cache
}

func NewRedisStateStore(cache Cache) StateStore {
	return &redisStateStore{cache: cache}
}

func (s *redisStateStore) RuntimeValues(input Input, ttl time.Duration) RuntimeValues {
	return newRedisRuntimeValues(s.cache, input, ttl)
}

func (s *redisStateStore) Initialize(ctx context.Context, input Input, batches []Batch, summary Summary, metadata map[string]any, ttl time.Duration) (string, string, error) {
	batchesListKey := fmt.Sprintf("%s:batches", input.RedisKey)
	partsListKey := fmt.Sprintf("%s:parts", input.RedisKey)
	summaryKey := fmt.Sprintf("%s:summary", input.RedisKey)

	_ = s.cache.Del(ctx, batchesListKey, partsListKey, summaryKey)

	summaryPayload, err := json.Marshal(map[string]any{
		"summary":  summary,
		"metadata": metadata,
	})
	if err != nil {
		return "", "", err
	}
	if err := s.cache.SetBytes(ctx, summaryKey, summaryPayload, ttl); err != nil {
		return "", "", err
	}

	for i, batch := range batches {
		batchKey := fmt.Sprintf("%s:batch:%06d", input.RedisKey, i)
		payload, err := json.Marshal(batch)
		if err != nil {
			return "", "", err
		}
		if err := s.cache.SetBytes(ctx, batchKey, payload, ttl); err != nil {
			return "", "", err
		}
		if err := s.cache.RPush(ctx, batchesListKey, batchKey); err != nil {
			return "", "", err
		}
	}

	if len(batches) == 0 {
		batchKey := fmt.Sprintf("%s:batch:%06d", input.RedisKey, 0)
		payload, err := json.Marshal(Batch{Items: []json.RawMessage{}})
		if err != nil {
			return "", "", err
		}
		if err := s.cache.SetBytes(ctx, batchKey, payload, ttl); err != nil {
			return "", "", err
		}
		if err := s.cache.RPush(ctx, batchesListKey, batchKey); err != nil {
			return "", "", err
		}
	}

	_ = s.cache.Expire(ctx, batchesListKey, ttl)
	_ = s.cache.Expire(ctx, partsListKey, ttl)
	_ = s.cache.Expire(ctx, summaryKey, ttl)

	return batchesListKey, partsListKey, nil
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

func (s *redisStateStore) AppendPartKey(ctx context.Context, partsListKey string, partKey string) error {
	return s.cache.RPush(ctx, partsListKey, partKey)
}

func (s *redisStateStore) LoadPartKeys(ctx context.Context, partsListKey string) ([]string, error) {
	return s.cache.LRange(ctx, partsListKey, 0, -1)
}

func (s *redisStateStore) Cleanup(ctx context.Context, input Input, batchesListKey string, partsListKey string) error {
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

	if partsListKey != "" {
		keysToDelete = append(keysToDelete, partsListKey)
	}

	runtimeKeys, err := s.cache.LRange(ctx, fmt.Sprintf("%s:runtime_keys", input.RedisKey), 0, -1)
	if err != nil {
		return err
	}
	keysToDelete = append(keysToDelete, runtimeKeys...)

	return s.cache.Del(ctx, keysToDelete...)
}
