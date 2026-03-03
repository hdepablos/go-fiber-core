package seeders

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func ProcessLifecycleAutoInvokeSeeder(pool *pgxpool.Pool) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultSeederTimeout)
	defer cancel()

	logger := slog.Default().With("seeder", "process_lifecycle_auto_invoke")
	logger.Info("Starting Process Lifecycle Auto Invoke Seeder")

	return executeInTransaction(ctx, pool, func(ctx context.Context, tx pgx.Tx) error {
		// 1. Create Process Type
		var processTypeID int64
		const processTypeName = "Test Auto Invoke Process"

		err := tx.QueryRow(ctx, "SELECT id FROM process_types WHERE name = $1", processTypeName).Scan(&processTypeID)
		if err != nil {
			if err == pgx.ErrNoRows {
				err = tx.QueryRow(ctx,
					`INSERT INTO process_types (name, description, is_visible)
                     VALUES ($1, $2, $3)
                     RETURNING id`,
					processTypeName,
					"Process to demonstrate auto-invoke/recursion capabilities",
					true,
				).Scan(&processTypeID)
				if err != nil {
					return fmt.Errorf("insert process_types auto_invoke: %w", err)
				}
				logger.Info("Process Type Created", "id", processTypeID)
			} else {
				return fmt.Errorf("select process_types auto_invoke: %w", err)
			}
		} else {
			logger.Info("Process Type Existing", "id", processTypeID)
		}

		// 2. Create Process Version
		var versionID int64
		err = tx.QueryRow(ctx,
			"SELECT id FROM process_versions WHERE process_type_id = $1 AND version_number = 1 AND sede_id IS NULL",
			processTypeID,
		).Scan(&versionID)

		if err != nil {
			if err == pgx.ErrNoRows {
				err = tx.QueryRow(ctx,
					`INSERT INTO process_versions (process_type_id, version_number, status, operator_id, sede_id)
                     VALUES ($1, $2, $3, $4, NULL)
                     RETURNING id`,
					processTypeID,
					1,
					"PROD",
					1,
				).Scan(&versionID)
				if err != nil {
					return fmt.Errorf("insert process_versions auto_invoke: %w", err)
				}
				logger.Info("Process Version Created", "id", versionID)
			} else {
				return fmt.Errorf("select process_versions auto_invoke: %w", err)
			}
		} else {
			logger.Info("Process Version Existing", "id", versionID)
		}

		// 3. Register History
		if err := EnsureHistory(ctx, tx, versionID, processTypeID, "Initial auto-invoke process seed", logger); err != nil {
			return err
		}

		// 4. Add Step
		// Step 1: The Auto Invoke Service
		config := `{"autoInvoke": true}`
		executionKey := "test/test_auto_invoke"
		stepName := "Auto Invoke Step"

		if err := UpsertStep(ctx, tx, versionID, 1, stepName, executionKey, config); err != nil {
			return fmt.Errorf("upsert step failed: %w", err)
		}

		logger.Info("Seeder Completed Successfully")
		return nil
	})
}
