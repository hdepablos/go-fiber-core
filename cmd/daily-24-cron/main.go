package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"runtime"

	"go-fiber-core/cmd/api/di"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

var (
	container *di.AppContainer
)

func init() {
	// Inicialización Warm Start para Cron
	// IMPORTANTE: En Lambda, el archivo está en internal/appconfig/config.yml según nuestro Dockerfile
	configPath := "internal/appconfig/config.yml"
	if os.Getenv("AWS_LAMBDA_FUNCTION_NAME") == "" {
		configPath = "config.yml" // Local fallback
	}

	res, _, err := di.InitializeAppContainer(configPath)
	if err != nil {
		slog.Error("💀 Fallo crítico inicializando Cron Daily", "error", err)
		os.Exit(1)
	}
	container = res
	slog.Info("🚀 Cron Daily: Infraestructura lista")
}

func handleRequest(ctx context.Context, event events.CloudWatchEvent) error {
	// --- LOGS DE RENDIMIENTO ---
	numCPU := runtime.NumCPU()
	numGoroutines := runtime.NumGoroutine()
	log.Printf("🚀 --- LOGS DE RENDIMIENTO ---\n")
	log.Printf("💻 CPUs disponibles: %d\n", numCPU)
	log.Printf("🔄 Goroutines iniciales: %d\n", numGoroutines)
	log.Printf("🏗️ Arquitectura: %s\n", runtime.GOARCH)
	// ---------------------------

	slog.Info("🗓️ Ejecutando Cron Diario (Daily)",
		"env", container.Config.App.AppEnv,
		"event_id", event.ID,
		"time", event.Time,
	)

	// --- LÓGICA DE NEGOCIO ---
	// Ejemplo: Reportes, Limpieza, Batch Jobs pesados
	// reportService := container.ReportService
	// err := reportService.GenerateDailyReport(ctx)

	slog.Info("✅ Tarea diaria completada exitosamente")
	return nil
}

func main() {
	if os.Getenv("AWS_LAMBDA_FUNCTION_NAME") != "" {
		lambda.Start(handleRequest)
	} else {
		// Modo CLI / K8s CronJob
		err := handleRequest(context.Background(), events.CloudWatchEvent{
			ID:     "cli-invocation",
			Source: "k8s-cronjob",
		})
		if err != nil {
			slog.Error("❌ Error executing cron", "error", err)
			os.Exit(1)
		}
	}
}
