package handlers

import (
	"go-fiber-core/internal/dtos/responses"
	"go-fiber-core/internal/services"

	fiber "github.com/gofiber/fiber/v2"
)

// DatabaseHandler define la interfaz para el manejador de la base de datos.
type DatabaseHandler interface {
	HealthCheck(c *fiber.Ctx) error
}

// databaseHandler es la implementación.
type databaseHandler struct {
	dbService *services.DatabaseService
}

// NewDatabaseHandler ahora acepta un puntero a DatabaseService.
func NewDatabaseHandler(dbService *services.DatabaseService) DatabaseHandler {
	return &databaseHandler{
		dbService: dbService,
	}
}

// HealthCheck es el único método necesario. Llama a todos los chequeos de salud
// del servicio y los agrupa en una sola respuesta.
func (h *databaseHandler) HealthCheck(c *fiber.Ctx) error {
	// Llamamos a cada uno de los métodos de chequeo de salud del servicio.
	gormWriteStatus := h.dbService.HealthGormWrite()
	gormReadStatus := h.dbService.HealthGormRead()
	pgxWriteStatus := h.dbService.HealthPgxWrite()
	pgxReadStatus := h.dbService.HealthPgxRead()
	redisStatus := h.dbService.HealthRedis()

	// Agregamos todos los resultados en un mapa para una respuesta clara y completa.
	fullStatus := map[string]any{
		"gorm_write": gormWriteStatus,
		"gorm_read":  gormReadStatus,
		"pgx_write":  pgxWriteStatus,
		"pgx_read":   pgxReadStatus,
		"redis":      redisStatus,
	}

	// Devuelve una respuesta estandarizada usando el helper 'Success'.
	return responses.Success(c, "Estado de las conexiones del sistema", fullStatus)
}
