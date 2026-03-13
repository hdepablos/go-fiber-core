package seeders

import (
	"context"
	"encoding/csv"
	"fmt"
	"os"
	"strings"
	"time"

	"log/slog"
	pgx "github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Shared constants across all seeders
const (
	// Timeouts
	defaultSeederTimeout = 30 * time.Second

	// CSV configuration
	csvDelimiter = ';'

	// Batch sizes for bulk operations
	// File paths base
	seedersFilesPath = "internal/database/seeders/files"
)

// parseCSV reads and parses a CSV file with semicolon delimiter.
func parseCSV(filename string) ([][]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("open file: %w", err)
	}
	defer func() {
		_ = file.Close()
	}()

	reader := csv.NewReader(file)
	reader.Comma = csvDelimiter
	reader.LazyQuotes = true
	reader.TrimLeadingSpace = true

	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("read CSV: %w", err)
	}

	if len(records) == 0 {
		return nil, fmt.Errorf("archivo CSV vacío")
	}

	return records, nil
}

// normalizeString removes quotes and trims whitespace from a string.
func normalizeString(s string) string {
	s = strings.TrimSpace(s)
	s = strings.Trim(s, "'\"")
	return strings.TrimSpace(s)
}

// parseBoolSafe converts string representations to boolean.
func parseBoolSafe(value string) (bool, error) {
	value = strings.TrimSpace(value)
	switch value {
	case "1", "true", "TRUE", "True":
		return true, nil
	case "0", "false", "FALSE", "False":
		return false, nil
	default:
		return false, fmt.Errorf("valor booleano inválido: %s", value)
	}
}

// executeInTransaction executes a function within a database transaction.
// Automatically handles rollback on error and commit on success.
func executeInTransaction(ctx context.Context, pool *pgxpool.Pool, fn func(context.Context, pgx.Tx) error) error {
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("acquire connection: %w", err)
	}
	defer conn.Release()

	tx, err := conn.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	defer func() {
		if rollbackErr := tx.Rollback(ctx); rollbackErr != nil && rollbackErr != pgx.ErrTxClosed {
			_ = rollbackErr
		}
	}()

	if err := fn(ctx, tx); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

// truncateTable truncates a table and restarts its identity sequence.
func truncateTable(ctx context.Context, tx pgx.Tx, tableName string) error {
	query := fmt.Sprintf("TRUNCATE %s RESTART IDENTITY CASCADE", tableName)
	_, err := tx.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("truncate %s: %w", tableName, err)
	}
	return nil
}

// validateRecords performs basic validation on CSV records.
func validateRecords(records [][]string, minColumns int) error {
	if len(records) <= 1 {
		return fmt.Errorf("csv vacío o solo contiene cabecera")
	}

	for i := 1; i < len(records); i++ {
		if len(records[i]) < minColumns {
			return fmt.Errorf("línea %d: insuficientes columnas (esperadas: %d, recibidas: %d)",
				i+1, minColumns, len(records[i]))
		}
	}

	return nil
}

// buildFilePath constructs the full path for a CSV file.
func buildFilePath(filename string) string {
	return fmt.Sprintf("%s/%s", seedersFilesPath, filename)
}


// parseUint converts string to uint safely.
func parseUint(s string) (uint, error) {
	s = normalizeString(s)

	if s == "" {
		return 0, fmt.Errorf("valor vacío")
	}

	var v uint64
	_, err := fmt.Sscan(s, &v)
	if err != nil {
		return 0, fmt.Errorf("uint inválido: %s", s)
	}

	return uint(v), nil
}

// logParseErrors logs parsing errors without stopping the seeder.
func logParseErrors(logger *slog.Logger, errs []error) {
	if len(errs) == 0 {
		return
	}

	logger.Warn("errores al parsear registros", "count", len(errs))
	for _, e := range errs {
		logger.Debug("parse error", "error", e)
	}
}

// EnsureHistory ensures a history record exists for the given version.
func EnsureHistory(ctx context.Context, tx pgx.Tx, versionID, typeID int64, comment string, logger *slog.Logger) error {
	var exists bool
	err := tx.QueryRow(ctx, 
		"SELECT EXISTS(SELECT 1 FROM process_version_history WHERE process_version_id = $1)", 
		versionID,
	).Scan(&exists)
	if err != nil {
		return fmt.Errorf("check history: %w", err)
	}

	if !exists {
		_, err := tx.Exec(ctx,
			`INSERT INTO process_version_history (
				process_version_id, process_type_id, promoted_from_status, promoted_at, promoted_by, comment
			) VALUES ($1, $2, $3, NOW(), $4, $5)`,
			versionID, typeID, "DRAFT", 1, comment,
		)
		if err != nil {
			return fmt.Errorf("insert history: %w", err)
		}
		logger.Info("Historial insertado", "version_id", versionID)
	}
	return nil
}

// UpsertStep inserts or updates a process step.
func UpsertStep(ctx context.Context, tx pgx.Tx, versionID int64, order int, name, key, config string) error {
	cmdTag, err := tx.Exec(ctx,
		`UPDATE process_steps 
		 SET step_order = $1, name = $2, config = $3::jsonb, roadmap = 0
		 WHERE process_version_id = $4 AND execution_key = $5`,
		order, name, config, versionID, key,
	)
	if err != nil {
		return fmt.Errorf("update step %s: %w", key, err)
	}

	if cmdTag.RowsAffected() == 0 {
		_, err := tx.Exec(ctx,
			`INSERT INTO process_steps (process_version_id, step_order, name, execution_key, config, roadmap)
			 VALUES ($1, $2, $3, $4, $5::jsonb, 0)`,
			versionID, order, name, key, config,
		)
		if err != nil {
			return fmt.Errorf("insert step %s: %w", key, err)
		}
	}
	return nil
}
