package seeders

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	pgx "github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const allMenusTimeout = 60 * time.Second

func AllMenusSeeder(pool *pgxpool.Pool, configPath string) error {
	ctx, cancel := context.WithTimeout(context.Background(), allMenusTimeout)
	defer cancel()

	logger := slog.Default().With("seeder", "all_menus")

	logger.Info("iniciando limpieza de tablas")

	// -------------------------
	// LIMPIEZA hijos → padres
	// -------------------------
	err := executeInTransaction(ctx, pool, func(ctx context.Context, tx pgx.Tx) error {
		tables := []string{
			"menu_role",
			"menu_user",
			"role_user",
			"roles",
			"menus",
			"users",
		}

		for _, t := range tables {
			logger.Debug("truncando tabla", "table", t)
			if err := truncateTable(ctx, tx, t); err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("cleanup: %w", err)
	}

	logger.Info("limpieza completada, iniciando seed")

	// -------------------------
	// SEED orden requerido
	// -------------------------

	if err := MenuSeeder(pool); err != nil {
		return fmt.Errorf("menus: %w", err)
	}

	if err := RoleSeeder(pool); err != nil {
		return fmt.Errorf("roles: %w", err)
	}

	if err := MenuRoleSeeder(pool); err != nil {
		return fmt.Errorf("menu_role: %w", err)
	}

	if err := CreateUserSeeder(configPath); err != nil {
		return fmt.Errorf("users: %w", err)
	}

	if err := RoleUserSeeder(pool); err != nil {
		return fmt.Errorf("role_user: %w", err)
	}

	if err := MenuUserSeeder(pool); err != nil {
		return fmt.Errorf("menu_user: %w", err)
	}

	logger.Info("all_menus finalizado correctamente")
	return nil
}