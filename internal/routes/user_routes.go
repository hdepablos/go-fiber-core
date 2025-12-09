// internal/routes/user_routes.go

package routes

import (
	"go-fiber-core/internal/handlers"

	fiber "github.com/gofiber/fiber/v2"
)

// SetupUserRoutes ahora recibe el UserHandler que crearemos en nuestro archivo principal.
func RegisterUserRoutes(router fiber.Router, userHandler handlers.UserHandler) {
	users := router.Group("/users")

	// Rutas principales
	users.Post("/", userHandler.CreateUser)
	// users.Post("/full", userHandler.CreateUserWithRelations) // 👈 Nuevo endpoint
	// users.Post("/full-existing", userHandler.CreateUserWithExistingRelations)
	// users.Post("/full-new-if-not-exist", userHandler.CreateUserWithNewProductsAndRolesIfNotExist)
	users.Get("/", userHandler.GetAllUsers)
	users.Get("/:id", userHandler.GetUserByID)
	users.Put("/:id", userHandler.UpdateUser)
	users.Delete("/:id", userHandler.SoftDelete) // Pendiente: cambiar password
	users.Delete("/hard/:id", userHandler.HardDelete)

	// Pendiente: cambiar password

	// Ruta para obtener usuarios paginados
	users.Post("/paginated", userHandler.GetAllPaginatedUsers)
}
