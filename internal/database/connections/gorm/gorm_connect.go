package gorm

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"go-fiber-core/internal/dtos/config"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// GormConnectService no cambia su estructura.
type GormConnectService struct {
	dbWrite    *gorm.DB
	sqlDBWrite *sql.DB
	dbRead     *gorm.DB
	sqlDBRead  *sql.DB
}

// createGormConnection crea una conexión GORM con logger configurado.
func createGormConnection(cfg config.GormConnectionConfig) (*gorm.DB, *sql.DB, error) {
	var dialector gorm.Dialector

	switch strings.ToLower(cfg.Driver) {
	case "postgres":
		dsn := fmt.Sprintf(
			"host=%s port=%d user=%s dbname=%s password=%s sslmode=disable search_path=%s",
			cfg.Host,
			cfg.Port,
			cfg.Username,
			cfg.Database,
			cfg.Password,
			cfg.Schema,
		)
		dialector = postgres.Open(dsn)

	case "mysql":
		dsn := fmt.Sprintf(
			"%s:%s@tcp(%s:%d)/%s?parseTime=true",
			cfg.Username,
			cfg.Password,
			cfg.Host,
			cfg.Port,
			cfg.Database,
		)
		dialector = mysql.Open(dsn)

	default:
		return nil, nil, fmt.Errorf("driver GORM no soportado: %s", cfg.Driver)
	}

	// --------------------------------------------------
	// Logger GORM (imprime SQL fuera de producción)
	// --------------------------------------------------
	gormLogger := getGormLogger(log.New(os.Stdout, "\r\n", log.LstdFlags))

	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: gormLogger,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("falló al abrir la conexión GORM hacia %s: %w", cfg.Host, err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, nil, fmt.Errorf("falló al obtener la instancia DB de GORM para %s: %w", cfg.Host, err)
	}

	if err := sqlDB.Ping(); err != nil {
		return nil, nil, fmt.Errorf("falló el ping a la base de datos GORM en %s: %w", cfg.Host, err)
	}

	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(time.Duration(cfg.MaxConnLifeTimeInSeconds) * time.Second)

	log.Printf("✅ Conexión GORM exitosa a %s", cfg.Host)
	return db, sqlDB, nil
}

func getGormLogger(writer logger.Writer) logger.Interface {
	logLevel := logger.Silent
	appEnv := os.Getenv("APP_ENV")
	if appEnv == "" {
		appEnv = "local"
	}

	// Default behavior based on APP_ENV
	if appEnv != "production" {
		logLevel = logger.Info
	}

	// Override with DB_LOG_LEVEL if set
	// Values: silent, error, warn, info
	if envLogLevel := os.Getenv("DB_LOG_LEVEL"); envLogLevel != "" {
		switch strings.ToLower(envLogLevel) {
		case "silent":
			logLevel = logger.Silent
		case "error":
			logLevel = logger.Error
		case "warn":
			logLevel = logger.Warn
		case "info":
			logLevel = logger.Info
		}
	}

	// Configurar Slow Threshold (Default: 1s)
	// Se puede sobrescribir con DB_SLOW_THRESHOLD_MS (en milisegundos)
	slowThreshold := time.Second
	if envSlowThreshold := os.Getenv("DB_SLOW_THRESHOLD_MS"); envSlowThreshold != "" {
		if ms, err := strconv.Atoi(envSlowThreshold); err == nil && ms > 0 {
			slowThreshold = time.Duration(ms) * time.Millisecond
		}
	}

	return logger.New(
		writer,
		logger.Config{
			SlowThreshold: slowThreshold,
			LogLevel:      logLevel,
			Colorful:      true,
		},
	)
}

// NewGormConnectService ahora retorna una función de cleanup.
func NewGormConnectService(cfg config.MultiDatabaseConfig) (*GormConnectService, func(), error) {
	dbWrite, sqlDBWrite, err := createGormConnection(cfg.Gorm.Write)
	if err != nil {
		return nil, nil, err
	}

	dbRead, sqlDBRead, err := createGormConnection(cfg.Gorm.Read)
	if err != nil {
		sqlDBWrite.Close()
		return nil, nil, err
	}

	// Permite forzar que las lecturas vayan al writer mediante variable de entorno.
	// Útil como interruptor de emergencia o en entornos sin réplica disponible.
	if v := strings.ToLower(os.Getenv("DB_FORCE_READS_TO_WRITE")); v == "1" || v == "true" || v == "yes" {
		log.Printf("⚠️  DB_FORCE_READS_TO_WRITE=%s habilitado: las lecturas usarán la conexión de escritura", os.Getenv("DB_FORCE_READS_TO_WRITE"))
		dbRead = dbWrite
		sqlDBRead = sqlDBWrite
	}

	service := &GormConnectService{
		dbWrite:    dbWrite,
		sqlDBWrite: sqlDBWrite,
		dbRead:     dbRead,
		sqlDBRead:  sqlDBRead,
	}

	cleanup := func() {
		log.Println("🔌 Desconectando de las bases de datos (GORM)...")

		if err := sqlDBWrite.Close(); err != nil {
			log.Printf("❌ Error cerrando la conexión de escritura GORM: %v", err)
		}
		// Evita double-close si ambas referencias apuntan al mismo pool.
		if sqlDBRead != sqlDBWrite {
			if err := sqlDBRead.Close(); err != nil {
				log.Printf("❌ Error cerrando la conexión de lectura GORM: %v", err)
			}
		}
	}

	return service, cleanup, nil
}

// Getters
func (s *GormConnectService) GetWriteDB() *gorm.DB {
	return s.dbWrite
}

func (s *GormConnectService) GetReadDB() *gorm.DB {
	return s.dbRead
}

func (s *GormConnectService) GetWriteSQLDB() *sql.DB {
	return s.sqlDBWrite
}

func (s *GormConnectService) GetReadSQLDB() *sql.DB {
	return s.sqlDBRead
}
