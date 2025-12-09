// internal/database/connections/gorm/gorm_connect.go
package gorm

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	"go-fiber-core/internal/dtos/config"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// GormConnectService no cambia su estructura.
type GormConnectService struct {
	dbWrite    *gorm.DB
	sqlDBWrite *sql.DB
	dbRead     *gorm.DB
	sqlDBRead  *sql.DB
}

// createGormConnection no necesita cambios.
func createGormConnection(cfg config.GormConnectionConfig) (*gorm.DB, *sql.DB, error) {
	// ... (sin cambios en esta función)
	var dialector gorm.Dialector
	switch strings.ToLower(cfg.Driver) {
	case "postgres":
		dsn := fmt.Sprintf("host=%s port=%d user=%s dbname=%s password=%s sslmode=disable search_path=%s",
			cfg.Host, cfg.Port, cfg.Username, cfg.Database, cfg.Password, cfg.Schema)
		dialector = postgres.Open(dsn)
	case "mysql":
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true",
			cfg.Username, cfg.Password, cfg.Host, cfg.Port, cfg.Database)
		dialector = mysql.Open(dsn)
	default:
		return nil, nil, fmt.Errorf("driver GORM no soportado: %s", cfg.Driver)
	}

	db, err := gorm.Open(dialector, &gorm.Config{})
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

// CAMBIO: NewGormConnectService ahora retorna una función de limpieza.
func NewGormConnectService(cfg config.MultiDatabaseConfig) (*GormConnectService, func(), error) {
	dbWrite, sqlDBWrite, err := createGormConnection(cfg.Gorm.Write)
	if err != nil {
		return nil, nil, err
	}

	dbRead, sqlDBRead, err := createGormConnection(cfg.Gorm.Read)
	if err != nil {
		// Si la conexión de lectura falla, cerramos la de escritura antes de salir.
		sqlDBWrite.Close()
		return nil, nil, err
	}

	service := &GormConnectService{
		dbWrite:    dbWrite,
		sqlDBWrite: sqlDBWrite,
		dbRead:     dbRead,
		sqlDBRead:  sqlDBRead,
	}

	// Esta es la función que Wire usará para limpiar los recursos.
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

// GetWriteDB y GetReadDB no cambian.
func (s *GormConnectService) GetWriteDB() *gorm.DB {
	return s.dbWrite
}

func (s *GormConnectService) GetReadDB() *gorm.DB {
	return s.dbRead
}

// GetWriteSQLDB y GetReadSQLDB no cambian.
func (s *GormConnectService) GetWriteSQLDB() *sql.DB {
	return s.sqlDBWrite
}

func (s *GormConnectService) GetReadSQLDB() *sql.DB {
	return s.sqlDBRead
}
