package handlers

import (
	"fmt" // <-- AÑADIDO: Para impresión de depuración
	"go-fiber-core/internal/domain"
	"go-fiber-core/internal/dtos"
	"go-fiber-core/internal/dtos/requests" //nolint
	"go-fiber-core/internal/dtos/responses"
	"go-fiber-core/internal/models"
	rolService "go-fiber-core/internal/services/rol"
	"log" // <-- AÑADIDO: Para logging de ejemplo

	fiber "github.com/gofiber/fiber/v2"
)

// Interfaz del Handler
type RolHandler interface {
	Create(c *fiber.Ctx) error
	GetAll(c *fiber.Ctx) error
	GetByID(c *fiber.Ctx) error
	Update(c *fiber.Ctx) error
	SoftDelete(c *fiber.Ctx) error
	HardDelete(c *fiber.Ctx) error
	GetAllPaginated(c *fiber.Ctx) error
}

type rolHandler struct {
	writer    rolService.RolWriterService
	reader    rolService.RolReaderService
	paginator rolService.RolPaginationService
}

// Constructor
func NewRolHandler(
	writer rolService.RolWriterService,
	reader rolService.RolReaderService,
	paginator rolService.RolPaginationService,
) RolHandler {
	return &rolHandler{
		writer:    writer,
		reader:    reader,
		paginator: paginator,
	}
}

// --- Métodos ---

func (h *rolHandler) Create(c *fiber.Ctx) error {
	ctx := c.UserContext()

	// AÑADIDO: Obtener el ID de usuario del contexto
	userID, err := getUserIDUint64FromCtx(ctx)
	if err != nil {
		return responses.Error(c, fiber.StatusUnauthorized, "Error de autenticación", err)
	}
	log.Printf("Usuario %d está creando un rol", userID)
	// ---

	var req requests.CreateRoleRequest
	if err := c.BodyParser(&req); err != nil {
		return responses.Error(c, fiber.StatusBadRequest, "Error al parsear el cuerpo de la solicitud", err)
	}

	newRol := models.Role{
		Name:       req.Name,
		IsActive:    true,
	}

	if err := h.writer.Create(ctx, &newRol); err != nil {
		return err
	}

	return responses.Success(c, "Rol creado exitosamente", newRol)
}

func (h *rolHandler) GetAll(c *fiber.Ctx) error {
	ctx := c.UserContext()

	// AÑADIDO: Obtener el ID de usuario del contexto (aunque no se use, valida la sesión)
	_, err := getUserIDUint64FromCtx(ctx)
	if err != nil {
		return responses.Error(c, fiber.StatusUnauthorized, "Error de autenticación", err)
	}
	// ---

	banks, err := h.reader.GetAll(ctx)
	if err != nil {
		return err
	}
	return responses.Success(c, "Roles obtenidos exitosamente", banks)
}

func (h *rolHandler) GetByID(c *fiber.Ctx) error {
	ctx := c.UserContext()

	// AÑADIDO: Obtener el ID de usuario del contexto
	_, err := getUserIDUint64FromCtx(ctx)
	if err != nil {
		return responses.Error(c, fiber.StatusUnauthorized, "Error de autenticación", err)
	}
	// ---

	id, err := getUintID(c) // Asumiendo que getUintID está en helpers.go
	if err != nil {
		return err
	}

	bank, err := h.reader.GetByID(ctx, uint(id))
	if err != nil {
		return err
	}
	return responses.Success(c, "Roles obtenido exitosamente", bank)
}

func (h *rolHandler) Update(c *fiber.Ctx) error {
	ctx := c.UserContext()
	fmt.Println("Entrando a Update del RoleHandler")

	// AÑADIDO: Obtener el ID de usuario del contexto
	userID, err := getUserIDUint64FromCtx(ctx)
	if err != nil {
		return responses.Error(c, fiber.StatusUnauthorized, "Error de autenticación", err)
	}
	// ---

	id, err := getUintID(c) // Asumiendo que getUintID está en helpers.go
	if err != nil {
		return err
	}

	log.Printf("Usuario %d está actualizando el rol %d", userID, id)

	var req requests.UpdateRoleRequest
	if err := c.BodyParser(&req); err != nil {
		return domain.ErrInvalidArgument
	}

	updatedRole, err := h.writer.Update(ctx, uint(id), &models.Role{
		Name:       req.Name,
		IsActive:   req.IsActive,
	})
	if err != nil {
		return err
	}

	return responses.Success(c, "Rol actualizado exitosamente", updatedRole)
}

func (h *rolHandler) SoftDelete(c *fiber.Ctx) error {
	ctx := c.UserContext()
	fmt.Println("Entrando a SoftDelete del RoleHandler")

	// AÑADIDO: Obtener el ID de usuario del contexto
	userID, err := getUserIDUint64FromCtx(ctx)
	if err != nil {
		return responses.Error(c, fiber.StatusUnauthorized, "Error de autenticación", err)
	}
	// ---

	id, err := getUintID(c) // Asumiendo que getUintID está en helpers.go
	if err != nil {
		return err
	}

	log.Printf("Usuario %d está borrando lógicamente el rol %d", userID, id)

	if err := h.writer.SoftDelete(ctx, uint(id)); err != nil {
		return err
	}
	return responses.Success(c, "Rol borrado lógicamente", nil)
}

func (h *rolHandler) HardDelete(c *fiber.Ctx) error {
	ctx := c.UserContext()

	// AÑADIDO: Obtener el ID de usuario del contexto
	userID, err := getUserIDUint64FromCtx(ctx)
	if err != nil {
		return responses.Error(c, fiber.StatusUnauthorized, "Error de autenticación", err)
	}
	// ---

	id, err := getUintID(c) // Asumiendo que getUintID está en helpers.go
	if err != nil {
		return err
	}

	log.Printf("Usuario %d está borrando permanentemente el rol %d", userID, id)

	if err := h.writer.HardDelete(ctx, uint(id)); err != nil {
		return err
	}
	return responses.Success(c, "Rol borrado permanentemente", nil)
}

func (h *rolHandler) GetAllPaginated(c *fiber.Ctx) error {
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

	response, err := h.paginator.GetAllPaginated(ctx, req)
	if err != nil {
		return err
	}
	return responses.Success(c, "Roles paginados obtenidos exitosamente", response)
}
