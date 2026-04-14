package test

import (
	"fmt"
	"go-fiber-core/internal/services/serviceconfig"
	"go-fiber-core/internal/services/serviceconfig/contracts"
	"math/rand"
	"time"
)

// TestAutoInvoke simulates a batch process with recursion capability.
type TestAutoInvoke struct {
	ctx         *contracts.ServiceContext
	servicePath string
}

// NewTestAutoInvoke creates a new instance of the service.
func NewTestAutoInvoke() contracts.Service {
	return &TestAutoInvoke{}
}

// Init initializes the service with context.
func (s *TestAutoInvoke) Init(ctx *contracts.ServiceContext, servicePath string) {
	s.ctx = ctx
	s.servicePath = servicePath
}

// Execute contains the business logic for the simulation.
func (s *TestAutoInvoke) Execute() error {
	lastID := 0
	if val, ok := s.ctx.GetInputValue("last_id_processed"); ok {
		switch v := val.(type) {
		case float64:
			lastID = int(v)
		case int:
			lastID = v
		case int64:
			lastID = int(v)
		}
	}

	// 2. Logic based on user requirements
	var newLastID int
	var isLastBatch bool
	var batchLabel string
	var batchConsoleMessage string
	var processedCount int

	switch lastID {
	case 0:
		batchLabel = "Ejecutando Lote 1/4"
		batchConsoleMessage = "Proceso de 1/4 Inicio (simulando lógica de negocio)"
		newLastID = 500
		isLastBatch = false
	case 500:
		batchLabel = "Ejecutando Lote 2/4"
		batchConsoleMessage = "Proceso de 2/4 Inicio (simulando lógica de negocio)"
		newLastID = 1500
		isLastBatch = false
	case 1500:
		batchLabel = "Ejecutando Lote 3/4"
		batchConsoleMessage = "Proceso de 3/4 Inicio (simulando lógica de negocio)"
		newLastID = 2500
		isLastBatch = false
	case 2500:
		batchLabel = "Ejecutando Lote 4/4"
		batchConsoleMessage = "Proceso de 4/4 Fin (simulando lógica de negocio)"
		newLastID = 3500
		isLastBatch = true
	default:
		// Safety fallback
		batchLabel = "Ejecutando Lote desconocido"
		batchConsoleMessage = "Proceso de lote desconocido (simulando lógica de negocio)"
		newLastID = lastID
		isLastBatch = true
	}
	processedCount = newLastID - lastID

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	processTime := rng.Intn(4) + 2 // (0-3) + 2 = 2-5
	time.Sleep(time.Duration(processTime) * time.Second)

	// 3. Check for autoInvoke config and propagate to Input
	// This ensures the SQS Consumer can see the flag even if it wasn't in the original request input
	if cfg, ok := s.ctx.CurrentStepConfig["autoInvoke"]; ok {
		// Propaga el flag al input para que el siguiente ciclo lo pueda leer.
		s.ctx.SetInputValue("autoInvoke", cfg)
	}

	fmt.Println(batchLabel)
	// 4. Set Result
	// We put the data in Data, which the consumer will read to update the Input for the next run
	result := contracts.StepResult{
		Status:  "success",
		Message: batchLabel,
		Data: map[string]any{
			"batch_label":       batchLabel,
			"batch_message":     batchConsoleMessage,
			"processed_count":   processedCount,
			"last_id_processed": newLastID,
			"is_last_batch":     isLastBatch,
		},
	}

	s.ctx.SetResult(s.servicePath, result)
	return nil
}

func init() {
	serviceconfig.Register("test/test_auto_invoke", NewTestAutoInvoke)
}
