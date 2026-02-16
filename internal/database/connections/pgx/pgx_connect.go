package pgx

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
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

	slowThreshold := time.Second
	if envSlow := os.Getenv("DB_SLOW_THRESHOLD_MS"); envSlow != "" {
		if ms, err := strconv.Atoi(envSlow); err == nil && ms > 0 {
			slowThreshold = time.Duration(ms) * time.Millisecond
		}
	}

	if v := strings.ToLower(os.Getenv("DB_SLOW_SQL_ENABLED")); v == "" || v == "0" || v == "false" || v == "no" {
		slowThreshold = 0
	}

	// Nivel de detalle: interpretamos DB_LOG_LEVEL (silent|error|warn|info)
	dbLogLevel := strings.ToLower(os.Getenv("DB_LOG_LEVEL"))
	if dbLogLevel == "" {
		// default: en local mostramos más
		if strings.ToLower(os.Getenv("APP_ENV")) == "production" {
			dbLogLevel = "silent"
		} else {
			dbLogLevel = "info"
		}
	}

	// PGX requiere un LogLevel mínimo para que el tracer reciba eventos.
	// Usamos Info y filtramos manualmente según DB_LOG_LEVEL para no perder eventos.
	logLevel := tracelog.LogLevelInfo

	// Escritor dual en local si se define DB_SLOW_LOG_FILE
	var multiWriter io.Writer = os.Stdout
	if strings.ToLower(os.Getenv("APP_ENV")) == "local" {
		if slowPath := os.Getenv("DB_SLOW_LOG_FILE"); slowPath != "" {
			if f, err := os.OpenFile(slowPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644); err == nil {
				multiWriter = io.MultiWriter(os.Stdout, f)
			} else {
				log.Printf("error abriendo DB_SLOW_LOG_FILE=%s: %v", slowPath, err)
			}
		}
	}
	stdLogger := log.New(multiWriter, "", log.LstdFlags)

	appEnv := os.Getenv("APP_ENV")
	if appEnv == "" {
		appEnv = "local"
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
				// Intentar obtener duración de la operación (pgx suele incluir "time" o "duration")
				var dur time.Duration
				if v, ok := data["time"]; ok {
					switch t := v.(type) {
					case time.Duration:
						dur = t
					case string:
						if parsed, err := time.ParseDuration(t); err == nil {
							dur = parsed
						}
					}
				} else if v, ok := data["duration"]; ok {
					switch t := v.(type) {
					case time.Duration:
						dur = t
					case string:
						if parsed, err := time.ParseDuration(t); err == nil {
							dur = parsed
						}
					}
				}

				// Extraer SQL si está disponible
				sql := ""
				if v, ok := data["sql"]; ok {
					if s, ok2 := v.(string); ok2 {
						sql = s
					}
				}

				// Existe error?
				var hasErr bool
				if v, ok := data["err"]; ok && v != nil {
					hasErr = true
				}

				// Determinar si es "lento"
				isSlow := dur > slowThreshold && slowThreshold > 0

				// Filtro según DB_LOG_LEVEL
				shouldLog := false
				switch dbLogLevel {
				case "silent":
					shouldLog = false
				case "error":
					shouldLog = hasErr
				case "warn":
					// Loguea errores y SLOW SQL
					shouldLog = hasErr || isSlow
				default: // info
					// Log completo en local; pero preferimos no saturar en prod
					if strings.ToLower(appEnv) == "production" {
						shouldLog = hasErr || isSlow
					} else {
						shouldLog = true
					}
				}

				if !shouldLog {
					return
				}

				// Mensaje formateado
				if isSlow {
					stdLogger.Printf("[PGX] SLOW SQL | duration=%s | sql=%q | %s | data=%v", dur.String(), sql, msg, data)
				} else if hasErr {
					stdLogger.Printf("[PGX] ERROR | sql=%q | %s | data=%v", sql, msg, data)
				} else {
					stdLogger.Printf("[PGX] %s | sql=%q | data=%v", msg, sql, data)
				}
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
