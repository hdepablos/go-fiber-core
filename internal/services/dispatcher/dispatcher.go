package dispatcher

import (
	"context"
	"encoding/json"
	"fmt"
	"go-fiber-core/internal/services/queue"
	"go-fiber-core/internal/services/serviceconfig/contracts"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

// QueueService define las operaciones necesarias para enviar mensajes.
// Desacopla la implementación concreta (SQS) del despachador.
type QueueService interface {
	SendMessageToUrl(ctx context.Context, queueURL string, message *queue.Message) error
}

// Dispatcher define la interfaz para despachar pasos de proceso
type Dispatcher interface {
	DispatchStep(ctx context.Context, servicePath string, order int, policy contracts.ExecutionPolicy, stepConfig map[string]any, svcCtx *contracts.ServiceContext) error
	SetQueueService(qs QueueService)
}

// ProcessDispatcherService implementa la lógica centralizada de despacho a colas
type ProcessDispatcherService struct {
	queueService QueueService
	mu           sync.RWMutex
}

// NewProcessDispatcherService crea una nueva instancia del dispatcher
func NewProcessDispatcherService() *ProcessDispatcherService {
	return &ProcessDispatcherService{}
}

// SetQueueService permite inyectar el servicio de colas después de la inicialización (Lazy Injection)
func (d *ProcessDispatcherService) SetQueueService(qs QueueService) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.queueService = qs
}

// DispatchStep envía un paso a la cola SQS configurada
func (d *ProcessDispatcherService) DispatchStep(ctx context.Context, servicePath string, order int, policy contracts.ExecutionPolicy, stepConfig map[string]any, svcCtx *contracts.ServiceContext) error {
	d.mu.RLock()
	qs := d.queueService
	d.mu.RUnlock()

	// Si no se ha inyectado un servicio de colas, intentamos inicializar uno por defecto (Fallback)
	// Esto mantiene la compatibilidad con el código legacy o pruebas rápidas, pero debería evitarse en producción.
	if qs == nil {
		fmt.Println("⚠️ Advertencia: QueueService no inyectado en Dispatcher. Inicializando conexión efímera (ineficiente).")
		awsSvc, err := queue.NewAWSService(ctx)
		if err != nil {
			return fmt.Errorf("error initializing AWS service (fallback): %w", err)
		}
		// Usamos un cliente SQS genérico, la URL se pasará en cada llamada
		qs = queue.NewSQSService(sqs.NewFromConfig(awsSvc.GetConfig()), "")
	}

	targetQueue := policy.QueueTarget
	defaultQueueName := os.Getenv("SQS_QUEUE_NAME")
	defaultQueueURL := os.Getenv("SQS_QUEUE_URL")
	if targetQueue == "" {
		targetQueue = defaultQueueName
	}

	fmt.Printf("☁️ Despachando servicio a cola SQS (%s): %s\n", targetQueue, servicePath)

	// Construir payload para el worker
	payload := map[string]any{
		"service_path":         servicePath,
		"process_execution_id": "TODO-UUID",
		"input":                svcCtx.SnapshotInput(),
		"step_order":           order,
		"execution_policy":     policy,
		"service_config":       stepConfig,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("error marshaling payload: %w", err)
	}

	queueURL := ""
	switch {
	case strings.HasPrefix(targetQueue, "http://") || strings.HasPrefix(targetQueue, "https://"):
		queueURL = targetQueue
	case defaultQueueURL != "" && targetQueue != "" && (targetQueue == defaultQueueName || policy.QueueTarget == ""):
		queueURL = defaultQueueURL
	default:
		endpoint := os.Getenv("LOCALSTACK_ENDPOINT_BASE")
		if endpoint == "" {
			endpoint = os.Getenv("AWS_ENDPOINT_URL")
		}
		if endpoint == "" {
			endpoint = "http://localhost:4566"
		}
		if targetQueue == "" {
			return fmt.Errorf("queue target is empty and SQS_QUEUE_URL is not configured")
		}
		queueURL = fmt.Sprintf("%s/000000000000/%s", strings.TrimRight(endpoint, "/"), targetQueue)
	}

	msg := &queue.Message{
		Source:  "process-lifecycle-dispatcher",
		Body:    string(payloadBytes),
		Created: "now",
	}

	// Usar la instancia inyectada o fallback
	dispatchCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := qs.SendMessageToUrl(dispatchCtx, queueURL, msg); err != nil {
		return fmt.Errorf("failed to send message to SQS: %w", err)
	}

	return nil
}

// Global instance
var DefaultDispatcher Dispatcher = NewProcessDispatcherService()
