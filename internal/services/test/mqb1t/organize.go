package mqb1t

import (
	"context"
	"fmt"
	"os"
	"time"

	"go-fiber-core/internal/services/runtimectx"
	"go-fiber-core/internal/services/serviceconfig"
	"go-fiber-core/internal/services/serviceconfig/contracts"

	"github.com/google/uuid"
)

type organizeService struct {
	ctx         *contracts.ServiceContext
	servicePath string
	batchSize   int64
	table       string
	queueTarget string
}

func NewOrganizeService() contracts.Service {
	// Crea el servicio con valores por defecto. Estos pueden ser sobreescritos desde CurrentStepConfig en Init.
	return &organizeService{
		batchSize: 50,
		table:     "multi_queue_batch_one_table",
	}
}

func (s *organizeService) Init(ctx *contracts.ServiceContext, servicePath string) {
	// Inyecta el contexto de ejecución (incluye config del step) y el path del servicio.
	s.ctx = ctx
	s.servicePath = servicePath

	// Lee configuración dinámica del step (si existe) para parametrizar la ejecución.
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

func (s *organizeService) Execute() error {
	// Este step:
	// 1) Cuenta registros en "pending"
	// 2) Planifica buckets (rangos de IDs) por batch_size
	// 3) Reserva registros cambiando status -> to_process y asignando run_id
	// 4) Despacha un mensaje por bucket al step process_batch
	execCtx := context.Background()
	if s.ctx != nil && s.ctx.Ctx != nil {
		execCtx = s.ctx.Ctx
	}

	// Validaciones básicas de configuración.
	if s.batchSize <= 0 {
		s.batchSize = 50
	}
	if s.table == "" {
		s.table = "multi_queue_batch_one_table"
	}
	if s.table != "multi_queue_batch_one_table" {
		return fmt.Errorf("table not allowed: %s", s.table)
	}

	// Dependencias compartidas del flujo (DB write + Redis).
	db, rdb, err := getDeps(execCtx)
	if err != nil {
		return err
	}

	// run_id identifica una corrida completa del proceso y se propaga a todos los buckets.
	runID := uuid.NewString()
	now := time.Now()

	// Si no hay registros pending, se corta la cadena (no hay nada para procesar).
	var totalPending int64
	if err := db.WithContext(execCtx).Table(s.table).Where("status = ?", "pending").Count(&totalPending).Error; err != nil {
		return err
	}
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
		return nil
	}

	// Plan de buckets: agrupa IDs en bloques de batch_size preservando orden.
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
		return err
	}
	if len(buckets) == 0 {
		if s.ctx != nil {
			s.ctx.SetInputValue("__stop_chain", true)
		}
		return nil
	}

	// Keys en Redis para coordinar:
	// - total: total de buckets
	// - done: contador de buckets procesados
	// - finalized: lock para que finalize se dispare una sola vez
	// - started_at_ms: timestamp para calcular duración total al finalizar
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
		return fmt.Errorf("redis set total failed (key=%s): %w", totalKey, err)
	}
	if err := rdb.Set(execCtx, doneKey, 0, ttl).Err(); err != nil {
		return fmt.Errorf("redis set done failed (key=%s): %w", doneKey, err)
	}
	if err := rdb.Set(execCtx, startedAtKey, now.UnixMilli(), ttl).Err(); err != nil {
		return fmt.Errorf("redis set started_at failed (key=%s): %w", startedAtKey, err)
	}
	_ = rdb.Del(execCtx, finalizeKey).Err()

	dispatched := 0
	updatedRows := int64(0)
	policy := contracts.ExecutionPolicy{
		QueueTarget: s.queueTarget,
	}

	loopStart := time.Now()
	var lastBucketStartID int64
	var lastBucketEndID int64

	for _, b := range buckets {
		// Reserva el bucket: cambia status -> to_process y asigna run_id.
		res := db.WithContext(execCtx).Table(s.table).
			Where("status = ?", "pending").
			Where("id BETWEEN ? AND ?", b.StartID, b.EndID).
			Updates(map[string]any{
				"status":     "to_process",
				"run_id":     runID,
				"updated_at": now,
			})
		if res.Error != nil {
			return fmt.Errorf("update to_process failed (bucket_id=%d start_id=%d end_id=%d): %w", b.BucketID, b.StartID, b.EndID, res.Error)
		}
		updatedRows += res.RowsAffected

		// Payload por bucket: define el rango que deberá procesar el worker.
		input := map[string]any{
			"run_id":     runID,
			"bucket_id":  b.BucketID,
			"start_id":   b.StartID,
			"end_id":     b.EndID,
			"table":      s.table,
			"batch_size": s.batchSize,
		}

		stepCtx := contracts.NewServiceContextFromInput(context.Background(), input)
		dispatcherSvc, ok := runtimectx.Dispatcher(s.ctx.Ctx)
		if !ok {
			return fmt.Errorf("dispatcher no disponible en contexto")
		}
		if err := dispatcherSvc.DispatchStep(execCtx, "test/mqb1t/process_batch", 2, policy, nil, stepCtx); err != nil {
			return fmt.Errorf("dispatch failed (bucket_id=%d): %w", b.BucketID, err)
		}
		dispatched++

		if b.BucketID == int64(len(buckets)) {
			lastBucketStartID = b.StartID
			lastBucketEndID = b.EndID
		}
	}
	dispatchLoopMS := time.Since(loopStart).Milliseconds()

	var reservedToProcess int64
	if err := db.WithContext(execCtx).Table(s.table).
		Where("run_id = ? AND status = ?", runID, "to_process").
		Count(&reservedToProcess).Error; err != nil {
		return err
	}

	if s.ctx != nil {
		s.ctx.SetInputValue("__stop_chain", true)
		s.ctx.SetResult(s.servicePath, contracts.StepResult{
			Status:  "completed",
			Message: "batches dispatched",
			Data: map[string]any{
				"run_id":               runID,
				"table":                s.table,
				"batch_size":           s.batchSize,
				"total_pending":        totalPending,
				"total_batches":        len(buckets),
				"dispatched":           dispatched,
				"updated_rows":         updatedRows,
				"reserved_to_process":  reservedToProcess,
				"queue_target":         s.queueTarget,
				"redis_total":          totalKey,
				"redis_done":           doneKey,
				"dispatched_at":        now.Format(time.RFC3339),
				"dispatch_loop_ms":     dispatchLoopMS,
				"last_bucket_start_id": lastBucketStartID,
				"last_bucket_end_id":   lastBucketEndID,
			},
		})
	}

	return nil
}

func init() {
	serviceconfig.Register("test/mqb1t/organize", NewOrganizeService)
}
