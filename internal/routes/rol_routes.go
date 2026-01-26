// internal/routes/bank_routes.go
package routes

import (
	"go-fiber-core/internal/dtos/requests"
	"go-fiber-core/internal/handlers"
	"go-fiber-core/internal/utils"

	fiber "github.com/gofiber/fiber/v2"
)

// RegisterRoleRoutes define todos los endpoints relacionados con el recurso de Roles.
func RegisterRoleRoutes(router fiber.Router, roleHandler handlers.RolHandler) {
	roleGroup := router.Group("/roles")

	// --- RUTAS DE ESCRITURA (Comandos) ---

	// POST /roles - Crear un nuevo rol
	roleGroup.Post(
		"/", // Usar la raíz del grupo es más estándar para "crear"
		utils.Validate(new(requests.CreateRoleRequest)),
		roleHandler.Create,
	)

	// PUT /roles/:id - Actualizar un rol existente
	// CAMBIO: Se añade el middleware para validar el body con UpdateRoleRequest.
	roleGroup.Put(
		"/:id", // Usar /:id en lugar de /edit/:id es más RESTful
		utils.Validate(new(requests.UpdateRoleRequest)),
		roleHandler.Update,
	)

	// DELETE /roles/:id - Borrado lógico
	roleGroup.Delete("/soft/:id", roleHandler.SoftDelete)
	// DELETE /roles/hard/:id - Borrado físico
	roleGroup.Delete("/hard/:id", roleHandler.HardDelete)

	// --- RUTAS DE LECTURA (Consultas) ---

	// GET /roles - Obtener todos los roles
	roleGroup.Get("/", roleHandler.GetAll)

	// GET /roles/:id - Obtener un rol por ID
	roleGroup.Get("/:id", roleHandler.GetByID)

	// POST /roles/paginated - Obtener roles paginados
	roleGroup.Post("/paginated", roleHandler.GetAllPaginated)

}
