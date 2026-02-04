package main

import (
	"context"
	"log/slog"
	"os"

	"go-fiber-core/cmd/api/di"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

var (
	container *di.AppContainer
)

func init() {
	// Inicialización Warm Start para Cron
	res, _, err := di.InitializeAppContainer("config.yml")
	if err != nil {
		slog.Error("💀 Fallo crítico inicializando Cron Daily", "error", err)
		os.Exit(1)
	}
	container = res
	slog.Info("🚀 Cron Daily: Infraestructura lista")
}

func handleRequest(ctx context.Context, event events.CloudWatchEvent) error {
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
	lambda.Start(handleRequest)
}
