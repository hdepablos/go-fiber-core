package seeders

import (
	"context"
	"fmt"
	"log/slog"

	pgx "github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ProcessLifecycleManagerSeeder seeds the process lifecycle tables and exercises
// the core PL/pgSQL functions for replication and promotion.
func ProcessLifecycleManagerSeeder(pool *pgxpool.Pool) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultSeederTimeout)
	defer cancel()

	logger := slog.Default().With("seeder", "process_lifecycle_manager")
	logger.Info("iniciando seeder de process_lifecycle_manager")

	if err := executeInTransaction(ctx, pool, func(ctx context.Context, tx pgx.Tx) error {
		for _, table := range []string{
			"process_version_history",
			"process_steps",
			"process_versions",
			"process_types",
		} {
			if err := truncateTable(ctx, tx, table); err != nil {
				return fmt.Errorf("truncate %s: %w", table, err)
			}
		}

		var processTypeID int64
		if err := tx.QueryRow(ctx,
			`INSERT INTO process_types (name, description, is_visible)
             VALUES ($1, $2, $3)
             RETURNING id`,
			"Order process lifecycle",
			"Base process type for order lifecycle testing",
			true,
		).Scan(&processTypeID); err != nil {
			return fmt.Errorf("insert process_types: %w", err)
		}

		var baseVersionID int64
		if err := tx.QueryRow(ctx,
			`INSERT INTO process_versions (process_type_id, version_number, status, operator_id)
             VALUES ($1, $2, $3, $4)
             RETURNING id`,
			processTypeID,
			1,
			"DRAFT",
			1,
		).Scan(&baseVersionID); err != nil {
			return fmt.Errorf("insert process_versions (TEST): %w", err)
		}

		type stepDef struct {
			Order        int
			Name         string
			ExecutionKey string
			Config       string
		}

		steps := []stepDef{
			{Order: 1, Name: "Validate input", ExecutionKey: "validate_input", Config: "{}"},
			{Order: 2, Name: "Apply business rules", ExecutionKey: "apply_business_rules", Config: "{}"},
			{Order: 3, Name: "Persist results", ExecutionKey: "persist_results", Config: "{}"},
		}

		for _, s := range steps {
			if _, err := tx.Exec(ctx,
				`INSERT INTO process_steps (process_version_id, step_order, name, execution_key, config)
                 VALUES ($1, $2, $3, $4, $5::jsonb)`,
				baseVersionID,
				s.Order,
				s.Name,
				s.ExecutionKey,
				s.Config,
			); err != nil {
				return fmt.Errorf("insert process_steps (order=%d): %w", s.Order, err)
			}
		}

		var newVersionID int64
		if err := tx.QueryRow(ctx,
			`SELECT replicate_process_version($1, $2)`,
			baseVersionID,
			1,
		).Scan(&newVersionID); err != nil {
			return fmt.Errorf("replicate_process_version: %w", err)
		}

		var loanProcessTypeID int64
		if err := tx.QueryRow(ctx,
			`INSERT INTO process_types (name, description, is_visible)
             VALUES ($1, $2, $3)
             RETURNING id`,
			"Loan risk lifecycle",
			"Lifecycle configurado para servicios de loanrisk",
			true,
		).Scan(&loanProcessTypeID); err != nil {
			return fmt.Errorf("insert process_types loanrisk: %w", err)
		}

		var loanBaseVersionID int64
		if err := tx.QueryRow(ctx,
			`INSERT INTO process_versions (process_type_id, version_number, status, operator_id)
             VALUES ($1, $2, $3, $4)
             RETURNING id`,
			loanProcessTypeID,
			1,
			"DRAFT",
			1,
		).Scan(&loanBaseVersionID); err != nil {
			return fmt.Errorf("insert process_versions loanrisk: %w", err)
		}

		loanSteps := []stepDef{
			{
				Order:        1,
				Name:         "Age validation",
				ExecutionKey: "loanrisk/NewAgeService",
				Config:       `{"error_tolerance":"inherit","required_keys":["age"],"min_age":40}`,
			},
			{
				Order:        2,
				Name:         "Special validation",
				ExecutionKey: "loanrisk/NewValidationService",
				Config:       `{"error_tolerance":"tolerable"}`,
			},
			{
				Order:        3,
				Name:         "Salary validation",
				ExecutionKey: "loanrisk/NewSalaryService",
				Config:       `{"error_tolerance":"critical","required_keys":["salary"],"min_salary":2500000}`,
			},
			{
				Order:        4,
				Name:         "Renovation check",
				ExecutionKey: "loanrisk/NewIsRenovationService",
				Config:       `{"error_tolerance":"inherit","required_keys":["min_salary","salary_bracket_k_usd","salary_checked"]}`,
			},
			{
				Order:        5,
				Name:         "Risk level",
				ExecutionKey: "loanrisk/NewRiskLevelService",
				Config:       `{"error_tolerance":"inherit","required_keys":["is_renovation"]}`,
			},
		}

		for _, s := range loanSteps {
			if _, err := tx.Exec(ctx,
				`INSERT INTO process_steps (process_version_id, step_order, name, execution_key, config)
                 VALUES ($1, $2, $3, $4, $5::jsonb)`,
				loanBaseVersionID,
				s.Order,
				s.Name,
				s.ExecutionKey,
				s.Config,
			); err != nil {
				return fmt.Errorf("insert process_steps loanrisk (order=%d): %w", s.Order, err)
			}
		}

		var loanNewVersionID int64
		if err := tx.QueryRow(ctx,
			`SELECT replicate_process_version($1, $2)`,
			loanBaseVersionID,
			1,
		).Scan(&loanNewVersionID); err != nil {
			return fmt.Errorf("replicate_process_version loanrisk: %w", err)
		}

		return nil
	}); err != nil {
		logger.Error("seeder process_lifecycle_manager falló", "error", err)
		return err
	}

	logger.Info("seeder process_lifecycle_manager completado exitosamente")
	return nil
}
