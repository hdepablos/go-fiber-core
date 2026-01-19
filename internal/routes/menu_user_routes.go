package routes

import (
	"go-fiber-core/internal/handlers"

	fiber "github.com/gofiber/fiber/v2"
)

func RegisterMenuUserRoutes(router fiber.Router, menuUserHandler handlers.MenuUserHandler) {
	menuUserGroup := router.Group("/menu-users")

	menuUserGroup.Post("/paginated", menuUserHandler.GetAllPaginated)

}
