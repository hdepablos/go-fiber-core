package pgx

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"go-fiber-core/internal/dtos/config"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/tracelog"
)

// NewPgxConnection crea un pool PGX con logging de SQL
func NewPgxConnection(cfg config.PgxConnectionConfig) (*pgxpool.Pool, func(), error) {
	dsn := fmt.Sprintf(
		"postgresql://%s:%s@%s:%d/%s",
		cfg.Username,
		cfg.Password,
		cfg.Host,
		cfg.Port,
		cfg.Database,
	)

	pgxConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, nil, fmt.Errorf("no se pudo parsear la configuración PGX: %w", err)
	}

	// --------------------------------------------------
	// Logger PGX (pgx v5)
	// --------------------------------------------------
	logLevel := tracelog.LogLevelNone
	appEnv := os.Getenv("APP_ENV")
	if appEnv == "" {
		appEnv = "local"
	}
	if appEnv != "production" {
		logLevel = tracelog.LogLevelInfo
	}

	pgxConfig.ConnConfig.Tracer = &tracelog.TraceLog{
		LogLevel: logLevel,
		Logger: tracelog.LoggerFunc(
			func(
				ctx context.Context,
				level tracelog.LogLevel,
				msg string,
				data map[string]interface{},
			) {
				log.Printf(
					"[PGX] %s | %s | %v",
					level,
					msg,
					data,
				)
			},
		),
	}

	// Pool config
	pgxConfig.MaxConns = int32(cfg.MaxConns)
	pgxConfig.MaxConnLifetime = 1 * time.Hour
	pgxConfig.MaxConnIdleTime = 30 * time.Minute

	pool, err := pgxpool.NewWithConfig(context.Background(), pgxConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("no se pudo crear el pool PGX: %w", err)
	}

	if err := pool.Ping(context.Background()); err != nil {
		pool.Close()
		return nil, nil, fmt.Errorf("no se pudo hacer ping PGX en %s: %w", cfg.Host, err)
	}

	log.Printf("✅ Conexión PGX exitosa a %s", cfg.Host)

	cleanup := func() {
		log.Printf("🔌 Desconectando pool PGX de %s...", cfg.Host)
		pool.Close()
	}

	return pool, cleanup, nil
}
