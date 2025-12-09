package connect

import (
	"github.com/jackc/pgx/v5/pgxpool"
	redis "github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// ConnectDTO ahora contiene todos los pools de conexiones que la aplicación necesita.
type ConnectDTO struct {
	// Conexiones para el ORM GORM
	ConnectGormWrite *gorm.DB
	ConnectGormRead  *gorm.DB

	// Conexiones para el driver PGX
	ConnectPgxWrite *pgxpool.Pool
	ConnectPgxRead  *pgxpool.Pool

	// Conexión para Redis (esta no cambia)
	ConnectRedis *redis.Client
}
