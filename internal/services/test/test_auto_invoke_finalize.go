package test

import (
	"go-fiber-core/internal/services/serviceconfig"
	"go-fiber-core/internal/services/serviceconfig/contracts"
)

type testAutoInvokeFinalize struct {
	ctx         *contracts.ServiceContext
	servicePath string
}

func NewTestAutoInvokeFinalize() contracts.Service {
	return &testAutoInvokeFinalize{}
}

func (s *testAutoInvokeFinalize) Init(ctx *contracts.ServiceContext, servicePath string) {
	s.ctx = ctx
	s.servicePath = servicePath
}

func (s *testAutoInvokeFinalize) Execute() error {
	// Lee el total acumulado (propagado por la cadena de ejecución) para exponerlo como resultado final.
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
