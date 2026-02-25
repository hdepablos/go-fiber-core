package seeders

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	pgx "github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	roleUserCSVPath   = "internal/database/seeders/files/role_user.csv"
	roleUserTableName = "role_user"
	roleUserTimeout   = 30 * time.Second
)

type RoleUser struct {
	RoleID uint
	UserID uint
}

func RoleUserSeeder(pool *pgxpool.Pool) error {
	ctx, cancel := context.WithTimeout(context.Background(), roleUserTimeout)
	defer cancel()

	logger := slog.Default().With("seeder", "role_user")

	records, err := parseCSV(roleUserCSVPath)
	if err != nil {
		return err
	}

	rows, errs := parseRoleUserRecords(records)
	logParseErrors(logger, errs)

	return seedRoleUsers(ctx, pool, rows, logger)
}

func parseRoleUserRecords(records [][]string) ([]*RoleUser, []error) {
	list := []*RoleUser{}
	errs := []error{}

	for i := 1; i < len(records); i++ {
		row := records[i]

		roleID, err1 := parseUint(row[0])
		userID, err2 := parseUint(row[1])

		if err1 != nil || err2 != nil {
			errs = append(errs, fmt.Errorf("línea %d inválida", i+1))
			continue
		}

		list = append(list, &RoleUser{roleID, userID})
	}

	return list, errs
}

func seedRoleUsers(ctx context.Context, pool *pgxpool.Pool, rows []*RoleUser, logger *slog.Logger) error {
	return executeInTransaction(ctx, pool, func(ctx context.Context, tx pgx.Tx) error {

		if err := truncateTable(ctx, tx, roleUserTableName); err != nil {
			return err
		}

		copyRows := make([][]any, 0, len(rows))
		for _, r := range rows {
			copyRows = append(copyRows, []any{r.RoleID, r.UserID})
		}

		_, err := tx.CopyFrom(
			ctx,
			pgx.Identifier{roleUserTableName},
			[]string{"role_id", "user_id"},
			pgx.CopyFromRows(copyRows),
		)

		logger.Info("role_user seed completado", "rows", len(rows))
		return err
	})
}