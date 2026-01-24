package routes

import (
	"go-fiber-core/internal/handlers"

	fiber "github.com/gofiber/fiber/v2"
)

func RegisterCatalogRoutes(router fiber.Router, catalogHandler handlers.CatalogHandler) {
	catalog := router.Group("/catalogs")
	catalog.Get("/", catalogHandler.GetAll)
}
