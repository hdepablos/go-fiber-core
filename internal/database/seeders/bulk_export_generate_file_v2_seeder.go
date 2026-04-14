package seeders

import (
	"context"
	"fmt"
	"log/slog"

	pgx "github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func BulkExportGenerateFileV2Seeder(pool *pgxpool.Pool) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultSeederTimeout)
	defer cancel()

	logger := slog.Default().With("seeder", "bulk_export_generate_file_v2")
	logger.Info("iniciando seeder de bulk_export_generate_file_v2")

	return executeInTransaction(ctx, pool, func(ctx context.Context, tx pgx.Tx) error {
		const processTypeName = "generar archivo v2"
		const processDescription = "Generación de archivo v2 usando exportmanager reusable"

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
					"PROD",
					1,
				).Scan(&versionID); err != nil {
					return fmt.Errorf("insert process_versions '%s': %w", processTypeName, err)
				}
			} else {
				return fmt.Errorf("select process_versions '%s': %w", processTypeName, err)
			}
		}

		if err := EnsureHistory(ctx, tx, versionID, processTypeID, "Seed inicial: generar archivo v2", logger); err != nil {
			return err
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
				ExecutionKey: "bulk/export/v2/start",
				Config:       `{"batch_size":5000,"redis_ttl_hours":24,"part_prefix":"exports/bulk_jobs/v2"}`,
			},
			{
				Order:        2,
				Name:         "Step 2: Procesar batch con header/body/footer",
				ExecutionKey: "bulk/export/v2/process_batch",
				Config: `{
					"execution_policy": {
						"mode": "ASYNC",
						"label": "generar archivo v2",
						"auto_invoke": {
							"enabled": true,
							"cursor_field": "batch_index",
							"stop_condition": "is_last_batch"
						},
						"next_step": "bulk/export/v2/finalize"
					}
				}`,
			},
			{
				Order:        3,
				Name:         "Step 3: Merge final y cierre del proceso",
				ExecutionKey: "bulk/export/v2/finalize",
				Config:       `{"file":"exports/bank/colombia/manager-colombia"}`,
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
