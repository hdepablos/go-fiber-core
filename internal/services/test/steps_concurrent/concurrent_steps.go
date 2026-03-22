package steps_concurrent

import (
	"fmt"
	"go-fiber-core/internal/services/serviceconfig"
	"go-fiber-core/internal/services/serviceconfig/contracts"
	"time"
)

// ConcurrentStepService simulates a process step with a configurable delay.
type ConcurrentStepService struct {
	Name        string
	Delay       time.Duration
	Ctx         *contracts.ServiceContext
	ServicePath string
}

// Init initializes the service with context and path.
func (s *ConcurrentStepService) Init(ctx *contracts.ServiceContext, servicePath string) {
	s.Ctx = ctx
	s.ServicePath = servicePath
	// Ensure delay is set, default to 1s if not configured
	if s.Delay == 0 {
		s.Delay = 1 * time.Second
	}
}

// Execute performs the step logic: wait for the specified delay.
func (s *ConcurrentStepService) Execute() error {
	select {
	case <-time.After(s.Delay):
		// Guardamos un resultado en el contexto para verificar luego
		s.Ctx.SetResult(s.ServicePath, contracts.StepResult{
			Status:  "completed",
			Message: fmt.Sprintf("Ejecutado en %v", s.Delay),
			// Limpiamos el Data para evitar ensuciar la respuesta en producción
			Data: nil,
		})
		return nil
	case <-s.Ctx.Ctx.Done():
		return s.Ctx.Ctx.Err()
	}
}

// Factory creators for the 5 steps

func NewStep1() contracts.Service {
	return &ConcurrentStepService{Name: "Step 1", Delay: 1 * time.Second}
}

func NewStep2() contracts.Service {
	return &ConcurrentStepService{Name: "Step 2", Delay: 1 * time.Second}
}

func NewStep3() contracts.Service {
	return &ConcurrentStepService{Name: "Step 3", Delay: 1 * time.Second}
}

func NewStep4() contracts.Service {
	return &ConcurrentStepService{Name: "Step 4", Delay: 1 * time.Second}
}

func NewStep5() contracts.Service {
	return &ConcurrentStepService{Name: "Step 5", Delay: 1 * time.Second}
}

// RegisterServices registers all 5 steps in the service registry.
// This function should be called during app initialization (e.g., in main.go or a specific init file).
func init() {
	serviceconfig.Register("test/concurrent/step1", NewStep1)
	serviceconfig.Register("test/concurrent/step2", NewStep2)
	serviceconfig.Register("test/concurrent/step3", NewStep3)
	serviceconfig.Register("test/concurrent/step4", NewStep4)
	serviceconfig.Register("test/concurrent/step5", NewStep5)
}
