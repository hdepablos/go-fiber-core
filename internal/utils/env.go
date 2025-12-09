package utils

import (
	"go-fiber-core/internal/dtos/config"
)

const (
	LocalEnvValue       = "local"
	StagingEnvValue     = "staging"
	LocalDatabaseHost   = "localhost"  // Host de la DB en entorno local
	StagingDatabaseHost = "staging-db" // Host de la DB en entorno de staging
)

// IsProduction verifica si la aplicación se está ejecutando en un entorno de producción.
// Retorna true si el entorno es de producción, false en caso contrario.
func IsProduction(appConfig config.AppConfig) bool {
	nonProductionEnvironments := []string{LocalEnvValue, StagingEnvValue, "development"}
	currentEnv := appConfig.App.AppEnv

	// CAMBIO: Apuntamos al host de la base de datos de escritura de GORM
	// como la fuente de verdad para determinar el entorno de la base de datos.
	dbHost := appConfig.MultiDatabaseConfig.Gorm.Write.Host

	isNonProductionEnv := false
	for _, env := range nonProductionEnvironments {
		if currentEnv == env {
			isNonProductionEnv = true
			break
		}
	}

	isNonProductionDB := (dbHost == LocalDatabaseHost || dbHost == StagingDatabaseHost)

	// Es producción si no es un entorno de no-producción Y la DB no es de no-producción.
	return !isNonProductionEnv && !isNonProductionDB
}

// IsLocal verifica si la aplicación se está ejecutando en un entorno local.
// Retorna true si el entorno es local, false en caso contrario.
func IsLocal(appConfig config.AppConfig) bool {
	currentEnv := appConfig.App.AppEnv

	// CAMBIO: Apuntamos al host de la base de datos de escritura de GORM.
	dbHost := appConfig.MultiDatabaseConfig.Gorm.Write.Host

	return currentEnv == LocalEnvValue && dbHost == LocalDatabaseHost
}
