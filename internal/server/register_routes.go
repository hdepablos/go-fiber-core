package server

import (
	"net"

	"go-fiber-core/internal/handlers"
	"go-fiber-core/internal/middleware"
	"go-fiber-core/internal/routes"
	"go-fiber-core/internal/services"
	authService "go-fiber-core/internal/services/auth"
	"go-fiber-core/internal/utils"

	fiber "github.com/gofiber/fiber/v2"
)

func (s *FiberServer) RegisterRoutes(
	authHandler handlers.AuthHandler,
	userHandler handlers.UserHandler,
	bankHandler handlers.BankHandler,
	// productHandler handlers.ProductHandler,
	menuHandler handlers.MenuHandler,
	dbHandler handlers.DatabaseHandler,
	tokenService authService.TokenService,
) {
	blacklistBankService := services.NewBlacklistBankService()
	utils.SetupValidator(blacklistBankService)

	// --- REGISTRO DE RUTAS ---
	s.App.Get("/", s.HelloWorldHandler)

	// Grupo base para la API v1
	api := s.App.Group("/api/v1")

	// --- Rutas Públicas ---
	// No requieren token de autenticación.
	routes.RegisterAuthRoutes(api, authHandler)   // Registra /login y /refresh
	routes.RegisterDatabaseRoutes(api, dbHandler) // Registra /health

	// --- Rutas Protegidas ---
	// Requieren un token de autenticación válido.
	authMiddleware := middleware.AuthMiddleware(tokenService)
	protected := api.Group("/", authMiddleware)

	// Registramos las rutas que usarán este grupo protegido.
	protected.Post("/auth/logout", authHandler.Logout)
	routes.RegisterBankRoutes(protected, bankHandler)
	routes.RegisterUserRoutes(protected, userHandler)
	// routes.RegisterProductRoutes(protected, productHandler)
	routes.RegisterMenuRoutes(protected, menuHandler)
}

// --- Handlers del Servidor ---
func (s *FiberServer) HelloWorldHandler(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"message": "Hello World!",
		"IP":      getLocalIP(),
	})
}

func getLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "unknown"
	}
	for _, addr := range addrs {
		if ipNet, ok := addr.(*net.IPNet); ok && !ipNet.IP.IsLoopback() {
			if ipNet.IP.To4() != nil {
				return ipNet.IP.String()
			}
		}
	}
	return "unknown"
}
