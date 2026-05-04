package heavy

import (
	"go-fiber-core/internal/services/serviceconfig"
	"go-fiber-core/internal/services/serviceconfig/contracts"
	"time"
)

type processService struct {
	ctx         *contracts.ServiceContext
	servicePath string
}

func NewProcessService() contracts.Service {
	return &processService{}
}

func (s *processService) Init(ctx *contracts.ServiceContext, servicePath string) {
	s.ctx = ctx
	s.servicePath = servicePath
}

func (s *processService) Execute() error {
	// Simular carga de trabajo real
	time.Sleep(500 * time.Millisecond)

	data := map[string]any{
		"processed":    true,
		"heavy_result": "done",
	}

	result := contracts.StepResult{
		Status: "completed",
		Data:   data,
	}

	s.ctx.SetResult(s.servicePath, result)
	return nil
}

func init() {
	serviceconfig.Register("heavy/process", NewProcessService)
}
