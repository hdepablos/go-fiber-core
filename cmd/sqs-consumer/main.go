package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"log/slog" // Uso de logger estructurado (Go 1.21+
	"os"
	"runtime"
	"strings"
	"time"

	"go-fiber-core/cmd/api/di"
	"go-fiber-core/internal/dtos/requests"
	ilogger "go-fiber-core/internal/logger"
	"go-fiber-core/internal/runtimebootstrap"
	"go-fiber-core/internal/services/batchflow"
	"go-fiber-core/internal/services/exportmanager"
	"go-fiber-core/internal/services/queue"
	"go-fiber-core/internal/services/serviceconfig"
	"go-fiber-core/internal/services/serviceconfig/contracts"

	// Import services to register them
	_ "go-fiber-core/internal/services/test"
	_ "go-fiber-core/internal/services/test/common"
	_ "go-fiber-core/internal/services/test/heavy"
	_ "go-fiber-core/internal/services/test/mqb1t"

	"github.com/google/uuid"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	_ "github.com/joho/godotenv/autoload"
	"go.uber.org/zap"

	_ "go-fiber-core/internal/services/bulkprocess"
	_ "go-fiber-core/internal/services/batchprocess/punitive"
)

var (
	appContainer *di.AppContainer
	runtimeDeps  *runtimebootstrap.Dependencies
	runControl   *batchflow.RunControl
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
	runtimeDeps, err = runtimebootstrap.Build(context.Background(), appContainer.Config, appContainer.Connect, appContainer.QueueService)
	if err != nil {
		slog.Warn("Runtime bootstrap parcial", "error", err)
	}
	if appContainer != nil && appContainer.Connect != nil && appContainer.Connect.ConnectRedis != nil {
		runControl = batchflow.NewRunControl(batchflow.NewRedisCache(appContainer.Connect.ConnectRedis), 24*time.Hour)
	}
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
		// slog.Info("⏳ Polling for messages...") // Comentado para no saturar
		messages, err := appContainer.QueueService.ReceiveMessages(ctx, 10)
		if err != nil {
			slog.Error("❌ Error recibiendo mensajes de SQS", "error", err)
			event := errorGuard.Record(err)
			if event.Triggered {
				ilogger.LogExecutionGuard(
					"consumer_receive_auto_pause",
					zap.String("component", "sqs_consumer_polling"),
					zap.String("fingerprint", event.Fingerprint),
					zap.Int("error_count", event.Count),
					zap.Int("threshold", event.Threshold),
					zap.String("cooldown", cooldown.String()),
				)
				time.Sleep(cooldown)
				errorGuard.Reset()
				continue
			}
			time.Sleep(5 * time.Second)
			continue
		}
		errorGuard.Reset()

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
	if runtimeDeps != nil {
		ctx = runtimeDeps.Inject(ctx)
	}
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
	// 1. Intentar procesar como RunProcessRequest (Proceso Completo)
	var req requests.RunProcessRequest
	var processErr error

	// Intentar unmarshal directo
	if err := json.Unmarshal([]byte(rawData), &req); err != nil || req.ProcessTypeID == 0 {
		processErr = err // Guardar error por si acaso

		// Intentar Unwrap de SQS/SNS si falló el directo
		var msg queue.Message
		if err2 := json.Unmarshal([]byte(rawData), &msg); err2 == nil && msg.ID != "" {
			_ = json.Unmarshal([]byte(msg.Body), &req)
		} else {
			var snsMsg struct {
				Message string `json:"Message"`
			}
			if err3 := json.Unmarshal([]byte(rawData), &snsMsg); err3 == nil && snsMsg.Message != "" {
				_ = json.Unmarshal([]byte(snsMsg.Message), &req)
			}
		}
	}

	// Si logramos decodificar un RunProcessRequest válido
	if req.ProcessTypeID != 0 {
		return executeRunProcessRequest(ctx, req)
	}

	// 2. Intentar procesar como DispatchStepRequest (Paso Individual - ASYNC)
	// Estructura usada en dispatcher.go:DispatchStep
	type DispatchStepRequest struct {
		ServicePath        string                    `json:"service_path"`
		ProcessExecutionID string                    `json:"process_execution_id"`
		Input              map[string]any            `json:"input"`
		StepOrder          int                       `json:"step_order"`
		ExecutionPolicy    contracts.ExecutionPolicy `json:"execution_policy,omitempty"`
		ServiceConfig      map[string]any            `json:"service_config,omitempty"`
	}

	var stepReq DispatchStepRequest
	var stepErr error

	// Intentar unmarshal directo para Step
	// IMPORTANTE: El mensaje puede venir envuelto en un objeto queue.Message (con id, body, source, created)
	// aunque no sea un evento SQS puro, porque así lo manda el dispatcher.
	// Primero intentamos parsear como queue.Message para sacar el body real.
	var wrapperMsg queue.Message
	if err := json.Unmarshal([]byte(rawData), &wrapperMsg); err == nil && wrapperMsg.Body != "" {
		// Es un wrapper, usamos el Body interno
		if err := json.Unmarshal([]byte(wrapperMsg.Body), &stepReq); err != nil {
			stepErr = err
		}
	} else {
		// No es wrapper, intentamos directo
		if err := json.Unmarshal([]byte(rawData), &stepReq); err != nil {
			stepErr = err
		}
	}

	// Si falló lo anterior, intentamos Unwrap de SNS por si acaso
	if stepReq.ServicePath == "" {
		var snsMsg struct {
			Message string `json:"Message"`
		}
		if err := json.Unmarshal([]byte(rawData), &snsMsg); err == nil && snsMsg.Message != "" {
			_ = json.Unmarshal([]byte(snsMsg.Message), &stepReq)
		}
	}

	// Si logramos decodificar un DispatchStepRequest válido
	if stepReq.ServicePath != "" {
		slog.Info("🔄 Executing Async Step", "service", stepReq.ServicePath, "order", stepReq.StepOrder)

		if stepReq.ServiceConfig == nil {
			stepReq.ServiceConfig = resolveStepConfig(ctx, stepReq.ServicePath, stepReq.Input)
		}
		if !stepReq.ExecutionPolicy.AutoInvoke.Enabled && stepReq.ExecutionPolicy.NextStep == "" && len(stepReq.ExecutionPolicy.Mode) == 0 {
			if cfgPolicy := extractExecutionPolicy(stepReq.ServiceConfig); cfgPolicy != nil {
				stepReq.ExecutionPolicy = *cfgPolicy
			}
		}

		// Reconstruir ServiceContext
		// Nota: El input viene serializado, necesitamos asegurarnos de que los tipos se respeten.
		// json.Unmarshal decodifica números como float64, hay que tenerlo en cuenta en los servicios.
		svcCtx := contracts.NewServiceContextFromInput(ctx, stepReq.Input)

		// Ejecutar el servicio individual
		if err := serviceconfig.ExecuteDispatchedServiceWithConfig(ctx, stepReq.ServicePath, stepReq.ServiceConfig, svcCtx); err != nil {
			slog.Error("❌ Error executing async step", "service", stepReq.ServicePath, "error", err)
			return handleRunExecutionError(ctx, stepReq.Input, stepReq.ServicePath, err)
		}

		slog.Info("✅ Async Step Completed", "service", stepReq.ServicePath)

		policy := stepReq.ExecutionPolicy
		if policy.AutoInvoke.Enabled {
			if policy.Label != "" {
				slog.Info("🏷️ AutoInvoke Process", "label", policy.Label)
			}
			if stepReq.Input == nil {
				stepReq.Input = make(map[string]any)
			}

			cursorField := policy.AutoInvoke.CursorField
			if cursorField == "" {
				cursorField = "last_id_processed"
			}
			stopField := policy.AutoInvoke.StopCondition
			if stopField == "" {
				stopField = "is_last_batch"
			}

			var nextCursor any
			var shouldStop bool
			var processedCount int
			var found bool

			extractFromMap := func(m map[string]any) {
				if m == nil {
					return
				}

				cursorVal, hasCursor := m[cursorField]
				stopVal, hasStop := m[stopField]
				if !hasCursor || !hasStop {
					return
				}

				nextCursor = cursorVal
				switch b := stopVal.(type) {
				case bool:
					shouldStop = b
					found = true
				case string:
					if b == "true" {
						shouldStop = true
						found = true
					} else if b == "false" {
						shouldStop = false
						found = true
					}
				}

				if val, ok := m["processed_count"]; ok {
					switch n := val.(type) {
					case int:
						processedCount = n
					case int64:
						processedCount = int(n)
					case float64:
						processedCount = int(n)
					}
				}
			}

			if stepRes, ok := svcCtx.GetResult(stepReq.ServicePath); ok && stepRes.Data != nil {
				extractFromMap(stepRes.Data)
			} else {
				for _, res := range svcCtx.Results {
					if stepRes, ok := res.(contracts.StepResult); ok {
						if stepRes.Data != nil {
							extractFromMap(stepRes.Data)
							if found {
								break
							}
						}
					} else if resMap, ok := res.(map[string]any); ok {
						if data, ok := resMap["data"]; ok {
							if dataMap, ok := data.(map[string]any); ok {
								extractFromMap(dataMap)
								if found {
									break
								}
							}
						}
						extractFromMap(resMap)
						if found {
							break
						}
					}
				}
			}

			if found {
				totalProcessed := 0
				if val, ok := stepReq.Input["total_processed"]; ok {
					switch n := val.(type) {
					case int:
						totalProcessed = n
					case int64:
						totalProcessed = int(n)
					case float64:
						totalProcessed = int(n)
					}
				}
				totalProcessed += processedCount
				stepReq.Input["total_processed"] = totalProcessed

				if !shouldStop {
					if shouldSkipQueuedDispatch(ctx, stepReq.Input, stepReq.ServicePath) {
						slog.Info("ℹ️ [Auto-Invoke] Re-queue omitido por corrida cancelada", "service", stepReq.ServicePath)
						return nil
					}
					slog.Info("🔄 [Auto-Invoke] Async step completed. Re-queuing next batch...", "cursor_field", cursorField, "cursor", nextCursor)

					stepReq.Input[cursorField] = nextCursor
					stepReq.Input[stopField] = false

					stepBytes, err := json.Marshal(stepReq)
					if err != nil {
						slog.Error("❌ Error marshaling auto-invoke step request", "error", err)
						return err
					}

					newMessage := &queue.Message{
						ID:      uuid.New().String(),
						Body:    string(stepBytes),
						Source:  "auto-invoke-step",
						Created: time.Now().Format(time.RFC3339),
					}
					if delaySeconds := resolveAutoInvokeDelaySeconds(policy, stepReq.ServiceConfig); delaySeconds > 0 {
						newMessage.DelaySeconds = int32(delaySeconds)
					}

					if err := appContainer.QueueService.SendMessage(ctx, newMessage); err != nil {
						slog.Error("❌ Failed to re-queue auto-invoke step message", "error", err)
						return err
					}
					slog.Info("✅ [Auto-Invoke] Step message re-queued successfully")
				} else {
					slog.Info("✅ [Auto-Invoke] All batches finished (stop_condition=true)", "stop_field", stopField, "cursor", nextCursor)

					shouldDispatchNextStep := true
					if raw, ok := stepReq.Input["should_dispatch_next_step"]; ok {
						if parsed, ok := raw.(bool); ok {
							shouldDispatchNextStep = parsed
						}
					}
					if stepRes, ok := svcCtx.GetResult(stepReq.ServicePath); ok && stepRes.Data != nil {
						if raw, ok := stepRes.Data["should_dispatch_next_step"]; ok {
							if parsed, ok := raw.(bool); ok {
								shouldDispatchNextStep = parsed
							}
						}
					}

					if policy.NextStep != "" && shouldDispatchNextStep {
						if shouldSkipQueuedDispatch(ctx, stepReq.Input, policy.NextStep) {
							slog.Info("ℹ️ [Auto-Invoke] Finalize omitido por corrida cancelada", "service", policy.NextStep)
							return nil
						}
						finalReq := DispatchStepRequest{
							ServicePath:        policy.NextStep,
							ProcessExecutionID: stepReq.ProcessExecutionID,
							Input:              stepReq.Input,
							StepOrder:          stepReq.StepOrder + 1,
							ServiceConfig:      resolveStepConfig(ctx, policy.NextStep, stepReq.Input),
						}
						if cfgPolicy := extractExecutionPolicy(finalReq.ServiceConfig); cfgPolicy != nil {
							finalReq.ExecutionPolicy = *cfgPolicy
						}

						finalBytes, err := json.Marshal(finalReq)
						if err != nil {
							slog.Error("❌ Error marshaling finalize step request", "error", err)
							return err
						}

						finalMsg := &queue.Message{
							ID:      uuid.New().String(),
							Body:    string(finalBytes),
							Source:  "auto-invoke-finalize",
							Created: time.Now().Format(time.RFC3339),
						}

						if err := appContainer.QueueService.SendMessage(ctx, finalMsg); err != nil {
							slog.Error("❌ Failed to queue finalize step message", "error", err)
							return err
						}
						slog.Info("✅ [Auto-Invoke] Finalize step message queued", "service", policy.NextStep, "total_processed", stepReq.Input["total_processed"])
					} else if policy.NextStep != "" && !shouldDispatchNextStep {
						slog.Info("ℹ️ [Auto-Invoke] Shard finalizado sin disparar next_step global", "service", stepReq.ServicePath)
					}
				}
			} else {
				slog.Warn("⚠️ [Auto-Invoke] Could not find cursor/stop fields in async step results", "cursor_field", cursorField, "stop_field", stopField)
			}
		}

		return nil
	}

	// 3. Si ninguno funcionó, reportar error detallado
	slog.Error("❌ Error unmarshalling message: Not a valid RunProcessRequest or DispatchStepRequest",
		"raw_data_preview", rawData[:min(len(rawData), 200)], // Preview seguro
		"process_err", processErr,
		"step_err", stepErr,
	)
	return nil // No reintentar si es basura, o retornar error si queremos DLQ
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func handleRunExecutionError(ctx context.Context, input map[string]any, component string, err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, batchflow.ErrRunCancelled) {
		slog.Info("ℹ️ Ejecución cancelada detectada; el mensaje no se reintentará", "component", component)
		return nil
	}
	if runControl == nil || input == nil {
		return err
	}

	runKey := getStringFromAny(input["key_redis"])
	parentID := getInt64FromAny(input["id"])
	if strings.TrimSpace(runKey) == "" {
		return err
	}

	record, recordErr := runControl.RecordError(ctx, batchflow.RunErrorRecordRequest{
		RunKey:    runKey,
		ParentID:  parentID,
		Component: component,
		Source:    "sqs_consumer",
		Err:       err,
	})
	if recordErr != nil {
		slog.Warn("⚠️ No se pudo registrar error para control de corrida", "run_key", runKey, "error", recordErr)
		return err
	}
	if record.AutoCancelled {
		triggerLifecycleFailOnAutoCancel(ctx, input, component, err)
		slog.Warn("🛑 Corrida auto-cancelada por umbral de errores", "run_key", runKey, "fingerprint", record.Fingerprint, "count", record.Count, "threshold", record.Threshold)
		return nil
	}

	cancelled, status, checkErr := runControl.IsCancelled(ctx, runKey)
	if checkErr == nil && cancelled {
		slog.Info("ℹ️ Corrida ya cancelada; se confirma el corte del mensaje", "run_key", runKey, "reason", status.Reason)
		return nil
	}
	return err
}

