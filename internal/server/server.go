package server

import (
	"go-fiber-core/internal/dtos/config"
	"go-fiber-core/internal/dtos/connect"
	"go-fiber-core/internal/handlers"
	"go-fiber-core/internal/middleware"
	authService "go-fiber-core/internal/services/auth"
	userService "go-fiber-core/internal/services/user"
	"time"

	fiber "github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
)

// FiberServer representa el servidor principal con todas sus dependencias.
type FiberServer struct {
	*fiber.App
	AppConfig         *config.AppConfig
	UserWriterService userService.UserWriterService // 👈 agregado para el comando createUser
}

// NewFiberServer crea e inicializa el servidor principal.
func NewFiberServer(
	appConfig *config.AppConfig,
	connect *connect.ConnectDTO,
	authHandler handlers.AuthHandler,
	userHandler handlers.UserHandler,
	bankHandler handlers.BankHandler,
	// productHandler handlers.ProductHandler,
	menuHandler handlers.MenuHandler,
	dbHandler handlers.DatabaseHandler,
	tokenService authService.TokenService,
	userWriterService userService.UserWriterService, // 👈 agregado
) (*FiberServer, func(), error) {

	server := &FiberServer{
		App: fiber.New(fiber.Config{
			ServerHeader: appConfig.Server.ServerHeader,
			AppName:      appConfig.App.AppName,
		}),
		AppConfig:         appConfig,
		UserWriterService: userWriterService, // 👈 guardamos la instancia para uso externo
	}

	// Middleware CORS
	server.App.Use(cors.New(cors.Config{
		AllowOrigins:     "http://localhost:9050",
		AllowCredentials: true,
		AllowMethods:     "GET, POST, PUT, DELETE, OPTIONS",
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization, X-Client-Code",
	}))

	// Rate Limiting
	rateLimitConfig := middleware.RateLimitConfig{
		Limit:  100,
		Window: 1 * time.Minute,
	}
	server.App.Use(middleware.RateLimitMiddleware(connect.ConnectRedis, rateLimitConfig))

	// Registrar rutas
	server.RegisterRoutes(authHandler, userHandler, bankHandler, menuHandler, dbHandler, tokenService)
	// server.RegisterRoutes(authHandler, userHandler, bankHandler, dbHandler, tokenService)

	// Cleanup combinado (Wire lo mezcla con cleanup global)
	cleanup := func() {}

	return server, cleanup, nil
}
