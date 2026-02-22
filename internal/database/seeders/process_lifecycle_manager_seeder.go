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
			`INSERT INTO process_types (name, description)
             VALUES ($1, $2)
             RETURNING id`,
			"Order process lifecycle",
			"Base process type for order lifecycle testing",
		).Scan(&processTypeID); err != nil {
			return fmt.Errorf("insert process_types: %w", err)
		}

		var baseVersionID int64
		if err := tx.QueryRow(ctx,
			`INSERT INTO process_versions (process_type_id, version_number, status)
             VALUES ($1, $2, $3)
             RETURNING id`,
			processTypeID,
			1,
			"DRAFT",
		).Scan(&baseVersionID); err != nil {
			return fmt.Errorf("insert process_versions (TEST): %w", err)
		}

		type stepDef struct {
			Order        int
			Name         string
			ExecutionKey string
		}

		steps := []stepDef{
			{Order: 1, Name: "Validate input", ExecutionKey: "validate_input"},
			{Order: 2, Name: "Apply business rules", ExecutionKey: "apply_business_rules"},
			{Order: 3, Name: "Persist results", ExecutionKey: "persist_results"},
		}

		for _, s := range steps {
			if _, err := tx.Exec(ctx,
				`INSERT INTO process_steps (process_version_id, step_order, name, execution_key, config)
                 VALUES ($1, $2, $3, $4, $5::jsonb)`,
				baseVersionID,
				s.Order,
				s.Name,
				s.ExecutionKey,
				"{}",
			); err != nil {
				return fmt.Errorf("insert process_steps (order=%d): %w", s.Order, err)
			}
		}

		const operatorID int64 = 1
		if _, err := tx.Exec(ctx,
			`SELECT promote_process_version($1, $2, $3)`,
			baseVersionID,
			operatorID,
			"Initial promotion from seeder",
		); err != nil {
			return fmt.Errorf("promote_process_version: %w", err)
		}

		var newVersionID int64
		if err := tx.QueryRow(ctx,
			`SELECT replicate_process_version($1)`,
			baseVersionID,
		).Scan(&newVersionID); err != nil {
			return fmt.Errorf("replicate_process_version: %w", err)
		}

		return nil
	}); err != nil {
		logger.Error("seeder process_lifecycle_manager falló", "error", err)
		return err
	}

	logger.Info("seeder process_lifecycle_manager completado exitosamente")
	return nil
}
