package main

import (
	"context"
	"log"
	"os"
	"runtime"
	"time"

	"go-fiber-core/cmd/api/di"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	fiberadapter "github.com/awslabs/aws-lambda-go-api-proxy/fiber"

	// Importación en blanco para asegurar el registro de servicios Loan Risk
	_ "go-fiber-core/internal/services/loanrisk"
	// Importación en blanco para registrar servicios de prueba concurrente
	_ "go-fiber-core/internal/services/test/steps_concurrent"
	_ "go-fiber-core/internal/services/test/imputation"
)

var fiberLambda *fiberadapter.FiberLambda

// initializeLambdaApp prepares the Fiber app for Lambda execution
func initializeLambdaApp() {
	log.Println("🚀 Initializing Fiber App for Lambda...")
	// In Lambda, we use the config file copied to the internal directory
	server, _, err := di.InitializeServer("internal/appconfig/config.yml")
	if err != nil {
		log.Printf("❌ Error initializing server: %v", err)
		return
	}

	fiberLambda = fiberadapter.New(server.App)
	log.Println("✅ Fiber Lambda Adapter initialized")
}

// Handler is the Lambda entry point
func Handler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// --- LOGS DE RENDIMIENTO ---
	numCPU := runtime.NumCPU()
	numGoroutines := runtime.NumGoroutine()
	log.Printf("🚀 --- LOGS DE RENDIMIENTO ---\n")
	log.Printf("💻 CPUs disponibles: %d\n", numCPU)
	log.Printf("🔄 Goroutines iniciales: %d\n", numGoroutines)
	log.Printf("🏗️ Arquitectura: %s\n", runtime.GOARCH)
	// ---------------------------

	if fiberLambda == nil {
		initializeLambdaApp()
	}

	// Timeout logic
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return fiberLambda.ProxyWithContext(ctx, req)
}

func main() {
	// Detect environment: AWS_LAMBDA_FUNCTION_NAME is only present in Lambda
	if os.Getenv("AWS_LAMBDA_FUNCTION_NAME") != "" {
		// Run as Lambda
		lambda.Start(Handler)
	} else {
		// Detect EKS/K8s Environment
		if os.Getenv("KUBERNETES_SERVICE_HOST") != "" {
			log.Println("🚀 Starting in EKS/K8s Cluster mode...")
		} else {
			// Run as Local Server (Air)
			log.Println("🚀 Starting in LOCAL mode...")
		}

		// Use the same config path as local development usually runs from root
		server, cleanup, err := di.InitializeServer("internal/appconfig/config.yml")
		if err != nil {
			log.Printf("❌ Error initializing server: %v", err)
			return
		}
		defer cleanup()

		port := server.AppConfig.Server.ServerPort
		if port == "" {
			port = "3000" // Fallback
		}

		log.Printf("✅ Server listening on port %s", port)
		if err := server.Listen(":" + port); err != nil {
			log.Printf("❌ Error starting server: %v", err)
		}
	}
}
