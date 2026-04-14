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
	"strconv"
	"strings"
	"time"

	fiber "github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
)

// FiberServer representa el servidor principal con todas sus dependencias.
type FiberServer struct {
	*fiber.App
	AppConfig         *config.AppConfig
	Connect           *connect.ConnectDTO
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
	importHandler handlers.ImportHandler,
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
		Connect:           connect,
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
	globalLimit := int64(100)
	if v := strings.TrimSpace(os.Getenv("RATE_LIMIT_GLOBAL_PER_MINUTE")); v != "" {
		if parsed, err := strconv.ParseInt(v, 10, 64); err == nil && parsed > 0 {
			globalLimit = parsed
		}
	}
	importsLimit := int64(5000)
	if v := strings.TrimSpace(os.Getenv("RATE_LIMIT_IMPORTS_PER_MINUTE")); v != "" {
		if parsed, err := strconv.ParseInt(v, 10, 64); err == nil && parsed > 0 {
			importsLimit = parsed
		}
	}

	server.App.Use(middleware.RateLimitByPathMiddleware(connect.ConnectRedis, middleware.RateLimitByPathConfig{
		Global: middleware.RateLimitConfig{
			Limit:  globalLimit,
			Window: 1 * time.Minute,
		},
		Imports: middleware.RateLimitConfig{
			Limit:  importsLimit,
			Window: 1 * time.Minute,
		},
		ImportsPath: "/api/v1/imports/",
	}))

	// Registrar rutas
	server.RegisterRoutes(authHandler, userHandler, bankHandler, catalogHandler, rolHandler, menuHandler, menuUserHandler, dbHandler, processLifecycleHandler, importHandler, tokenService, connect.ConnectRedis)

	// Cleanup combinado (Wire lo mezcla con cleanup global)
	cleanup := func() {}

	return server, cleanup, nil
}
