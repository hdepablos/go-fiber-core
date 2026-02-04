package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog" // Uso de logger estructurado (Go 1.21+)
	"os"

	"go-fiber-core/cmd/api/di"
	"go-fiber-core/internal/services/queue"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	_ "github.com/joho/godotenv/autoload"
)

var (
	appContainer *di.AppContainer
)

func init() {
	// Inicializamos una sola vez al levantar el microcontenedor
	res, _, err := di.InitializeAppContainer("config.yml")
	if err != nil {
		slog.Error("Fallo crítico en DI", "error", err)
		os.Exit(1)
	}
	appContainer = res
}

// Handler usa SQSEventResponse para evitar re-procesar mensajes exitosos
func Handler(ctx context.Context, event events.SQSEvent) (events.SQSEventResponse, error) {
	batchItemFailures := []events.SQSBatchItemFailure{}

	for _, record := range event.Records {
		if err := processMessage(ctx, record); err != nil {
			slog.Error("Error procesando mensaje", "id", record.MessageId, "error", err)
			// Agregamos el ID fallido para que SQS solo reintente este
			batchItemFailures = append(batchItemFailures, events.SQSBatchItemFailure{
				ItemIdentifier: record.MessageId,
			})
		}
	}

	return events.SQSEventResponse{BatchItemFailures: batchItemFailures}, nil
}

func processMessage(ctx context.Context, record events.SQSMessage) error {
	// 1. Unmarshal inicial para detectar origen
	var wrapper struct {
		Type    string `json:"Type"`
		Message string `json:"Message"`
	}

	bodyBytes := []byte(record.Body)
	_ = json.Unmarshal(bodyBytes, &wrapper)

	// 2. Lógica de Unwrapping de SNS
	var finalPayload string
	if wrapper.Type == "Notification" {
		finalPayload = wrapper.Message
	} else {
		finalPayload = record.Body
	}

	// 3. Lógica de negocio
	return handleBusinessLogic(ctx, finalPayload)
}

func handleBusinessLogic(ctx context.Context, rawData string) error {
	var msg queue.Message
	if err := json.Unmarshal([]byte(rawData), &msg); err != nil {
		// Error de formato: No reintentar (mensaje venenoso)
		slog.Warn("Mensaje malformado omitido", "body", rawData)
		return nil
	}

	if msg.ID == "999" {
		return fmt.Errorf("simulated failure for DLQ: %s", msg.ID)
	}

	slog.Info("Mensaje procesado con éxito", "msgID", msg.ID)
	return nil
}

func main() {
	lambda.Start(Handler)
}
