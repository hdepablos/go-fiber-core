package seeders

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"go-fiber-core/internal/database/connections/pgx"
	"go-fiber-core/internal/dtos/config"
)

const (
	defaultConfigPath = "internal/appconfig/config.yml"
	seedTimeout       = 5 * time.Minute
)

type Seeder func() error

type SeederService struct {
	seeders []SeederConfig
	logger  *slog.Logger
}

type SeederConfig struct {
	Name   string
	Seeder Seeder
}

func ListSeedersNames() []string {
	return []string{
		"banks",
		"roles",
		"menus",
		"catalog_items",
		"create_test_user",
		"role_user",
		"menu_user",
		"process_lifecycle_manager",
		"test_process_scenarios",
	}
}

func NewSeederService(logger *slog.Logger) *SeederService {
	if logger == nil {
		logger = slog.Default()
	}

	return &SeederService{
		seeders: make([]SeederConfig, 0),
		logger:  logger.With("component", "seeder_service"),
	}
}

func (s *SeederService) AddSeeder(name string, seeder Seeder) {
	s.seeders = append(s.seeders, SeederConfig{
		Name:   name,
		Seeder: seeder,
	})
}

func (s *SeederService) Run(ctx context.Context) error {
	if len(s.seeders) == 0 {
		s.logger.Warn("no hay seeders registrados")
		return nil
	}

	s.logger.Info("iniciando ejecución de seeders", "total", len(s.seeders))
	startTime := time.Now()

	for i, sc := range s.seeders {
		seederLogger := s.logger.With("seeder", sc.Name, "index", i+1)
		seederLogger.Info("ejecutando seeder")

		seederStart := time.Now()
		if err := sc.Seeder(); err != nil {
			seederLogger.Error("seeder falló", "error", err, "duration", time.Since(seederStart))
			return fmt.Errorf("seeder '%s' falló: %w", sc.Name, err)
		}

		seederLogger.Info("seeder completado", "duration", time.Since(seederStart))
	}

	s.logger.Info("todos los seeders completados exitosamente",
		"total", len(s.seeders),
		"duration", time.Since(startTime))

	return nil
}

func (s *SeederService) RunSelected(ctx context.Context, names []string) error {
	if len(names) == 0 {
		return s.Run(ctx)
	}

	selected := make(map[string]struct{}, len(names))
	for _, n := range names {
		selected[n] = struct{}{}
	}

	filtered := make([]SeederConfig, 0, len(names))
	for _, sc := range s.seeders {
		if _, ok := selected[sc.Name]; ok {
			filtered = append(filtered, sc)
		}
	}

	if len(filtered) == 0 {
		s.logger.Warn("no se encontraron seeders para los nombres solicitados", "requested", names)
		return nil
	}

	s.logger.Info("iniciando ejecución de seeders filtrados",
		"total", len(filtered),
		"requested", names)
	startTime := time.Now()

	for i, sc := range filtered {
		seederLogger := s.logger.With("seeder", sc.Name, "index", i+1)
		seederLogger.Info("ejecutando seeder")

		seederStart := time.Now()
		if err := sc.Seeder(); err != nil {
			seederLogger.Error("seeder falló", "error", err, "duration", time.Since(seederStart))
			return fmt.Errorf("seeder '%s' falló: %w", sc.Name, err)
		}

		seederLogger.Info("seeder completado", "duration", time.Since(seederStart))
	}

	s.logger.Info("seeders filtrados completados exitosamente",
		"total", len(filtered),
		"duration", time.Since(startTime))

	return nil
}

