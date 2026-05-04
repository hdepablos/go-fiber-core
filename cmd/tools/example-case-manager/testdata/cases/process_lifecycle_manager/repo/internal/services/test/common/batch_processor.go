package common

import (
	"go-fiber-core/internal/services/serviceconfig"
	"go-fiber-core/internal/services/serviceconfig/contracts"
	"time"
)

type batchProcessorService struct {
	ctx         *contracts.ServiceContext
	servicePath string
}

func NewBatchProcessorService() contracts.Service {
	return &batchProcessorService{}
}

func (s *batchProcessorService) Init(ctx *contracts.ServiceContext, servicePath string) {
	s.ctx = ctx
	s.servicePath = servicePath
}

func (s *batchProcessorService) Execute() error {
	// Simular procesamiento de lote
	lastID, _ := s.ctx.GetInputValue("last_id_processed")

	time.Sleep(100 * time.Millisecond)

	data := map[string]any{
		"last_id_processed": lastID,
		"processed_count":   100,
		"next_id":           100,
		"status":            "partial",
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
