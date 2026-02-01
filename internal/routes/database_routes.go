package routes

import (
	"go-fiber-core/internal/handlers"

	fiber "github.com/gofiber/fiber/v2"
)

// RegisterDatabaseRoutes ahora recibe el handler ya creado y registra la ruta de health check.
// Ya no necesita la configuración ni las conexiones, siguiendo el principio de Inyección de Dependencias.
func RegisterDatabaseRoutes(router fiber.Router, dbHandler handlers.DatabaseHandler) {
	// Grupo de rutas para base de datos
	db := router.Group("/database")

	// Endpoint general que devuelve todo
	db.Get("/health", dbHandler.HealthCheck)

	// Endpoints granulares
	db.Get("/health/redis", dbHandler.HealthRedis)
	db.Get("/health/gorm", dbHandler.HealthGorm)
	db.Get("/health/pgx", dbHandler.HealthPgx)
}
