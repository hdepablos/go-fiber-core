package handlers

import (
	"crypto/rand"
	"encoding/base64"
	"go-fiber-core/internal/contextkeys"
	"go-fiber-core/internal/domain"
	"go-fiber-core/internal/dtos"
	"go-fiber-core/internal/dtos/requests"
	"go-fiber-core/internal/dtos/responses"
	"go-fiber-core/internal/logger"
	authService "go-fiber-core/internal/services/auth"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	fiber "github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type AuthHandler interface {
	Login(c *fiber.Ctx) error
	Refresh(c *fiber.Ctx) error
	Logout(c *fiber.Ctx) error
	RevokeSession(c *fiber.Ctx) error
	RevokeUserSessions(c *fiber.Ctx) error
	RevokeAllSessions(c *fiber.Ctx) error
	GetActiveSessions(c *fiber.Ctx) error

	GoogleAuth(c *fiber.Ctx) error
	GoogleCallback(c *fiber.Ctx) error
	GoogleExchange(c *fiber.Ctx) error
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

func isLoginMethodEnabled(method string) bool {
	val := strings.ToLower(strings.TrimSpace(os.Getenv("AUTH_LOGIN_METHODS")))
	if val == "" {
		return true
	}

	for _, part := range strings.FieldsFunc(val, func(r rune) bool { return r == ',' || r == ';' || r == ' ' }) {
		if part == method {
			return true
		}
	}

	return false
}

func authFileLogger(component string) *zap.Logger {
	return logger.GetLoggerToFile("auth", logger.ResolveProjectPath("pkg/logs/auth.log")).With(zap.String("component", component))
}

func (h *authHandler) Login(c *fiber.Ctx) error {
	log := authFileLogger("auth_local")
	if !isLoginMethodEnabled("local") {
		log.Warn("local login: method disabled", zap.String("ip", c.IP()), zap.String("ua", c.Get("User-Agent")))
		return fiber.NewError(fiber.StatusForbidden, "login con base de datos deshabilitado")
	}

	var req requests.LoginRequest
	if err := c.BodyParser(&req); err != nil {
		log.Warn("local login: invalid body", zap.Error(err), zap.String("ip", c.IP()))
		return domain.ErrInvalidArgument
	}

	// fmt.Printf("Intentando login para usuario: %s\n", req.Email)

	userAgent := c.Get("User-Agent")
	clientIP := c.IP()
	requestID := strings.TrimSpace(c.Get("X-Request-Id"))
	if requestID == "" {
		requestID = strings.TrimSpace(c.Get("X-Request-ID"))
	}
	origin := strings.TrimSpace(c.Get("X-Origin"))
	if origin == "" {
		if strings.TrimSpace(c.Get("Origin")) != "" {
			origin = "web"
		} else {
			origin = "api"
		}
	}

	log.Info("local login: attempt", zap.String("email", req.Email), zap.String("ip", clientIP), zap.String("ua", userAgent))
	resp, err := h.authService.Login(c.UserContext(), req, userAgent, clientIP, origin, requestID)
	if err != nil {
		log.Warn("local login: failed", zap.String("email", req.Email), zap.String("ip", clientIP), zap.Error(err))
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

	log.Info("local login: success", zap.String("email", req.Email), zap.String("ip", clientIP))
	return responses.Success(c, "Inicio de sesión exitoso v5", data)
}

func (h *authHandler) GoogleAuth(c *fiber.Ctx) error {
	log := authFileLogger("auth_google")
	if !isLoginMethodEnabled("google") {
		log.Warn("google auth: method disabled", zap.String("ip", c.IP()))
		return fiber.NewError(fiber.StatusForbidden, "login con Google deshabilitado")
	}
	// "state" se usa como protección CSRF; se guarda en cookie HttpOnly y se valida en el callback.
	stateBytes := make([]byte, 32)
	if _, err := rand.Read(stateBytes); err != nil {
		log.Error("google auth: error generando state", zap.Error(err))
		return domain.ErrInternal
	}
	state := base64.RawURLEncoding.EncodeToString(stateBytes)

	authURL, err := h.authService.GoogleAuthURL(state)
	if err != nil {
		log.Error("google auth: error generando auth url", zap.Error(err), zap.String("ip", c.IP()))
		return err
	}

	if err := h.authService.SaveGoogleOAuthState(c.UserContext(), state); err != nil {
		log.Error("google auth: error guardando oauth state", zap.Error(err), zap.String("ip", c.IP()))
		return domain.ErrInternal
	}

	c.Cookie(&fiber.Cookie{
		Name:     "google_oauth_state",
		Value:    state,
		Path:     "/api/v1/oauth/google/callback",
		MaxAge:   600,
		HTTPOnly: true,
		Secure:   c.Protocol() == "https",
		SameSite: "Lax",
		Expires:  time.Now().Add(10 * time.Minute),
	})

	log.Info("google auth: redirecting to provider", zap.String("ip", c.IP()), zap.String("ua", c.Get("User-Agent")), zap.Int("location_len", len(authURL)))
	if strings.Contains(strings.ToLower(c.Get("Accept")), "application/json") || c.Query("mode") == "json" {
		return responses.Success(c, "Google OAuth listo", fiber.Map{"auth_url": authURL})
	}
	return c.Redirect(authURL, fiber.StatusFound)
}

func (h *authHandler) GoogleCallback(c *fiber.Ctx) error {
	log := authFileLogger("auth_google")
	if !isLoginMethodEnabled("google") {
		log.Warn("google callback: method disabled", zap.String("ip", c.IP()))
		return fiber.NewError(fiber.StatusForbidden, "login con Google deshabilitado")
	}
	// Callback OAuth: valida state/code, intercambia token y retorna JWT propios del API.
	if oauthErr := c.Query("error"); oauthErr != "" {
		log.Warn("google callback: provider returned error", zap.String("error", oauthErr), zap.String("ip", c.IP()))
		return domain.ErrAuthentication
	}

	code := c.Query("code")
	if code == "" {
		log.Warn("google callback: missing code", zap.String("ip", c.IP()))
		return domain.ErrInvalidArgument
	}

	state := c.Query("state")
	if state == "" {
		log.Warn("google callback: missing state", zap.String("ip", c.IP()))
		return domain.ErrAuthentication
	}

	cookieState := c.Cookies("google_oauth_state")
	if cookieState != "" {
		if cookieState != state {
			log.Warn(
				"google callback: state mismatch (cookie)",
				zap.String("ip", c.IP()),
				zap.Int("cookie_state_len", len(cookieState)),
				zap.Int("query_state_len", len(state)),
			)
			return domain.ErrAuthentication
		}
		log.Info("google callback: state ok (cookie)", zap.String("ip", c.IP()))
	} else {
		ok, err := h.authService.ConsumeGoogleOAuthState(c.UserContext(), state)
		if err != nil {
			log.Error("google callback: error validando oauth state (redis)", zap.Error(err), zap.String("ip", c.IP()))
			return domain.ErrInternal
		}
		if !ok {
			log.Warn("google callback: invalid state (redis)", zap.String("ip", c.IP()))
			return domain.ErrAuthentication
		}
		log.Info("google callback: state ok (redis)", zap.String("ip", c.IP()))
	}

	c.Cookie(&fiber.Cookie{
		Name:     "google_oauth_state",
		Value:    "",
		Path:     "/api/v1/oauth/google/callback",
		MaxAge:   -1,
		HTTPOnly: true,
		Secure:   c.Protocol() == "https",
		SameSite: "Lax",
	})

	userAgent := c.Get("User-Agent")
	clientIP := c.IP()
	requestID := strings.TrimSpace(c.Get("X-Request-Id"))
	if requestID == "" {
		requestID = strings.TrimSpace(c.Get("X-Request-ID"))
	}
	origin := strings.TrimSpace(c.Get("X-Origin"))
	if origin == "" {
		if strings.TrimSpace(c.Get("Origin")) != "" {
			origin = "web"
		} else {
			origin = "api"
		}
	}

	log.Info("google callback: starting login", zap.String("ip", clientIP), zap.String("ua", userAgent))
	resp, err := h.authService.GoogleCallbackLogin(c.UserContext(), code, userAgent, clientIP, origin, requestID)
	if err != nil {
		log.Error("google callback: login failed", zap.Error(err), zap.String("ip", clientIP))
		successRedirect := strings.TrimSpace(os.Getenv("GOOGLE_SUCCESS_REDIRECT_URL"))
		accept := strings.ToLower(c.Get("Accept"))
		if successRedirect != "" && (strings.Contains(accept, "text/html") || c.Query("mode") == "redirect") {
			u, parseErr := url.Parse(successRedirect)
			if parseErr != nil {
				log.Error("google callback: invalid GOOGLE_SUCCESS_REDIRECT_URL", zap.Error(parseErr))
				return domain.ErrInternal
			}

			q := url.Values{}
			q.Set("error", err.Error())
			u.Fragment = q.Encode()
			log.Info("google callback: redirecting with error", zap.String("ip", clientIP), zap.String("redirect_host", u.Host))
			return c.Redirect(u.String(), fiber.StatusFound)
		}

		return err
	}

	log.Info("google callback: login ok", zap.String("ip", clientIP), zap.Uint64("user_id", resp.UserID))

	successRedirect := strings.TrimSpace(os.Getenv("GOOGLE_SUCCESS_REDIRECT_URL"))
	accept := strings.ToLower(c.Get("Accept"))
	if successRedirect != "" && (strings.Contains(accept, "text/html") || c.Query("mode") == "redirect") {
		u, err := url.Parse(successRedirect)
		if err != nil {
			log.Error("google callback: invalid GOOGLE_SUCCESS_REDIRECT_URL", zap.Error(err))
			return domain.ErrInternal
		}

		q := url.Values{}
		exchangeCode := uuid.NewString()
		if err := h.authService.SaveGoogleOAuthLoginResult(c.UserContext(), exchangeCode, resp); err != nil {
			log.Error("google callback: error guardando oauth login result", zap.Error(err), zap.String("ip", clientIP))
			return domain.ErrInternal
		}
		q.Set("oauth_code", exchangeCode)
		q.Set("provider", "google")
		u.Fragment = q.Encode()

		log.Info("google callback: redirecting to success url", zap.String("ip", clientIP), zap.String("redirect_host", u.Host))
		return c.Redirect(u.String(), fiber.StatusFound)
	}

	return responses.Success(c, "Inicio de sesión con Google exitoso", resp)
}

type googleExchangeRequest struct {
	Code string `json:"code"`
}

func (h *authHandler) GoogleExchange(c *fiber.Ctx) error {
	log := authFileLogger("auth_google")
	if !isLoginMethodEnabled("google") {
		log.Warn("google exchange: method disabled", zap.String("ip", c.IP()))
		return fiber.NewError(fiber.StatusForbidden, "login con Google deshabilitado")
	}

	var req googleExchangeRequest
	if err := c.BodyParser(&req); err != nil {
		log.Warn("google exchange: invalid body", zap.Error(err), zap.String("ip", c.IP()))
		return domain.ErrInvalidArgument
	}
	if strings.TrimSpace(req.Code) == "" {
		return domain.ErrInvalidArgument
	}

	result, err := h.authService.ConsumeGoogleOAuthLoginResult(c.UserContext(), strings.TrimSpace(req.Code))
	if err != nil {
		log.Warn("google exchange: invalid code", zap.Error(err), zap.String("ip", c.IP()))
		return err
	}

	data := fiber.Map{
		"access_token":  result.AccessToken,
		"refresh_token": result.RefreshToken,
		"user_name":     result.UserName,
		"role_ids":      result.RoleIDs,
		"roles":         result.Roles,
		"menu":          result.Menu,
	}

	log.Info("google exchange: success", zap.String("ip", c.IP()), zap.Uint64("user_id", result.UserID))
	return responses.Success(c, "Inicio de sesión con Google exitoso", data)
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
	log := authFileLogger("auth_local")
	userIDStr, ok := contextkeys.GetUserID(c.UserContext())
	if !ok || userIDStr == "" {
		log.Warn("logout: missing user_id", zap.String("ip", c.IP()), zap.String("ua", c.Get("User-Agent")))
		return domain.ErrInvalidArgument
	}

	userID, err := strconv.ParseUint(userIDStr, 10, 64)
	if err != nil {
		log.Warn("logout: invalid user_id", zap.String("ip", c.IP()), zap.String("ua", c.Get("User-Agent")))
		return domain.ErrInvalidArgument
	}

	userAgent := c.Get("User-Agent")
	clientIP := c.IP()
	requestID := strings.TrimSpace(c.Get("X-Request-Id"))
	if requestID == "" {
		requestID = strings.TrimSpace(c.Get("X-Request-ID"))
	}
	origin := strings.TrimSpace(c.Get("X-Origin"))
	if origin == "" {
		if strings.TrimSpace(c.Get("Origin")) != "" {
			origin = "web"
		} else {
			origin = "api"
		}
	}

	log.Info("logout: attempt", zap.Uint64("user_id", userID), zap.String("ip", clientIP), zap.String("ua", userAgent))
	if err := h.authService.Logout(c.Context(), userID, userAgent, clientIP, origin, requestID); err != nil {
		log.Warn("logout: failed", zap.Uint64("user_id", userID), zap.String("ip", clientIP), zap.Error(err))
		return err
	}

	log.Info("logout: success", zap.Uint64("user_id", userID), zap.String("ip", clientIP))
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

func (h *authHandler) RevokeAllSessions(c *fiber.Ctx) error {
	if err := h.authService.RevokeAllSessions(c.Context()); err != nil {
		return err
	}

	return responses.Success(c, "todas las sesiones fueron revocadas", nil)
}
