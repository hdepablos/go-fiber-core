package common

import (
	"go-fiber-core/internal/services/serviceconfig"
	"go-fiber-core/internal/services/serviceconfig/contracts"
)

type notifyService struct {
	ctx         *contracts.ServiceContext
	servicePath string
}

func NewNotifyService() contracts.Service {
	return &notifyService{}
}

func (s *notifyService) Init(ctx *contracts.ServiceContext, servicePath string) {
	s.ctx = ctx
	s.servicePath = servicePath
}

func (s *notifyService) Execute() error {
	// Aquí se integraría con el servicio de email real (AWS SES / SMTP)
	// Por ahora simulamos el envío exitoso.

	data := map[string]any{
		"sent":      true,
		"channel":   "email",
		"timestamp": "now",
	}

	result := contracts.StepResult{
		Status: "completed",
		Data:   data,
	}

	s.ctx.SetResult(s.servicePath, result)
	return nil
}

func init() {
	serviceconfig.Register("common/notify", NewNotifyService)
}
