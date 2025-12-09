//go:build wireinject
// +build wireinject

package di

import (
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
	"github.com/redis/go-redis/v9"
)

type PgxWritePool *pgxpool.Pool
type PgxReadPool *pgxpool.Pool

func InitializeServer(configPath string) (*server.FiberServer, func(), error) {
	wire.Build(
		provideAppConfig,
		connectionSet,
		repositorySet,
		serviceSet,
		handlerSet,
		server.NewFiberServer,
	)
	return nil, nil, nil
}

// --- PROVIDERS ---

func provideAppConfig(configPath string) (*config.AppConfig, error) {
	return config.NewAppConfig(configPath)
}

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

func provideConnectDTO(
	gormService *gorm.GormConnectService,
	writePool PgxWritePool,
	readPool PgxReadPool,
	redisClient *redis.Client,
) *connect.ConnectDTO {
	return &connect.ConnectDTO{
		ConnectGormWrite: gormService.GetWriteDB(),
		ConnectGormRead:  gormService.GetReadDB(),
		ConnectPgxWrite:  (*pgxpool.Pool)(writePool),
		ConnectPgxRead:   (*pgxpool.Pool)(readPool),
		ConnectRedis:     redisClient,
	}
}

func provideTokenService(cfg *config.AppConfig) auth.TokenService {
	return auth.NewTokenService(cfg)
}

func provideUserPaginationService() *pagination.PaginationService[models.User] {
	return pagination.NewPaginationService[models.User]()
}

func provideBankPaginationService() *pagination.PaginationService[models.Bank] {
	return pagination.NewPaginationService[models.Bank]()
}

var connectionSet = wire.NewSet(
	provideGormService,
	provideRedisClient,
	providePgxWritePool,
	providePgxReadPool,
	provideConnectDTO,
)

var repositorySet = wire.NewSet(
	user.NewUserReaderRepo,
	user.NewUserWriterRepo,
	user.NewUserPaginatorRepo,
	user.NewUserRepository,

	bank.NewBankReaderRepo,
	bank.NewBankWriterRepo,
	bank.NewBankCrudRepository,
	bank.NewBankPaginationRepo,

	refreshtoken.NewRefreshTokenReaderRepo,
	refreshtoken.NewRefreshTokenWriterRepo,
	refreshtoken.NewRefreshTokenRepository,

	// --- Repositorios de Menú (Solo Lector) ---
	// Cambiamos el nombre del constructor a la implementación existente:
	menu.NewMenuReaderRepository,
	// Comentamos los constructores de escritura y CRUD por ahora:
	// menu.NewMenuWriterRepo,
	// menu.NewMenuCrudRepo,
)

var serviceSet = wire.NewSet(
	provideTokenService,
	auth.NewAuthService,

	provideUserPaginationService,
	provideBankPaginationService,

	services.NewTransactionManager,
	services.NewDatabaseService,

	user2.NewUserReaderService,
	user2.NewUserWriterService,

	bank2.NewBankReaderService,
	bank2.NewBankWriterService,
	bank2.NewBankPaginationService,
	bank2.NewDeactivationService,

	menu2.NewMenuReaderService,
	// Comentamos el servicio de escritura de menús:
	// menu2.NewMenuWriterService,
)

var handlerSet = wire.NewSet(
	handlers.NewAuthHandler,
	handlers.NewUserHandler,
	handlers.NewBankHandler,
	handlers.NewDatabaseHandler,
	// NOTA: Si handlers.NewMenuHandler inyecta MenuWriterService,
	// necesitarás actualizar su constructor también.
	// handlers.NewMenuHandler,
)
