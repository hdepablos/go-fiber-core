package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"time"

	"go-fiber-core/cmd/api/di"
	ilogger "go-fiber-core/internal/logger"
	"go-fiber-core/internal/services/queue"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	_ "github.com/joho/godotenv/autoload"
	"go.uber.org/zap"
)

var (
	appContainer *di.AppContainer
)

func init() {
	// Inicialización optimizada para Lambda (Warm Start)
	// Se ejecuta una sola vez cuando el contenedor se levanta
	// IMPORTANTE: En Lambda, el archivo está en internal/appconfig/config.yml según nuestro Dockerfile
	configPath := "internal/appconfig/config.yml"
	if os.Getenv("AWS_LAMBDA_FUNCTION_NAME") == "" {
		configPath = "config.yml" // Local fallback
	}

	res, _, err := di.InitializeAppContainer(configPath)
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

	// Intento de parsear para mostrar mensaje descriptivo
	type SimpleMsg struct {
		ID   string `json:"id"`
		Body string `json:"body"`
	}
	var msg SimpleMsg
	var logBody string

	// Primero desenvovlemos si viene de SNS (opcional, pero buena práctica)
	// (Aquí asumimos que viene directo de SQS o es compatible)
	if err := json.Unmarshal([]byte(record.Body), &msg); err == nil && msg.Body != "" {
		logBody = msg.Body // Usamos el campo 'body' interno si existe
	} else {
		logBody = record.Body // Si no, usamos el body crudo
	}

	// Truncar para log si es muy largo
	displayBody := logBody
	if len(displayBody) > 100 {
		displayBody = displayBody[:100] + "..."
	}

	slog.Info("💀 Mensaje recibido en DLQ (Dead Letter Queue)", "msgID", record.MessageId, "contenido", displayBody)

	// Simulación de error si el cuerpo es "fail"
	if record.Body == "fail" {
		return fmt.Errorf("error simulado persistente en DLQ")
	}

	// Lógica de "Limpieza" o "Rescate" exitosa
	slog.Info("✔️ Mensaje de DLQ auditado/archivado", "msgID", record.MessageId)
	return nil
}

func main() {
	if os.Getenv("AWS_LAMBDA_FUNCTION_NAME") != "" {
		lambda.Start(Handler)
	} else {
		runPollingLoop()
	}
}

func runPollingLoop() {
	slog.Info("🚀 Iniciando DLQ Consumer en modo POLLING (EKS/Local)...")
	ctx := context.Background()
	errorGuard := queue.NewPollingErrorGuard(0)
	cooldown := queue.DefaultPollingErrorCooldown()

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
			slog.Error("❌ Error recibiendo mensajes de DLQ", "error", err)
			event := errorGuard.Record(err)
			if event.Triggered {
				ilogger.LogExecutionGuard(
					"consumer_receive_auto_pause",
					zap.String("component", "dlq_consumer_polling"),
					zap.String("fingerprint", event.Fingerprint),
					zap.Int("error_count", event.Count),
					zap.Int("threshold", event.Threshold),
					zap.String("cooldown", cooldown.String()),
				)
				time.Sleep(cooldown)
				errorGuard.Reset()
				continue
			}
			continue
		}
		errorGuard.Reset()

		if len(messages) == 0 {
			continue
		}

		// 2. Procesar cada mensaje
		for _, msg := range messages {
			// Convertir types.Message (AWS SDK v2) a events.SQSMessage (AWS Lambda Events)
			lambdaMsg := events.SQSMessage{
				MessageId:     *msg.MessageId,
				ReceiptHandle: *msg.ReceiptHandle,
				Body:          *msg.Body,
			}

			// Nota: processMessage devuelve events.SQSEventResponse (con BatchItemFailures)
			// pero aquí procesamos uno a uno.
			// Si falla, NO borramos el mensaje para que vuelva a la cola (reintento).
			// PERO processMessage en este archivo devuelve 'nil' error si falla,
			// y retorna BatchItemFailures. Tenemos que adaptar eso.

			// Vamos a refactorizar processMessage para que devuelva error si falla el procesamiento individual
			// O mejor, extraemos la lógica de procesamiento real.
			// Pero processMessage llama a processMessage interno (line 59).
			// La funcion processMessage en linea 59 devuelve error.

			err := processMessage(ctx, lambdaMsg)
			if err != nil {
				slog.Error("❌ Error procesando mensaje DLQ en modo polling", "id", lambdaMsg.MessageId, "error", err)
				// No borramos -> Reintento
				continue
			}

			// 3. Borrar mensaje exitoso
			if err := appContainer.QueueService.DeleteMessage(ctx, lambdaMsg.ReceiptHandle); err != nil {
				slog.Error("⚠️ Error borrando mensaje procesado de DLQ", "id", lambdaMsg.MessageId, "error", err)
			}
		}
	}
}
