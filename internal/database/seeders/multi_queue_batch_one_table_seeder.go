package seeders

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func MultiQueueBatchOneTableSeeder(pool *pgxpool.Pool) error {
	if pool == nil {
		return fmt.Errorf("db pool is nil")
	}

	ctx, cancel := context.WithTimeout(context.Background(), seedTimeout)
	defer cancel()

	logger := slog.Default().With("seeder", "multi_queue_batch_one_table")

	return executeInTransaction(ctx, pool, func(ctx context.Context, tx pgx.Tx) error {
		if err := seedMultiQueueBatchOneTableRecords(ctx, tx, 10000); err != nil {
			return err
		}

		if err := seedMultiQueueBatchOneTableProcessLifecycle(ctx, tx, logger); err != nil {
			return err
		}

		return nil
	})
}

func MultiQueueBatchOneTableProcessLifecycleSeeder(pool *pgxpool.Pool) error {
	if pool == nil {
		return fmt.Errorf("db pool is nil")
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultSeederTimeout)
	defer cancel()

	logger := slog.Default().With("seeder", "multi_queue_batch_one_table_process_lifecycle")

	return executeInTransaction(ctx, pool, func(ctx context.Context, tx pgx.Tx) error {
		return seedMultiQueueBatchOneTableProcessLifecycle(ctx, tx, logger)
	})
}

func MultiQueueBatchOneTableRecreateRecordsSeeder(pool *pgxpool.Pool) error {
	if pool == nil {
		return fmt.Errorf("db pool is nil")
	}

	ctx, cancel := context.WithTimeout(context.Background(), seedTimeout)
	defer cancel()

	logger := slog.Default().With("seeder", "multi_queue_batch_one_table_recreate_records")

	return executeInTransaction(ctx, pool, func(ctx context.Context, tx pgx.Tx) error {
		if err := seedMultiQueueBatchOneTableRecords(ctx, tx, 200000); err != nil {
			return err
		}

		if err := seedMultiQueueBatchOneTableProcessLifecycle(ctx, tx, logger); err != nil {
			return err
		}

		return nil
	})
}

func seedMultiQueueBatchOneTableRecords(ctx context.Context, tx pgx.Tx, count int64) error {
	if err := truncateTable(ctx, tx, "multi_queue_batch_one_table"); err != nil {
		return err
	}

	if count <= 0 {
		count = 10000
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO multi_queue_batch_one_table (status)
		SELECT 'pending'
		FROM generate_series(1, $1)
	`, count); err != nil {
		return fmt.Errorf("seed multi_queue_batch_one_table: %w", err)
	}

	return nil
}

func seedMultiQueueBatchOneTableProcessLifecycle(ctx context.Context, tx pgx.Tx, logger *slog.Logger) error {
	const processTypeName = "MultiQueueBatchProcessorOneTableV1"
	var processTypeID int64
	err := tx.QueryRow(ctx, "SELECT id FROM process_types WHERE name = $1", processTypeName).Scan(&processTypeID)
	if err != nil {
		if err == pgx.ErrNoRows {
			if err := tx.QueryRow(ctx,
				`INSERT INTO process_types (name, description, is_visible)
				VALUES ($1, $2, $3)
				RETURNING id`,
				processTypeName,
				"Fan-out masivo a colas procesando una sola tabla por lotes",
				true,
			).Scan(&processTypeID); err != nil {
				return fmt.Errorf("insert process_types: %w", err)
			}
		} else {
			return fmt.Errorf("select process_types: %w", err)
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
				return fmt.Errorf("insert process_versions: %w", err)
			}
		} else {
			return fmt.Errorf("select process_versions: %w", err)
		}
	}

	if err := EnsureHistory(ctx, tx, versionID, processTypeID, "Seed MultiQueueBatchProcessorOneTable", logger); err != nil {
		return err
	}

	step1Config := `{"batch_size":50,"table":"multi_queue_batch_one_table"}`
	if err := UpsertStep(ctx, tx, versionID, 1, "Organize records and dispatch batches", "test/mqb1t/organize", step1Config); err != nil {
		return err
	}

	step2Config := `{"batch_size":50,"table":"multi_queue_batch_one_table"}`
	if err := UpsertStep(ctx, tx, versionID, 2, "Process batch (async)", "test/mqb1t/process_batch", step2Config); err != nil {
		return err
	}

	step3Config := `{"table":"multi_queue_batch_one_table"}`
	if err := UpsertStep(ctx, tx, versionID, 3, "Finalize and print stats", "test/mqb1t/finalize", step3Config); err != nil {
		return err
	}

	return nil
}
