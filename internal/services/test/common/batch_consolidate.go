package common

import (
	"go-fiber-core/internal/services/serviceconfig"
	"go-fiber-core/internal/services/serviceconfig/contracts"
)

type batchConsolidateService struct {
	ctx         *contracts.ServiceContext
	servicePath string
}

func NewBatchConsolidateService() contracts.Service {
	return &batchConsolidateService{}
}

func (s *batchConsolidateService) Init(ctx *contracts.ServiceContext, servicePath string) {
	s.ctx = ctx
	s.servicePath = servicePath
}

func (s *batchConsolidateService) Execute() error {
	// Servicio de ejemplo: simula una consolidación al final de un batch.
	data := map[string]any{
		"status":          "success",
		"consolidated_at": "now",
	}

	result := contracts.StepResult{
		Status: "completed",
		Data:   data,
	}

	if s.ctx != nil {
		s.ctx.SetResult(s.servicePath, result)
	}
	return nil
}

func init() {
	serviceconfig.Register("batch/consolidate", NewBatchConsolidateService)
}
