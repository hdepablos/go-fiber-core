package handlers

import (
	"go-fiber-core/internal/dtos/requests"
	"go-fiber-core/internal/dtos/responses"
	"log"

	"github.com/gofiber/fiber/v2"

	menuService "go-fiber-core/internal/services/menu"
)

// ─────────────────────────────────────────────
// INTERFAZ DEL HANDLER
// ─────────────────────────────────────────────
type MenuHandler interface {
	AddBulkUsers(c *fiber.Ctx) error
	BulkRemoveUsers(c *fiber.Ctx) error
	GetMenuByUser(c *fiber.Ctx) error
}

// ─────────────────────────────────────────────
// HANDLER
// ─────────────────────────────────────────────
type menuHandler struct {
	writer menuService.MenuWriterService
	reader menuService.MenuReaderService
}

// ─────────────────────────────────────────────
// CONSTRUCTOR
// ─────────────────────────────────────────────
func NewMenuHandler(
	writer menuService.MenuWriterService,
	reader menuService.MenuReaderService,
) MenuHandler {
	return &menuHandler{
		writer: writer,
		reader: reader,
	}
}

// ─────────────────────────────────────────────
// ASSIGN USERS → AddBulkUsers
// ─────────────────────────────────────────────
func (h *menuHandler) AddBulkUsers(c *fiber.Ctx) error {
	ctx := c.UserContext()

	// Validar sesión
	userID, err := getUserIDUint64FromCtx(ctx)
	if err != nil {
		return responses.Error(c, fiber.StatusUnauthorized, "Error de autenticación", err)
	}

	log.Printf("Usuario %d está asignando usuarios a menús", userID)

	var req requests.BulkAssignMenuUsersRequest
	if err := c.BodyParser(&req); err != nil {
		return responses.Error(c, fiber.StatusBadRequest, "Error al parsear el cuerpo de la solicitud", err)
	}

	if err := h.writer.AddBulkUsers(ctx, req.MenuIDs, req.UserIDs); err != nil {
		return err
	}

	return responses.Success(c, "Usuarios asignados correctamente a los menús", nil)
}

// ─────────────────────────────────────────────
// REMOVE USERS → BulkRemoveUsers
// ─────────────────────────────────────────────
func (h *menuHandler) BulkRemoveUsers(c *fiber.Ctx) error {
	ctx := c.UserContext()

	// Validar sesión
	userID, err := getUserIDUint64FromCtx(ctx)
	if err != nil {
		return responses.Error(c, fiber.StatusUnauthorized, "Error de autenticación", err)
	}

	log.Printf("Usuario %d está removiendo usuarios de menús", userID)

	var req requests.BulkAssignMenuUsersRequest // mismo DTO
	if err := c.BodyParser(&req); err != nil {
		return responses.Error(c, fiber.StatusBadRequest, "Error al parsear solicitud", err)
	}

	if err := h.writer.BulkRemoveUsers(ctx, req.MenuIDs, req.UserIDs); err != nil {
		return err
	}

	return responses.Success(c, "Usuarios removidos correctamente de los menús", nil)
}

// ─────────────────────────────────────────────
// GET MENU BY USER (árbol)
// ─────────────────────────────────────────────
func (h *menuHandler) GetMenuByUser(c *fiber.Ctx) error {
	ctx := c.UserContext()

	// Validar token
	userID, err := getUserIDUint64FromCtx(ctx)
	if err != nil {
		return responses.Error(c, fiber.StatusUnauthorized, "Error de autenticación", err)
	}

	log.Printf("Usuario %d está obteniendo su menú", userID)

	tree, err := h.reader.GetMenuByUser(ctx, userID)
	if err != nil {
		return err
	}

	return responses.Success(c, "Menú obtenido correctamente", tree)
}
