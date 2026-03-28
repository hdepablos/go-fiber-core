// internal/routes/auth_routes.go
package routes

import (
	"go-fiber-core/internal/handlers"

	fiber "github.com/gofiber/fiber/v2"
)

// SetupAuthRoutes ahora acepta un fiber.Router y la interfaz del handler.
func RegisterAuthRoutes(router fiber.Router, authHandler handlers.AuthHandler) {
	auth := router.Group("/auth")

	auth.Post("/login", authHandler.Login)
	auth.Post("/refresh", authHandler.Refresh)
	auth.Get("/google", authHandler.GoogleAuth)
	auth.Post("/google/exchange", authHandler.GoogleExchange)

	router.Get("/oauth/google/callback", authHandler.GoogleCallback)
}

// RegisterProtectedAuthRoutes registra las rutas de autenticación que requieren protección (middleware).
func RegisterProtectedAuthRoutes(router fiber.Router, authHandler handlers.AuthHandler) {
	auth := router.Group("/auth")

	auth.Post("/logout", authHandler.Logout)
	auth.Post("/revoke-session", authHandler.RevokeSession)
	auth.Post("/revoke-user-sessions", authHandler.RevokeUserSessions)
	auth.Post("/active-sessions", authHandler.GetActiveSessions)
	auth.Post("/revoke-all-sessions", authHandler.RevokeAllSessions)
}
