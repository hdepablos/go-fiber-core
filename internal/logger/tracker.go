package logger

import (
	"runtime"
	"time"

	"go.uber.org/zap"
)

// MetricsTracker helps track performance and results for structured logging
type MetricsTracker struct {
	StartTime     time.Time
	ReadDuration  time.Duration
	WriteDuration time.Duration

	// Results tracking
	RecordsTotal   int
	RecordsSuccess int
	RecordsFailed  int
	ErrorDetails   []string
}

// NewMetricsTracker creates a new tracker with StartTime set to now
func NewMetricsTracker() *MetricsTracker {
	return &MetricsTracker{
		StartTime:    time.Now(),
		ErrorDetails: make([]string, 0),
	}
}

// StopRead marks the end of a read operation and accumulates duration
func (t *MetricsTracker) StopRead(start time.Time) {
	t.ReadDuration += time.Since(start)
}

// StopWrite marks the end of a write operation and accumulates duration
func (t *MetricsTracker) StopWrite(start time.Time) {
	t.WriteDuration += time.Since(start)
}

// AddError records a failure and its detail
func (t *MetricsTracker) AddError(detail string) {
	t.RecordsFailed++
	t.ErrorDetails = append(t.ErrorDetails, detail)
}

// IncrementSuccess increments the success counter
func (t *MetricsTracker) IncrementSuccess() {
	t.RecordsSuccess++
}

// Finish prepares the final log fields
func (t *MetricsTracker) Finish() []zap.Field {
	totalDuration := time.Since(t.StartTime)

	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	memoryUsedMB := float64(m.Alloc) / 1024 / 1024

	// Performance Object
	performance := map[string]interface{}{
		"total_duration_ms": totalDuration.Milliseconds(),
		"db_read_ms":        t.ReadDuration.Milliseconds(),
		"db_write_ms":       t.WriteDuration.Milliseconds(),
		"memory_used_mb":    memoryUsedMB,
		"goroutines":        runtime.NumGoroutine(),
	}

	// Results Object
	// Only include if there are relevant metrics
	results := map[string]interface{}{
		"records_total":   t.RecordsTotal,
		"records_success": t.RecordsSuccess,
		"records_failed":  t.RecordsFailed,
	}

	if len(t.ErrorDetails) > 0 {
		results["error_details"] = t.ErrorDetails
	}

	return []zap.Field{
		zap.Any("performance", performance),
		zap.Any("results", results),
	}
}

// Log logs the summary using the provided logger
func (t *MetricsTracker) Log(l *zap.Logger, msg string, fields ...zap.Field) {
	finalFields := append(fields, t.Finish()...)
	l.Info(msg, finalFields...)
}

// Marshaler implementation for Performance (optional optimization, map is easier for now)
