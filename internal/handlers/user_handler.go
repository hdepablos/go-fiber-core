package handlers

import (
	"errors"
	"fmt"
	"go-fiber-core/internal/domain"
	"go-fiber-core/internal/dtos"
	"go-fiber-core/internal/dtos/requests"
	"go-fiber-core/internal/dtos/responses"
	"go-fiber-core/internal/models"
	userService "go-fiber-core/internal/services/user"
	"log"
	"strconv"

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
	ActivateUser(c *fiber.Ctx) error
	DeactivateUser(c *fiber.Ctx) error
	SetActiveBulk(c *fiber.Ctx) error
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

	// 1️⃣ Obtenemos el operatorID del contexto
	operatorID, err := getUserIDUint64FromCtx(ctx)
	if err != nil {
		return responses.Error(c, fiber.StatusUnauthorized, "Error de autenticación", err)
	}

	// 2️⃣ Obtenemos el ID del usuario a actualizar desde la URL
	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return domain.ErrInvalidArgument
	}

	// 3️⃣ Parseamos la request
	var req requests.UpdateUserRequest
	if err := c.BodyParser(&req); err != nil {
		return domain.ErrInvalidArgument
	}

	// 4️⃣ Preparamos los roles
	var roleIDs []uint64
	if req.RoleIDs != nil {
		roleIDs = *req.RoleIDs
	}

	// 5️⃣ Llamamos al service para actualizar usuario + roles
	user, err := h.userWriter.UpdateUserWithRoles(
		ctx,
		id,
		userService.UpdateUserDTO{
			Name:  &req.Name,
			Email: &req.Email,
			// No hay Password ni IsActive, solo Name y Email
		},
		roleIDs,     // roles separados
		operatorID,  // operador que hace el cambio
	)
	if err != nil {
		return responses.Error(c, fiber.StatusInternalServerError, "No se pudo actualizar el usuario", err)
	}

	// 6️⃣ Respondemos con el usuario actualizado
	return responses.Success(c, "Usuario actualizado exitosamente", user)
}


func (h *userHandler) SoftDelete(c *fiber.Ctx) error {
	ctx := c.UserContext()

	userID, err := getUserIDUint64FromCtx(ctx)
	if err != nil {
		return responses.Error(c, fiber.StatusUnauthorized, "Error de autenticación", err)
	}

	id, err := getUintID(c)
	if err != nil {
		return err
	}

	log.Printf("Usuario %d está desactivando lógicamente el usuario %d", userID, id)

	if err := h.userWriter.SoftDelete(ctx, uint64(id)); err != nil {
		return err
	}

	return responses.Success(c, "Usuario desactivado lógicamente", nil)
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

func (h *userHandler) ActivateUser(c *fiber.Ctx) error {
	ctx := c.UserContext()

	operatorID, err := getUserIDUint64FromCtx(ctx)
	if err != nil {
		return responses.Error(c, fiber.StatusUnauthorized, "Error de autenticación", err)
	}

	id, err := getUintID(c)
	if err != nil {
		return err
	}

	log.Printf("Usuario %d está reactivando al usuario %d", operatorID, id)

	if err := h.userWriter.Activate(ctx, uint64(id), operatorID); err != nil {
		return err
	}

	return responses.Success(c, "Usuario activado correctamente", nil)
}

func (h *userHandler) DeactivateUser(c *fiber.Ctx) error {
	ctx := c.UserContext()

	operatorID, err := getUserIDUint64FromCtx(ctx)
	if err != nil {
		return responses.Error(c, fiber.StatusUnauthorized, "Error de autenticación", err)
	}

	id, err := getUintID(c)
	if err != nil {
		return err
	}

	log.Printf("Usuario %d está desactivando al usuario %d", operatorID, id)

	if err := h.userWriter.Deactivate(ctx, uint64(id), operatorID); err != nil {
		return err
	}

	return responses.Success(c, "Usuario desactivado correctamente", nil)
}

func (h *userHandler) SetActiveBulk(c *fiber.Ctx) error {
	ctx := c.UserContext()

	var dto requests.BulkSetActiveDTO
	if err := c.BodyParser(&dto); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "JSON inválido")
	}

	if len(dto.IDs) == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "IDs no pueden estar vacíos")
	}

	// Convertir a []uint64 explícitamente
for i, id := range dto.IDs {
    dto.IDs[i] = uint64(id)
}

	// Obtener el operatorID desde el contexto
	operatorID, err := getUserIDUint64FromCtx(ctx)
	if err != nil {
		return responses.Error(c, fiber.StatusUnauthorized, "Error de autenticación", err)
	}

	if err := h.userWriter.SetActiveBulk(ctx, dto.IDs, dto.Active, operatorID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	action := "activados"
	if !dto.Active {
		action = "desactivados"
	}

	return responses.Success(c, fmt.Sprintf("Usuarios %s correctamente", action), nil)

}
