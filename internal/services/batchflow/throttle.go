package batchflow

import (
	"context"
	"fmt"
	"time"
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
		return fmt.Errorf("throttle cooldown activo para %s: %s", cfg.Key, ttl)
	}

	windowTTL := time.Duration(cfg.PerSeconds) * time.Second
	windowKey := fmt.Sprintf("throttle:%s:window", cfg.Key)
	current, err := c.cache.IncrBy(ctx, windowKey, 1)
	if err != nil {
		return err
	}
	_ = c.cache.Expire(ctx, windowKey, windowTTL)

	if cfg.MaxInFlight > 0 {
		inFlightKey := fmt.Sprintf("throttle:%s:inflight", cfg.Key)
		inFlight, inFlightErr := c.cache.IncrBy(ctx, inFlightKey, 1)
		if inFlightErr != nil {
			return inFlightErr
		}
		_ = c.cache.Expire(ctx, inFlightKey, windowTTL)
		if inFlight > cfg.MaxInFlight {
			_, _ = c.cache.IncrBy(ctx, inFlightKey, -1)
			return fmt.Errorf("max_in_flight excedido para %s", cfg.Key)
		}
	}

	if current > cfg.MaxRequests {
		if cfg.CooldownSeconds > 0 {
			_ = c.cache.SetString(ctx, cooldownKey, "1", time.Duration(cfg.CooldownSeconds)*time.Second)
		}
		return fmt.Errorf("rate limit alcanzado para %s", cfg.Key)
	}
	return nil
}

func (c *redisThrottleCoordinator) Release(ctx context.Context, cfg ThrottleConfig) error {
	if !cfg.Enabled || cfg.Key == "" || cfg.MaxInFlight <= 0 {
		return nil
	}
	inFlightKey := fmt.Sprintf("throttle:%s:inflight", cfg.Key)
	_, err := c.cache.IncrBy(ctx, inFlightKey, -1)
	return err
}
