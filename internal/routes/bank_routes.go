// internal/routes/bank_routes.go
package routes

import (
	"go-fiber-core/internal/dtos/requests"
	"go-fiber-core/internal/handlers"
	"go-fiber-core/internal/utils"

	fiber "github.com/gofiber/fiber/v2"
)

// RegisterBankRoutes define todos los endpoints relacionados con el recurso de Bancos.
func RegisterBankRoutes(router fiber.Router, bankHandler handlers.BankHandler) {
	bankGroup := router.Group("/banks")

	// --- RUTAS DE ESCRITURA (Comandos) ---

	// POST /banks - Crear un nuevo banco
	bankGroup.Post(
		"/", // Usar la raíz del grupo es más estándar para "crear"
		utils.Validate(new(requests.CreateBankRequest)),
		bankHandler.Create,
	)

	// PUT /banks/:id - Actualizar un banco existente
	// CAMBIO: Se añade el middleware para validar el body con UpdateBankRequest.
	bankGroup.Put(
		"/:id", // Usar /:id en lugar de /edit/:id es más RESTful
		utils.Validate(new(requests.UpdateBankRequest)),
		bankHandler.Update,
	)

	// DELETE /banks/:id - Borrado lógico
	bankGroup.Delete("/:id", bankHandler.SoftDelete)

	// DELETE /banks/hard/:id - Borrado físico
	bankGroup.Delete("/hard/:id", bankHandler.HardDelete)

	// --- RUTAS DE LECTURA (Consultas) ---

	// GET /banks - Obtener todos los bancos
	bankGroup.Get("/", bankHandler.GetAll)

	// GET /banks/:id - Obtener un banco por ID
	bankGroup.Get("/:id", bankHandler.GetByID)

	// POST /banks/paginated - Obtener bancos paginados
	bankGroup.Post("/paginated", bankHandler.GetAllPaginated)

}
