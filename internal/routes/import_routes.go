package routes

import (
	"go-fiber-core/internal/handlers"

	fiber "github.com/gofiber/fiber/v2"
)

func RegisterImportRoutes(router fiber.Router, handler handlers.ImportHandler) {
	group := router.Group("/imports")
	group.Post("/all/:branchId/:refCode/:total/:key", handler.Process)
}
