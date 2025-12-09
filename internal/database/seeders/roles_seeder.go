package seeders

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	rolesCSVPath   = "internal/database/seeders/files/roles.csv"
	rolesBatchSize = 100
	rolesTableName = "roles"
	defaultTimeout = 30 * time.Second
)

// RoleSeeder seeds the roles table from a CSV file using pgx COPY.
func RoleSeeder(pool *pgxpool.Pool) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	logger := slog.Default().With("seeder", "roles")
	logger.Info("iniciando seeder de roles")

	records, err := parseCSV(rolesCSVPath)
	if err != nil {
		return fmt.Errorf("parseCSV: %w", err)
	}

	roles, parseErrs := parseRoleRecords(records)
	if len(parseErrs) > 0 {
		logger.Warn("errores al parsear registros", "count", len(parseErrs))
		for _, e := range parseErrs {
			logger.Debug("error de parseo", "error", e)
		}
	}

	if len(roles) == 0 {
		return fmt.Errorf("no hay roles válidos para insertar")
	}

	if err := seedRoles(ctx, pool, roles, logger); err != nil {
		return fmt.Errorf("seedRoles: %w", err)
	}

	logger.Info("seeder completado exitosamente", "roles_insertados", len(roles))
	return nil
}

// Role represents a role entity for seeding.
type Role struct {
	Name     string
	IsActive bool
}

// parseRoleRecords parses multiple CSV records into Role structs.
// Returns valid roles and a slice of parsing errors.
func parseRoleRecords(records [][]string) ([]*Role, []error) {
	if len(records) <= 1 {
		return nil, []error{fmt.Errorf("csv vacío o solo contiene cabecera")}
	}

	roles := make([]*Role, 0, len(records)-1)
	errs := make([]error, 0)

	for i := 1; i < len(records); i++ {
		role, err := parseRoleRecord(records[i], i+1)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		roles = append(roles, role)
	}

	return roles, errs
}

// parseRoleRecord parses a single CSV record into a Role struct.
func parseRoleRecord(row []string, lineNum int) (*Role, error) {
	if len(row) < 1 {
		return nil, fmt.Errorf("línea %d: insuficientes campos", lineNum)
	}

	name := normalizeString(row[0])
	if name == "" {
		return nil, fmt.Errorf("línea %d: nombre de rol vacío", lineNum)
	}

	if len(name) > 100 {
		return nil, fmt.Errorf("línea %d: nombre excede 100 caracteres", lineNum)
	}

	return &Role{
		Name:     name,
		IsActive: true,
	}, nil
}

// seedRoles executes the database seeding operation within a transaction.
func seedRoles(ctx context.Context, pool *pgxpool.Pool, roles []*Role, logger *slog.Logger) error {
	return executeInTransaction(ctx, pool, func(ctx context.Context, tx pgx.Tx) error {
		if err := truncateTable(ctx, tx, rolesTableName); err != nil {
			return fmt.Errorf("truncate: %w", err)
		}
		logger.Debug("tabla truncada", "table", rolesTableName)

		rows := rolesToCopyRows(roles)
		count, err := tx.CopyFrom(
			ctx,
			pgx.Identifier{rolesTableName},
			[]string{"name", "is_active"},
			pgx.CopyFromRows(rows),
		)
		if err != nil {
			return fmt.Errorf("CopyFrom: %w", err)
		}

		logger.Debug("roles insertados vía COPY", "count", count)
		return nil
	})
}

// rolesToCopyRows converts Role structs to the format required by CopyFrom.
func rolesToCopyRows(roles []*Role) [][]any {
	rows := make([][]any, 0, len(roles))
	for _, r := range roles {
		rows = append(rows, []any{r.Name, r.IsActive})
	}
	return rows
}
