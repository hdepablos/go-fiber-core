package mqb1t

import (
	"context"
	"fmt"
	"os"
	"time"

	"go-fiber-core/internal/logger"
	"go-fiber-core/internal/services/dispatcher"
	"go-fiber-core/internal/services/serviceconfig"
	"go-fiber-core/internal/services/serviceconfig/contracts"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type OrganizeService struct {
	ctx         *contracts.ServiceContext
	servicePath string
	batchSize   int64
	table       string
	queueTarget string
}

func NewOrganizeService() contracts.Service {
	fmt.Println("si está llegando a la clase.... organize v.1.0.0")

	return &OrganizeService{
		batchSize: 50,
		table:     "multi_queue_batch_one_table",
	}
}

func (s *OrganizeService) Init(ctx *contracts.ServiceContext, servicePath string) {

	fmt.Println("si está llegando a la clase.... organize")
	s.ctx = ctx
	s.servicePath = servicePath

	if s.ctx != nil && s.ctx.CurrentStepConfig != nil {
		if v, ok := s.ctx.CurrentStepConfig["batch_size"]; ok {
			switch n := v.(type) {
			case int:
				s.batchSize = int64(n)
			case int64:
				s.batchSize = n
			case float64:
				s.batchSize = int64(n)
			}
		}
		if v, ok := s.ctx.CurrentStepConfig["table"]; ok {
			if str, ok := v.(string); ok && str != "" {
				s.table = str
			}
		}
		if v, ok := s.ctx.CurrentStepConfig["queue_target"]; ok {
			if str, ok := v.(string); ok && str != "" {
				s.queueTarget = str
			}
		}
	}
}

type bucketRangeRow struct {
	BucketID int64 `gorm:"column:bucket_id"`
	StartID  int64 `gorm:"column:start_id"`
	EndID    int64 `gorm:"column:end_id"`
	RowCount int64 `gorm:"column:row_count"`
}

func (s *OrganizeService) Execute() error {
	const baseLoggerName = "MultiQueueBatchProcessorOneTable"

	execCtx := context.Background()
	if s.ctx != nil && s.ctx.Ctx != nil {
		execCtx = s.ctx.Ctx
	}

	wd, _ := os.Getwd()
	logOutput := os.Getenv("LOG_OUTPUT")
	appEnv := os.Getenv("APP_ENV")

	base := logger.GetLoggerToFile(baseLoggerName, "pkg/logs/MultiQueueBatchProcessorOneTable.log")
	defer func() { _ = base.Sync() }()

	log := base.With(
		zap.String("component", "mqb1t"),
		zap.String("step", "organize"),
		zap.String("service_path", s.servicePath),
		zap.String("cwd", wd),
		zap.String("app_env", appEnv),
		zap.String("log_output", logOutput),
		zap.String("log_file", "pkg/logs/MultiQueueBatchProcessorOneTable.log"),
	)

	if s.batchSize <= 0 {
		s.batchSize = 50
	}
	if s.table == "" {
		s.table = "multi_queue_batch_one_table"
	}
	if s.table != "multi_queue_batch_one_table" {
		return fmt.Errorf("table not allowed: %s", s.table)
	}

	log.Info("starting",
		zap.String("table", s.table),
		zap.Int64("batch_size", s.batchSize),
		zap.String("queue_target", s.queueTarget),
	)

	db, rdb, err := getDeps(execCtx)
	if err != nil {
		log.Error("deps error", zap.Error(err))
		return err
	}
	log.Info("deps ready")

	runID := uuid.NewString()
	now := time.Now()
	log = log.With(zap.String("run_id", runID))
	log.Info("run initialized", zap.String("now", now.Format(time.RFC3339)))

	var totalPending int64
	if err := db.WithContext(execCtx).Table(s.table).Where("status = ?", "pending").Count(&totalPending).Error; err != nil {
		log.Error("count pending failed", zap.Error(err))
		return err
	}
	log.Info("counted pending", zap.Int64("total_pending", totalPending))
	if totalPending == 0 {
		if s.ctx != nil {
			s.ctx.SetInputValue("__stop_chain", true)
			s.ctx.SetResult(s.servicePath, contracts.StepResult{
				Status:  "completed",
				Message: "no pending records",
				Data: map[string]any{
					"run_id":        runID,
					"total_pending": 0,
					"total_batches": 0,
					"batch_size":    s.batchSize,
					"table":         s.table,
					"dispatched":    0,
					"queue_target":  s.queueTarget,
					"dispatched_at": now.Format(time.RFC3339),
				},
			})
		}
		log.Info("no pending records, stopping")
		return nil
	}

	query := fmt.Sprintf(`
		WITH base AS (
			SELECT id, ROW_NUMBER() OVER (ORDER BY id) AS rn
			FROM %s
			WHERE status = 'pending'
		),
		bucketed AS (
			SELECT id, ((rn - 1) / ?) + 1 AS bucket_id
			FROM base
		)
		SELECT bucket_id, MIN(id) AS start_id, MAX(id) AS end_id, COUNT(*) AS row_count
		FROM bucketed
		GROUP BY bucket_id
		ORDER BY bucket_id
	`, s.table)

	var buckets []bucketRangeRow
	if err := db.WithContext(execCtx).Raw(query, s.batchSize).Scan(&buckets).Error; err != nil {
		log.Error("bucket planning failed", zap.Error(err))
		return err
	}
	if len(buckets) == 0 {
		log.Info("no buckets generated, stopping")
		if s.ctx != nil {
			s.ctx.SetInputValue("__stop_chain", true)
		}
		return nil
	}
	log.Info("buckets planned",
		zap.Int("total_batches", len(buckets)),
		zap.Int64("first_start_id", buckets[0].StartID),
		zap.Int64("first_end_id", buckets[0].EndID),
	)

	projectPrefix := os.Getenv("APP_NAME")
	if projectPrefix == "" {
		projectPrefix = "go-fiber-core"
	}
	totalKey := fmt.Sprintf("%s:mqb1t:%s:total", projectPrefix, runID)
	doneKey := fmt.Sprintf("%s:mqb1t:%s:done", projectPrefix, runID)
	finalizeKey := fmt.Sprintf("%s:mqb1t:%s:finalized", projectPrefix, runID)
	startedAtKey := fmt.Sprintf("%s:mqb1t:%s:started_at_ms", projectPrefix, runID)

	ttl := 24 * time.Hour
	if err := rdb.Set(execCtx, totalKey, int64(len(buckets)), ttl).Err(); err != nil {
		log.Error("redis set total failed", zap.String("key", totalKey), zap.Error(err))
		return err
	}
	if err := rdb.Set(execCtx, doneKey, 0, ttl).Err(); err != nil {
		log.Error("redis set done failed", zap.String("key", doneKey), zap.Error(err))
		return err
	}
	if err := rdb.Set(execCtx, startedAtKey, now.UnixMilli(), ttl).Err(); err != nil {
		log.Error("redis set started_at failed", zap.String("key", startedAtKey), zap.Error(err))
		return err
	}
	_ = rdb.Del(execCtx, finalizeKey).Err()
	log.Info("redis initialized",
		zap.String("total_key", totalKey),
		zap.String("done_key", doneKey),
		zap.String("finalized_key", finalizeKey),
		zap.String("started_at_key", startedAtKey),
	)

	dispatched := 0
	updatedRows := int64(0)
	policy := contracts.ExecutionPolicy{
		QueueTarget: s.queueTarget,
	}

	loopStart := time.Now()
	log.Info("dispatch loop started",
		zap.Int("total_batches", len(buckets)),
		zap.Int64("batch_size", s.batchSize),
		zap.String("table", s.table),
		zap.String("queue_target", s.queueTarget),
	)

	for _, b := range buckets {
		res := db.WithContext(execCtx).Table(s.table).
			Where("status = ?", "pending").
			Where("id BETWEEN ? AND ?", b.StartID, b.EndID).
			Updates(map[string]any{
				"status":     "to_process",
				"run_id":     runID,
				"updated_at": now,
			})
		if res.Error != nil {
			log.Error("update to_process failed",
				zap.Int64("bucket_id", b.BucketID),
				zap.Int64("start_id", b.StartID),
				zap.Int64("end_id", b.EndID),
				zap.Error(res.Error),
			)
			return res.Error
		}
		updatedRows += res.RowsAffected

		input := map[string]any{
			"run_id":     runID,
			"bucket_id":  b.BucketID,
			"start_id":   b.StartID,
			"end_id":     b.EndID,
			"table":      s.table,
			"batch_size": s.batchSize,
		}

		stepCtx := contracts.NewServiceContextFromInput(context.Background(), input)
		if err := dispatcher.DefaultDispatcher.DispatchStep(execCtx, "test/mqb1t/process_batch", 2, policy, stepCtx); err != nil {
			log.Error("dispatch failed",
				zap.Int64("bucket_id", b.BucketID),
				zap.Error(err),
			)
			return err
		}
		dispatched++

		if b.BucketID == int64(len(buckets)) {
			log.Info("last batch detected",
				zap.Int64("bucket_id", b.BucketID),
				zap.Int64("start_id", b.StartID),
				zap.Int64("end_id", b.EndID),
				zap.Int("dispatched_total", dispatched),
				zap.Int64("dispatch_loop_ms", time.Since(loopStart).Milliseconds()),
			)
		}
	}
	log.Info("dispatch loop finished", zap.Int64("dispatch_loop_ms", time.Since(loopStart).Milliseconds()))

	var reservedToProcess int64
	if err := db.WithContext(execCtx).Table(s.table).
		Where("run_id = ? AND status = ?", runID, "to_process").
		Count(&reservedToProcess).Error; err != nil {
		log.Error("count reserved_to_process failed", zap.Error(err))
		return err
	}

	if s.ctx != nil {
		s.ctx.SetInputValue("__stop_chain", true)
		s.ctx.SetResult(s.servicePath, contracts.StepResult{
			Status:  "completed",
			Message: "batches dispatched",
			Data: map[string]any{
				"run_id":              runID,
				"table":               s.table,
				"batch_size":          s.batchSize,
				"total_pending":       totalPending,
				"total_batches":       len(buckets),
				"dispatched":          dispatched,
				"updated_rows":        updatedRows,
				"reserved_to_process": reservedToProcess,
				"queue_target":        s.queueTarget,
				"redis_total":         totalKey,
				"redis_done":          doneKey,
				"dispatched_at":       now.Format(time.RFC3339),
			},
		})
	}

	fmt.Printf("mqb1t dispatched run_id=%s table=%s batches=%d batch_size=%d\n", runID, s.table, len(buckets), s.batchSize)
	return nil
}

func init() {
	serviceconfig.Register("test/mqb1t/organize", NewOrganizeService)
}
