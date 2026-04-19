package routes

import (
	"go-fiber-core/internal/handlers"

	fiber "github.com/gofiber/fiber/v2"
)

func RegisterImportRoutes(router fiber.Router, handler handlers.ImportHandler) {
	group := router.Group("/imports")
	group.Post("/", handler.Create)
	group.Post("/all/:branchId/:refCode/:total/:key", handler.CreateLegacy)
	group.Post("/bulk-jobs/paginated", handler.GetBulkJobsPaginated)
}
