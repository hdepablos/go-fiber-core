package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"go-fiber-core/cmd/api/di"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	_ "github.com/joho/godotenv/autoload"
)

var (
	appContainer *di.AppContainer
)

func init() {
	// Inicialización optimizada para Lambda (Warm Start)
	// Se ejecuta una sola vez cuando el contenedor se levanta
	res, _, err := di.InitializeAppContainer("config.yml")
	if err != nil {
		slog.Error("Fallo crítico inicializando dependencias (DLQ Consumer)", "error", err)
		os.Exit(1)
	}
	appContainer = res
	slog.Info("🚀 DLQ Consumer: Infraestructura lista")
}

// Handler procesa mensajes de la DLQ usando SQSEventResponse para manejo parcial de fallos
func Handler(ctx context.Context, event events.SQSEvent) (events.SQSEventResponse, error) {
	batchItemFailures := []events.SQSBatchItemFailure{}

	slog.Info("🌍 Evento DLQ recibido", "num_records", len(event.Records))

	for _, record := range event.Records {
		if err := processMessage(ctx, record); err != nil {
			slog.Error("❌ Error procesando mensaje de DLQ", "msgID", record.MessageId, "error", err)
			// Marcamos el mensaje para reintento (aunque en DLQ a veces se prefiere borrar y alertar)
			// Dependiendo de la estrategia, podrías NO devolver el fallo para "drenar" la DLQ
			// Pero por seguridad, mantenemos el patrón de no perder datos.
			batchItemFailures = append(batchItemFailures, events.SQSBatchItemFailure{
				ItemIdentifier: record.MessageId,
			})
		}
	}

	return events.SQSEventResponse{BatchItemFailures: batchItemFailures}, nil
}

func processMessage(ctx context.Context, record events.SQSMessage) error {
	// Aquí normalmente implementarías lógica de:
	// 1. Almacenar el mensaje fallido en una DB para auditoría
	// 2. Enviar una alerta a Slack/Email
	// 3. Intentar un reprocesamiento manual si aplica

	slog.Info("📥 Analizando mensaje de DLQ", "msgID", record.MessageId, "body_preview", record.Body[:min(len(record.Body), 50)])

	// Simulación de error si el cuerpo es "fail"
	if record.Body == "fail" {
		return fmt.Errorf("error simulado persistente en DLQ")
	}

	// Lógica de "Limpieza" o "Rescate" exitosa
	slog.Info("✔️ Mensaje de DLQ procesado/auditado correctamente", "msgID", record.MessageId)
	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func main() {
	lambda.Start(Handler)
}
