package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"log/slog" // Uso de logger estructurado (Go 1.21+)
	"os"
	"runtime"
	"time"

	"go-fiber-core/cmd/api/di"
	"go-fiber-core/internal/dtos/requests"
	"go-fiber-core/internal/services/queue"
	"go-fiber-core/internal/services/serviceconfig/contracts"

	// Import services to register them
	_ "go-fiber-core/internal/services/test"

	"github.com/google/uuid"

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
	// if os.Getenv("AWS_LAMBDA_FUNCTION_NAME") == "" {
	// 	configPath = "config.yml" // Local fallback
	// }

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
		// slog.Info("⏳ Polling for messages...") // Comentado para no saturar
		messages, err := appContainer.QueueService.ReceiveMessages(ctx, 10)
		if err != nil {
			slog.Error("❌ Error recibiendo mensajes de SQS", "error", err)
			// Backoff simple para no saturar logs si SQS está caído
			time.Sleep(5 * time.Second)
			continue
		}

		if len(messages) == 0 {
			continue
		}

		slog.Info("📨 Received messages", "count", len(messages))

		// 2. Procesar cada mensaje
		for _, msg := range messages {
			slog.Info("🔄 Processing message", "message_id", *msg.MessageId)

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
	var req requests.RunProcessRequest

	// 1. Intentar unmarshal directo
	if err := json.Unmarshal([]byte(rawData), &req); err != nil || req.ProcessTypeID == 0 {
		// 2. Si falla o está vacío, intentar como queue.Message
		var msg queue.Message
		if err2 := json.Unmarshal([]byte(rawData), &msg); err2 == nil && msg.ID != "" {
			// Es un queue.Message. Intentamos sacar el request del Body
			if err3 := json.Unmarshal([]byte(msg.Body), &req); err3 != nil {
				slog.Error("Failed to unmarshal RunProcessRequest from queue.Message body", "error", err3)
				return nil
			}
		} else {
			// Try one more time assuming it might be wrapped in "Message" field from SNS/SQS raw body
			var snsMsg struct {
				Message string `json:"Message"`
			}
			if err4 := json.Unmarshal([]byte(rawData), &snsMsg); err4 == nil && snsMsg.Message != "" {
				if err5 := json.Unmarshal([]byte(snsMsg.Message), &req); err5 != nil {
					slog.Error("Error unmarshalling RunProcessRequest from SNS Message", "error", err5)
					return nil
				}
			} else {
				slog.Error("Error unmarshalling RunProcessRequest", "error", err)
				return nil
			}
		}
	}

	slog.Info("🔄 Executing Process Version", "process_version_id", req.OverrideProcessVersionID, "process_type_id", req.ProcessTypeID, "input", req.Input)

	if req.Input == nil {
		req.Input = make(map[string]any)
	}

	// 🚨 Check for forced error (DLQ testing)
	if val, ok := req.Input["force_error"]; ok {
		if forceError, isBool := val.(bool); isBool && forceError {
			slog.Error("🚨 FORCED ERROR TRIGGERED (DLQ TEST)")
			return fmt.Errorf("🚨 Error forzado para prueba de DLQ")
		}
	}

	// Validate autoInvoke keys if present in Input
	if val, ok := req.Input["autoInvoke"]; ok && val == true {
		_, hasLastID := req.Input["last_id_processed"]
		_, hasLastBatch := req.Input["is_last_batch"]

		if !hasLastID || !hasLastBatch {
			slog.Error("❌ autoInvoke=true pero faltan keys requeridas", "input", req.Input)
			return fmt.Errorf("missing required keys for autoInvoke")
		}
		slog.Info("🚀 [Auto-Invoke] Starting batch processing", "last_id_processed", req.Input["last_id_processed"])
	}

	_, svcCtx, err := appContainer.ProcessLifecycleService.Run(ctx, req)
	if err != nil {
		slog.Error("❌ Error executing process", "error", err)
		return err
	}

	// Check for autoInvoke recursion
	autoInvokeVal, ok := svcCtx.Input["autoInvoke"]
	if ok && autoInvokeVal == true {
		slog.Info("🔄 AutoInvoke detected")

		var newLastID any
		var isLastBatch bool
		var found bool

		// Helper function to extract values from map
		extractFromMap := func(m map[string]any) {
			if val, ok := m["last_id_processed"]; ok {
				newLastID = val
				if batchVal, ok := m["is_last_batch"]; ok {
					if b, ok := batchVal.(bool); ok {
						isLastBatch = b
						found = true
					}
				}
			}
		}

		// Check Results
		for _, res := range svcCtx.Results {
			if stepRes, ok := res.(contracts.StepResult); ok {
				// Check stepRes.Data
				if stepRes.Data != nil {
					extractFromMap(stepRes.Data)
					if found {
						break
					}
				}
			} else if resMap, ok := res.(map[string]any); ok {
				// Check inside "data" key first (common pattern)
				if data, ok := resMap["data"]; ok {
					if dataMap, ok := data.(map[string]any); ok {
						extractFromMap(dataMap)
						if found {
							break
						}
					}
				}
				// Check directly in result map
				extractFromMap(resMap)
				if found {
					break
				}
			}
		}

		if found {
			if !isLastBatch {
				slog.Info("🔄 [Auto-Invoke] Batch completed. Re-queuing for next batch...", "last_id", newLastID)

				// Update input for next iteration
				req.Input["last_id_processed"] = newLastID
				req.Input["is_last_batch"] = false

				// Serialize request
				reqBytes, err := json.Marshal(req)
				if err != nil {
					slog.Error("❌ Error marshaling re-queue request", "error", err)
					return err
				}

				// Create new message
				newMessage := &queue.Message{
					ID:      uuid.New().String(),
					Body:    string(reqBytes),
					Source:  "auto-invoke",
					Created: time.Now().Format(time.RFC3339),
				}

				// Send to queue
				if err := appContainer.QueueService.SendMessage(ctx, newMessage); err != nil {
					slog.Error("❌ Failed to re-queue auto-invoke message", "error", err)
					return err
				}
				slog.Info("✅ Message re-queued successfully")

			} else {
				slog.Info("✅ [Auto-Invoke] All batches finished (is_last_batch=true)", "last_id", newLastID)
			}
		} else {
			slog.Warn("⚠️ [Auto-Invoke] Could not find last_id_processed/is_last_batch in results")
		}
	}

	return nil
}
