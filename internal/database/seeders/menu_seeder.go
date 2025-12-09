package seeders

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Menu-specific constants
const (
	menusJSONFile       = "menus.json"
	menusTableName      = "menus"
	menusRequiredFields = 3 // item_type, item_name, order_index
)

// MenuSeeder seeds the menus table from a JSON file using pgx COPY.
func MenuSeeder(pool *pgxpool.Pool) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultSeederTimeout)
	defer cancel()

	logger := slog.Default().With("seeder", "menus")
	logger.Info("iniciando seeder de menús")

	jsonPath := buildFilePath(menusJSONFile)
	menuItems, err := parseMenuJSON(jsonPath)
	if err != nil {
		return fmt.Errorf("parseMenuJSON: %w", err)
	}

	logger.Debug("menús parseados desde JSON", "count", len(menuItems))

	if len(menuItems) == 0 {
		return fmt.Errorf("no hay menús para insertar")
	}

	// Flatten the menu structure (parents first, then children)
	flatMenus := flattenMenus(menuItems)
	logger.Debug("estructura de menús aplanada", "total_items", len(flatMenus))

	if err := seedMenus(ctx, pool, flatMenus, logger); err != nil {
		return fmt.Errorf("seedMenus: %w", err)
	}

	logger.Info("seeder completado exitosamente", "menus_insertados", len(flatMenus))
	return nil
}

// MenuJSON represents the JSON structure for menu items.
// IDs are not needed - the structure is inferred from the children array.
type MenuJSON struct {
	ItemType   string     `json:"item_type"`
	ItemName   string     `json:"item_name"`
	ToPath     *string    `json:"to_path,omitempty"`
	Icon       *string    `json:"icon,omitempty"`
	OrderIndex int        `json:"order_index"`
	Children   []MenuJSON `json:"children,omitempty"`
}

// Menu represents a flattened menu item for database insertion.
type Menu struct {
	ItemType   string
	ItemName   string
	ToPath     *string
	Icon       *string
	ParentID   *uint
	OrderIndex int
	IsActive   bool
}

// parseMenuJSON reads and parses the menu JSON file.
func parseMenuJSON(filename string) ([]MenuJSON, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	var menuItems []MenuJSON
	if err := json.Unmarshal(data, &menuItems); err != nil {
		return nil, fmt.Errorf("unmarshal JSON: %w", err)
	}

	return menuItems, nil
}

// flattenMenus converts hierarchical menu structure to flat list.
// Parents are inserted first, then their children with the correct parent_id.
// The parent_id is automatically assigned based on the insertion order.
func flattenMenus(menuItems []MenuJSON) []*Menu {
	var result []*Menu

	// Process each top-level menu item
	for _, item := range menuItems {
		// Insert the parent with no parent_id (top-level item)
		parentMenu := jsonToMenu(&item, nil)
		result = append(result, parentMenu)

		// Calculate what will be the database ID of this parent
		// (based on current position in the result slice)
		parentDBID := uint(len(result))

		// Process children if they exist
		if len(item.Children) > 0 {
			for _, child := range item.Children {
				// The child's parent_id is the calculated parent ID
				childMenu := jsonToMenu(&child, &parentDBID)
				result = append(result, childMenu)
			}
		}
	}

	return result
}

// jsonToMenu converts a MenuJSON to a Menu struct.
// If parentID is nil, this is a top-level menu item.
// If parentID is set, this is a child of another menu item.
func jsonToMenu(item *MenuJSON, parentID *uint) *Menu {
	menu := &Menu{
		ItemType:   item.ItemType,
		ItemName:   item.ItemName,
		ToPath:     item.ToPath,
		Icon:       item.Icon,
		ParentID:   parentID,
		OrderIndex: item.OrderIndex,
		IsActive:   true, // All menus are active by default
	}

	return menu
}

// seedMenus executes the database seeding operation within a transaction.
func seedMenus(ctx context.Context, pool *pgxpool.Pool, menus []*Menu, logger *slog.Logger) error {
	return executeInTransaction(ctx, pool, func(ctx context.Context, tx pgx.Tx) error {
		if err := truncateTable(ctx, tx, menusTableName); err != nil {
			return fmt.Errorf("truncate: %w", err)
		}
		logger.Debug("tabla truncada", "table", menusTableName)

		rows := menusToCopyRows(menus)
		count, err := tx.CopyFrom(
			ctx,
			pgx.Identifier{menusTableName},
			[]string{"item_type", "item_name", "to_path", "icon", "parent_id", "order_index", "is_active"},
			pgx.CopyFromRows(rows),
		)
		if err != nil {
			return fmt.Errorf("CopyFrom: %w", err)
		}

		logger.Debug("menús insertados vía COPY", "count", count)
		return nil
	})
}

// menusToCopyRows converts Menu structs to the format required by CopyFrom.
func menusToCopyRows(menus []*Menu) [][]any {
	rows := make([][]any, 0, len(menus))
	for _, m := range menus {
		rows = append(rows, []any{
			m.ItemType,
			m.ItemName,
			m.ToPath,
			m.Icon,
			m.ParentID,
			m.OrderIndex,
			m.IsActive,
		})
	}
	return rows
}
