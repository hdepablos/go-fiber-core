package gorm

import (
	"database/sql"
	"fmt"
	"log"
	"os"
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
	logLevel := logger.Silent
	APP_ENV := "local"
	if APP_ENV != "production" {
		logLevel = logger.Info
	}

	gormLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold: time.Second,
			LogLevel:      logLevel,
			Colorful:      true,
		},
	)

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
		if err := sqlDBRead.Close(); err != nil {
			log.Printf("❌ Error cerrando la conexión de lectura GORM: %v", err)
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
