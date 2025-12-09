package services

import (
	"context"
	"fmt"
	"go-fiber-core/internal/dtos/config"
	"go-fiber-core/internal/dtos/connect"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"gorm.io/gorm"
)

// DatabaseService no necesita cambios en su estructura.
type DatabaseService struct {
	// CAMBIO: AppConfig y Db ahora son punteros.
	AppConfig *config.AppConfig
	Db        *connect.ConnectDTO
}

// CAMBIO: Los parámetros appConfig y connect ahora son punteros.
func NewDatabaseService(appConfig *config.AppConfig, connect *connect.ConnectDTO) *DatabaseService {
	return &DatabaseService{
		AppConfig: appConfig,
		Db:        connect,
	}
}

// --- CHEQUEOS DE SALUD PARA GORM ---

// checkGormHealth es un helper interno para no repetir código.
func (s *DatabaseService) checkGormHealth(db *gorm.DB, cfg config.GormConnectionConfig, connectionName string) map[string]any {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	sqlDB, err := db.DB()
	if err != nil {
		return map[string]any{
			"database":     cfg.Database,
			"connection":   connectionName,
			"connect_type": "gorm",
			"status":       "Down",
			"error":        fmt.Sprintf("failed to get underlying sql.DB: %v", err),
		}
	}

	if err = sqlDB.PingContext(ctx); err != nil {
		return map[string]any{
			"database":     cfg.Database,
			"connection":   connectionName,
			"connect_type": "gorm",
			"status":       "Down",
			"error":        fmt.Sprintf("db down: %v", err),
		}
	}

	dbStats := sqlDB.Stats()
	return map[string]any{
		"database":            cfg.Database,
		"connection":          connectionName,
		"connect_type":        "gorm",
		"status":              "Up",
		"message":             "It's healthy",
		"open_connections":    dbStats.OpenConnections,
		"in_use":              dbStats.InUse,
		"idle":                dbStats.Idle,
		"wait_count":          dbStats.WaitCount,
		"wait_duration":       dbStats.WaitDuration.String(),
		"max_idle_closed":     dbStats.MaxIdleClosed,
		"max_lifetime_closed": dbStats.MaxLifetimeClosed,
	}
}

// HealthGormWrite verifica la conexión de escritura de GORM.
func (s *DatabaseService) HealthGormWrite() map[string]any {
	return s.checkGormHealth(s.Db.ConnectGormWrite, s.AppConfig.MultiDatabaseConfig.Gorm.Write, "gorm-write")
}

// HealthGormRead verifica la conexión de lectura de GORM.
func (s *DatabaseService) HealthGormRead() map[string]any {
	return s.checkGormHealth(s.Db.ConnectGormRead, s.AppConfig.MultiDatabaseConfig.Gorm.Read, "gorm-read")
}

// --- CHEQUEOS DE SALUD PARA PGX ---

// checkPgxHealth es un helper interno para PGX.
func (s *DatabaseService) checkPgxHealth(pool *pgxpool.Pool, cfg config.PgxConnectionConfig, connectionName string) map[string]any {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := pool.Ping(ctx); err != nil {
		return map[string]any{
			"database":     cfg.Database,
			"connection":   connectionName,
			"status":       "Down",
			"connect_type": "pgx",
			"error":        fmt.Sprintf("db down: %v", err),
		}
	}

	poolStats := pool.Stat()
	return map[string]any{
		"database":           cfg.Database,
		"connection":         connectionName,
		"status":             "Up",
		"connect_type":       "pgx",
		"message":            "It's healthy",
		"total_conns":        poolStats.TotalConns(),
		"idle_conns":         poolStats.IdleConns(),
		"acquired_conns":     poolStats.AcquiredConns(),
		"constructing_conns": poolStats.ConstructingConns(),
		"max_conns":          poolStats.MaxConns(),
	}
}

// HealthPgxWrite verifica la conexión de escritura de PGX.
func (s *DatabaseService) HealthPgxWrite() map[string]any {
	return s.checkPgxHealth(s.Db.ConnectPgxWrite, s.AppConfig.MultiDatabaseConfig.Pgx.Write, "pgx-write")
}

// HealthPgxRead verifica la conexión de lectura de PGX.
func (s *DatabaseService) HealthPgxRead() map[string]any {
	return s.checkPgxHealth(s.Db.ConnectPgxRead, s.AppConfig.MultiDatabaseConfig.Pgx.Read, "pgx-read")
}

// --- CHEQUEO DE SALUD PARA REDIS (sin cambios) ---

func (s *DatabaseService) HealthRedis() map[string]any {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Usamos el host como nombre ya que Redis no tiene un "nombre de base de datos" como SQL.
	nameDatabase := s.AppConfig.Redis.RedisHost

	if err := s.Db.ConnectRedis.Ping(ctx).Err(); err != nil {
		return map[string]any{
			"database":     nameDatabase,
			"status":       "Down",
			"connect_type": "Redis",
			"error":        fmt.Sprintf("redis down: %v", err),
		}
	}

	return map[string]any{
		"database":     nameDatabase,
		"status":       "Up",
		"connect_type": "redis",
		"message":      "It's healthy",
	}
}
