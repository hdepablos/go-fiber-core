package routes

import (
	"go-fiber-core/internal/handlers"

	fiber "github.com/gofiber/fiber/v2"
)

// RegisterDatabaseRoutes ahora recibe el handler ya creado y registra la ruta de health check.
// Ya no necesita la configuración ni las conexiones, siguiendo el principio de Inyección de Dependencias.
func RegisterDatabaseRoutes(router fiber.Router, dbHandler handlers.DatabaseHandler) {
	// CAMBIO: Se registra una única ruta '/health' que devuelve el estado de todas las conexiones.
	// Esta es una práctica estándar para los endpoints de monitoreo.
	router.Get("/health", dbHandler.HealthCheck)
}
