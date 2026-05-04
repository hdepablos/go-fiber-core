package mqb1t

import (
	"fmt"
	"os"
	"time"

	"go-fiber-core/internal/services/serviceconfig"
	"go-fiber-core/internal/services/serviceconfig/contracts"
)

type finalizeService struct {
	ctx         *contracts.ServiceContext
	servicePath string
}

func NewFinalizeService() contracts.Service {
	return &finalizeService{}
}

func (s *finalizeService) Init(ctx *contracts.ServiceContext, servicePath string) {
	s.ctx = ctx
	s.servicePath = servicePath
}

type statusCountRow struct {
	Status string `gorm:"column:status"`
	Count  int64  `gorm:"column:count"`
}

func (s *finalizeService) Execute() error {
	// Este step se ejecuta una sola vez al final del procesamiento:
	// - Calcula conteos por status dentro del run_id
	// - Calcula duración total usando started_at_ms (seteado por organize.go)
	// - Limpia keys de Redis usadas para coordinación
	db, rdb, err := getDeps(s.ctx.Ctx)
	if err != nil {
		return err
	}

	rawRunID, _ := s.ctx.GetInputValue("run_id")
	runID, _ := rawRunID.(string)
	if runID == "" {
		return fmt.Errorf("run_id missing")
	}

	// Seguridad: este flujo solo permite operar sobre la tabla esperada.
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

	// El resultado queda persistido en el ServiceContext para inspección desde el engine.
	s.ctx.SetResult(s.servicePath, contracts.StepResult{
		Status: "completed",
		Data: map[string]any{
			"run_id":                      runID,
			"table":                       table,
			"total_to_process":            total,
			"total_processed":             stats["processed"],
			"total_processed_with_detail": stats["processed_with_details"],
			"duration_seconds":            durationSeconds,
			"duration_ms":                 durationMS,
			"duration":                    fmt.Sprintf("%.3fs", durationSeconds),
		},
	})
	return nil
}

func init() {
	serviceconfig.Register("test/mqb1t/finalize", NewFinalizeService)
}
