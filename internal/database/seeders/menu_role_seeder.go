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
	menuRoleCSVPath   = "internal/database/seeders/files/menu_role.csv"
	menuRoleTableName = "menu_role"
	menuRoleTimeout   = 30 * time.Second
)

type MenuRole struct {
	MenuID uint
	RoleID uint
}

func MenuRoleSeeder(pool *pgxpool.Pool) error {
	ctx, cancel := context.WithTimeout(context.Background(), menuRoleTimeout)
	defer cancel()

	logger := slog.Default().With("seeder", "menu_role")
	logger.Info("iniciando seeder menu_role")

	records, err := parseCSV(menuRoleCSVPath)
	if err != nil {
		return err
	}

	rows, errs := parseMenuRoleRecords(records)
	logParseErrors(logger, errs)

	return seedMenuRoles(ctx, pool, rows, logger)
}

func parseMenuRoleRecords(records [][]string) ([]*MenuRole, []error) {
	list := []*MenuRole{}
	errs := []error{}

	for i := 1; i < len(records); i++ {
		row := records[i]

		menuID, err1 := parseUint(row[0])
		roleID, err2 := parseUint(row[1])

		if err1 != nil || err2 != nil {
			errs = append(errs, fmt.Errorf("línea %d inválida", i+1))
			continue
		}

		list = append(list, &MenuRole{menuID, roleID})
	}

	return list, errs
}

func seedMenuRoles(ctx context.Context, pool *pgxpool.Pool, rows []*MenuRole, logger *slog.Logger) error {
	return executeInTransaction(ctx, pool, func(ctx context.Context, tx pgx.Tx) error {

		if err := truncateTable(ctx, tx, menuRoleTableName); err != nil {
			return err
		}

		copyRows := make([][]any, 0, len(rows))
		for _, r := range rows {
			copyRows = append(copyRows, []any{r.MenuID, r.RoleID})
		}

		_, err := tx.CopyFrom(
			ctx,
			pgx.Identifier{menuRoleTableName},
			[]string{"menu_id", "role_id"},
			pgx.CopyFromRows(copyRows),
		)

		logger.Info("menu_role seed completado", "rows", len(rows))
		return err
	})
}