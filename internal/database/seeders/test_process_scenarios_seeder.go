package seeders

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TestProcessScenariosSeeder creates a process type with concurrent steps
// to demonstrate parallel execution capabilities and multi-sede logic.
func TestProcessScenariosSeeder(pool *pgxpool.Pool) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultSeederTimeout)
	defer cancel()

	logger := slog.Default().With("seeder", "test_process_scenarios")
	logger.Info("iniciando seeder de test_process_scenarios")

	return executeInTransaction(ctx, pool, func(ctx context.Context, tx pgx.Tx) error {
		// =================================================================================
		// PARTE 1: Proceso de Steps Concurrentes (Original)
		// =================================================================================
		if err := seedConcurrentProcess(ctx, tx, logger); err != nil {
			return err
		}

		// =================================================================================
		// PARTE 2: Proceso Multi-Sede (Nuevo - Lógica de resolución)
		// =================================================================================
		if err := seedMultiSedeProcess(ctx, tx, logger); err != nil {
			return err
		}

		return nil
	})
}

func seedConcurrentProcess(ctx context.Context, tx pgx.Tx, logger *slog.Logger) error {
	// 1. Crear o Buscar Process Type
	var processTypeID int64
	const processTypeName = "Test Proceso de steps concurrente"
	
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
				return fmt.Errorf("insert process_types concurrent: %w", err)
			}
			logger.Info("Process Type (Concurrent) creado", "id", processTypeID)
		} else {
			return fmt.Errorf("select process_types concurrent: %w", err)
		}
	} else {
		logger.Info("Process Type (Concurrent) existente", "id", processTypeID)
	}

	// 2. Crear o Buscar Versión Global en PROD
	var versionID int64
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
				1,
			).Scan(&versionID)
			if err != nil {
				return fmt.Errorf("insert process_versions concurrent: %w", err)
			}
			logger.Info("Process Version (Concurrent) creada", "id", versionID)
		} else {
			return fmt.Errorf("select process_versions concurrent: %w", err)
		}
	} else {
		logger.Info("Process Version (Concurrent) existente", "id", versionID)
	}

	// 3. Registrar en Historial
	if err := ensureHistory(ctx, tx, versionID, processTypeID, "Initial concurrent process seed", logger); err != nil {
		return err
	}

	// 4. Upsert Steps
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
		if err := upsertStep(ctx, tx, versionID, s.Order, s.Name, s.ExecutionKey, s.Config); err != nil {
			return err
		}
	}

	return nil
}

