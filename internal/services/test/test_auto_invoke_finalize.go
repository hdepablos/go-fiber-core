package test

import (
	"fmt"
	"go-fiber-core/internal/services/serviceconfig"
	"go-fiber-core/internal/services/serviceconfig/contracts"
)

type TestAutoInvokeFinalize struct {
	ctx         *contracts.ServiceContext
	servicePath string
}

func NewTestAutoInvokeFinalize() contracts.Service {
	return &TestAutoInvokeFinalize{}
}

func (s *TestAutoInvokeFinalize) Init(ctx *contracts.ServiceContext, servicePath string) {
	s.ctx = ctx
	s.servicePath = servicePath
}

func (s *TestAutoInvokeFinalize) Execute() error {
	var totalProcessed int
	if s.ctx != nil {
		if v, ok := s.ctx.GetInputValue("total_processed"); ok {
			switch n := v.(type) {
			case int:
				totalProcessed = n
			case int64:
				totalProcessed = int(n)
			case float64:
				totalProcessed = int(n)
			}
		}
	}

	fmt.Printf("🏁 Ejecutado luego de finalizar todo el proceso de lote. Total registros procesados: %d\n", totalProcessed)

	result := contracts.StepResult{
		Status: "success",
		Data: map[string]any{
			"total_processed": totalProcessed,
		},
	}
	if s.ctx != nil {
		s.ctx.SetResult(s.servicePath, result)
	}
	return nil
}

func init() {
	serviceconfig.Register("test/test_auto_invoke_finalize", NewTestAutoInvokeFinalize)
}
