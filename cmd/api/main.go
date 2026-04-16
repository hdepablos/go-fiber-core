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

	// Registro de servicios (Process Lifecycle)
	//
	// El Process Lifecycle executor resuelve cada step por su `execution_key` (string) y busca una
	// factory registrada en `serviceconfig` con esa misma key.
	//
	// En Go, si un paquete no se importa, su `init()` no corre y por lo tanto no se registra el servicio.
	// Por eso estos imports en blanco (`_`) son obligatorios para que el registry tenga las keys.
	//
	// Si en base de datos hay un step con `execution_key = "validate_input"` pero NO existe un
	// `serviceconfig.Register("validate_input", ...)` incluido por alguno de estos paquetes,
	// el runtime fallará con:
	//   "servicio no encontrado en el registro: validate_input"
	//
	// Paquetes de servicios incluidos para pruebas/demo de Process Lifecycle:
	// - internal/services/test/common: common/validate, common/calculate, common/notify, batch/processor, batch/consolidate
	// - internal/services/test/heavy: heavy/process
	// - internal/services/test/loanrisk: loanrisk/age, loanrisk/salary, loanrisk/validation, loanrisk/is_renovation, loanrisk/risk_level
	// - internal/services/test: test/auto_invoke (y otros helpers de prueba)
	_ "go-fiber-core/internal/services/test/loanrisk"
	_ "go-fiber-core/internal/services/test/mqb1t"

	// Servicios de prueba concurrente (no pertenecen al seed principal de Process Lifecycle)
	_ "go-fiber-core/internal/services/test/imputation"
	_ "go-fiber-core/internal/services/test/steps_concurrent"

	// Servicios de prueba de auto-invoke (loop/batch)
	_ "go-fiber-core/internal/services/test"

	// Servicios de demo para escenarios seed de Process Lifecycle
	_ "go-fiber-core/internal/services/test/bulkexportV2"
	_ "go-fiber-core/internal/services/test/bulkexportv1"
	_ "go-fiber-core/internal/services/test/common"
	_ "go-fiber-core/internal/services/test/heavy"

	_ "go-fiber-core/internal/services/bulkprocess"
	_ "go-fiber-core/internal/services/generar_archivo_banco_galicia"

	_ "go-fiber-core/internal/services/punitorios")

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
