package test

import (
	"go-fiber-core/internal/services/serviceconfig"
	"go-fiber-core/internal/services/serviceconfig/contracts"
)

type validateInput struct {
	ctx         *contracts.ServiceContext
	servicePath string
}

func NewValidateInputService() contracts.Service {
	return &validateInput{}
}

func (s *validateInput) Init(ctx *contracts.ServiceContext, servicePath string) {
	s.ctx = ctx
	s.servicePath = servicePath
}

func (s *validateInput) Execute() error {
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
