package migrations

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	// Se importa el paquete 'gorm' que contiene el 'GormConnectService'.
	"go-fiber-core/internal/database/connections/gorm"
	"go-fiber-core/internal/dtos/config"

	goose "github.com/pressly/goose/v3"
)

// MigrationService gestiona las operaciones de migración de la base de datos.
type MigrationService struct {
	DB             *sql.DB
	migrationsPath string
}

// MigrationServiceInterface define los métodos disponibles para el servicio de migraciones.
type MigrationServiceInterface interface {
	Create(name string) error
	CurrentVersion() (int64, error)
	Up() error
	Down(step int) error
	DownToZero() error
	Refresh() error
	PrintStatus() error
}

// NewMigrationService crea una nueva instancia del servicio de migraciones.
func NewMigrationService(multiConfig config.MultiDatabaseConfig) (*MigrationService, func(), error) {
	// CAMBIO: Capturamos la función de limpieza (`gormCleanup`) devuelta por el constructor.
	gormService, gormCleanup, err := gorm.NewGormConnectService(multiConfig)
	if err != nil {
		// No necesitamos llamar a gormCleanup aquí porque si el servicio falla, no hay nada que limpiar.
		return nil, nil, fmt.Errorf("no se pudo crear el GormConnectService para las migraciones: %w", err)
	}

	// Obtenemos el *sql.DB específico de la conexión de ESCRITURA.
	sqlDB := gormService.GetWriteSQLDB()
	if sqlDB == nil {
		gormCleanup() // Cerramos el servicio si no podemos obtener el sql.DB
		return nil, nil, fmt.Errorf("se obtuvo un puntero nulo para la conexión de escritura de la base de datos")
	}

	// Obtenemos la ruta de las migraciones.
	writeConfig := multiConfig.Gorm.Write
	migrationsPath, err := getMigrationsPath(writeConfig.Driver)
	if err != nil {
		// CAMBIO: En caso de error, usamos la función `gormCleanup` para cerrar la conexión.
		gormCleanup()
		return nil, nil, fmt.Errorf("error al obtener la ruta de las migraciones: %w", err)
	}
	log.Printf("Usando migraciones desde: %s", migrationsPath)

	// CAMBIO: Ya no creamos una función de limpieza manual.
	// Simplemente devolvemos la que nos proporcionó el GormConnectService.
	return &MigrationService{
		DB:             sqlDB,
		migrationsPath: migrationsPath,
	}, gormCleanup, nil
}

// --- El resto de los métodos no necesita ningún cambio ---

// Create genera un nuevo archivo de migración SQL.
func (m *MigrationService) Create(name string) error {
	return goose.Create(nil, m.migrationsPath, name, "sql")
}

// CurrentVersion devuelve la versión actual de la base de datos.
func (m *MigrationService) CurrentVersion() (int64, error) {
	return goose.GetDBVersion(m.DB)
}

// Up aplica todas las migraciones pendientes.
func (m *MigrationService) Up() error {
	return goose.Up(m.DB, m.migrationsPath)
}

// Down revierte la última migración o un número de pasos especificado.
func (m *MigrationService) Down(step int) error {
	if step == 0 {
		return m.DownToZero()
	}
	if step < 1 {
		step = 1
	}
	for i := 0; i < step; i++ {
		if err := goose.Down(m.DB, m.migrationsPath); err != nil {
			return fmt.Errorf("error revirtiendo migración #%d: %w", i+1, err)
		}
	}
	return nil
}

// DownToZero revierte todas las migraciones.
func (m *MigrationService) DownToZero() error {
	return goose.DownTo(m.DB, m.migrationsPath, 0)
}

// Refresh revierte todas las migraciones y las vuelve a aplicar.
func (m *MigrationService) Refresh() error {
	if err := m.DownToZero(); err != nil {
		return err
	}
	return m.Up()
}

// PrintStatus muestra el estado de todas las migraciones.
func (m *MigrationService) PrintStatus() error {
	return goose.Status(m.DB, m.migrationsPath)
}

// getMigrationsPath construye la ruta al directorio de migraciones.
func getMigrationsPath(driver string) (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	path := filepath.Join(wd, "internal", "database", "migrations", strings.ToLower(driver))
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return "", fmt.Errorf("el directorio de migraciones no existe para el driver '%s': %s", driver, path)
	}
	return path, nil
}
