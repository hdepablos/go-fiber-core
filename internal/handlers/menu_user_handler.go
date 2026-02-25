package handlers

import (
	"go-fiber-core/internal/dtos"
	"go-fiber-core/internal/services/menu_user"
	"log"
	"go-fiber-core/internal/domain"
	"go-fiber-core/internal/dtos/responses"
	"strconv"

	fiber  "github.com/gofiber/fiber/v2"
)

type MenuUserHandler interface {
	GetAllPaginated(c *fiber.Ctx) error
	GetMenusByUser(c *fiber.Ctx) error
	GetMenusNotByUser(c *fiber.Ctx) error
	GetMenuAssignmentStatus(c *fiber.Ctx) error
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
func (h *menuUserHandler) GetMenusByUser(c *fiber.Ctx) error {
    ctx := c.UserContext()
    userIDParam := c.Params("userId") // Coincide con :userId en la ruta

    userID64, err := strconv.ParseUint(userIDParam, 10, 64)
    if err != nil {
        return responses.Error(c, fiber.StatusBadRequest, "ID de usuario inválido", err)
    }

    var req dtos.PaginationRequest
    // Si la tabla envía datos en el cuerpo, BodyParser está bien.
    // Si falla, prueba cambiar a c.QueryParser(&req)
    if err := c.BodyParser(&req); err != nil {
        return domain.ErrInvalidArgument
    }

    response, err := h.menuUserPaginationService.GetMenusByUser(ctx, uint(userID64), req)
    if err != nil {
        return err
    }

    return responses.Success(c, "Menus asociados obtenidos exitosamente", response)
}

func (h *menuUserHandler) GetMenusNotByUser(c *fiber.Ctx) error {

	ctx := c.UserContext()

	userIDParam := c.Params("userId")

	userID64, err := strconv.ParseUint(userIDParam, 10, 64)
	if err != nil {
		return domain.ErrInvalidArgument
	}

	var req dtos.PaginationRequest
	if err := c.BodyParser(&req); err != nil {
		return domain.ErrInvalidArgument
	}

	response, err := h.menuUserPaginationService.GetMenusNotByUser(
		ctx,
		uint(userID64),
		req,
	)
	if err != nil {
		return err
	}

	return responses.Success(
		c,
		"Menus no asociados obtenidos exitosamente",
		response,
	)
}

func (h *menuUserHandler) GetMenuAssignmentStatus(c *fiber.Ctx) error {
	ctx := c.UserContext()

	userIDParam := c.Params("userId")

	userID64, err := strconv.ParseUint(userIDParam, 10, 64)
	if err != nil {
		return domain.ErrInvalidArgument
	}

	total, assigned, err := h.menuUserPaginationService.GetMenuAssignmentStatus(ctx, uint(userID64))
	if err != nil {
		return err
	}

	return responses.Success(c, "Estado de asignación de menús obtenido", map[string]uint{
		"total":    total,
		"assigned": assigned,
	})
}