func SeedDatabase(only ...string) error {
	// Setup structured logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// Parse configuration path from flags
	configPath := flag.String("config", defaultConfigPath, "ruta al archivo de configuración YAML")
	flag.Parse()

	logger.Info("iniciando proceso de seeding", "config_path", *configPath)

	// Load application configuration
	appConfig, err := config.NewAppConfig(*configPath)
	if err != nil {
		return fmt.Errorf("cargar configuración: %w", err)
	}

	// Create context with timeout for the entire seeding process
	ctx, cancel := context.WithTimeout(context.Background(), seedTimeout)
	defer cancel()

	// Establish database connection pool
	dbPool, cleanup, err := pgx.NewPgxConnection(appConfig.MultiDatabaseConfig.Pgx.Write)
	if err != nil {
		return fmt.Errorf("conectar a base de datos: %w", err)
	}
	defer cleanup()

	logger.Info("conexión a base de datos establecida")

	service := NewSeederService(logger)

	registerSeeders(service, dbPool, *configPath)

	if err := service.RunSelected(ctx, only); err != nil {
		logger.Error("error ejecutando seeders", "error", err)
		return err
	}

	logger.Info("proceso de seeding finalizado exitosamente")
	return nil
}

// registerSeeders registers all available seeders with the service.
// This function centralizes seeder registration for better maintainability.
//
// Seeder execution order matters when there are foreign key dependencies:
// 1. Base tables (banks, roles, menus) - no dependencies
// 2. User creation - may depend on roles
// 3. Relationship tables - depend on both users and other entities
func registerSeeders(service *SeederService, dbPool interface{}, configPath string) {
	// Cast dbPool to the correct type for CSV/JSON-based seeders
	pool, ok := dbPool.(*pgxpool.Pool)
	if !ok {
		panic("invalid pool type")
	}

	// ═══════════════════════════════════════════════════════════════
	// PHASE 1: Base Tables (no dependencies)
	// ═══════════════════════════════════════════════════════════════

	service.AddSeeder("banks", func() error {
		return BankSeeder(pool)
	})

	service.AddSeeder("roles", func() error {
		return RoleSeeder(pool)
	})

	service.AddSeeder("menus", func() error {
		return MenuSeeder(pool)
	})

	service.AddSeeder("catalog_items", func() error {
		return CatalogItemsSeeder(pool)
	})

	// ═══════════════════════════════════════════════════════════════
	// PHASE 2: Users (requires DI container for services)
	// ═══════════════════════════════════════════════════════════════

	service.AddSeeder("create_test_user", func() error {
		return CreateUserSeeder(configPath)
	})

	// Example: Create additional users with different roles
	// service.AddSeeder("create_coord_user", func() error {
	//     return CreateUserSeederWithCustomData(configPath, "Coordinador", "coord@test.com", "coord123")
	// })
	//
	// service.AddSeeder("create_super_user", func() error {
	//     return CreateUserSeederWithCustomData(configPath, "Supervisor", "super@test.com", "super123")
	// })
	//
	// service.AddSeeder("create_operator_user", func() error {
	//     return CreateUserSeederWithCustomData(configPath, "Operador", "operator@test.com", "op123")
	// })

	// ═══════════════════════════════════════════════════════════════
	// PHASE 3: Relationship Tables (depend on users and other entities)
	// ═══════════════════════════════════════════════════════════════

	// Assign role to user
	// This creates: user_id=1 → role_id=1 (Admin)
	service.AddSeeder("role_user", func() error {
		return RoleUserSeeder(pool)
	})

	// Assign menus to users based on their role templates
	// User 1 has role "Admin", so will get all 15 menus
	// The seeder automatically:
	// 1. Queries the user's role from role_user table
	// 2. Looks up the menu template for that role
	// 3. Inserts all menu permissions for the user
	service.AddSeeder("menu_user", func() error {
		return MenuUserSeeder(pool)
	})

	// ═══════════════════════════════════════════════════════════════
	// PHASE 4: Process lifecycle (versioned workflows)
	// ═══════════════════════════════════════════════════════════════

	service.AddSeeder("process_lifecycle_manager", func() error {
		return ProcessLifecycleManagerSeeder(pool)
	})

	service.AddSeeder("test_process_scenarios", func() error {
		return TestProcessScenariosSeeder(pool)
	})

	// Example: Seed menus for multiple users at once
	// Uncomment this if you created multiple users above
	// service.AddSeeder("menu_user_multiple", func() error {
	//     return MenuUserSeederForMultipleUsers(pool, []uint{1, 2, 3, 4})
	// })
}
