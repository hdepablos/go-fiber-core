package handlers

import (
	"go-fiber-core/internal/dtos/responses"
	catalogService "go-fiber-core/internal/services/catalog"

	fiber "github.com/gofiber/fiber/v2"
)

type CatalogHandler interface {
	GetAll(c *fiber.Ctx) error
}

type catalogHandler struct {
	catalogService catalogService.CatalogService
}

func NewCatalogHandler(catalogService catalogService.CatalogService) CatalogHandler {
	return &catalogHandler{
		catalogService: catalogService,
	}
}

func (h *catalogHandler) GetAll(c *fiber.Ctx) error {
	response, err := h.catalogService.GetAll(c.Context())
	if err != nil {
		return err
	}
	return responses.Success(c, "Catálogos obtenidos exitosamente", response)
}
