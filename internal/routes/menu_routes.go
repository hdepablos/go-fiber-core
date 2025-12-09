// internal/routes/menu_routes.go
package routes

import (
	"go-fiber-core/internal/dtos/requests"
	"go-fiber-core/internal/handlers"
	"go-fiber-core/internal/utils"

	fiber "github.com/gofiber/fiber/v2"
)

func RegisterMenuRoutes(router fiber.Router, menuHandler handlers.MenuHandler) {
	menuGroup := router.Group("/menus")

	// --- 1) OBTENER MENÚ DEL USUARIO AUTENTICADO ---
	// GET /menus/my
	menuGroup.Get("/my", menuHandler.GetMenuByUser)

	// --- 2) ASIGNACIÓN MASIVA MENÚS ↔ USUARIOS ---
	// POST /menus/users/bulk
	menuGroup.Post(
		"/users/bulk",
		utils.Validate(new(requests.BulkAssignMenuUsersRequest)),
		menuHandler.AddBulkUsers,
	)

	// --- 3) REMOCIÓN MASIVA MENÚS ↔ USUARIOS ---
	// DELETE /menus/users/bulk
	menuGroup.Delete(
		"/users/bulk",
		menuHandler.BulkRemoveUsers,
	)
}
