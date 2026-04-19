package batchflow

import (
	"context"
	"fmt"
	"time"

	"go-fiber-core/internal/logger"

	"go.uber.org/zap"
)

type ThrottleConfig struct {
	Enabled         bool   `json:"enabled"`
	Key             string `json:"key"`
	MaxRequests     int64  `json:"max_requests"`
	PerSeconds      int64  `json:"per_seconds"`
	CooldownSeconds int64  `json:"cooldown_seconds"`
	MaxInFlight     int64  `json:"max_in_flight"`
}

type ThrottleCoordinator interface {
	Acquire(ctx context.Context, cfg ThrottleConfig) error
	Release(ctx context.Context, cfg ThrottleConfig) error
}

type redisThrottleCoordinator struct {
	cache Cache
}

func NewThrottleCoordinator(cache Cache) ThrottleCoordinator {
	return &redisThrottleCoordinator{cache: cache}
}

func (c *redisThrottleCoordinator) Acquire(ctx context.Context, cfg ThrottleConfig) error {
	if !cfg.Enabled || cfg.Key == "" || cfg.MaxRequests <= 0 || cfg.PerSeconds <= 0 {
		return nil
	}

	cooldownKey := fmt.Sprintf("throttle:%s:cooldown", cfg.Key)
	if ttl, err := c.cache.TTL(ctx, cooldownKey); err == nil && ttl > 0 {
		rateLimitErr := fmt.Errorf("throttle cooldown activo para %s: %s", cfg.Key, ttl)
		logger.LogInternalRateLimit(
			"internal_rate_limit_cooldown",
			rateLimitErr,
			zap.String("component", "batchflow_throttle"),
			zap.String("throttle_key", cfg.Key),
			zap.Duration("cooldown_ttl", ttl),
		)
		return rateLimitErr
	}

	windowTTL := time.Duration(cfg.PerSeconds) * time.Second
	windowKey := fmt.Sprintf("throttle:%s:window", cfg.Key)
	current, err := c.cache.IncrBy(ctx, windowKey, 1)
	if err != nil {
		logger.LogRedisError("throttle.acquire_window", err, zap.String("component", "batchflow_throttle"), zap.String("throttle_key", cfg.Key), zap.String("window_key", windowKey))
		return err
	}
	_ = c.cache.Expire(ctx, windowKey, windowTTL)

	if cfg.MaxInFlight > 0 {
		inFlightKey := fmt.Sprintf("throttle:%s:inflight", cfg.Key)
		inFlight, inFlightErr := c.cache.IncrBy(ctx, inFlightKey, 1)
		if inFlightErr != nil {
			logger.LogRedisError("throttle.acquire_inflight", inFlightErr, zap.String("component", "batchflow_throttle"), zap.String("throttle_key", cfg.Key), zap.String("inflight_key", inFlightKey))
			return inFlightErr
		}
		_ = c.cache.Expire(ctx, inFlightKey, windowTTL)
		if inFlight > cfg.MaxInFlight {
			_, _ = c.cache.IncrBy(ctx, inFlightKey, -1)
			rateLimitErr := fmt.Errorf("max_in_flight excedido para %s", cfg.Key)
			logger.LogInternalRateLimit(
				"internal_rate_limit_max_inflight",
				rateLimitErr,
				zap.String("component", "batchflow_throttle"),
				zap.String("throttle_key", cfg.Key),
				zap.Int64("max_in_flight", cfg.MaxInFlight),
				zap.Int64("current_in_flight", inFlight),
			)
			return rateLimitErr
		}
	}

	if current > cfg.MaxRequests {
		if cfg.CooldownSeconds > 0 {
			if setErr := c.cache.SetString(ctx, cooldownKey, "1", time.Duration(cfg.CooldownSeconds)*time.Second); setErr != nil {
				logger.LogRedisError("throttle.set_cooldown", setErr, zap.String("component", "batchflow_throttle"), zap.String("throttle_key", cfg.Key), zap.String("cooldown_key", cooldownKey))
			}
		}
		rateLimitErr := fmt.Errorf("rate limit alcanzado para %s", cfg.Key)
		logger.LogInternalRateLimit(
			"internal_rate_limit_window",
			rateLimitErr,
			zap.String("component", "batchflow_throttle"),
			zap.String("throttle_key", cfg.Key),
			zap.Int64("max_requests", cfg.MaxRequests),
			zap.Int64("current_requests", current),
			zap.Int64("per_seconds", cfg.PerSeconds),
		)
		return rateLimitErr
	}
	return nil
}

func (c *redisThrottleCoordinator) Release(ctx context.Context, cfg ThrottleConfig) error {
	if !cfg.Enabled || cfg.Key == "" || cfg.MaxInFlight <= 0 {
		return nil
	}
	inFlightKey := fmt.Sprintf("throttle:%s:inflight", cfg.Key)
	_, err := c.cache.IncrBy(ctx, inFlightKey, -1)
	if err != nil {
		logger.LogRedisError("throttle.release_inflight", err, zap.String("component", "batchflow_throttle"), zap.String("throttle_key", cfg.Key), zap.String("inflight_key", inFlightKey))
	}
	return err
}
