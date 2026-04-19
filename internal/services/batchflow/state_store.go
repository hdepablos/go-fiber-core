package batchflow

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"go-fiber-core/internal/logger"

	"go.uber.org/zap"
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
	stateKeysRegistry := fmt.Sprintf("%s:state_keys", input.RedisKey)

	_ = s.cache.Del(ctx, batchesListKey, summaryKey, stateKeysRegistry)

	summaryPayload, err := json.Marshal(map[string]any{
		"summary":  summary,
		"metadata": metadata,
	})
	if err != nil {
		return "", err
	}
	if err := s.cache.SetBytes(ctx, summaryKey, summaryPayload, ttl); err != nil {
		logStateStoreError("initialize.set_summary", input, err, zap.String("summary_key", summaryKey))
		return "", err
	}

	for i, batch := range batches {
		batchKey := fmt.Sprintf("%s:batch:%06d", input.RedisKey, i)
		payload, err := json.Marshal(batch)
		if err != nil {
			return "", err
		}
		if err := s.cache.SetBytes(ctx, batchKey, payload, ttl); err != nil {
			logStateStoreError("initialize.set_batch", input, err, zap.String("batch_key", batchKey), zap.Int("batch_index", i))
			return "", err
		}
		if err := s.cache.RPush(ctx, batchesListKey, batchKey); err != nil {
			logStateStoreError("initialize.push_batch_key", input, err, zap.String("batches_list_key", batchesListKey), zap.String("batch_key", batchKey), zap.Int("batch_index", i))
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
			logStateStoreError("initialize.set_empty_batch", input, err, zap.String("batch_key", batchKey))
			return "", err
		}
		if err := s.cache.RPush(ctx, batchesListKey, batchKey); err != nil {
			logStateStoreError("initialize.push_empty_batch", input, err, zap.String("batches_list_key", batchesListKey), zap.String("batch_key", batchKey))
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
		logStateStoreError("load_summary.get", input, err, zap.String("summary_key", summaryKey))
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
		logger.LogRedisError("load_batch.lindex", err, zap.String("component", "batchflow_state_store"), zap.String("batches_list_key", batchesListKey), zap.Int("batch_index", batchIndex))
		return Batch{}, err
	}
	payload, err := s.cache.GetBytes(ctx, batchKey)
	if err != nil {
		logger.LogRedisError("load_batch.get", err, zap.String("component", "batchflow_state_store"), zap.String("batches_list_key", batchesListKey), zap.String("batch_key", batchKey), zap.Int("batch_index", batchIndex))
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
			logger.LogRedisError("cleanup.lrange_batches", err, zap.String("component", "batchflow_state_store"), zap.String("batches_list_key", batchesListKey), zap.String("redis_key", input.RedisKey), zap.Int64("parent_id", input.ParentID))
			return err
		}
		keysToDelete = append(keysToDelete, batchKeys...)
		keysToDelete = append(keysToDelete, batchesListKey)
	}

	runtimeKeys, err := s.cache.LRange(ctx, fmt.Sprintf("%s:runtime_keys", input.RedisKey), 0, -1)
	if err != nil {
		logStateStoreError("cleanup.lrange_runtime_keys", input, err, zap.String("runtime_registry_key", fmt.Sprintf("%s:runtime_keys", input.RedisKey)))
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

	stateKeys, err := s.cache.LRange(ctx, fmt.Sprintf("%s:state_keys", input.RedisKey), 0, -1)
	if err == nil {
		keysToDelete = append(keysToDelete, stateKeys...)
		keysToDelete = append(keysToDelete, fmt.Sprintf("%s:state_keys", input.RedisKey))
	}

	if err := s.cache.Del(ctx, keysToDelete...); err != nil {
		logStateStoreError("cleanup.del", input, err, zap.Int("keys_to_delete", len(keysToDelete)))
		return err
	}
	return nil
}

func (s *redisStateStore) SetCounter(ctx context.Context, key string, value int64, ttl time.Duration) error {
	if err := s.cache.SetString(ctx, key, fmt.Sprintf("%d", value), ttl); err != nil {
		logger.LogRedisError("set_counter.set", err, zap.String("component", "batchflow_state_store"), zap.String("counter_key", key), zap.Int64("value", value))
		return err
	}
	runKey := counterRunRegistryKey(key)
	if runKey != "" {
		if err := s.cache.RPush(ctx, runKey, key); err != nil {
			logger.LogRedisError("set_counter.push_registry", err, zap.String("component", "batchflow_state_store"), zap.String("counter_key", key), zap.String("registry_key", runKey))
			return err
		}
	}
	return nil
}

