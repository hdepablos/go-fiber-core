package batchflow

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"go-fiber-core/internal/domain"
	"go-fiber-core/internal/logger"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

var (
	ErrRunCancelled = errors.New("ejecucion cancelada")

	reHexToken = regexp.MustCompile(`\b[0-9a-f]{8,}\b`)
	reDigits   = regexp.MustCompile(`\b\d+\b`)
)

const (
	RunStateRunning   = "running"
	RunStateCancelled = "cancelled"
	RunStateFailed    = "failed"
	RunStateCompleted = "completed"
)

type RunStatus struct {
	RunKey           string `json:"run_key"`
	ParentID         int64  `json:"parent_id,omitempty"`
	State            string `json:"state"`
	Cancelled        bool   `json:"cancelled"`
	Reason           string `json:"reason,omitempty"`
	RequestedBy      string `json:"requested_by,omitempty"`
	Source           string `json:"source,omitempty"`
	CancelledAt      string `json:"cancelled_at,omitempty"`
	LastError        string `json:"last_error,omitempty"`
	ErrorFingerprint string `json:"error_fingerprint,omitempty"`
	ErrorCount       int64  `json:"error_count,omitempty"`
	Threshold        int64  `json:"threshold,omitempty"`
}

type RunCancelRequest struct {
	RunKey      string
	ParentID    int64
	Reason      string
	RequestedBy string
	Source      string
	TTL         time.Duration
}

type RunErrorRecordRequest struct {
	RunKey    string
	ParentID  int64
	Component string
	Source    string
	Err       error
	TTL       time.Duration
}

type ErrorRecordResult struct {
	RunKey        string `json:"run_key"`
	Fingerprint   string `json:"fingerprint"`
	Count         int64  `json:"count"`
	Threshold     int64  `json:"threshold"`
	AutoCancelled bool   `json:"auto_cancelled"`
}

type RunControl struct {
	cache               Cache
	defaultTTL          time.Duration
	autoCancelThreshold int64
}

func NewRunControl(cache Cache, defaultTTL time.Duration) *RunControl {
	if defaultTTL <= 0 {
		defaultTTL = 24 * time.Hour
	}
	threshold := int64(25)
	if raw := strings.TrimSpace(os.Getenv("BATCHFLOW_AUTO_CANCEL_ERROR_THRESHOLD")); raw != "" {
		if parsed, err := strconv.ParseInt(raw, 10, 64); err == nil && parsed > 0 {
			threshold = parsed
		}
	}
	return &RunControl{
		cache:               cache,
		defaultTTL:          defaultTTL,
		autoCancelThreshold: threshold,
	}
}

func (c *RunControl) RegisterRun(ctx context.Context, input Input, ttl time.Duration) error {
	if c == nil || c.cache == nil || strings.TrimSpace(input.RedisKey) == "" || input.ParentID <= 0 {
		return nil
	}
	ttl = c.normalizeTTL(ttl)

	status := RunStatus{
		RunKey:    input.RedisKey,
		ParentID:  input.ParentID,
		State:     RunStateRunning,
		Cancelled: false,
	}
	if err := c.saveStatus(ctx, status, ttl); err != nil {
		return err
	}
	return c.cache.SetString(ctx, activeRunByParentKey(input.ParentID), input.RedisKey, ttl)
}

func (c *RunControl) ResolveRunKey(ctx context.Context, parentID int64) (string, error) {
	if c == nil || c.cache == nil {
		return "", domain.ErrInternal
	}
	if parentID <= 0 {
		return "", domain.ErrInvalidArgument
	}
	raw, err := c.cache.GetString(ctx, activeRunByParentKey(parentID))
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return "", domain.ErrNotFound
		}
		return "", err
	}
	if strings.TrimSpace(raw) == "" {
		return "", domain.ErrNotFound
	}
	return raw, nil
}

func (c *RunControl) Status(ctx context.Context, runKey string) (RunStatus, error) {
	if c == nil || c.cache == nil {
		return RunStatus{}, domain.ErrInternal
	}
	runKey = strings.TrimSpace(runKey)
	if runKey == "" {
		return RunStatus{}, domain.ErrInvalidArgument
	}
	raw, err := c.cache.GetBytes(ctx, runStatusKey(runKey))
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return RunStatus{RunKey: runKey}, nil
		}
		return RunStatus{}, err
	}
	var status RunStatus
	if err := json.Unmarshal(raw, &status); err != nil {
		return RunStatus{}, err
	}
	if status.RunKey == "" {
		status.RunKey = runKey
	}
	return status, nil
}

