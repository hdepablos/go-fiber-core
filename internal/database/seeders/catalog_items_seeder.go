package seeders

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"

	pgx "github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	catalogItemsJSONFile  = "catalog_items.jsonc"
	catalogItemsTableName = "catalog_items"
)

type RawCatalogItem struct {
	ID           int             `json:"id"`
	Name         string          `json:"name"`
	CodeRaw      json.RawMessage `json:"code"`
	ParentJSONID int             `json:"parent_id"`
	SedeID       int             `json:"sede_id"`
	Order        int             `json:"order"`
	Active       int             `json:"active"`
	CreatedAt    *string         `json:"created_at"`
	UpdatedAt    *string         `json:"updated_at"`
}

type CatalogItem struct {
	Name     string
	Code     string
	ParentID *int64
}

type catalogItemsStats struct {
	Inserted int
	Reused   int
}

func CatalogItemsSeeder(pool *pgxpool.Pool) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultSeederTimeout)
	defer cancel()

	logger := slog.Default().With("seeder", "catalog_items")
	logger.Info("iniciando seeder de catalog_items")

	jsonPath := buildFilePath(catalogItemsJSONFile)
	rawItems, err := parseCatalogItemsJSONC(jsonPath)
	if err != nil {
		return fmt.Errorf("parseCatalogItemsJSON: %w", err)
	}

	if len(rawItems) == 0 {
		return fmt.Errorf("no hay catalog_items para insertar")
	}

	// Execute idempotent inserts (no truncate).
	if err := seedCatalogItems(ctx, pool, rawItems, logger); err != nil {
		return fmt.Errorf("seedCatalogItems idempotent: %w", err)
	}

	logger.Info("seeder catalog_items completado", "items_procesados", len(rawItems))
	return nil
}

// parseCatalogItemsJSONC reads a JSON (optionally with // and /* */ comments) and parses it.
func parseCatalogItemsJSONC(filename string) ([]RawCatalogItem, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	clean := stripJSONComments(string(data))

	var items []RawCatalogItem
	if err := json.Unmarshal([]byte(clean), &items); err != nil {
		return nil, fmt.Errorf("unmarshal JSON: %w", err)
	}

	return items, nil
}

// stripJSONComments removes // line comments and /* */ block comments from a JSONC-like string.
func stripJSONComments(s string) string {
	// Remove block comments
	for {
		start := strings.Index(s, "/*")
		if start == -1 {
			break
		}
		end := strings.Index(s[start+2:], "*/")
		if end == -1 {
			// Unclosed comment, drop the rest
			s = s[:start]
			break
		}
		s = s[:start] + s[start+2+end+2:]
	}
	// Remove line comments
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		if idx := strings.Index(line, "//"); idx != -1 {
			lines[i] = line[:idx]
		}
	}
	return strings.Join(lines, "\n")
}

// seedCatalogItems inserts only missing records based on (code, parent_id) uniqueness.
// It resolves parent-child relationships using RelTable that references the JSON item ID.
func seedCatalogItems(ctx context.Context, pool *pgxpool.Pool, items []RawCatalogItem, logger *slog.Logger) error {
	return executeInTransaction(ctx, pool, func(ctx context.Context, tx pgx.Tx) error {
		// No truncation: idempotent add-only behavior

		// Build map for quick lookup by JSON ID
		byID := make(map[int]RawCatalogItem, len(items))
		for _, it := range items {
			byID[it.ID] = it
		}

		resolved := make(map[int]int64)
		stats := &catalogItemsStats{}
		var processed int
		for _, it := range items {
			if _, err := ensureItem(ctx, tx, it.ID, byID, resolved, stats); err != nil {
				return err
			}
			processed++
		}
		logger.Info("catalog_items procesados",
			"count", processed,
			"inserted", stats.Inserted,
			"reused", stats.Reused,
		)
		return nil
	})
}

// ensureItem guarantees that the JSON item with given id exists in DB, inserting it if missing.
// Returns the DB id of the record.
func ensureItem(ctx context.Context, tx pgx.Tx, jsonID int, source map[int]RawCatalogItem, cache map[int]int64, stats *catalogItemsStats) (int64, error) {
	if v, ok := cache[jsonID]; ok {
		return v, nil
	}

	it, ok := source[jsonID]
	if !ok {
		return 0, fmt.Errorf("json id %d no encontrado en el archivo", jsonID)
	}

	// Resolve parent first if needed
	var parentDBID *int64
	if it.ParentJSONID != 0 {
		parentID, err := ensureItem(ctx, tx, it.ParentJSONID, source, cache, stats)
		if err != nil {
			return 0, fmt.Errorf("resolver padre para id %d: %w", jsonID, err)
		}
		parentDBID = &parentID
	}

	code, err := codeToString(it.CodeRaw)
	if err != nil {
		return 0, fmt.Errorf("code inválido para json id %d: %w", jsonID, err)
	}
	// Check existence by unique key (code, parent_id) where deleted_at IS NULL
	var existingID int64
	err = tx.QueryRow(ctx,
		`SELECT id FROM catalog_items 
         WHERE code=$1 
           AND ((parent_id IS NULL AND $2::bigint IS NULL) OR parent_id=$2) 
           AND deleted_at IS NULL`,
		code, parentDBID,
	).Scan(&existingID)
	if err == nil {
		cache[jsonID] = existingID
		if stats != nil {
			stats.Reused++
		}
		return existingID, nil
	}
	if err != nil && err != pgx.ErrNoRows {
		return 0, fmt.Errorf("consultar existencia (code=%s): %w", code, err)
	}

	// Build metadata to keep original fields
	meta := map[string]any{
		"source_id":      it.ID,
		"sede_id":        it.SedeID,
		"order":          it.Order,
		"parent_json_id": it.ParentJSONID,
	}
	metaBytes, _ := json.Marshal(meta)

	isActive := it.Active == 1

	// Insert missing
	var newID int64
	if err := tx.QueryRow(ctx,
		`INSERT INTO catalog_items (name, code, parent_id, sort_order, is_active, metadata)
         VALUES ($1, $2, $3, $4, $5, $6)
         RETURNING id`,
		it.Name, code, parentDBID, it.Order, isActive, metaBytes,
	).Scan(&newID); err != nil {
		return 0, fmt.Errorf("insertar catalog_item (code=%s): %w", code, err)
	}

	cache[jsonID] = newID
	if stats != nil {
		stats.Inserted++
	}
	return newID, nil
}

func codeToString(raw json.RawMessage) (string, error) {
	var asString string
	if err := json.Unmarshal(raw, &asString); err == nil {
		return strings.TrimSpace(asString), nil
	}
	var asInt int
	if err := json.Unmarshal(raw, &asInt); err == nil {
		return strconv.Itoa(asInt), nil
	}
	var asFloat float64
	if err := json.Unmarshal(raw, &asFloat); err == nil {
		return strconv.Itoa(int(asFloat)), nil
	}
	return "", fmt.Errorf("no soportado")
}
