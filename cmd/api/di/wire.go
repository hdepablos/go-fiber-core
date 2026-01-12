//go:build wireinject
// +build wireinject

package di

import (
	"log"
	"os"

	"go-fiber-core/internal/database/connections/gorm"
	"go-fiber-core/internal/database/connections/pgx"
	redis2 "go-fiber-core/internal/database/connections/redis"
	"go-fiber-core/internal/dtos/config"
	"go-fiber-core/internal/dtos/connect"
	"go-fiber-core/internal/handlers"
	"go-fiber-core/internal/models"
	"go-fiber-core/internal/repositories/bank"
	"go-fiber-core/internal/repositories/menu"
	"go-fiber-core/internal/repositories/refreshtoken"
	"go-fiber-core/internal/repositories/user"
	"go-fiber-core/internal/server"
	"go-fiber-core/internal/services"
	"go-fiber-core/internal/services/auth"
	bank2 "go-fiber-core/internal/services/bank"
	menu2 "go-fiber-core/internal/services/menu"
	"go-fiber-core/internal/services/pagination"
	user2 "go-fiber-core/internal/services/user"

	"github.com/google/wire"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
)

type PgxWritePool *pgxpool.Pool
type PgxReadPool *pgxpool.Pool

// AppContainer contiene la lógica de negocio completa sin el servidor HTTP
type AppContainer struct {
	Config            *config.AppConfig
	Connect           *connect.ConnectDTO
	UserWriterService user2.UserWriterService
	UserReaderService user2.UserReaderService
	BankWriterService bank2.BankWriterService
	BankReaderService bank2.BankReaderService
	AuthService       auth.AuthService
	DatabaseService   *services.DatabaseService
}

// AWSConfigResponse se mantiene para compatibilidad si lo usas en otros lados
type AWSConfigResponse struct {
	Config  *config.AppConfig
	Connect *connect.ConnectDTO
}

// ──────────────────────────────
// INITIALIZERS
// ──────────────────────────────

// InitializeServer para API (Local/Docker)
func InitializeServer(configPath string) (*server.FiberServer, func(), error) {
	wire.Build(
		provideAppConfigServer,
		connectionSet,
		repositorySet,
		serviceSet,
		handlerSet,
		server.NewFiberServer,
	)
	return nil, nil, nil
}

// InitializeAppContainer para Crons, Lambdas y CLI (Lógica pura)
func InitializeAppContainer(configPath string) (*AppContainer, func(), error) {
	wire.Build(
		provideAppConfigAWS, // Usa validación de ENV sin .env
		connectionSet,
		repositorySet,
		serviceSet,
		wire.Struct(new(AppContainer), "*"),
	)
	return nil, nil, nil
}

// InitializeAWS (Mantenido por compatibilidad simple)
func InitializeAWS(configPath string) (*AWSConfigResponse, func(), error) {
	wire.Build(
		provideAppConfigAWS,
		connectionSet,
		wire.Struct(new(AWSConfigResponse), "*"),
	)
	return nil, nil, nil
}

// ──────────────────────────────
// CONFIG PROVIDERS
// ──────────────────────────────

func provideAppConfigServer(configPath string) (*config.AppConfig, error) {
	_ = godotenv.Load()
	return config.NewAppConfig(configPath)
}

func provideAppConfigAWS(configPath string) (*config.AppConfig, error) {
	_ = godotenv.Load()
	validateLambdaEnv()
	return config.NewAppConfig(configPath)
}

func validateLambdaEnv() {
	required := []string{"JWT_ACCESS_SECRET", "JWT_REFRESH_SECRET"}
	for _, v := range required {
		if os.Getenv(v) == "" {
			log.Fatalf("❌ Variable de entorno requerida no definida: %s", v)
		}
	}
}

// ──────────────────────────────
// CONNECTION PROVIDERS
// ──────────────────────────────

func provideGormService(cfg *config.AppConfig) (*gorm.GormConnectService, func(), error) {
	return gorm.NewGormConnectService(cfg.MultiDatabaseConfig)
}

func provideRedisClient(cfg *config.AppConfig) (*redis.Client, func(), error) {
	return redis2.NewRedisClient(cfg.Redis)
}

func providePgxWritePool(cfg *config.AppConfig) (PgxWritePool, func(), error) {
	pool, cleanup, err := pgx.NewPgxConnection(cfg.MultiDatabaseConfig.Pgx.Write)
	return PgxWritePool(pool), cleanup, err
}

func providePgxReadPool(cfg *config.AppConfig) (PgxReadPool, func(), error) {
	pool, cleanup, err := pgx.NewPgxConnection(cfg.MultiDatabaseConfig.Pgx.Read)
	return PgxReadPool(pool), cleanup, err
}

func provideConnectDTO(gormService *gorm.GormConnectService, w PgxWritePool, r PgxReadPool, rd *redis.Client) *connect.ConnectDTO {
	return &connect.ConnectDTO{
		ConnectGormWrite: gormService.GetWriteDB(),
		ConnectGormRead:  gormService.GetReadDB(),
		ConnectPgxWrite:  (*pgxpool.Pool)(w),
		ConnectPgxRead:   (*pgxpool.Pool)(r),
		ConnectRedis:     rd,
	}
}

// ──────────────────────────────
// SERVICE PROVIDERS
// ──────────────────────────────

func provideTokenService(cfg *config.AppConfig) auth.TokenService {
	return auth.NewTokenService(cfg)
}

func provideUserPaginationService() *pagination.PaginationService[models.User] {
	return pagination.NewPaginationService[models.User]()
}

func provideBankPaginationService() *pagination.PaginationService[models.Bank] {
	return pagination.NewPaginationService[models.Bank]()
}

// ──────────────────────────────
// SETS
// ──────────────────────────────

var connectionSet = wire.NewSet(
	provideGormService,
	provideRedisClient,
	providePgxWritePool,
	providePgxReadPool,
	provideConnectDTO,
)

var repositorySet = wire.NewSet(
	user.NewUserReaderRepo, user.NewUserWriterRepo, user.NewUserPaginatorRepo, user.NewUserRepository,
	bank.NewBankReaderRepo, bank.NewBankWriterRepo, bank.NewBankCrudRepository, bank.NewBankPaginationRepo,
	refreshtoken.NewRefreshTokenReaderRepo, refreshtoken.NewRefreshTokenWriterRepo, refreshtoken.NewRefreshTokenRepository,
	menu.NewMenuReaderRepository,
)

var serviceSet = wire.NewSet(
	provideTokenService, auth.NewAuthService,
	provideUserPaginationService, provideBankPaginationService,
	services.NewTransactionManager, services.NewDatabaseService,
	user2.NewUserReaderService, user2.NewUserWriterService,
	bank2.NewBankReaderService, bank2.NewBankWriterService, bank2.NewBankPaginationService, bank2.NewDeactivationService,
	menu2.NewMenuReaderService,
)

var handlerSet = wire.NewSet(
	handlers.NewAuthHandler, handlers.NewUserHandler, handlers.NewBankHandler, handlers.NewDatabaseHandler,
)
