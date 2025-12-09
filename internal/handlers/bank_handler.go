package handlers

import (
	"go-fiber-core/internal/domain"
	"go-fiber-core/internal/dtos"
	"go-fiber-core/internal/dtos/requests" //nolint
	"go-fiber-core/internal/dtos/responses"
	"go-fiber-core/internal/models"
	bankService "go-fiber-core/internal/services/bank"
	"log" // <-- AÑADIDO: Para logging de ejemplo

	fiber "github.com/gofiber/fiber/v2"
)

// Interfaz del Handler
type BankHandler interface {
	Create(c *fiber.Ctx) error
	GetAll(c *fiber.Ctx) error
	GetByID(c *fiber.Ctx) error
	Update(c *fiber.Ctx) error
	SoftDelete(c *fiber.Ctx) error
	HardDelete(c *fiber.Ctx) error
	GetAllPaginated(c *fiber.Ctx) error
}

// Handler concreto
type bankHandler struct {
	writer    bankService.BankWriterService
	reader    bankService.BankReaderService
	paginator bankService.BankPaginationService
}

// Constructor
func NewBankHandler(
	writer bankService.BankWriterService,
	reader bankService.BankReaderService,
	paginator bankService.BankPaginationService,
) BankHandler {
	return &bankHandler{
		writer:    writer,
		reader:    reader,
		paginator: paginator,
	}
}

// --- Métodos ---

func (h *bankHandler) Create(c *fiber.Ctx) error {
	ctx := c.UserContext()

	// AÑADIDO: Obtener el ID de usuario del contexto
	userID, err := getUserIDUint64FromCtx(ctx)
	if err != nil {
		return responses.Error(c, fiber.StatusUnauthorized, "Error de autenticación", err)
	}
	log.Printf("Usuario %d está creando un banco", userID)
	// ---

	var req requests.CreateBankRequest
	if err := c.BodyParser(&req); err != nil {
		return responses.Error(c, fiber.StatusBadRequest, "Error al parsear el cuerpo de la solicitud", err)
	}

	newBank := models.Bank{
		Name:       req.Name,
		EntityCode: req.EntityCode,
		Enabled:    true,
		// Opcional: Podrías añadir CreatedByUserID: userID aquí si tu modelo lo soporta
	}

	if err := h.writer.Create(ctx, &newBank); err != nil {
		return err
	}

	return responses.Success(c, "Banco creado exitosamente", newBank)
}

func (h *bankHandler) GetAll(c *fiber.Ctx) error {
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
	return responses.Success(c, "Bancos obtenidos exitosamente", banks)
}

func (h *bankHandler) GetByID(c *fiber.Ctx) error {
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
	return responses.Success(c, "Banco obtenido exitosamente", bank)
}

func (h *bankHandler) Update(c *fiber.Ctx) error {
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

	log.Printf("Usuario %d está actualizando el banco %d", userID, id)

	var req requests.UpdateBankRequest
	if err := c.BodyParser(&req); err != nil {
		return domain.ErrInvalidArgument
	}

	updatedBank, err := h.writer.Update(ctx, uint(id), &models.Bank{
		Name:       req.Name,
		EntityCode: req.EntityCode,
		Enabled:    req.Enabled,
	})
	if err != nil {
		return err
	}

	return responses.Success(c, "Banco actualizado exitosamente", updatedBank)
}

func (h *bankHandler) SoftDelete(c *fiber.Ctx) error {
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

	log.Printf("Usuario %d está borrando lógicamente el banco %d", userID, id)

	if err := h.writer.SoftDelete(ctx, uint(id)); err != nil {
		return err
	}
	return responses.Success(c, "Banco borrado lógicamente", nil)
}

func (h *bankHandler) HardDelete(c *fiber.Ctx) error {
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

	log.Printf("Usuario %d está borrando permanentemente el banco %d", userID, id)

	if err := h.writer.HardDelete(ctx, uint(id)); err != nil {
		return err
	}
	return responses.Success(c, "Banco borrado permanentemente", nil)
}

func (h *bankHandler) GetAllPaginated(c *fiber.Ctx) error {
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
	return responses.Success(c, "Bancos paginados obtenidos exitosamente", response)
}
