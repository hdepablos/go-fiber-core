package handlers

import (
	"go-fiber-core/internal/domain"
	"go-fiber-core/internal/dtos/requests"
	"go-fiber-core/internal/dtos/responses"
	"go-fiber-core/internal/services/processlifecycle"

	fiber "github.com/gofiber/fiber/v2"
)

type ProcessLifecycleHandler interface {
	ReplicateScenario(c *fiber.Ctx) error
	PromoteScenario(c *fiber.Ctx) error
	ResolveScenario(c *fiber.Ctx) error
	MoveToTestScenario(c *fiber.Ctx) error
}

type processLifecycleHandler struct {
	service processlifecycle.Service
}

func NewProcessLifecycleHandler(service processlifecycle.Service) ProcessLifecycleHandler {
	return &processLifecycleHandler{
		service: service,
	}
}

func (h *processLifecycleHandler) ReplicateScenario(c *fiber.Ctx) error {
	ctx := c.UserContext()

	var req requests.ReplicateScenarioRequest
	if err := c.BodyParser(&req); err != nil {
		return domain.ErrInvalidArgument
	}

	newID, err := h.service.ReplicateProcessVersion(ctx, req.ProcessVersionID, req.OperatorID)
	if err != nil {
		return err
	}

	return responses.Success(c, "Escenario replicado exitosamente", fiber.Map{
		"new_process_version_id": newID,
	})
}

func (h *processLifecycleHandler) PromoteScenario(c *fiber.Ctx) error {
	ctx := c.UserContext()

	var req requests.PromoteScenarioRequest
	if err := c.BodyParser(&req); err != nil {
		return domain.ErrInvalidArgument
	}

	if err := h.service.PromoteProcessVersion(ctx, req.ProcessVersionID, req.PromotedBy, req.Comment); err != nil {
		return err
	}

	return responses.Success(c, "Escenario promovido a producción exitosamente", nil)
}

func (h *processLifecycleHandler) ResolveScenario(c *fiber.Ctx) error {
	ctx := c.UserContext()

	var req requests.ResolveScenarioRequest
	if err := c.BodyParser(&req); err != nil {
		return domain.ErrInvalidArgument
	}

	resolvedID, steps, err := h.service.ResolveProcessVersion(ctx, req.ProcessTypeID, req.SedeID, req.OverrideProcessVersionID)
	if err != nil {
		return err
	}

	return responses.Success(c, "Escenario resuelto exitosamente", fiber.Map{
		"process_version_id": resolvedID,
		"process_steps":      steps,
	})
}

func (h *processLifecycleHandler) MoveToTestScenario(c *fiber.Ctx) error {
	ctx := c.UserContext()

	var req requests.MoveToTestScenarioRequest
	if err := c.BodyParser(&req); err != nil {
		return domain.ErrInvalidArgument
	}

	if err := h.service.MoveProcessVersionToTest(ctx, req.ProcessVersionID); err != nil {
		return err
	}

	return responses.Success(c, "Escenario movido a TEST exitosamente", nil)
}
