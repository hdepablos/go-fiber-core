package server

import (
	"net"
	"time"

	"go-fiber-core/internal/handlers"
	"go-fiber-core/internal/middleware"
	"go-fiber-core/internal/routes"
	"go-fiber-core/internal/services"
	authService "go-fiber-core/internal/services/auth"
	"go-fiber-core/internal/utils"

	fiber "github.com/gofiber/fiber/v2"
	redis "github.com/redis/go-redis/v9"
)

func (s *FiberServer) RegisterRoutes(
	authHandler handlers.AuthHandler,
	userHandler handlers.UserHandler,
	bankHandler handlers.BankHandler,
	catalogHandler handlers.CatalogHandler,
	rolHandler handlers.RolHandler,
	menuHandler handlers.MenuHandler,
	menuUserHandler handlers.MenuUserHandler,
	dbHandler handlers.DatabaseHandler,
	tokenService authService.TokenService,
	redisClient *redis.Client,
) {
	blacklistBankService := services.NewBlacklistBankService()
	utils.SetupValidator(blacklistBankService)

	// --- REGISTRO DE RUTAS ---
	s.App.Get("/", s.HelloWorldHandler)

	// Grupo base para la API v1
	api := s.App.Group("/api/v1")

	// --- Rutas Públicas ---
	// No requieren token de autenticación.
	api.Get("/health", s.HealthCheckHandler)      // Healthcheck de la aplicación
	routes.RegisterAuthRoutes(api, authHandler)   // Registra /login y /refresh
	routes.RegisterDatabaseRoutes(api, dbHandler) // Registra /health

	// --- Rutas Protegidas ---
	// Requieren un token de autenticación válido.
	authMiddleware := middleware.AuthMiddleware(tokenService, redisClient)
	protected := api.Group("/", authMiddleware)

	// Registramos las rutas que usarán este grupo protegido.
	routes.RegisterProtectedAuthRoutes(protected, authHandler)
	routes.RegisterBankRoutes(protected, bankHandler)
	routes.RegisterUserRoutes(protected, userHandler)
	routes.RegisterCatalogRoutes(protected, catalogHandler)
	routes.RegisterRoleRoutes(protected, rolHandler)
	routes.RegisterMenuRoutes(protected, menuHandler)
	routes.RegisterMenuUserRoutes(protected, menuUserHandler)
}

// --- Handlers del Servidor ---
func (s *FiberServer) HelloWorldHandler(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"message": "Hello World!",
		"IP":      getLocalIP(),
	})
}

func (s *FiberServer) HealthCheckHandler(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status":    "UP",
		"timestamp": time.Now().Format(time.RFC3339),
		"service":   "go-fiber-core",
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
