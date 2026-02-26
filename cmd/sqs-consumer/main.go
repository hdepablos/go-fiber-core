package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"log/slog" // Uso de logger estructurado (Go 1.21+)
	"os"
	"runtime"

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
	// IMPORTANTE: En Lambda, el archivo está en internal/appconfig/config.yml según nuestro Dockerfile
	configPath := "internal/appconfig/config.yml"
	if os.Getenv("AWS_LAMBDA_FUNCTION_NAME") == "" {
		configPath = "config.yml" // Local fallback
	}

	res, _, err := di.InitializeAppContainer(configPath)
	if err != nil {
		slog.Error("Fallo crítico en DI", "error", err)
		os.Exit(1)
	}
	appContainer = res
}

func main() {
	if os.Getenv("AWS_LAMBDA_FUNCTION_NAME") != "" {
		// Modo Lambda: AWS invoca el Handler
		lambda.Start(Handler)
	} else {
		// Modo Polling (EKS/Local): Nosotros invocamos a SQS
		runPollingLoop()
	}
}

func runPollingLoop() {
	slog.Info("🚀 Iniciando SQS Consumer en modo POLLING (EKS/Local)...")
	ctx := context.Background()
	
	// Validar que tengamos QueueService
	if appContainer.QueueService == nil {
		slog.Error("❌ QueueService no inicializado")
		os.Exit(1)
	}

	for {
		// 1. Recibir mensajes (Long Polling de 20s ya configurado en sqs_service.go)
		// Pedimos máximo 10 mensajes por lote
		messages, err := appContainer.QueueService.ReceiveMessages(ctx, 10)
		if err != nil {
			slog.Error("❌ Error recibiendo mensajes de SQS", "error", err)
			// Backoff simple para no saturar logs si SQS está caído
			continue
		}

		if len(messages) == 0 {
			continue
		}

		// 2. Procesar cada mensaje
		for _, msg := range messages {
			// Convertir types.Message (AWS SDK v2) a events.SQSMessage (AWS Lambda Events)
			// Son estructuras diferentes pero compatibles en datos clave
			lambdaMsg := events.SQSMessage{
				MessageId:     *msg.MessageId,
				ReceiptHandle: *msg.ReceiptHandle,
				Body:          *msg.Body,
				// Nota: No copiamos atributos aquí por simplicidad, pero se podría si fuera necesario
			}

			if err := processMessage(ctx, lambdaMsg); err != nil {
				slog.Error("❌ Error procesando mensaje en modo polling", "id", lambdaMsg.MessageId, "error", err)
				// En modo polling, NO borramos el mensaje si falla.
				// SQS lo volverá a hacer visible después del VisibilityTimeout.
				continue
			}

			// 3. Borrar mensaje exitoso
			if err := appContainer.QueueService.DeleteMessage(ctx, lambdaMsg.ReceiptHandle); err != nil {
				slog.Error("⚠️ Error borrando mensaje procesado", "id", lambdaMsg.MessageId, "error", err)
			} else {
				// Log opcional de borrado exitoso
				// slog.Debug("✅ Mensaje borrado", "id", lambdaMsg.MessageId)
			}
		}
	}
}

// Handler usa SQSEventResponse para evitar re-procesar mensajes exitosos
func Handler(ctx context.Context, event events.SQSEvent) (events.SQSEventResponse, error) {
	// --- LOGS DE RENDIMIENTO ---
	numCPU := runtime.NumCPU()
	numGoroutines := runtime.NumGoroutine()
	log.Printf("🚀 --- LOGS DE RENDIMIENTO ---\n")
	log.Printf("💻 CPUs disponibles: %d\n", numCPU)
	log.Printf("🔄 Goroutines iniciales: %d\n", numGoroutines)
	log.Printf("🏗️ Arquitectura: %s\n", runtime.GOARCH)
	// ---------------------------

	batchItemFailures := []events.SQSBatchItemFailure{}

	for _, record := range event.Records {
		if err := processMessage(ctx, record); err != nil {
			slog.Error("❌ Error procesando mensaje", "id", record.MessageId, "error", err)
			// Retornamos el error globalmente para que Lambda marque todo el lote como fallido.
			// Esto fuerza el reintento gestionado por la política de SQS (VisibilityTimeout + maxReceiveCount).
			// Si usamos BatchItemFailures, SQS borra los exitosos y reencola los fallidos,
			// pero para asegurar el comportamiento clásico de reintento visible en logs,
			// devolver el error es más directo en pruebas unitarias/simples.
			// Sin embargo, para producción con batch > 1, BatchItemFailures es mejor.
			// En este caso, LocalStack a veces requiere el error explícito para incrementar el ReceiveCount rápidamente.

			// OPCIÓN HÍBRIDA: Reportar fallo parcial PERO asegurarnos que SQS lo vea.
			batchItemFailures = append(batchItemFailures, events.SQSBatchItemFailure{
				ItemIdentifier: record.MessageId,
			})
		}
	}

	// Si hay fallos, el retorno de SQSEventResponse le dice a Lambda "estos mensajes fallaron, no los borres".
	// SQS incrementará el ReceiveCount y volverá a entregarlos después del VisibilityTimeout.
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
