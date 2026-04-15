package test

import (
	"go-fiber-core/internal/services/serviceconfig"
	"go-fiber-core/internal/services/serviceconfig/contracts"
)

type persistResults struct {
	ctx         *contracts.ServiceContext
	servicePath string
}

func NewPersistResultsService() contracts.Service {
	return &persistResults{}
}

func (s *persistResults) Init(ctx *contracts.ServiceContext, servicePath string) {
	s.ctx = ctx
	s.servicePath = servicePath
}

func (s *persistResults) Execute() error {
	res := contracts.StepResult{
		Status: "ok",
		Data: map[string]any{
			"persisted": true,
		},
	}
	s.ctx.SetResult(s.servicePath, res)
	return nil
}

func init() {
	serviceconfig.Register("persist_results", NewPersistResultsService)
}
