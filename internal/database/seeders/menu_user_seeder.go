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
	menuUserCSVPath   = "internal/database/seeders/files/menu_user.csv"
	menuUserTableName = "menu_user"
	menuUserTimeout   = 30 * time.Second
)

type MenuUser struct {
	MenuID uint
	UserID uint
}

func MenuUserSeeder(pool *pgxpool.Pool) error {
	ctx, cancel := context.WithTimeout(context.Background(), menuUserTimeout)
	defer cancel()

	logger := slog.Default().With("seeder", "menu_user")

	records, err := parseCSV(menuUserCSVPath)
	if err != nil {
		return err
	}

	rows, errs := parseMenuUserRecords(records)
	logParseErrors(logger, errs)

	return seedMenuUsers(ctx, pool, rows, logger)
}

func parseMenuUserRecords(records [][]string) ([]*MenuUser, []error) {
	list := []*MenuUser{}
	errs := []error{}

	for i := 1; i < len(records); i++ {
		row := records[i]

		menuID, err1 := parseUint(row[0])
		userID, err2 := parseUint(row[1])

		if err1 != nil || err2 != nil {
			errs = append(errs, fmt.Errorf("línea %d inválida", i+1))
			continue
		}

		list = append(list, &MenuUser{menuID, userID})
	}

	return list, errs
}

func seedMenuUsers(ctx context.Context, pool *pgxpool.Pool, rows []*MenuUser, logger *slog.Logger) error {
	return executeInTransaction(ctx, pool, func(ctx context.Context, tx pgx.Tx) error {

		if err := truncateTable(ctx, tx, menuUserTableName); err != nil {
			return err
		}

		copyRows := make([][]any, 0, len(rows))
		for _, r := range rows {
			copyRows = append(copyRows, []any{r.MenuID, r.UserID})
		}

		_, err := tx.CopyFrom(
			ctx,
			pgx.Identifier{menuUserTableName},
			[]string{"menu_id", "user_id"},
			pgx.CopyFromRows(copyRows),
		)

		logger.Info("menu_user seed completado", "rows", len(rows))
		return err
	})
}