func shouldSkipQueuedDispatch(ctx context.Context, input map[string]any, component string) bool {
	if runControl == nil || input == nil {
		return false
	}
	runKey := getStringFromAny(input["key_redis"])
	if strings.TrimSpace(runKey) == "" {
		return false
	}
	cancelled, status, err := runControl.IsCancelled(ctx, runKey)
	if err != nil || !cancelled {
		return false
	}
	ilogger.LogExecutionGuard(
		"run_cancel_skip_reinvoke",
		zap.String("run_key", runKey),
		zap.Int64("parent_id", getInt64FromAny(input["id"])),
		zap.String("component", component),
		zap.String("reason", status.Reason),
	)
	return true
}

func triggerLifecycleFailOnAutoCancel(ctx context.Context, input map[string]any, servicePath string, cause error) {
	if runControl == nil || input == nil {
		return
	}

	runKey := getStringFromAny(input["key_redis"])
	parentID := getInt64FromAny(input["id"])
	if strings.TrimSpace(runKey) == "" || parentID <= 0 {
		return
	}

	locked, err := runControl.AcquireStopLock(ctx, runKey, 24*time.Hour)
	if err != nil {
		slog.Warn("⚠️ No se pudo adquirir stop lock para auto-cancel", "run_key", runKey, "error", err)
		return
	}
	if !locked {
		return
	}

	failer, err := resolveManagedFailer(ctx, servicePath)
	if err != nil {
		slog.Warn("⚠️ Auto-cancel sin manager registrado para ejecutar Fail", "service", servicePath, "run_key", runKey, "error", err)
		return
	}

	if err := failer.Fail(ctx, runKey, parentID, batchflow.ErrRunCancelled); err != nil {
		slog.Error("❌ Falló lifecycle.Fail durante auto-cancel", "service", servicePath, "run_key", runKey, "parent_id", parentID, "error", err)
		return
	}

	ilogger.LogExecutionGuard(
		"run_auto_cancel_fail_applied",
		zap.String("run_key", runKey),
		zap.Int64("parent_id", parentID),
		zap.String("component", servicePath),
		zap.Error(cause),
	)
}

