package seeders

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Bank-specific constants
const (
	banksCSVFile         = "banks.csv"
	banksTableName       = "banks"
	banksRequiredColumns = 4
)

// BankSeeder seeds the banks table from a CSV file using pgx COPY.
func BankSeeder(pool *pgxpool.Pool) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultSeederTimeout)
	defer cancel()

	logger := slog.Default().With("seeder", "banks")
	logger.Info("iniciando seeder de bancos")

	csvPath := buildFilePath(banksCSVFile)
	records, err := parseCSV(csvPath)
	if err != nil {
		return fmt.Errorf("parseCSV: %w", err)
	}

	if err := validateRecords(records, banksRequiredColumns); err != nil {
		return fmt.Errorf("validateRecords: %w", err)
	}

	banks, parseErrs := parseBankRecords(records)
	if len(parseErrs) > 0 {
		logger.Warn("errores al parsear registros", "count", len(parseErrs))
		for _, e := range parseErrs {
			logger.Debug("error de parseo", "error", e)
		}
	}

	if len(banks) == 0 {
		return fmt.Errorf("no hay bancos válidos para insertar")
	}

	if err := seedBanks(ctx, pool, banks, logger); err != nil {
		return fmt.Errorf("seedBanks: %w", err)
	}

	logger.Info("seeder completado exitosamente", "bancos_insertados", len(banks))
	return nil
}

// Bank represents a bank entity for seeding.
type Bank struct {
	ID         int
	Name       string
	EntityCode string
	Enabled    bool
}

// parseBankRecords parses multiple CSV records into Bank structs.
// Returns valid banks and a slice of parsing errors.
func parseBankRecords(records [][]string) ([]*Bank, []error) {
	banks := make([]*Bank, 0, len(records)-1)
	errs := make([]error, 0)

	for i := 1; i < len(records); i++ {
		bank, err := parseBankRecord(records[i], i+1)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		banks = append(banks, bank)
	}

	return banks, errs
}

// parseBankRecord parses a single CSV record into a Bank struct.
func parseBankRecord(row []string, lineNum int) (*Bank, error) {
	if len(row) < banksRequiredColumns {
		return nil, fmt.Errorf("línea %d: insuficientes campos (esperados: %d, recibidos: %d)",
			lineNum, banksRequiredColumns, len(row))
	}

	id, err := strconv.Atoi(normalizeString(row[0]))
	if err != nil {
		return nil, fmt.Errorf("línea %d: ID inválido '%s': %w", lineNum, row[0], err)
	}

	if id <= 0 {
		return nil, fmt.Errorf("línea %d: ID debe ser positivo", lineNum)
	}

	name := normalizeString(row[1])
	if name == "" {
		return nil, fmt.Errorf("línea %d: nombre vacío", lineNum)
	}

	entityCode := normalizeString(row[2])
	if entityCode == "" {
		return nil, fmt.Errorf("línea %d: código de entidad vacío", lineNum)
	}

	enabled, err := parseBoolSafe(row[3])
	if err != nil {
		return nil, fmt.Errorf("línea %d: 'enabled' inválido '%s': %w", lineNum, row[3], err)
	}

	return &Bank{
		ID:         id,
		Name:       name,
		EntityCode: entityCode,
		Enabled:    enabled,
	}, nil
}

// seedBanks executes the database seeding operation within a transaction.
func seedBanks(ctx context.Context, pool *pgxpool.Pool, banks []*Bank, logger *slog.Logger) error {
	return executeInTransaction(ctx, pool, func(ctx context.Context, tx pgx.Tx) error {
		if err := truncateTable(ctx, tx, banksTableName); err != nil {
			return fmt.Errorf("truncate: %w", err)
		}
		logger.Debug("tabla truncada", "table", banksTableName)

		rows := banksToCopyRows(banks)
		count, err := tx.CopyFrom(
			ctx,
			pgx.Identifier{banksTableName},
			[]string{"id", "name", "entity_code", "enabled"},
			pgx.CopyFromRows(rows),
		)
		if err != nil {
			return fmt.Errorf("CopyFrom: %w", err)
		}

		logger.Debug("bancos insertados vía COPY", "count", count)
		return nil
	})
}

// banksToCopyRows converts Bank structs to the format required by CopyFrom.
func banksToCopyRows(banks []*Bank) [][]any {
	rows := make([][]any, 0, len(banks))
	for _, b := range banks {
		rows = append(rows, []any{b.ID, b.Name, b.EntityCode, b.Enabled})
	}
	return rows
}
