package seeders

import (
	"context"
	"encoding/csv"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Shared constants across all seeders
const (
	// Timeouts
	defaultSeederTimeout = 30 * time.Second

	// CSV configuration
	csvDelimiter = ';'

	// Batch sizes for bulk operations
	defaultBatchSize = 100

	// File paths base
	seedersFilesPath = "internal/database/seeders/files"
)

// parseCSV reads and parses a CSV file with semicolon delimiter.
func parseCSV(filename string) ([][]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("open file: %w", err)
	}
	defer file.Close()

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
		if err := tx.Rollback(ctx); err != nil && err != pgx.ErrTxClosed {
			// Log rollback errors but don't override the original error
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
