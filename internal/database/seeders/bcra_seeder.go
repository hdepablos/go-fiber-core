package seeders

import (
	"context"
	"fmt"
	"log/slog"

	pgx "github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func ExportManagerBcraSeeder(pool *pgxpool.Pool) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultSeederTimeout)
	defer cancel()

	logger := slog.Default().With("seeder", "export_manager_bcra")
	logger.Info("iniciando seeder")

	return executeInTransaction(ctx, pool, func(ctx context.Context, tx pgx.Tx) error {
		const processTypeName = "generar archivo BCRA"
		const processDescription = "Scaffold de exportmanager generado automaticamente"

		var processTypeID int64
		err := tx.QueryRow(ctx, "SELECT id FROM process_types WHERE name = $1 AND archived_at IS NULL", processTypeName).Scan(&processTypeID)
		if err != nil {
			if err == pgx.ErrNoRows {
				if err := tx.QueryRow(ctx,
					`INSERT INTO process_types (name, description, is_visible)
					VALUES ($1, $2, $3)
					RETURNING id`,
					processTypeName,
					processDescription,
					true,
				).Scan(&processTypeID); err != nil {
					return fmt.Errorf("insert process_types '%s': %w", processTypeName, err)
				}
			} else {
				return fmt.Errorf("select process_types '%s': %w", processTypeName, err)
			}
		}

		var versionID int64
		err = tx.QueryRow(ctx,
			`SELECT id
			 FROM process_versions
			 WHERE process_type_id = $1 AND version_number = 1 AND sede_id IS NULL AND archived_at IS NULL`,
			processTypeID,
		).Scan(&versionID)
		if err != nil {
			if err == pgx.ErrNoRows {
				if err := tx.QueryRow(ctx,
					`INSERT INTO process_versions (process_type_id, version_number, status, operator_id, sede_id)
					VALUES ($1, $2, $3, $4, NULL)
					RETURNING id`,
					processTypeID,
					1,
					"TEST",
					1,
				).Scan(&versionID); err != nil {
					return fmt.Errorf("insert process_versions '%s': %w", processTypeName, err)
				}
			} else {
				return fmt.Errorf("select process_versions '%s': %w", processTypeName, err)
			}
		}

		steps := []struct {
			Order        int
			Name         string
			ExecutionKey string
			Config       string
		}{
			{
				Order:        1,
				Name:         "Step 1: Start + organize batches con ExportManager",
				ExecutionKey: "bulk/export/bcra/start",
				Config:       `{"batch_size":5000,"part_prefix":"exports/bulk_jobs/bcra","redis_ttl_hours":24}`,
			},
			{
				Order:        2,
				Name:         "Step 2: Procesar batch con header/body/footer",
				ExecutionKey: "bulk/export/bcra/process_batch",
				Config: `{
					"execution_policy": {
						"mode": "ASYNC",
						"label": "generar archivo BCRA",
						"auto_invoke": {
							"enabled": true,
							"cursor_field": "batch_index",
							"stop_condition": "is_last_batch"
						},
						"next_step": "bulk/export/bcra/finalize"
					}
				}`,
			},
			{
				Order:        3,
				Name:         "Step 3: Merge final y cierre del proceso",
				ExecutionKey: "bulk/export/bcra/finalize",
				Config:       `{"file":"exports/bcra"}`,
			},
		}

		validExecutionKeys := make([]string, 0, len(steps))
		for _, s := range steps {
			validExecutionKeys = append(validExecutionKeys, s.ExecutionKey)
		}

		if _, err := tx.Exec(ctx,
			`DELETE FROM process_steps
			 WHERE process_version_id = $1
			   AND NOT (execution_key = ANY($2))`,
			versionID,
			validExecutionKeys,
		); err != nil {
			return fmt.Errorf("delete obsolete process_steps for '%s': %w", processTypeName, err)
		}

		for _, s := range steps {
			if err := UpsertStep(ctx, tx, versionID, s.Order, s.Name, s.ExecutionKey, s.Config); err != nil {
				return fmt.Errorf("upsert step failed (%s): %w", s.ExecutionKey, err)
			}
		}

		logger.Info("seeder completado exitosamente", "process_type_id", processTypeID, "version_id", versionID)
		return nil
	})
}
