package mqb1t

import (
	"fmt"
	"os"
	"time"

	"go-fiber-core/internal/logger"
	"go-fiber-core/internal/services/serviceconfig"
	"go-fiber-core/internal/services/serviceconfig/contracts"

	"go.uber.org/zap"
)

type FinalizeService struct {
	ctx         *contracts.ServiceContext
	servicePath string
}

func NewFinalizeService() contracts.Service {
	return &FinalizeService{}
}

func (s *FinalizeService) Init(ctx *contracts.ServiceContext, servicePath string) {
	s.ctx = ctx
	s.servicePath = servicePath
}

type statusCountRow struct {
	Status string `gorm:"column:status"`
	Count  int64  `gorm:"column:count"`
}

func (s *FinalizeService) Execute() error {
	const baseLoggerName = "MultiQueueBatchProcessorOneTable"

	db, rdb, err := getDeps(s.ctx.Ctx)
	if err != nil {
		return err
	}

	rawRunID, _ := s.ctx.GetInputValue("run_id")
	runID, _ := rawRunID.(string)
	if runID == "" {
		return fmt.Errorf("run_id missing")
	}

	wd, _ := os.Getwd()
	logOutput := os.Getenv("LOG_OUTPUT")
	appEnv := os.Getenv("APP_ENV")

	base := logger.GetLoggerToFile(baseLoggerName, "pkg/logs/MultiQueueBatchProcessorOneTable.log")
	defer func() { _ = base.Sync() }()

	log := base.With(
		zap.String("component", "mqb1t"),
		zap.String("step", "finalize"),
		zap.String("service_path", s.servicePath),
		zap.String("cwd", wd),
		zap.String("app_env", appEnv),
		zap.String("log_output", logOutput),
		zap.String("log_file", "pkg/logs/MultiQueueBatchProcessorOneTable.log"),
		zap.String("run_id", runID),
	)

	table := "multi_queue_batch_one_table"
	if v, ok := s.ctx.GetInputValue("table"); ok {
		if str, ok := v.(string); ok && str != "" {
			table = str
		}
	}
	if table != "multi_queue_batch_one_table" {
		return fmt.Errorf("table not allowed: %s", table)
	}

	var rows []statusCountRow
	if err := db.WithContext(s.ctx.Ctx).
		Table(table).
		Select("status, COUNT(*) AS count").
		Where("run_id = ?", runID).
		Group("status").
		Scan(&rows).Error; err != nil {
		return err
	}

	stats := make(map[string]int64, len(rows))
	var total int64
	for _, r := range rows {
		stats[r.Status] = r.Count
		total += r.Count
	}

	projectPrefix := os.Getenv("APP_NAME")
	if projectPrefix == "" {
		projectPrefix = "go-fiber-core"
	}
	totalKey := fmt.Sprintf("%s:mqb1t:%s:total", projectPrefix, runID)
	doneKey := fmt.Sprintf("%s:mqb1t:%s:done", projectPrefix, runID)
	finalizeKey := fmt.Sprintf("%s:mqb1t:%s:finalized", projectPrefix, runID)
	startedAtKey := fmt.Sprintf("%s:mqb1t:%s:started_at_ms", projectPrefix, runID)

	startedAtMs, err := rdb.Get(s.ctx.Ctx, startedAtKey).Int64()
	if err != nil {
		startedAtMs = 0
	}
	var durationMS int64
	if startedAtMs > 0 {
		durationMS = time.Since(time.UnixMilli(startedAtMs)).Milliseconds()
	}
	durationSeconds := float64(durationMS) / 1000.0

	_ = rdb.Del(s.ctx.Ctx, totalKey, doneKey, startedAtKey).Err()
	_ = rdb.Expire(s.ctx.Ctx, finalizeKey, 24*time.Hour).Err()

	log.Info("final stats",
		zap.String("table", table),
		zap.Int64("total_to_process", total),
		zap.Int64("total_processed", stats["processed"]),
		zap.Int64("total_processed_with_detail", stats["processed_with_details"]),
		zap.Float64("duration_seconds", durationSeconds),
		zap.Int64("duration_ms", durationMS),
		zap.String("duration", fmt.Sprintf("%.3fs", durationSeconds)),
	)

	fmt.Printf(
		"mqb1t finalize run_id=%s total_to_process=%d total_processed=%d total_processed_with_detail=%d duration=%.3fs duration_ms=%d\n",
		runID,
		total,
		stats["processed"],
		stats["processed_with_details"],
		durationSeconds,
		durationMS,
	)

	s.ctx.SetResult(s.servicePath, contracts.StepResult{
		Status: "completed",
		Data: map[string]any{
			"run_id":                 runID,
			"table":                  table,
			"total_to_process":       total,
			"total_processed":        stats["processed"],
			"total_processed_with_detail": stats["processed_with_details"],
			"duration_ms":            durationMS,
		},
	})
	return nil
}

func init() {
	serviceconfig.Register("test/mqb1t/finalize", NewFinalizeService)
}
