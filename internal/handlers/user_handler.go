package handlers

import (
	"go-fiber-core/internal/domain"
	"go-fiber-core/internal/dtos"
	"go-fiber-core/internal/dtos/requests"
	"go-fiber-core/internal/dtos/responses"
	"go-fiber-core/internal/models"
	userService "go-fiber-core/internal/services/user"
	"log"
	"strconv"
	"errors"
	"gorm.io/gorm"

	fiber "github.com/gofiber/fiber/v2"
)

// La interfaz del Handler no cambia
type UserHandler interface {
	CreateUser(c *fiber.Ctx) error
	// CreateUserWithRelations(c *fiber.Ctx) error // 👈 Nuevo método
	// CreateUserWithExistingRelations(c *fiber.Ctx) error
	// CreateUserWithNewProductsAndRolesIfNotExist(c *fiber.Ctx) error
	AssignRolesToUsers(c *fiber.Ctx) error
	GetAllUsers(c *fiber.Ctx) error
	GetUserByID(c *fiber.Ctx) error
	RemoveRoles(c *fiber.Ctx) error
	UpdateUser(c *fiber.Ctx) error
	SoftDelete(c *fiber.Ctx) error
	HardDelete(c *fiber.Ctx) error
	GetAllPaginatedUsers(c *fiber.Ctx) error
}

type userHandler struct {
	userWriter userService.UserWriterService
	userReader userService.UserReaderService
	// userDeactivation userService.DeactivationService
}

func NewUserHandler(writer userService.UserWriterService, reader userService.UserReaderService) UserHandler {
	return &userHandler{
		userWriter: writer,
		userReader: reader,
	}
}

// --- Handler Methods ---

func (h *userHandler) CreateUser(c *fiber.Ctx) error {
	ctx := c.UserContext()

	var req requests.CreateUserRequest
	if err := c.BodyParser(&req); err != nil {
		return domain.ErrInvalidArgument
	}

	// Validación mínima
	if len(req.RoleIDs) == 0 {
		return domain.ErrInvalidArgument
	}
	for _, id := range req.RoleIDs {
		if id == 0 {
			return domain.ErrInvalidArgument
		}
	}

	user := &models.User{
		Name:     req.Name,
		Email:    req.Email,
		Password: req.Password,
	}

	if err := h.userWriter.CreateWithRole(ctx, user, req.RoleIDs); err != nil {
		return err
	}

	return responses.Success(c, "Usuario creado exitosamente", user)
}


func (h *userHandler) GetAllUsers(c *fiber.Ctx) error {
	ctx := c.UserContext()

	// AÑADIDO: Obtener el ID de usuario del contexto
	_, err := getUserIDUint64FromCtx(ctx)
	if err != nil {
		return responses.Error(c, fiber.StatusUnauthorized, "Error de autenticación", err)
	}
	// ---

	users, err := h.userReader.GetAll(ctx)
	if err != nil {
		return err
	}
	return responses.Success(c, "Usuarios obtenidos exitosamente", users)
}

func (h *userHandler) GetUserByID(c *fiber.Ctx) error {
	ctx := c.UserContext()

	// AÑADIDO: Obtener el ID de usuario del contexto
	requestingUserID, err := getUserIDUint64FromCtx(ctx)
	if err != nil {
		return responses.Error(c, fiber.StatusUnauthorized, "Error de autenticación", err)
	}
	// ---

	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return domain.ErrInvalidArgument
	}

	log.Printf("Usuario %d está solicitando el usuario %d", requestingUserID, id)

	user, err := h.userReader.GetByID(ctx, id)
	if err != nil {
		return err
	}

	return responses.Success(c, "Usuario obtenido exitosamente", user)
}

func (h *userHandler) UpdateUser(c *fiber.Ctx) error {
	ctx := c.UserContext()

	// AÑADIDO: Obtener el ID de usuario del contexto
	requestingUserID, err := getUserIDUint64FromCtx(ctx)
	if err != nil {
		return responses.Error(c, fiber.StatusUnauthorized, "Error de autenticación", err)
	}
	// ---

	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return domain.ErrInvalidArgument
	}

	log.Printf("Usuario %d está intentando actualizar al usuario %d", requestingUserID, id)

	// LÓGICA DE NEGOCIO: Aquí podrías verificar permisos
	// ej: if requestingUserID != id && !esAdmin(requestingUserID) {
	// 	   return responses.Error(c, fiber.StatusForbidden, "No tiene permisos", nil)
	// }

	var req requests.UpdateUserRequest
	if err := c.BodyParser(&req); err != nil {
		return domain.ErrInvalidArgument
	}

	updateDTO := userService.UpdateUserDTO{
		Name:  &req.Name,
		Email: &req.Email,
	}

	updatedUser, err := h.userWriter.Update(ctx, id, updateDTO)
	if err != nil {
		return err
	}

	return responses.Success(c, "Usuario actualizado exitosamente", updatedUser)
}

func (h *userHandler) SoftDelete(c *fiber.Ctx) error {
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

	if err := h.userWriter.SoftDelete(ctx, uint64(id)); err != nil {
		return err
	}
	return responses.Success(c, "Banco borrado lógicamente", nil)
}

func (h *userHandler) HardDelete(c *fiber.Ctx) error {
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

	if err := h.userWriter.HardDelete(ctx, uint(id)); err != nil {
		return err
	}
	return responses.Success(c, "Banco borrado permanentemente", nil)
}

func (h *userHandler) GetAllPaginatedUsers(c *fiber.Ctx) error {
	ctx := c.UserContext()

	// AÑADIDO: Obtener el ID de usuario del contexto
	_, err := getUserIDUint64FromCtx(ctx)
	if err != nil {
		return responses.Error(c, fiber.StatusUnauthorized, "Error de autenticación", err)
	}
	// ---

	var req dtos.PaginationRequest
	if err := c.BodyParser(&req); err != nil {
		return domain.ErrInvalidArgument
	}

	response, err := h.userReader.GetAllPaginated(ctx, req)
	if err != nil {
		return err
	}

	return responses.Success(c, "Usuarios paginados obtenidos exitosamente", response)
}

func (h *userHandler) RemoveRoles(c *fiber.Ctx) error {
	ctx := c.UserContext()

	var req requests.RemoveUserRolesRequest
	if err := c.BodyParser(&req); err != nil {
		return domain.ErrInvalidArgument
	}

	if len(req.UserIDs) == 0 || len(req.RoleIDs) == 0 {
		return domain.ErrInvalidArgument
	}

	for _, id := range req.UserIDs {
		if id == 0 {
			return domain.ErrInvalidArgument
		}
	}
	for _, id := range req.RoleIDs {
		if id == 0 {
			return domain.ErrInvalidArgument
		}
	}

	if err := h.userWriter.RemoveRolesFromUsers(ctx, req.UserIDs, req.RoleIDs); err != nil {
		return err
	}

	return responses.Success(c, "Roles desasignados correctamente", nil)
}

func (h *userHandler) AssignRolesToUsers(c *fiber.Ctx) error {
	var req requests.AssignRolesRequest

	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "json inválido")
	}

	if len(req.UserIDs) == 0 || len(req.RoleIDs) == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "user_ids y role_ids son obligatorios")
	}

	if err := h.userWriter.AssignRolesToUsers(
		c.Context(),
		req.UserIDs,
		req.RoleIDs,
	); err != nil {

		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "usuario o rol inexistente")
		}

		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status":  "ok",
		"message": "roles asignados correctamente",
	})
}