type managedFailer interface {
	Fail(ctx context.Context, runKey string, parentID int64, cause error) error
}

type batchManagerFailer struct {
	manager batchflow.Manager
}

func (f batchManagerFailer) Fail(ctx context.Context, runKey string, parentID int64, cause error) error {
	return f.manager.Fail(ctx, batchflow.Input{
		RedisKey: runKey,
		ParentID: parentID,
	}, cause)
}

type exportManagerFailer struct {
	manager exportmanager.Manager
}

func (f exportManagerFailer) Fail(ctx context.Context, runKey string, parentID int64, cause error) error {
	return f.manager.Fail(ctx, exportmanager.Input{
		RedisKey: runKey,
		ParentID: parentID,
	}, cause)
}

func resolveManagedFailer(ctx context.Context, servicePath string) (managedFailer, error) {
	if manager, err := batchflow.ResolveManagedBatchManager(ctx, servicePath); err == nil && manager != nil {
		return batchManagerFailer{manager: manager}, nil
	}
	if manager, err := exportmanager.ResolveManagedExportManager(ctx, servicePath); err == nil && manager != nil {
		return exportManagerFailer{manager: manager}, nil
	}
	return nil, fmt.Errorf("no existe manager batch/export registrado para %s", servicePath)
}

func resolveStepConfig(ctx context.Context, servicePath string, input map[string]any) map[string]any {
	if appContainer == nil || appContainer.ProcessLifecycleService == nil || input == nil {
		return nil
	}

	processTypeID := getInt64FromAny(input["process_type_id"])
	sedeID := getInt64FromAny(input["sede_id"])
	roadmap := getIntFromAny(input["roadmap"])
	overrideVal, hasOverride := input["override_process_version_id"]
	if processTypeID <= 0 || sedeID < 0 || roadmap < 0 || !hasOverride {
		return nil
	}

	overrideIDValue := getInt64FromAny(overrideVal)
	overrideID := &overrideIDValue

	_, steps, err := appContainer.ProcessLifecycleService.ResolveProcessVersion(ctx, processTypeID, sedeID, overrideID, roadmap, true)
	if err != nil {
		slog.Warn("⚠️ No se pudo resolver config del next_step", "service", servicePath, "error", err)
		return nil
	}

	for _, step := range steps {
		if step.ExecutionKey != servicePath {
			continue
		}
		if len(step.Config) == 0 {
			return nil
		}
		var cfg map[string]any
		if err := json.Unmarshal(step.Config, &cfg); err != nil {
			slog.Warn("⚠️ Config inválida para next_step", "service", servicePath, "error", err)
			return nil
		}
		return cfg
	}

	return nil
}