func (c *RunControl) IsCancelled(ctx context.Context, runKey string) (bool, RunStatus, error) {
	status, err := c.Status(ctx, runKey)
	if err != nil {
		return false, RunStatus{}, err
	}
	return status.Cancelled, status, nil
}

func (c *RunControl) Cancel(ctx context.Context, req RunCancelRequest) (RunStatus, error) {
	if c == nil || c.cache == nil {
		return RunStatus{}, domain.ErrInternal
	}

	runKey := strings.TrimSpace(req.RunKey)
	if runKey == "" && req.ParentID > 0 {
		resolved, err := c.ResolveRunKey(ctx, req.ParentID)
		if err != nil {
			return RunStatus{}, err
		}
		runKey = resolved
	}
	if runKey == "" {
		return RunStatus{}, domain.ErrInvalidArgument
	}

	ttl := c.normalizeTTL(req.TTL)
	status, err := c.Status(ctx, runKey)
	if err != nil {
		return RunStatus{}, err
	}
	if status.ParentID == 0 {
		status.ParentID = req.ParentID
	}
	status.RunKey = runKey
	status.State = RunStateCancelled
	status.Cancelled = true
	if strings.TrimSpace(req.Reason) != "" {
		status.Reason = strings.TrimSpace(req.Reason)
	} else if strings.TrimSpace(status.Reason) == "" {
		status.Reason = "manual_cancel"
	}
	status.RequestedBy = strings.TrimSpace(req.RequestedBy)
	status.Source = strings.TrimSpace(req.Source)
	status.CancelledAt = time.Now().UTC().Format(time.RFC3339)

	if err := c.saveStatus(ctx, status, ttl); err != nil {
		return RunStatus{}, err
	}
	if status.ParentID > 0 {
		_ = c.cache.SetString(ctx, activeRunByParentKey(status.ParentID), runKey, ttl)
	}

	logger.LogExecutionGuard(
		"run_cancel_requested",
		zap.String("run_key", runKey),
		zap.Int64("parent_id", status.ParentID),
		zap.String("reason", status.Reason),
		zap.String("requested_by", status.RequestedBy),
		zap.String("source", status.Source),
	)
	return status, nil
}

func (c *RunControl) MarkFailed(ctx context.Context, input Input, cause error) error {
	if c == nil || c.cache == nil || strings.TrimSpace(input.RedisKey) == "" {
		return nil
	}
	status, err := c.Status(ctx, input.RedisKey)
	if err != nil {
		return err
	}
	if status.State == RunStateCancelled {
		return nil
	}
	status.RunKey = input.RedisKey
	status.ParentID = input.ParentID
	status.State = RunStateFailed
	status.LastError = trimForMeta(cause)
	return c.saveStatus(ctx, status, c.defaultTTL)
}

func (c *RunControl) MarkCompleted(ctx context.Context, input Input) error {
	if c == nil || c.cache == nil || strings.TrimSpace(input.RedisKey) == "" {
		return nil
	}
	status, err := c.Status(ctx, input.RedisKey)
	if err != nil {
		return err
	}
	if status.State == RunStateCancelled {
		return nil
	}
	status.RunKey = input.RedisKey
	status.ParentID = input.ParentID
	status.State = RunStateCompleted
	return c.saveStatus(ctx, status, c.defaultTTL)
}

