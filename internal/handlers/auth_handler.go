package handlers

import (
	"go-fiber-core/internal/domain"
	"go-fiber-core/internal/dtos/requests"
	"go-fiber-core/internal/dtos/responses"
	authService "go-fiber-core/internal/services/auth"
	"strconv"

	fiber "github.com/gofiber/fiber/v2"
)

type AuthHandler interface {
	Login(c *fiber.Ctx) error
	Refresh(c *fiber.Ctx) error
	Logout(c *fiber.Ctx) error
}

type authHandler struct {
	authService authService.AuthService
}

func NewAuthHandler(authService authService.AuthService) AuthHandler {
	return &authHandler{
		authService: authService,
	}
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

func (h *authHandler) Login(c *fiber.Ctx) error {

	var req requests.LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return domain.ErrInvalidArgument
	}

	// fmt.Printf("Intentando login para usuario: %s\n", req.Email)

	resp, err := h.authService.Login(c.Context(), req)
	if err != nil {
		return err
	}

	data := fiber.Map{
		"access_token":  resp.Token,
		"refresh_token": resp.RefreshToken,
		"user_name":     resp.UserName,
		"role_ids":      resp.RoleIDs,
		"roles":         resp.Roles,
		"menu":          resp.Menu,
	}

	return responses.Success(c, "Inicio de sesión exitoso", data)
}

func (h *authHandler) Refresh(c *fiber.Ctx) error {
	var req refreshRequest
	if err := c.BodyParser(&req); err != nil {
		return domain.ErrInvalidArgument
	}

	if req.RefreshToken == "" {
		return domain.ErrInvalidArgument
	}

	newAccessToken, newRefreshToken, err := h.authService.Refresh(c.Context(), req.RefreshToken)
	if err != nil {
		return err
	}

	data := fiber.Map{
		"access_token":  newAccessToken,
		"refresh_token": newRefreshToken,
	}
	return responses.Success(c, "Token refrescado exitosamente", data)
}

func (h *authHandler) Logout(c *fiber.Ctx) error {
	userIDStr, ok := c.Locals("userID").(string)
	if !ok || userIDStr == "" {
		// Este es un error de autorización que el middleware de errores
		// puede traducir a un 401 si lo configuramos.
		return domain.ErrInvalidArgument
	}

	userID, err := strconv.ParseUint(userIDStr, 10, 64)
	if err != nil {
		return domain.ErrInvalidArgument
	}

	if err := h.authService.Logout(c.Context(), userID); err != nil {
		return err
	}

	return responses.Success(c, "Cierre de sesión exitoso", nil)
}
