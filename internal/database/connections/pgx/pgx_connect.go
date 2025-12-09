// internal/database/connections/pgx/pgx_connect.go
package pgx

import (
	"context"
	"fmt"
	"log"
	"time"

	"go-fiber-core/internal/dtos/config"

	"github.com/jackc/pgx/v5/pgxpool"
)

// CAMBIO: NewPgxConnection ahora retorna una función de limpieza.
func NewPgxConnection(cfg config.PgxConnectionConfig) (*pgxpool.Pool, func(), error) {
	dsn := fmt.Sprintf("postgresql://%s:%s@%s:%d/%s",
		cfg.Username, cfg.Password, cfg.Host, cfg.Port, cfg.Database)

	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, nil, fmt.Errorf("no se pudo parsear la configuración de la base de datos para pgx: %w", err)
	}

	config.MaxConns = int32(cfg.MaxConns)
	config.MaxConnLifetime = 1 * time.Hour
	config.MaxConnIdleTime = 30 * time.Minute

	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return nil, nil, fmt.Errorf("no se pudo crear el pool de conexiones pgx: %w", err)
	}

	if err := pool.Ping(context.Background()); err != nil {
		pool.Close()
		return nil, nil, fmt.Errorf("no se pudo hacer ping a la base de datos pgx en %s: %w", cfg.Host, err)
	}

	log.Printf("✅ Conexión PGX exitosa a %s", cfg.Host)

	// Esta es la función de limpieza para este pool específico.
	cleanup := func() {
		log.Printf("🔌 Desconectando pool PGX de %s...", cfg.Host)
		pool.Close()
	}

	return pool, cleanup, nil
}
