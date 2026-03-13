package seeders

import (
	"context"
	"fmt"
	"log/slog"

	pgx "github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ConcurrentProcessSeeder creates a process type with concurrent steps
// to demonstrate parallel execution capabilities.
func ConcurrentProcessSeeder(pool *pgxpool.Pool) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultSeederTimeout)
	defer cancel()

	logger := slog.Default().With("seeder", "concurrent_process")
	logger.Info("iniciando seeder de concurrent_process")

	return executeInTransaction(ctx, pool, func(ctx context.Context, tx pgx.Tx) error {
		// 1. Crear o Buscar Process Type
		var processTypeID int64
		const processTypeName = "Proceso de steps concurrente"

		err := tx.QueryRow(ctx, "SELECT id FROM process_types WHERE name = $1", processTypeName).Scan(&processTypeID)
		if err != nil {
			if err == pgx.ErrNoRows {
				// No existe, insertar
				err = tx.QueryRow(ctx,
					`INSERT INTO process_types (name, description, is_visible)
						VALUES ($1, $2, $3)
						RETURNING id`,
					processTypeName,
					"Proceso para demostrar ejecución concurrente de pasos",
					true,
				).Scan(&processTypeID)
				if err != nil {
					return fmt.Errorf("insert process_types: %w", err)
				}
				logger.Info("Process Type creado", "id", processTypeID)
			} else {
				return fmt.Errorf("select process_types: %w", err)
			}
		} else {
			logger.Info("Process Type existente", "id", processTypeID)
		}

		// 2. Crear o Buscar Versión Global en PROD
		var versionID int64
		// Buscamos la versión 1 global para este tipo
		err = tx.QueryRow(ctx,
			"SELECT id FROM process_versions WHERE process_type_id = $1 AND version_number = 1 AND sede_id IS NULL",
			processTypeID,
		).Scan(&versionID)

		if err != nil {
			if err == pgx.ErrNoRows {
				// No existe, insertar
				err = tx.QueryRow(ctx,
					`INSERT INTO process_versions (process_type_id, version_number, status, operator_id, sede_id)
						VALUES ($1, $2, $3, $4, NULL)
						RETURNING id`,
					processTypeID,
					1,
					"PROD",
					1, // Operator ID 1 (admin/system)
				).Scan(&versionID)
				if err != nil {
					return fmt.Errorf("insert process_versions: %w", err)
				}
				logger.Info("Process Version creada", "id", versionID)
			} else {
				return fmt.Errorf("select process_versions: %w", err)
			}
		} else {
			logger.Info("Process Version existente", "id", versionID)
		}

		// 4. Registrar en Historial (Si no existe)
		var historyExists bool
		err = tx.QueryRow(ctx,
			"SELECT EXISTS(SELECT 1 FROM process_version_history WHERE process_version_id = $1)",
			versionID,
		).Scan(&historyExists)
		if err != nil {
			return fmt.Errorf("check process_version_history: %w", err)
		}

		if !historyExists {
			if _, err := tx.Exec(ctx,
				`INSERT INTO process_version_history (
					process_version_id,
					process_type_id,
					promoted_from_status,
					promoted_at,
					promoted_by,
					comment
				) VALUES ($1, $2, $3, NOW(), $4, $5)`,
				versionID,
				processTypeID,
				"DRAFT", // promoted_from_status
				1,       // promoted_by (operator_id)
				"Initial concurrent process seed",
			); err != nil {
				return fmt.Errorf("insert process_version_history: %w", err)
			}
			logger.Info("Historial insertado", "version_id", versionID)
		} else {
			logger.Info("Historial ya existente", "version_id", versionID)
		}

		// 3. Upsert Steps
		// Configuración solicitada: 4 primeros concurrentes, último secuencial.
		// Esto significa:
		// Steps 1-4: Order 1 (se ejecutan en paralelo)
		// Step 5: Order 2 (se ejecuta después de que terminen los del Order 1)
		// Tiempo esperado: max(1s, 1s, 1s, 1s) + 1s = 2s.

		// Para cambiar a secuencial (tiempo 5s), cambiar los Orders a 1, 2, 3, 4, 5.

		type stepDef struct {
			Order        int
			Name         string
			ExecutionKey string
			Config       string
		}

		steps := []stepDef{
			{Order: 1, Name: "Step 1 (Concurrent)", ExecutionKey: "test/concurrent/step1", Config: "{}"},
			{Order: 1, Name: "Step 2 (Concurrent)", ExecutionKey: "test/concurrent/step2", Config: "{}"},
			{Order: 1, Name: "Step 3 (Concurrent)", ExecutionKey: "test/concurrent/step3", Config: "{}"},
			{Order: 1, Name: "Step 4 (Concurrent)", ExecutionKey: "test/concurrent/step4", Config: "{}"},
			{Order: 2, Name: "Step 5 (Sequential)", ExecutionKey: "test/concurrent/step5", Config: "{}"},
		}

		for _, s := range steps {
			// Intentar actualizar primero
			cmdTag, err := tx.Exec(ctx,
				`UPDATE process_steps
					SET step_order = $1, name = $2, config = $3::jsonb, roadmap = 0
					WHERE process_version_id = $4 AND execution_key = $5`,
				s.Order, s.Name, s.Config, versionID, s.ExecutionKey,
			)
			if err != nil {
				return fmt.Errorf("update process_steps (key=%s): %w", s.ExecutionKey, err)
			}

			if cmdTag.RowsAffected() == 0 {
				// No existía, insertar
				if _, err := tx.Exec(ctx,
					`INSERT INTO process_steps (process_version_id, step_order, name, execution_key, config, roadmap)
						VALUES ($1, $2, $3, $4, $5::jsonb, 0)`,
					versionID,
					s.Order,
					s.Name,
					s.ExecutionKey,
					s.Config,
				); err != nil {
					return fmt.Errorf("insert process_steps (key=%s): %w", s.ExecutionKey, err)
				}
			}
		}

		logger.Info("Seeder completado exitosamente", "process_type_id", processTypeID, "version_id", versionID)
		return nil
	})
}
