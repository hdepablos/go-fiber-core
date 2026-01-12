// cmd/api/main.go
package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go-fiber-core/cmd/api/di"
	"go-fiber-core/internal/dtos/config"
	"go-fiber-core/internal/dtos/connect" // Importado el server struct
	"go-fiber-core/internal/services/product"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	fiberadapter "github.com/awslabs/aws-lambda-go-api-proxy/fiber"
	"github.com/gofiber/fiber/v2"

	_ "github.com/joho/godotenv/autoload"
)

// Variables globales para el estado y el adaptador
var (
	fiberApp    *fiber.App                // La aplicación Fiber extraída
	appConfig   *config.AppConfig         // Configuración extraída
	connectDTO  *connect.ConnectDTO       // Conexiones DTO (Nota: no se extrae directamente de FiberServer, pero se puede añadir si lo necesitas)
	appCleanup  func()                    // La función de limpieza retornada por Wire
	fiberLambda *fiberadapter.FiberLambda // El adaptador para Lambda
	appEnv      string                    // Entorno de ejecución (local, lambda)
	configPath  string                    // Ruta al archivo de configuración
)

// isLambdaEnvironment detecta si estamos en modo Lambda
func isLambdaEnvironment() bool {
	// 1. Detección por variable de entorno APP_ENV (primaria)
	if appEnv == "local" {
		return true
	}
	// 2. Detección por variables de entorno de AWS Lambda (secundaria)
	return os.Getenv("AWS_LAMBDA_FUNCTION_NAME") != "" ||
		os.Getenv("_LAMBDA_SERVER_PORT") != ""
}

// initializeApp se encarga de la Inyección de Dependencias (Wire) y la configuración inicial.
func initializeApp() func() {
	// La DI se ejecuta una sola vez.
	if fiberApp != nil {
		return appCleanup
	}

	// --- INICIALIZACIÓN CON WIRE ---
	// CORRECCIÓN: Ahora InitializeServer retorna *server.FiberServer, func(), error
	serverInstance, cleanup, err := di.InitializeServer(configPath)
	if err != nil {
		log.Fatalf("💀 Failed to initialize server: %v", err)
	}

	// Extraemos los componentes del FiberServer
	fiberApp = serverInstance.App
	appConfig = serverInstance.AppConfig
	// El connectDTO no se guarda en FiberServer, pero el AppConfig sí.
	appCleanup = cleanup

	// Ejemplo de uso de un servicio para prueba (usando la AppConfig extraída)
	testService := product.NewProductAPIService(appConfig)
	testService.PrintRedisConfig(context.Background())

	// Si estamos en Lambda, inicializamos el adaptador Lambda
	if isLambdaEnvironment() {
		fiberLambda = fiberadapter.New(fiberApp)
		log.Println("✅ Adapter Lambda configurado")
	}

	log.Println("🚀 Dependencies initialized successfully!")
	return appCleanup
}

// Handler principal para Lambda (solo se usa en modo Lambda)
func Handler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	if fiberLambda == nil {
		log.Println("⚠️  fiberLambda no inicializado. Llamando a initializeApp()...")
		initializeApp()
		if fiberLambda == nil {
			return events.APIGatewayProxyResponse{StatusCode: 500}, errors.New("fiber Lambda adapter not initialized")
		}
	}

	return fiberLambda.ProxyWithContext(ctx, req)
}

func main() {
	// Carga de configuración desde el flag
	flagPath := flag.String("config", "internal/appconfig/config.yml", "Ruta al archivo de configuración YAML")
	flag.Parse()
	configPath = *flagPath // Guardar la ruta para usarla en initializeApp

	// 1. Determinar el entorno (de la variable de entorno)
	appEnv = os.Getenv("APP_ENV")
	if appEnv == "" {
		appEnv = "local" // Valor por defecto
	}
	log.Printf("⚙️  Modo de ejecución detectado: %s", appEnv)

	// 2. Inicializar la aplicación con Wire (Crea FiberServer, DI, Conexiones, Rutas)
	cleanup := initializeApp()
	defer cleanup()

	// 3. Ejecutar la lógica de arranque según el entorno
	if isLambdaEnvironment() {
		// --- ARRANQUE EN MODO LAMBDA ---
		var BuildMarker = "lambda0api"
		_ = BuildMarker

		log.Println("🔥 Iniciando en modo AWS lambda0api")
		lambda.Start(Handler)
	} else {
		// --- ARRANQUE EN MODO HTTP TRADICIONAL (Local/Server) ---

		// Obtener puerto de la variable de entorno o un valor por defecto
		port := os.Getenv("PORT")
		if port == "" {
			// Usamos el puerto 9009
			port = "9009"
		}

		// --- GESTIÓN DE CIERRE ORDENADO ---
		done := make(chan bool, 1)

		// Goroutine que escucha señales de cierre (SIGINT, SIGTERM)
		go func() {
			ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
			defer stop()

			<-ctx.Done()

			log.Println("⬇️ Shutting down gracefully, press Ctrl+C again to force")

			shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			// fiberApp (extraído del FiberServer) es la instancia a apagar
			if err := fiberApp.ShutdownWithContext(shutdownCtx); err != nil {
				if !errors.Is(err, http.ErrServerClosed) {
					log.Printf("❌ Server forced to shutdown with error: %v", err)
				}
			}
			log.Println("✅ HTTP server stopped.")
			done <- true
		}()

		// --- ARRANQUE DEL SERVIDOR HTTP ---
		log.Printf("🚀 Starting server on port :%s", port)
		if err := fiberApp.Listen(":" + port); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("❌ HTTP server error: %s", err)
		}

		<-done // Esperar a que el cierre ordenado termine
		log.Println("👋 Graceful shutdown complete. Exiting.")
	}
}
