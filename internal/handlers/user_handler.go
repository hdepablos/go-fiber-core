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

	fiber "github.com/gofiber/fiber/v2"
)

// La interfaz del Handler no cambia
type UserHandler interface {
	CreateUser(c *fiber.Ctx) error
	// CreateUserWithRelations(c *fiber.Ctx) error // 👈 Nuevo método
	// CreateUserWithExistingRelations(c *fiber.Ctx) error
	// CreateUserWithNewProductsAndRolesIfNotExist(c *fiber.Ctx) error
	GetAllUsers(c *fiber.Ctx) error
	GetUserByID(c *fiber.Ctx) error
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

	requestingUserID, err := getUserIDUint64FromCtx(ctx)
	if err != nil {
		return responses.Error(c, fiber.StatusUnauthorized, "Error de autenticación", err)
	}
	log.Printf("Usuario %d está creando un nuevo usuario", requestingUserID)

	var req requests.CreateUserRequest
	if err := c.BodyParser(&req); err != nil {
		return domain.ErrInvalidArgument
	}

	user := &models.User{
		Name:     req.Name,
		Email:    req.Email,
		Password: req.Password,
	}

	if err := h.userWriter.Create(ctx, user); err != nil {
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

// func (h *userHandler) CreateUserWithRelations(c *fiber.Ctx) error {
// 	ctx := c.UserContext()
// 	var req requests.CreateUserWithRelationsRequest

// 	if err := c.BodyParser(&req); err != nil {
// 		// return responses.Error(c, 400, err.Error())
// 		return responses.Error(c, 422, domain.ErrInvalidArgument.Error())
// 		// return domain.ErrInvalidArgument
// 	}

// 	// Convertir los productos y roles del request a modelos
// 	user := &models.User{
// 		Name:     req.Name,
// 		Email:    req.Email,
// 		Password: req.Password,
// 	}

// 	// Asignar productos (si hay)
// 	for _, p := range req.Products {
// 		user.Products = append(user.Products, models.Product{
// 			Name:  p.Name,
// 			Price: p.Price,
// 		})
// 	}

// 	// Obtener los IDs de roles
// 	roleIDs := make([]uint64, len(req.Roles))
// 	for i, r := range req.Roles {
// 		roleIDs[i] = r.ID
// 	}

// 	// Crear usuario + relaciones
// 	if err := h.userDeactivation.CreateWithProductsAndRoles(ctx, user, roleIDs); err != nil {
// 		return responses.Error(c, 400, err.Error())
// 	}

// 	return responses.Success(c, "Usuario creado con productos y roles exitosamente", user)
// }

// // Crear usuario con productos y roles ya existentes (por IDs)
// func (h *userHandler) CreateUserWithExistingRelations(c *fiber.Ctx) error {
// 	ctx := c.UserContext()
// 	var req requests.CreateUserWithExistingRelationsRequest

// 	if err := c.BodyParser(&req); err != nil {
// 		return responses.Error(c, 422, domain.ErrInvalidArgument.Error())
// 	}

// 	user := &models.User{
// 		Name:     req.Name,
// 		Email:    req.Email,
// 		Password: req.Password,
// 	}

// 	// Llama al service que maneja productos y roles existentes
// 	if err := h.userDeactivation.CreateWithExistingProductsAndRoles(ctx, user, req.ProductIDs, req.RoleIDs); err != nil {
// 		return responses.Error(c, 400, err.Error())
// 	}

// 	return responses.Success(c, "Usuario creado con productos y roles existentes", user)
// }

// func (h *userHandler) CreateUserWithNewProductsAndRolesIfNotExist(c *fiber.Ctx) error {
// 	ctx := c.UserContext()
// 	var req requests.CreateUserWithNewProductsAndRolesRequest

// 	if err := c.BodyParser(&req); err != nil {
// 		return responses.Error(c, 422, domain.ErrInvalidArgument.Error())
// 	}

// 	user := &models.User{
// 		Name:     req.Name,
// 		Email:    req.Email,
// 		Password: req.Password,
// 	}

// 	if err := h.userDeactivation.CreateUserWithNewProductsAndRolesIfNotExist(ctx, user, req.Products, req.Roles); err != nil {
// 		return responses.Error(c, 400, err.Error())
// 	}

// 	return responses.Success(c, "Usuario creado con productos y roles nuevos exitosamente", user)
// }

// func (h *userHandler) CreateUserWithRelations(c *fiber.Ctx) error {
// 	ctx := c.UserContext()
// 	var req requests.CreateUserWithRelationsRequest

// 	if err := c.BodyParser(&req); err != nil {
// 		// return responses.Error(c, 400, err.Error())
// 		return responses.Error(c, 422, domain.ErrInvalidArgument.Error())
// 		// return domain.ErrInvalidArgument
// 	}

// 	// Convertir los productos y roles del request a modelos
// 	user := &models.User{
// 		Name:     req.Name,
// 		Email:    req.Email,
// 		Password: req.Password,
// 	}

// 	// Asignar productos (si hay)
// 	for _, p := range req.Products {
// 		user.Products = append(user.Products, models.Product{
// 			Name:  p.Name,
// 			Price: p.Price,
// 		})
// 	}

// 	// Obtener los IDs de roles
// 	roleIDs := make([]uint64, len(req.Roles))
// 	for i, r := range req.Roles {
// 		roleIDs[i] = r.ID
// 	}

// 	// Crear usuario + relaciones
// 	if err := h.userDeactivation.CreateWithProductsAndRoles(ctx, user, roleIDs); err != nil {
// 		return responses.Error(c, 400, err.Error())
// 	}

// 	return responses.Success(c, "Usuario creado con productos y roles exitosamente", user)
// }

// Crear usuario con productos y roles ya existentes (por IDs)
// func (h *userHandler) CreateUserWithExistingRelations(c *fiber.Ctx) error {
// 	ctx := c.UserContext()
// 	var req requests.CreateUserWithExistingRelationsRequest

// 	if err := c.BodyParser(&req); err != nil {
// 		return responses.Error(c, 422, domain.ErrInvalidArgument.Error())
// 	}

// 	user := &models.User{
// 		Name:     req.Name,
// 		Email:    req.Email,
// 		Password: req.Password,
// 	}

// 	// Llama al service que maneja productos y roles existentes
// 	if err := h.userDeactivation.CreateWithExistingProductsAndRoles(ctx, user, req.ProductIDs, req.RoleIDs); err != nil {
// 		return responses.Error(c, 400, err.Error())
// 	}

// 	return responses.Success(c, "Usuario creado con productos y roles existentes", user)
// }

// func (h *userHandler) CreateUserWithNewProductsAndRolesIfNotExist(c *fiber.Ctx) error {
// 	ctx := c.UserContext()
// 	var req requests.CreateUserWithNewProductsAndRolesRequest

// 	if err := c.BodyParser(&req); err != nil {
// 		return responses.Error(c, 422, domain.ErrInvalidArgument.Error())
// 	}

// 	user := &models.User{
// 		Name:     req.Name,
// 		Email:    req.Email,
// 		Password: req.Password,
// 	}

// 	if err := h.userDeactivation.CreateUserWithNewProductsAndRolesIfNotExist(ctx, user, req.Products, req.Roles); err != nil {
// 		return responses.Error(c, 400, err.Error())
// 	}

// 	return responses.Success(c, "Usuario creado con productos y roles nuevos exitosamente", user)
// }