func (c *RunControl) RecordError(ctx context.Context, req RunErrorRecordRequest) (ErrorRecordResult, error) {
	if c == nil || c.cache == nil || req.Err == nil {
		return ErrorRecordResult{}, nil
	}
	runKey := strings.TrimSpace(req.RunKey)
	if runKey == "" && req.ParentID > 0 {
		resolved, err := c.ResolveRunKey(ctx, req.ParentID)
		if err != nil {
			return ErrorRecordResult{}, err
		}
		runKey = resolved
	}
	if runKey == "" {
		return ErrorRecordResult{}, nil
	}

	ttl := c.normalizeTTL(req.TTL)
	fingerprint := buildErrorFingerprint(req.Component, req.Err)
	countKey := runErrorCountKey(runKey, fingerprint)
	count, err := c.cache.IncrBy(ctx, countKey, 1)
	if err != nil {
		return ErrorRecordResult{}, err
	}
	if ttl > 0 {
		_ = c.cache.Expire(ctx, countKey, ttl)
	}
	_ = c.cache.RPush(ctx, runErrorRegistryKey(runKey), countKey)
	_ = c.cache.Expire(ctx, runErrorRegistryKey(runKey), ttl)

	result := ErrorRecordResult{
		RunKey:      runKey,
		Fingerprint: fingerprint,
		Count:       count,
		Threshold:   c.autoCancelThreshold,
	}
	if count < c.autoCancelThreshold {
		return result, nil
	}

	status, err := c.Cancel(ctx, RunCancelRequest{
		RunKey:      runKey,
		ParentID:    req.ParentID,
		Reason:      "auto_cancel_threshold",
		RequestedBy: "system",
		Source:      firstNonEmpty(req.Source, req.Component, "auto_threshold"),
		TTL:         ttl,
	})
	if err != nil {
		return result, err
	}
	status.ErrorFingerprint = fingerprint
	status.ErrorCount = count
	status.Threshold = c.autoCancelThreshold
	status.LastError = trimForMeta(req.Err)
	if saveErr := c.saveStatus(ctx, status, ttl); saveErr != nil {
		return result, saveErr
	}

	logger.LogExecutionGuard(
		"run_auto_cancel_threshold",
		zap.String("run_key", runKey),
		zap.Int64("parent_id", status.ParentID),
		zap.String("fingerprint", fingerprint),
		zap.Int64("error_count", count),
		zap.Int64("threshold", c.autoCancelThreshold),
		zap.String("component", req.Component),
		zap.String("source", req.Source),
		zap.Error(req.Err),
	)
	result.AutoCancelled = true
	return result, nil
}

func (c *RunControl) AcquireStopLock(ctx context.Context, runKey string, ttl time.Duration) (bool, error) {
	if c == nil || c.cache == nil || strings.TrimSpace(runKey) == "" {
		return false, nil
	}
	return c.cache.SetNXString(ctx, runStopLockKey(runKey), "1", c.normalizeTTL(ttl))
}

func (c *RunControl) normalizeTTL(ttl time.Duration) time.Duration {
	if ttl > 0 {
		return ttl
	}
	return c.defaultTTL
}

func (c *RunControl) saveStatus(ctx context.Context, status RunStatus, ttl time.Duration) error {
	payload, err := json.Marshal(status)
	if err != nil {
		return err
	}
	return c.cache.SetBytes(ctx, runStatusKey(status.RunKey), payload, c.normalizeTTL(ttl))
}

func runStatusKey(runKey string) string {
	return fmt.Sprintf("%s:control:status", runKey)
}

func runStopLockKey(runKey string) string {
	return fmt.Sprintf("%s:control:stop_lock", runKey)
}

func runErrorRegistryKey(runKey string) string {
	return fmt.Sprintf("%s:control:error_keys", runKey)
}

func runErrorCountKey(runKey string, fingerprint string) string {
	sum := sha1.Sum([]byte(fingerprint))
	return fmt.Sprintf("%s:control:error:%s", runKey, hex.EncodeToString(sum[:]))
}

func activeRunByParentKey(parentID int64) string {
	return fmt.Sprintf("batchflow:parent:%d:active_run", parentID)
}

func buildErrorFingerprint(component string, err error) string {
	raw := strings.ToLower(strings.TrimSpace(trimForMeta(err)))
	raw = reHexToken.ReplaceAllString(raw, ":token")
	raw = reDigits.ReplaceAllString(raw, ":n")
	raw = strings.Join(strings.Fields(raw), " ")
	if component != "" {
		return strings.ToLower(strings.TrimSpace(component)) + "|" + raw
	}
	return raw
}

func trimForMeta(err error) string {
	if err == nil {
		return ""
	}
	msg := strings.TrimSpace(err.Error())
	if len(msg) > 300 {
		return msg[:300]
	}
	return msg
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}
