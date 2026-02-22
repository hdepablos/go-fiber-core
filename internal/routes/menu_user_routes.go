package routes

import (
	"go-fiber-core/internal/handlers"

	fiber "github.com/gofiber/fiber/v2"
)

func RegisterMenuUserRoutes(router fiber.Router, menuUserHandler handlers.MenuUserHandler) {
	menuUserGroup := router.Group("/menu-users")

	menuUserGroup.Post("/paginated", menuUserHandler.GetAllPaginated)
	menuUserGroup.Post("/eliminar/:userId", menuUserHandler.GetMenusByUser)
	menuUserGroup.Post("/agregar/:userId", menuUserHandler.GetMenusNotByUser)

	// Estado de asignación
	menuUserGroup.Get("/status/:userId", menuUserHandler.GetMenuAssignmentStatus)

}
