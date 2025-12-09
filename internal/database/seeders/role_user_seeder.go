package seeders

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// RoleUser-specific constants
const (
	roleUserTableName = "role_user"
)

// RoleUserSeeder seeds the role_user relationship table.
// Assigns role_id = 1 to user_id = 1.
func RoleUserSeeder(pool *pgxpool.Pool) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultSeederTimeout)
	defer cancel()

	logger := slog.Default().With("seeder", "role_user")
	logger.Info("iniciando seeder de relación role_user")

	// Create the relationships
	roleUsers := []*RoleUser{
		{
			RoleID: 1, // Primer rol (ej: Admin)
			UserID: 1, // Usuario test
		},
		// Puedes agregar más relaciones aquí:
		// {RoleID: 2, UserID: 1}, // Si el usuario 1 tiene múltiples roles
	}

	if err := seedRoleUsers(ctx, pool, roleUsers, logger); err != nil {
		return fmt.Errorf("seedRoleUsers: %w", err)
	}

	logger.Info("seeder completado exitosamente", "relaciones_insertadas", len(roleUsers))
	return nil
}

// RoleUser represents a role-user relationship for seeding.
type RoleUser struct {
	RoleID uint64
	UserID uint64
}

// seedRoleUsers executes the database seeding operation within a transaction.
func seedRoleUsers(ctx context.Context, pool *pgxpool.Pool, roleUsers []*RoleUser, logger *slog.Logger) error {
	return executeInTransaction(ctx, pool, func(ctx context.Context, tx pgx.Tx) error {
		if err := truncateTable(ctx, tx, roleUserTableName); err != nil {
			return fmt.Errorf("truncate: %w", err)
		}
		logger.Debug("tabla truncada", "table", roleUserTableName)

		rows := roleUsersToCopyRows(roleUsers)
		count, err := tx.CopyFrom(
			ctx,
			pgx.Identifier{roleUserTableName},
			[]string{"role_id", "user_id"},
			pgx.CopyFromRows(rows),
		)
		if err != nil {
			return fmt.Errorf("CopyFrom: %w", err)
		}

		logger.Debug("relaciones insertadas vía COPY", "count", count)
		return nil
	})
}

// roleUsersToCopyRows converts RoleUser structs to the format required by CopyFrom.
func roleUsersToCopyRows(roleUsers []*RoleUser) [][]any {
	rows := make([][]any, 0, len(roleUsers))
	for _, ru := range roleUsers {
		rows = append(rows, []any{ru.RoleID, ru.UserID})
	}
	return rows
}
