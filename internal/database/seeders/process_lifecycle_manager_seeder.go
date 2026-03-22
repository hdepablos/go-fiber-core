package seeders

import (
	"context"
	"fmt"
	"log/slog"

	pgx "github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ProcessLifecycleManagerSeeder seeds the process lifecycle tables and exercises
// the core PL/pgSQL functions for replication and promotion.
func ProcessLifecycleManagerSeeder(pool *pgxpool.Pool) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultSeederTimeout)
	defer cancel()

	logger := slog.Default().With("seeder", "process_lifecycle_manager")
	logger.Info("iniciando seeder de process_lifecycle_manager")

	if err := executeInTransaction(ctx, pool, func(ctx context.Context, tx pgx.Tx) error {
		for _, table := range []string{
			"process_version_history",
			"process_steps",
			"process_versions",
			"process_types",
		} {
			if err := truncateTable(ctx, tx, table); err != nil {
				return fmt.Errorf("truncate %s: %w", table, err)
			}
		}

		var processTypeID int64
		if err := tx.QueryRow(ctx,
			`INSERT INTO process_types (name, description, is_visible)
			VALUES ($1, $2, $3)
			RETURNING id`,
			"Order process lifecycle",
			"Base process type for order lifecycle testing",
			true,
		).Scan(&processTypeID); err != nil {
			return fmt.Errorf("insert process_types: %w", err)
		}

		var baseVersionID int64
		if err := tx.QueryRow(ctx,
			`INSERT INTO process_versions (process_type_id, version_number, status, operator_id)
            VALUES ($1, $2, $3, $4)
            RETURNING id`,
			processTypeID,
			1,
			"DRAFT",
			1,
		).Scan(&baseVersionID); err != nil {
			return fmt.Errorf("insert process_versions (TEST): %w", err)
		}

		type stepDef struct {
			Order        int
			Name         string
			ExecutionKey string
			Config       string
		}

		steps := []stepDef{
			{Order: 1, Name: "Validate input", ExecutionKey: "validate_input", Config: "{}"},
			{Order: 2, Name: "Apply business rules", ExecutionKey: "apply_business_rules", Config: "{}"},
			{Order: 3, Name: "Persist results", ExecutionKey: "persist_results", Config: "{}"},
		}

		for _, s := range steps {
			if _, err := tx.Exec(ctx,
				`INSERT INTO process_steps (process_version_id, step_order, name, execution_key, config, roadmap)
                VALUES ($1, $2, $3, $4, $5::jsonb, 0)`,
				baseVersionID,
				s.Order,
				s.Name,
				s.ExecutionKey,
				s.Config,
			); err != nil {
				return fmt.Errorf("insert process_steps (order=%d): %w", s.Order, err)
			}
		}

		// =================================================================
		// Caso 1: Ejecución 1x1 (Secuencial Simple)
		// =================================================================
		var case1ProcessTypeID int64
		if err := tx.QueryRow(ctx,
			`INSERT INTO process_types (name, description, is_visible) VALUES ($1, $2, $3) RETURNING id`,
			"Case 1: Sequential Execution", "Caso de uso: Ejecución 1 a 1 de pasos simples (SYNC)", true,
		).Scan(&case1ProcessTypeID); err != nil {
			return fmt.Errorf("insert case 1 process_types: %w", err)
		}

		// Crear Versión DRAFT
		var case1VersionID int64
		if err := tx.QueryRow(ctx,
			`INSERT INTO process_versions (process_type_id, version_number, status, operator_id) VALUES ($1, $2, $3, $4) RETURNING id`,
			case1ProcessTypeID, 1, "DRAFT", 1,
		).Scan(&case1VersionID); err != nil {
			return fmt.Errorf("insert case 1 process_versions: %w", err)
		}

		// Steps Secuenciales (1, 2, 3)
		// Configuración mínima, default SYNC
		case1Steps := []struct {
			Order        int
			Name         string
			ExecutionKey string
			Config       string
		}{
			{1, "Step 1: Validate", "common/validate", `{"min_age": 18, "required_keys": ["age"]}`},
			{2, "Step 2: Calculate", "common/calculate", `{"factor": 1.5, "execution_policy": {"mode": "SYNC"}}`},
			{3, "Step 3: Notify", "common/notify", `{"channel": "email"}`},
		}

		for _, s := range case1Steps {
			if _, err := tx.Exec(ctx,
				`INSERT INTO process_steps (process_version_id, step_order, name, execution_key, config, roadmap) VALUES ($1, $2, $3, $4, $5::jsonb, 0)`,
				case1VersionID, s.Order, s.Name, s.ExecutionKey, s.Config,
			); err != nil {
				return fmt.Errorf("insert case 1 steps: %w", err)
			}
		}

		// =================================================================
		// Caso 2: Ejecución Paralela y Recursiva (Batching Async)
		// =================================================================
		var case2ProcessTypeID int64
		if err := tx.QueryRow(ctx,
			`INSERT INTO process_types (name, description, is_visible) VALUES ($1, $2, $3) RETURNING id`,
			"Case 2: Parallel Batch Processing", "Caso de uso: 4 Workers Async en Paralelo con Recursión (Auto-Invoke)", true,
		).Scan(&case2ProcessTypeID); err != nil {
			return fmt.Errorf("insert case 2 process_types: %w", err)
		}

		// Crear Versión DRAFT
		var case2VersionID int64
		if err := tx.QueryRow(ctx,
			`INSERT INTO process_versions (process_type_id, version_number, status, operator_id) VALUES ($1, $2, $3, $4) RETURNING id`,
			case2ProcessTypeID, 1, "DRAFT", 1,
		).Scan(&case2VersionID); err != nil {
			return fmt.Errorf("insert case 2 process_versions: %w", err)
		}

		// 4 Steps Paralelos (Orden 1) con Auto-Invoke
		// Simula 4 colas/workers procesando lotes distintos o particiones
		parallelConfig := `{
			"batch_size": 500,
			"required_keys": ["partition_id", "last_id_processed", "is_last_batch"],
			"execution_policy": {
				"mode": "ASYNC",
				"auto_invoke": {
					"enabled": true,
					"cursor_field": "last_id_processed",
					"stop_condition": "is_last_batch"
				}
			}
		}`

		for i := 1; i <= 4; i++ {
			if _, err := tx.Exec(ctx,
				`INSERT INTO process_steps (process_version_id, step_order, name, execution_key, config, roadmap) VALUES ($1, $2, $3, $4, $5::jsonb, 0)`,
				case2VersionID, 1, fmt.Sprintf("Parallel Worker %d", i), "batch/processor", parallelConfig,
			); err != nil {
				return fmt.Errorf("insert case 2 parallel steps: %w", err)
			}
		}

		// Step Final (Orden 2) - Se ejecuta cuando TODOS los paralelos terminen (o se despachen)
		finalConfig := `{"execution_policy": {"mode": "SYNC"}}`
		if _, err := tx.Exec(ctx,
			`INSERT INTO process_steps (process_version_id, step_order, name, execution_key, config, roadmap) VALUES ($1, $2, $3, $4, $5::jsonb, 0)`,
			case2VersionID, 2, "Final Consolidation", "batch/consolidate", finalConfig,
		); err != nil {
			return fmt.Errorf("insert case 2 final step: %w", err)
		}

		// =================================================================
		// Caso 3: Híbrido (Sync -> Async -> Sync)
		// =================================================================
		var case3ProcessTypeID int64
		if err := tx.QueryRow(ctx,
			`INSERT INTO process_types (name, description, is_visible) VALUES ($1, $2, $3) RETURNING id`,
			"Case 3: Hybrid Flow", "Caso de uso: Validación Sync -> Proceso Async -> Notificación Final", true,
		).Scan(&case3ProcessTypeID); err != nil {
			return fmt.Errorf("insert case 3 process_types: %w", err)
		}

		var case3VersionID int64
		if err := tx.QueryRow(ctx,
			`INSERT INTO process_versions (process_type_id, version_number, status, operator_id) VALUES ($1, $2, $3, $4) RETURNING id`,
			case3ProcessTypeID, 1, "DRAFT", 1,
		).Scan(&case3VersionID); err != nil {
			return fmt.Errorf("insert case 3 process_versions: %w", err)
		}

		case3Steps := []struct {
			Order        int
			Name         string
			ExecutionKey string
			Config       string
		}{
			{1, "Validación Rápida", "common/validate", `{"execution_policy": {"mode": "SYNC"}}`},
			{2, "Proceso Pesado (Cola)", "heavy/process", `{"execution_policy": {"mode": "ASYNC", "queue_target": "vip-queue"}}`},
			{3, "Notificación (Si termina)", "common/notify", `{"execution_policy": {"mode": "SYNC"}}`},
		}

		for _, s := range case3Steps {
			if _, err := tx.Exec(ctx,
				`INSERT INTO process_steps (process_version_id, step_order, name, execution_key, config, roadmap) VALUES ($1, $2, $3, $4, $5::jsonb, 0)`,
				case3VersionID, s.Order, s.Name, s.ExecutionKey, s.Config,
			); err != nil {
				return fmt.Errorf("insert case 3 steps: %w", err)
			}
		}

		var newVersionID int64
		if err := tx.QueryRow(ctx,
			`SELECT replicate_process_version($1, $2)`,
			baseVersionID,
			1,
		).Scan(&newVersionID); err != nil {
			return fmt.Errorf("replicate_process_version: %w", err)
		}

		var loanProcessTypeID int64
		if err := tx.QueryRow(ctx,
			`INSERT INTO process_types (name, description, is_visible)
			VALUES ($1, $2, $3)
			RETURNING id`,
			"Loan risk lifecycle",
			"Lifecycle configurado para servicios de loanrisk",
			true,
		).Scan(&loanProcessTypeID); err != nil {
			return fmt.Errorf("insert process_types loanrisk: %w", err)
		}

		var loanBaseVersionID int64
		if err := tx.QueryRow(ctx,
			`INSERT INTO process_versions (process_type_id, version_number, status, operator_id)
			VALUES ($1, $2, $3, $4)
			RETURNING id`,
			loanProcessTypeID,
			1,
			"DRAFT",
			1,
		).Scan(&loanBaseVersionID); err != nil {
			return fmt.Errorf("insert process_versions loanrisk: %w", err)
		}

		loanSteps := []stepDef{
			{
				Order:        1,
				Name:         "Age validation",
				ExecutionKey: "loanrisk/age",
				Config:       `{"error_tolerance":"inherit","required_keys":["age"],"min_age":40}`,
			},
			{
				Order:        2,
				Name:         "Special validation",
				ExecutionKey: "loanrisk/validation",
				Config:       `{"error_tolerance":"tolerable"}`,
			},
			{
				Order:        3,
				Name:         "Salary validation",
				ExecutionKey: "loanrisk/salary",
				Config:       `{"error_tolerance":"critical","required_keys":["salary"],"min_salary":2500000}`,
			},
			{
				Order:        4,
				Name:         "Renovation check",
				ExecutionKey: "loanrisk/is_renovation",
				Config:       `{"error_tolerance":"inherit","required_keys":["min_salary","salary_bracket_k_usd","salary_checked"]}`,
			},
			{
				Order:        5,
				Name:         "Risk level",
				ExecutionKey: "loanrisk/risk_level",
				Config:       `{"error_tolerance":"inherit","required_keys":["is_renovation"]}`,
			},
		}

		for _, s := range loanSteps {
			if _, err := tx.Exec(ctx,
				`INSERT INTO process_steps (process_version_id, step_order, name, execution_key, config, roadmap)
                VALUES ($1, $2, $3, $4, $5::jsonb, 0)`,
				loanBaseVersionID,
				s.Order,
				s.Name,
				s.ExecutionKey,
				s.Config,
			); err != nil {
				return fmt.Errorf("insert process_steps loanrisk (order=%d): %w", s.Order, err)
			}
		}

		var loanNewVersionID int64
		if err := tx.QueryRow(ctx,
			`SELECT replicate_process_version($1, $2)`,
			loanBaseVersionID,
			1,
		).Scan(&loanNewVersionID); err != nil {
			return fmt.Errorf("replicate_process_version loanrisk: %w", err)
		}

		return nil
	}); err != nil {
		logger.Error("seeder process_lifecycle_manager falló", "error", err)
		return err
	}

	logger.Info("seeder process_lifecycle_manager completado exitosamente")
	return nil
}
