package test

import (
	"go-fiber-core/internal/services/serviceconfig"
	"go-fiber-core/internal/services/serviceconfig/contracts"
)

type ValidateInput struct {
	ctx         *contracts.ServiceContext
	servicePath string
}

func NewValidateInputService() contracts.Service {
	return &ValidateInput{}
}

func (s *ValidateInput) Init(ctx *contracts.ServiceContext, servicePath string) {
	s.ctx = ctx
	s.servicePath = servicePath
}

func (s *ValidateInput) Execute() error {
	res := contracts.StepResult{
		Status: "ok",
		Data: map[string]any{
			"validated": true,
		},
	}
	s.ctx.SetResult(s.servicePath, res)
	return nil
}

func init() {
	serviceconfig.Register("validate_input", NewValidateInputService)
}