func (s *redisStateStore) IncrCounter(ctx context.Context, key string, delta int64, ttl time.Duration) (int64, error) {
	value, err := s.cache.IncrBy(ctx, key, delta)
	if err != nil {
		logger.LogRedisError("incr_counter.incrby", err, zap.String("component", "batchflow_state_store"), zap.String("counter_key", key), zap.Int64("delta", delta))
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
		logger.LogRedisError("get_counter.get", err, zap.String("component", "batchflow_state_store"), zap.String("counter_key", key))
		return 0, err
	}
	return parseInt64(raw)
}

func (s *redisStateStore) RegisterShards(ctx context.Context, input Input, totalShards int, ttl time.Duration) error {
	if totalShards <= 0 {
		totalShards = 1
	}
	totalKey := fmt.Sprintf("%s:shards:total", input.RedisKey)
	completedKey := fmt.Sprintf("%s:shards:completed", input.RedisKey)
	if err := s.cache.SetString(ctx, totalKey, fmt.Sprintf("%d", totalShards), ttl); err != nil {
		logStateStoreError("register_shards.set_total", input, err, zap.String("total_key", totalKey), zap.Int("total_shards", totalShards))
		return err
	}
	if err := s.cache.SetString(ctx, completedKey, "0", ttl); err != nil {
		logStateStoreError("register_shards.set_completed", input, err, zap.String("completed_key", completedKey), zap.Int("total_shards", totalShards))
		return err
	}
	if err := s.registerStateKey(ctx, input, totalKey, ttl); err != nil {
		return err
	}
	return s.registerStateKey(ctx, input, completedKey, ttl)
}

func (s *redisStateStore) CompleteShard(ctx context.Context, input Input, shardIndex int, totalShards int, ttl time.Duration) (ShardCompletion, error) {
	doneKey := fmt.Sprintf("%s:shard:%d:done", input.RedisKey, shardIndex)
	if err := s.registerStateKey(ctx, input, doneKey, ttl); err != nil {
		return ShardCompletion{}, err
	}
	isNew, err := s.cache.SetNXString(ctx, doneKey, "1", ttl)
	if err != nil {
		logStateStoreError("complete_shard.set_done", input, err, zap.String("done_key", doneKey), zap.Int("shard_index", shardIndex), zap.Int("total_shards", totalShards))
		return ShardCompletion{}, err
	}

	completedKey := fmt.Sprintf("%s:shards:completed", input.RedisKey)
	var completed int64
	if isNew {
		completed, err = s.cache.IncrBy(ctx, completedKey, 1)
		if err != nil {
			logStateStoreError("complete_shard.incr_completed", input, err, zap.String("completed_key", completedKey), zap.Int("shard_index", shardIndex), zap.Int("total_shards", totalShards))
			return ShardCompletion{}, err
		}
		if ttl > 0 {
			_ = s.cache.Expire(ctx, completedKey, ttl)
		}
	} else {
		completed, err = s.GetCounter(ctx, completedKey)
		if err != nil {
			logStateStoreError("complete_shard.get_completed", input, err, zap.String("completed_key", completedKey), zap.Int("shard_index", shardIndex), zap.Int("total_shards", totalShards))
			return ShardCompletion{}, err
		}
	}

	lockKey := fmt.Sprintf("%s:finalize_lock", input.RedisKey)
	if registerErr := s.registerStateKey(ctx, input, lockKey, ttl); registerErr != nil {
		return ShardCompletion{}, registerErr
	}
	shouldFinalize := false
	if completed >= int64(totalShards) {
		shouldFinalize, err = s.cache.SetNXString(ctx, lockKey, "1", ttl)
		if err != nil {
			logStateStoreError("complete_shard.set_finalize_lock", input, err, zap.String("lock_key", lockKey), zap.Int("shard_index", shardIndex), zap.Int64("completed_shards", completed), zap.Int("total_shards", totalShards))
			return ShardCompletion{}, err
		}
	}

	return ShardCompletion{
		CompletedShards: completed,
		ShouldFinalize:  shouldFinalize,
	}, nil
}

func (s *redisStateStore) registerStateKey(ctx context.Context, input Input, key string, ttl time.Duration) error {
	registryKey := fmt.Sprintf("%s:state_keys", input.RedisKey)
	if err := s.cache.RPush(ctx, registryKey, key); err != nil {
		logger.LogRedisError("register_state_key.push", err, zap.String("component", "batchflow_state_store"), zap.String("redis_key", input.RedisKey), zap.Int64("parent_id", input.ParentID), zap.String("registry_key", registryKey), zap.String("state_key", key))
		return err
	}
	if ttl > 0 {
		_ = s.cache.Expire(ctx, registryKey, ttl)
	}
	return nil
}

func logStateStoreError(operation string, input Input, err error, fields ...zap.Field) {
	baseFields := []zap.Field{
		zap.String("component", "batchflow_state_store"),
		zap.String("redis_key", input.RedisKey),
		zap.Int64("parent_id", input.ParentID),
	}
	logger.LogRedisError(operation, err, append(baseFields, fields...)...)
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
