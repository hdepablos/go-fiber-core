package test

import (
	"go-fiber-core/internal/services/serviceconfig"
	"go-fiber-core/internal/services/serviceconfig/contracts"
)

type ApplyBusinessRules struct {
	ctx         *contracts.ServiceContext
	servicePath string
}

func NewApplyBusinessRulesService() contracts.Service {
	return &ApplyBusinessRules{}
}

func (s *ApplyBusinessRules) Init(ctx *contracts.ServiceContext, servicePath string) {
	s.ctx = ctx
	s.servicePath = servicePath
}

func (s *ApplyBusinessRules) Execute() error {
	res := contracts.StepResult{
		Status: "ok",
		Data: map[string]any{
			"business_rules_applied": true,
		},
	}
	s.ctx.SetResult(s.servicePath, res)
	return nil
}

func init() {
	serviceconfig.Register("apply_business_rules", NewApplyBusinessRulesService)
}
