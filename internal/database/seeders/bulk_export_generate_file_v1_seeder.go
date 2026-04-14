package seeders

import (
	"context"
	"fmt"
	"log/slog"

	pgx "github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func BulkExportGenerateFileV1Seeder(pool *pgxpool.Pool) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultSeederTimeout)
	defer cancel()

	logger := slog.Default().With("seeder", "bulk_export_generate_file_v1")
	logger.Info("iniciando seeder de bulk_export_generate_file_v1")

	return executeInTransaction(ctx, pool, func(ctx context.Context, tx pgx.Tx) error {
		const processTypeName = "generar archivo v1"
		const processDescription = "Generación de archivo v1 a partir de bulk_jobs (IMPORTED) con batching + S3"

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

		if err := EnsureHistory(ctx, tx, versionID, processTypeID, "Seed inicial: generar archivo v1", logger); err != nil {
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
				Name:         "Step 1: Organizar data (bulk_jobs IMPORTED -> lotes 5000 -> Redis)",
				ExecutionKey: "bulk/export/v1/organize",
				Config:       `{"batch_size":5000,"redis_ttl_hours":24}`,
			},
			{
				Order:        2,
				Name:         "Step 2: CSV por lote y subir a S3 (AutoInvoke)",
				ExecutionKey: "bulk/export/v1/write_csv_batch",
				Config: `{
					"batch_size": 5000,
					"execution_policy": {
						"mode": "ASYNC",
						"label": "generar archivo v1",
						"auto_invoke": {
							"enabled": true,
							"cursor_field": "batch_index",
							"stop_condition": "is_last_batch"
						},
						"next_step": "bulk/export/v1/merge_multipart"
					}
				}`,
			},
			{
				Order:        3,
				Name:         "Step 3: Integrar archivos S3 en uno (Multipart Upload)",
				ExecutionKey: "bulk/export/v1/merge_multipart",
				Config:       `{"file":"exports/bank/colombia/pago-colombia"}`,
			},
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
