package heavy

import (
	"fmt"
	"go-fiber-core/internal/services/serviceconfig"
	"go-fiber-core/internal/services/serviceconfig/contracts"
	"time"
)

type ProcessService struct {
	ctx         *contracts.ServiceContext
	servicePath string
}

func NewProcessService() contracts.Service {
	return &ProcessService{}
}

func (s *ProcessService) Init(ctx *contracts.ServiceContext, servicePath string) {
	s.ctx = ctx
	s.servicePath = servicePath
}

func (s *ProcessService) Execute() error {
	fmt.Printf("🏗️ Ejecutando HeavyProcessService: %s\n", s.servicePath)
	
	// Simular carga de trabajo real
	time.Sleep(500 * time.Millisecond)
	
	data := map[string]any{
		"processed": true,
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
