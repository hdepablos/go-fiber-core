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
		slog.Error("💀 Fallo crítico inicializando Cron 1min", "error", err)
		os.Exit(1)
	}
	container = res
	slog.Info("🚀 Cron 1min: Infraestructura lista")
}

func handleRequest(ctx context.Context, event events.CloudWatchEvent) error {
	slog.Info("⏰ Ejecutando Cron (Cada 1 Minuto)",
		"env", container.Config.App.AppEnv,
		"event_id", event.ID,
		"source", event.Source,
	)

	// Lógica del Cron
	// Ejemplo: Enviar heartbeat a SQS o limpiar caché
	// msg := queue.Message{ID: uuid.New(), Type: "heartbeat"}
	// err := container.QueueService.SendMessage(ctx, &msg)

	slog.Info("✅ Tarea programada completada exitosamente")
	return nil
}

func main() {
	lambda.Start(handleRequest)
}