func seedMultiSedeProcess(ctx context.Context, tx pgx.Tx, logger *slog.Logger) error {
	// 1. Crear o Buscar Process Type
	var processTypeID int64
	const processTypeName = "Test Multi-Sede Logic"
	
	err := tx.QueryRow(ctx, "SELECT id FROM process_types WHERE name = $1", processTypeName).Scan(&processTypeID)
	if err != nil {
		if err == pgx.ErrNoRows {
			err = tx.QueryRow(ctx,
				`INSERT INTO process_types (name, description, is_visible)
				 VALUES ($1, $2, $3)
				 RETURNING id`,
				processTypeName,
				"Proceso para demostrar resolución de versiones (Global vs Sede)",
				true,
			).Scan(&processTypeID)
			if err != nil {
				return fmt.Errorf("insert process_types multisede: %w", err)
			}
			logger.Info("Process Type (MultiSede) creado", "id", processTypeID)
		} else {
			return fmt.Errorf("select process_types multisede: %w", err)
		}
	} else {
		logger.Info("Process Type (MultiSede) existente", "id", processTypeID)
	}

	// ============================================================
	// CASO A: Versión Global (Sede NULL) -> Estándar
	// ============================================================
	var globalVersionID int64
	err = tx.QueryRow(ctx, 
		"SELECT id FROM process_versions WHERE process_type_id = $1 AND version_number = 1 AND sede_id IS NULL", 
		processTypeID,
	).Scan(&globalVersionID)

	if err != nil {
		if err == pgx.ErrNoRows {
			err = tx.QueryRow(ctx,
				`INSERT INTO process_versions (process_type_id, version_number, status, operator_id, sede_id)
				 VALUES ($1, $2, $3, $4, NULL)
				 RETURNING id`,
				processTypeID,
				1,
				"PROD",
				1,
			).Scan(&globalVersionID)
			if err != nil {
				return fmt.Errorf("insert global version: %w", err)
			}
			logger.Info("Global Version (MultiSede) creada", "id", globalVersionID)
		} else {
			return fmt.Errorf("select global version: %w", err)
		}
	}

	if err := ensureHistory(ctx, tx, globalVersionID, processTypeID, "Standard Global Flow", logger); err != nil {
		return err
	}

	// Steps Globales (Reusamos concurrent steps solo como placeholders)
	globalSteps := []struct{Order int; Name string; Key string}{
		{1, "Standard Step A", "test/concurrent/step1"},
		{2, "Standard Step B", "test/concurrent/step2"},
	}
	for _, s := range globalSteps {
		if err := upsertStep(ctx, tx, globalVersionID, s.Order, s.Name, s.Key, "{}"); err != nil {
			return err
		}
	}

	// ============================================================
	// CASO B: Versión Específica (Sede 2 - Galicia) -> Custom
	// ============================================================
	targetSedeID := 2
	var customVersionID int64
	err = tx.QueryRow(ctx, 
		"SELECT id FROM process_versions WHERE process_type_id = $1 AND version_number = 2 AND sede_id = $2", 
		processTypeID, targetSedeID,
	).Scan(&customVersionID)

	if err != nil {
		if err == pgx.ErrNoRows {
			err = tx.QueryRow(ctx,
				`INSERT INTO process_versions (process_type_id, version_number, status, operator_id, sede_id)
				 VALUES ($1, $2, $3, $4, $5)
				 RETURNING id`,
				processTypeID,
				2, // Version 2 (arbitrario, podría ser 1 pero con sede_id)
				"PROD",
				1,
				targetSedeID,
			).Scan(&customVersionID)
			if err != nil {
				return fmt.Errorf("insert custom version: %w", err)
			}
			logger.Info("Custom Version (MultiSede) creada", "id", customVersionID, "sede_id", targetSedeID)
		} else {
			return fmt.Errorf("select custom version: %w", err)
		}
	}

	if err := ensureHistory(ctx, tx, customVersionID, processTypeID, "Custom Flow for Galicia", logger); err != nil {
		return err
	}

	// Steps Custom (Más pasos o diferentes)
	customSteps := []struct{Order int; Name string; Key string}{
		{1, "Custom Step A", "test/concurrent/step1"},
		{2, "Custom Step B", "test/concurrent/step2"},
		{3, "Custom Step C (Extra)", "test/concurrent/step3"},
	}
	for _, s := range customSteps {
		if err := upsertStep(ctx, tx, customVersionID, s.Order, s.Name, s.Key, "{}"); err != nil {
			return err
		}
	}

	return nil
}

// Helpers

func ensureHistory(ctx context.Context, tx pgx.Tx, versionID, typeID int64, comment string, logger *slog.Logger) error {
	var exists bool
	err := tx.QueryRow(ctx, 
		"SELECT EXISTS(SELECT 1 FROM process_version_history WHERE process_version_id = $1)", 
		versionID,
	).Scan(&exists)
	if err != nil {
		return fmt.Errorf("check history: %w", err)
	}

	if !exists {
		_, err := tx.Exec(ctx,
			`INSERT INTO process_version_history (
				process_version_id, process_type_id, promoted_from_status, promoted_at, promoted_by, comment
			) VALUES ($1, $2, $3, NOW(), $4, $5)`,
			versionID, typeID, "DRAFT", 1, comment,
		)
		if err != nil {
			return fmt.Errorf("insert history: %w", err)
		}
		logger.Info("Historial insertado", "version_id", versionID)
	}
	return nil
}

func upsertStep(ctx context.Context, tx pgx.Tx, versionID int64, order int, name, key, config string) error {
	cmdTag, err := tx.Exec(ctx,
		`UPDATE process_steps 
		 SET step_order = $1, name = $2, config = $3::jsonb, roadmap = 0
		 WHERE process_version_id = $4 AND execution_key = $5`,
		order, name, config, versionID, key,
	)
	if err != nil {
		return fmt.Errorf("update step %s: %w", key, err)
	}

	if cmdTag.RowsAffected() == 0 {
		_, err := tx.Exec(ctx,
			`INSERT INTO process_steps (process_version_id, step_order, name, execution_key, config, roadmap)
			 VALUES ($1, $2, $3, $4, $5::jsonb, 0)`,
			versionID, order, name, key, config,
		)
		if err != nil {
			return fmt.Errorf("insert step %s: %w", key, err)
		}
	}
	return nil
}
