package seeders

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// UserMenu-specific constants
const (
	menuUserTableName = "menu_user" // Singular, según tu modelo
)

// MenuTemplate defines which menus each role has access to.
// These are the BASE permissions for each role type.
var MenuTemplates = map[string][]uint{
	"Admin": {
		// Admin tiene acceso a TODO
		1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15,
	},
	"Coordinador": {
		// Coordinador: Dashboard, Mensajes, Segmentos, Tablas (sin hijos), Mis Archivos
		1, 3, 4, 6, 14,
	},
	"Supervisor": {
		// Supervisor: Dashboard, Mensajes, Mis Archivos, Notificaciones
		1, 3, 14, 15,
	},
	"Operador": {
		// Operador: Solo Dashboard y Notificaciones
		1, 15,
	},
}

// MenuUserSeeder seeds the menu_user relationship table.
// Assigns menus to user_id = 1 based on role templates.
func MenuUserSeeder(pool *pgxpool.Pool) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultSeederTimeout)
	defer cancel()

	logger := slog.Default().With("seeder", "menu_user")
	logger.Info("iniciando seeder de relación menu_user")

	// Get the role name for user_id = 1
	roleName, err := getUserRoleName(ctx, pool, 1)
	if err != nil {
		return fmt.Errorf("getUserRoleName: %w", err)
	}

	logger.Info("rol del usuario obtenido", "user_id", 1, "role_name", roleName)

	// Get menu IDs from the template for this role
	menuIDs, exists := MenuTemplates[roleName]
	if !exists {
		return fmt.Errorf("no existe plantilla de menús para el rol '%s'", roleName)
	}

	logger.Debug("plantilla de menús encontrada", "role", roleName, "menu_count", len(menuIDs))

	// Create relationships: assign template menus to user_id = 1
	now := time.Now()
	menuUsers := make([]*MenuUser, 0, len(menuIDs))
	for _, menuID := range menuIDs {
		menuUsers = append(menuUsers, &MenuUser{
			MenuID:    menuID,
			UserID:    1,
			CreatedAt: now,
			UpdatedAt: now,
		})
	}

	if err := seedMenuUsers(ctx, pool, menuUsers, logger); err != nil {
		return fmt.Errorf("seedMenuUsers: %w", err)
	}

	logger.Info("seeder completado exitosamente",
		"user_id", 1,
		"role", roleName,
		"menus_asignados", len(menuUsers))

	return nil
}

// MenuUserSeederForMultipleUsers seeds menus for multiple users based on their roles.
// This is useful when you have multiple test users with different roles.
func MenuUserSeederForMultipleUsers(pool *pgxpool.Pool, userIDs []uint) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultSeederTimeout)
	defer cancel()

	logger := slog.Default().With("seeder", "menu_user_multiple")
	logger.Info("iniciando seeder de menu_user para múltiples usuarios", "user_count", len(userIDs))

	now := time.Now()
	allMenuUsers := make([]*MenuUser, 0)

	for _, userID := range userIDs {
		// Get role name for this user
		roleName, err := getUserRoleName(ctx, pool, uint64(userID))
		if err != nil {
			logger.Warn("no se pudo obtener el rol del usuario", "user_id", userID, "error", err)
			continue
		}

		// Get menu template for this role
		menuIDs, exists := MenuTemplates[roleName]
		if !exists {
			logger.Warn("no existe plantilla para el rol", "user_id", userID, "role", roleName)
			continue
		}

		// Create relationships for this user
		for _, menuID := range menuIDs {
			allMenuUsers = append(allMenuUsers, &MenuUser{
				MenuID:    menuID,
				UserID:    userID,
				CreatedAt: now,
				UpdatedAt: now,
			})
		}

		logger.Debug("menús asignados", "user_id", userID, "role", roleName, "menu_count", len(menuIDs))
	}

	if len(allMenuUsers) == 0 {
		logger.Warn("no hay relaciones menu_user para insertar")
		return nil
	}

	if err := seedMenuUsers(ctx, pool, allMenuUsers, logger); err != nil {
		return fmt.Errorf("seedMenuUsers: %w", err)
	}

	logger.Info("seeder completado exitosamente", "total_relaciones", len(allMenuUsers))
	return nil
}

// MenuUser represents a menu-user relationship for seeding.
type MenuUser struct {
	MenuID    uint
	UserID    uint
	CreatedAt time.Time
	UpdatedAt time.Time
}

// getUserRoleName retrieves the role name for a given user ID.
func getUserRoleName(ctx context.Context, pool *pgxpool.Pool, userID uint64) (string, error) {
	query := `
		SELECT r.name
		FROM roles r
		INNER JOIN role_user ru ON r.id = ru.role_id
		WHERE ru.user_id = $1
		LIMIT 1
	`

	var roleName string
	err := pool.QueryRow(ctx, query, userID).Scan(&roleName)
	if err != nil {
		return "", fmt.Errorf("query role name: %w", err)
	}

	return roleName, nil
}

// seedMenuUsers executes the database seeding operation within a transaction.
func seedMenuUsers(ctx context.Context, pool *pgxpool.Pool, menuUsers []*MenuUser, logger *slog.Logger) error {
	return executeInTransaction(ctx, pool, func(ctx context.Context, tx pgx.Tx) error {
		if err := truncateTable(ctx, tx, menuUserTableName); err != nil {
			return fmt.Errorf("truncate: %w", err)
		}
		logger.Debug("tabla truncada", "table", menuUserTableName)

		rows := menuUsersToCopyRows(menuUsers)
		count, err := tx.CopyFrom(
			ctx,
			pgx.Identifier{menuUserTableName},
			[]string{"menu_id", "user_id", "created_at", "updated_at"},
			pgx.CopyFromRows(rows),
		)
		if err != nil {
			return fmt.Errorf("CopyFrom: %w", err)
		}

		logger.Debug("relaciones insertadas vía COPY", "count", count)
		return nil
	})
}

// menuUsersToCopyRows converts MenuUser structs to the format required by CopyFrom.
func menuUsersToCopyRows(menuUsers []*MenuUser) [][]any {
	rows := make([][]any, 0, len(menuUsers))
	for _, mu := range menuUsers {
		rows = append(rows, []any{
			mu.MenuID,
			mu.UserID,
			mu.CreatedAt,
			mu.UpdatedAt,
		})
	}
	return rows
}
