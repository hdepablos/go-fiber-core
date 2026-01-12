package handlers

import (
	"go-fiber-core/internal/dtos"
	"go-fiber-core/internal/services/menu_user"
	"log"
	"go-fiber-core/internal/domain"
	"go-fiber-core/internal/dtos/responses"

	fiber  "github.com/gofiber/fiber/v2"
)

type MenuUserHandler interface {
	GetAllPaginated(c *fiber.Ctx) error
}

type menuUserHandler struct {
	menuUserPaginationService menu_user.MenuUserPaginationService
}

func NewMenuUserHandler(
	menuUserPaginationService menu_user.MenuUserPaginationService,
) MenuUserHandler {
	return &menuUserHandler{
		menuUserPaginationService: menuUserPaginationService,
	}
}

func (h *menuUserHandler) GetAllPaginated(c *fiber.Ctx) error {
	ctx := c.UserContext()

	// AÑADIDO: Obtener el ID de usuario del contexto
	userID, err := getUserIDUint64FromCtx(ctx)
	if err != nil {
		return responses.Error(c, fiber.StatusUnauthorized, "Error de autenticación", err)
	}
	// ---

	log.Printf("Usuario %d la solicita la paginación", userID)

	var req dtos.PaginationRequest
	if err := c.BodyParser(&req); err != nil {
		return domain.ErrInvalidArgument
	}

	response, err := h.menuUserPaginationService.GetAllPaginated(ctx, req)
	if err != nil {
		return err
	}
	return responses.Success(c, "Menus y Usuarios paginados obtenidos exitosamente", response)
}
