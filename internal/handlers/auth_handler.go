package handlers

import (
	"fmt"
	"go-fiber-core/internal/contextkeys"
	"go-fiber-core/internal/domain"
	"go-fiber-core/internal/dtos"
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
	RevokeSession(c *fiber.Ctx) error
	RevokeUserSessions(c *fiber.Ctx) error
	GetActiveSessions(c *fiber.Ctx) error
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

	userAgent := c.Get("User-Agent")
	clientIP := c.IP()

	resp, err := h.authService.Login(c.UserContext(), req, userAgent, clientIP)
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

	return responses.Success(c, "Inicio de sesión exitoso v5", data)
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
	fmt.Println("Entro a cerrar sessión")
	userIDStr, ok := contextkeys.GetUserID(c.UserContext())
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

type revokeSessionRequest struct {
	SessionID string `json:"session_id"`
}

type revokeUserSessionsRequest struct {
	UserID uint64 `json:"user_id"`
}

func (h *authHandler) RevokeSession(c *fiber.Ctx) error {
	var req revokeSessionRequest
	if err := c.BodyParser(&req); err != nil {
		return domain.ErrInvalidArgument
	}
	if req.SessionID == "" {
		return domain.ErrInvalidArgument
	}

	if err := h.authService.RevokeSession(c.Context(), req.SessionID); err != nil {
		return err
	}

	return responses.Success(c, "Sesión revocada exitosamente", nil)
}

func (h *authHandler) RevokeUserSessions(c *fiber.Ctx) error {
	var req revokeUserSessionsRequest
	if err := c.BodyParser(&req); err != nil {
		return domain.ErrInvalidArgument
	}
	if req.UserID == 0 {
		return domain.ErrInvalidArgument
	}

	if err := h.authService.RevokeUserSessions(c.Context(), req.UserID); err != nil {
		return err
	}

	return responses.Success(c, "Sesiones del usuario revocadas exitosamente", nil)
}

func (h *authHandler) GetActiveSessions(c *fiber.Ctx) error {
	var req dtos.PaginationRequest
	if err := c.BodyParser(&req); err != nil {
		return domain.ErrInvalidArgument
	}

	response, err := h.authService.GetActiveSessions(c.Context(), req)
	if err != nil {
		return err
	}

	return responses.Success(c, "Sesiones activas obtenidas exitosamente", response)
}