func extractExecutionPolicy(cfg map[string]any) *contracts.ExecutionPolicy {
	if cfg == nil {
		return nil
	}
	raw, ok := cfg["execution_policy"]
	if !ok || raw == nil {
		return nil
	}
	bytes, err := json.Marshal(raw)
	if err != nil {
		return nil
	}
	var policy contracts.ExecutionPolicy
	if err := json.Unmarshal(bytes, &policy); err != nil {
		return nil
	}
	return &policy
}

func resolveAutoInvokeDelaySeconds(policy contracts.ExecutionPolicy, cfg map[string]any) int {
	if policy.AutoInvoke.DelaySeconds > 0 {
		return policy.AutoInvoke.DelaySeconds
	}
	if cfg == nil {
		return 0
	}
	raw, ok := cfg["dispatch_pacing"]
	if !ok || raw == nil {
		return 0
	}
	bytes, err := json.Marshal(raw)
	if err != nil {
		return 0
	}
	var pacing struct {
		Enabled         bool `json:"enabled"`
		IntervalSeconds int  `json:"interval_seconds"`
	}
	if err := json.Unmarshal(bytes, &pacing); err != nil {
		return 0
	}
	if pacing.Enabled && pacing.IntervalSeconds > 0 {
		return pacing.IntervalSeconds
	}
	return 0
}

func getInt64FromAny(v any) int64 {
	switch n := v.(type) {
	case int:
		return int64(n)
	case int64:
		return n
	case int32:
		return int64(n)
	case float64:
		return int64(n)
	case json.Number:
		i, _ := n.Int64()
		return i
	default:
		return 0
	}
}

func getIntFromAny(v any) int {
	switch n := v.(type) {
	case int:
		return n
	case int64:
		return int(n)
	case int32:
		return int(n)
	case float64:
		return int(n)
	case json.Number:
		i, _ := n.Int64()
		return int(i)
	default:
		return 0
	}
}

func getStringFromAny(v any) string {
	switch raw := v.(type) {
	case string:
		return raw
	case []byte:
		return string(raw)
	case fmt.Stringer:
		return raw.String()
	default:
		return ""
	}
}

func executeRunProcessRequest(ctx context.Context, req requests.RunProcessRequest) error {

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
		return handleRunExecutionError(ctx, req.Input, "run_process_request", err)
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
				if shouldSkipQueuedDispatch(ctx, req.Input, "run_process_request") {
					slog.Info("ℹ️ [Auto-Invoke] Re-queue omitido por corrida cancelada", "process_type_id", req.ProcessTypeID)
					return nil
				}
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
