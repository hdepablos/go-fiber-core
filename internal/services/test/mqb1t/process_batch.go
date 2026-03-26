package mqb1t

import (
	"context"
	"fmt"
	"os"
	"time"

	"go-fiber-core/internal/services/dispatcher"
	"go-fiber-core/internal/services/serviceconfig"
	"go-fiber-core/internal/services/serviceconfig/contracts"

	"github.com/redis/go-redis/v9"
)

type ProcessBatchService struct {
	ctx         *contracts.ServiceContext
	servicePath string
}

func NewProcessBatchService() contracts.Service {
	return &ProcessBatchService{}
}

func (s *ProcessBatchService) Init(ctx *contracts.ServiceContext, servicePath string) {
	s.ctx = ctx
	s.servicePath = servicePath
}

func getInt64(v any) int64 {
	switch n := v.(type) {
	case int:
		return int64(n)
	case int64:
		return n
	case float64:
		return int64(n)
	case float32:
		return int64(n)
	default:
		return 0
	}
}

func (s *ProcessBatchService) Execute() error {
	// 1) Dependencias: base de datos (para actualizar estados) y Redis (para coordinar batches).
	db, rdb, err := getDeps(s.ctx.Ctx)
	if err != nil {
		return err
	}

	// 2) run_id: identifica una corrida completa del proceso (todas las filas reservadas por organize.go).
	rawRunID, _ := s.ctx.GetInputValue("run_id")
	runID, _ := rawRunID.(string)
	if runID == "" {
		return fmt.Errorf("run_id missing")
	}

	// 3) Rango del lote: cada mensaje representa un bucket con [start_id, end_id] reservado.
	startID := getInt64(mustGet(s.ctx, "start_id"))
	endID := getInt64(mustGet(s.ctx, "end_id"))
	if startID <= 0 || endID <= 0 {
		return fmt.Errorf("invalid batch range start_id=%d end_id=%d", startID, endID)
	}

	// 4) batch_size: viene del payload solo para trazabilidad (no controla la query del update).
	batchSize := getInt64(mustGet(s.ctx, "batch_size"))

	// 5) Tabla target: acotado por seguridad (evita updates arbitrarios).
	table := "multi_queue_batch_one_table"
	if v, ok := s.ctx.GetInputValue("table"); ok {
		if str, ok := v.(string); ok && str != "" {
			table = str
		}
	}
	if table != "multi_queue_batch_one_table" {
		return fmt.Errorf("table not allowed: %s", table)
	}

	now := time.Now()

	// 6) Procesamiento "con detalle": marca un subconjunto determinístico del lote.
	//    Regla: MOD(id, 10) = 0  → aproximadamente 10% de los IDs (múltiplos de 10).
	//    Esto NO es aleatorio. Es reproducible para una misma distribución de IDs.
	withDetails := db.WithContext(s.ctx.Ctx).Table(table).
		Where("run_id = ? AND status = ? AND id BETWEEN ? AND ? AND MOD(id, 10) = 0", runID, "to_process", startID, endID).
		Updates(map[string]any{
			"status":     "processed_with_details",
			"detail":     "detail",
			"updated_at": now,
		})
	if withDetails.Error != nil {
		return withDetails.Error
	}

	// 7) Procesamiento normal: el resto del lote (IDs que NO cumplen la regla anterior).
	processed := db.WithContext(s.ctx.Ctx).Table(table).
		Where("run_id = ? AND status = ? AND id BETWEEN ? AND ? AND MOD(id, 10) <> 0", runID, "to_process", startID, endID).
		Updates(map[string]any{
			"status":     "processed",
			"detail":     nil,
			"updated_at": now,
		})
	if processed.Error != nil {
		return processed.Error
	}

	// 8) Coordinación en Redis:
	//    - totalKey: total de batches que organizó organize.go (len(buckets)).
	//    - doneKey: contador atómico (INCR) de batches completados.
	//    - finalizeKey: lock NX para disparar finalize solo una vez.
	projectPrefix := os.Getenv("APP_NAME")
	if projectPrefix == "" {
		projectPrefix = "go-fiber-core"
	}
	totalKey := fmt.Sprintf("%s:mqb1t:%s:total", projectPrefix, runID)
	doneKey := fmt.Sprintf("%s:mqb1t:%s:done", projectPrefix, runID)
	finalizeKey := fmt.Sprintf("%s:mqb1t:%s:finalized", projectPrefix, runID)

	done, err := rdb.Incr(s.ctx.Ctx, doneKey).Result()
	if err != nil {
		return err
	}
	total, err := rdb.Get(s.ctx.Ctx, totalKey).Int64()
	if err != nil {
		return err
	}

	// 9) Detección de último lote: cuando done == total.
	isLast := done == total && total > 0
	if isLast {
		// 10) Lock NX: evita que múltiples workers disparen finalize al mismo tiempo.
		status, err := rdb.SetArgs(s.ctx.Ctx, finalizeKey, "1", redis.SetArgs{Mode: "NX", TTL: 24 * time.Hour}).Result()
		if err != nil && err != redis.Nil {
			return err
		}
		if status == "OK" {
			// 11) Disparar finalize: calcula totales y duración completa (desde started_at_ms en Redis).
			input := map[string]any{
				"run_id": runID,
				"table":  table,
			}
			stepCtx := contracts.NewServiceContextFromInput(context.Background(), input)
			policy := contracts.ExecutionPolicy{}
			if err := dispatcher.DefaultDispatcher.DispatchStep(s.ctx.Ctx, "test/mqb1t/finalize", 3, policy, stepCtx); err != nil {
				return err
			}
		}
	}

	// 12) Resultado del step: información útil para debugging/observabilidad.
	s.ctx.SetResult(s.servicePath, contracts.StepResult{
		Status: "completed",
		Data: map[string]any{
			"run_id":                      runID,
			"batch_size":                  batchSize,
			"start_id":                    startID,
			"end_id":                      endID,
			"processed_rows":              processed.RowsAffected,
			"processed_with_details_rows": withDetails.RowsAffected,
			"done_batches":                done,
			"total_batches":               total,
			"is_last_batch":               isLast,
		},
	})

	if isLast {
		fmt.Printf("mqb1t last batch run_id=%s range=%d-%d done=%d/%d\n", runID, startID, endID, done, total)
	}
	return nil
}

func mustGet(ctx *contracts.ServiceContext, key string) any {
	v, _ := ctx.GetInputValue(key)
	return v
}

func init() {
	serviceconfig.Register("test/mqb1t/process_batch", NewProcessBatchService)
}
