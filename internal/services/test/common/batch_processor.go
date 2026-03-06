package common

import (
	"fmt"
	"go-fiber-core/internal/services/serviceconfig"
	"go-fiber-core/internal/services/serviceconfig/contracts"
	"time"
)

type BatchProcessorService struct {
	ctx         *contracts.ServiceContext
	servicePath string
}

func NewBatchProcessorService() contracts.Service {
	return &BatchProcessorService{}
}

func (s *BatchProcessorService) Init(ctx *contracts.ServiceContext, servicePath string) {
	s.ctx = ctx
	s.servicePath = servicePath
}

func (s *BatchProcessorService) Execute() error {
	fmt.Printf("📦 Ejecutando BatchProcessorService: %s\n", s.servicePath)
	
	// Simular procesamiento de lote
	lastID, _ := s.ctx.GetInputValue("last_id_processed")
	fmt.Printf("🔄 Procesando desde ID: %v\n", lastID)

	time.Sleep(100 * time.Millisecond)

	data := map[string]any{
		"processed_count": 100,
		"next_id": 100,
		"status": "partial",
	}

	result := contracts.StepResult{
		Status: "completed",
		Data:   data,
	}
	
	s.ctx.SetResult(s.servicePath, result)
	return nil
}

func init() {
	serviceconfig.Register("batch/processor", NewBatchProcessorService)
}
