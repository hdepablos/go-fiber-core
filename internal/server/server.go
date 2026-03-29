package server

import (
	"go-fiber-core/internal/dtos/config"
	"go-fiber-core/internal/dtos/connect"
	"go-fiber-core/internal/handlers"
	"go-fiber-core/internal/middleware"
	authService "go-fiber-core/internal/services/auth"
	"go-fiber-core/internal/services/queue"
	userService "go-fiber-core/internal/services/user"
	"os"
	"strings"
	"time"

	fiber "github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
)

// FiberServer representa el servidor principal con todas sus dependencias.
type FiberServer struct {
	*fiber.App
	AppConfig         *config.AppConfig
	UserWriterService userService.UserWriterService
	QueueService      *queue.SQSService // 👈 Nuevo campo para acceso global
}

// NewFiberServer crea e inicializa el servidor principal.
func NewFiberServer(
	appConfig *config.AppConfig,
	connect *connect.ConnectDTO,
	authHandler handlers.AuthHandler,
	userHandler handlers.UserHandler,
	bankHandler handlers.BankHandler,
	catalogHandler handlers.CatalogHandler,
	rolHandler handlers.RolHandler,
	menuHandler handlers.MenuHandler,
	menuUserHandler handlers.MenuUserHandler,
	dbHandler handlers.DatabaseHandler,
	processLifecycleHandler handlers.ProcessLifecycleHandler,
	tokenService authService.TokenService,
	userWriterService userService.UserWriterService,
	queueService *queue.SQSService, // 👈 Nuevo parámetro
) (*FiberServer, func(), error) {

	server := &FiberServer{
		App: fiber.New(fiber.Config{
			ServerHeader: appConfig.Server.ServerHeader,
			AppName:      appConfig.App.AppName,
			ErrorHandler: middleware.GlobalErrorHandler,
		}),
		AppConfig:         appConfig,
		UserWriterService: userWriterService,
		QueueService:      queueService, // 👈 Asignación
	}

	allowOrigins := strings.TrimSpace(os.Getenv("CORS_ALLOW_ORIGINS"))
	if allowOrigins == "" {
		allowOrigins = "http://localhost:9050,http://127.0.0.1:9050,http://localhost:9000,http://127.0.0.1:9000"
	}

	// Middleware CORS
	server.App.Use(cors.New(cors.Config{
		AllowOrigins:     allowOrigins,
		AllowCredentials: true,
		AllowMethods:     "GET, POST, PUT, DELETE, OPTIONS",
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization, X-Client-Code, X-Origin, X-Request-Id, X-Request-ID",
	}))

	// Rate Limiting
	rateLimitConfig := middleware.RateLimitConfig{
		Limit:  100,
		Window: 1 * time.Minute,
	}
	server.App.Use(middleware.RateLimitMiddleware(connect.ConnectRedis, rateLimitConfig))

	// Registrar rutas
	server.RegisterRoutes(authHandler, userHandler, bankHandler, catalogHandler, rolHandler, menuHandler, menuUserHandler, dbHandler, processLifecycleHandler, tokenService, connect.ConnectRedis)

	// Cleanup combinado (Wire lo mezcla con cleanup global)
	cleanup := func() {}

	return server, cleanup, nil
}
