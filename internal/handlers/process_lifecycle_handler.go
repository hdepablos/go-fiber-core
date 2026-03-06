package handlers

import (
	"errors"
	"sort"

	"go-fiber-core/internal/domain"
	"go-fiber-core/internal/dtos"
	"go-fiber-core/internal/dtos/requests"
	"go-fiber-core/internal/dtos/responses"
	"go-fiber-core/internal/logger"
	"go-fiber-core/internal/services/processlifecycle"
	"go-fiber-core/internal/services/serviceconfig/contracts"

	fiber "github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
)

type ProcessLifecycleHandler interface {
	ReplicateScenario(c *fiber.Ctx) error
	PromoteScenario(c *fiber.Ctx) error
	ResolveScenario(c *fiber.Ctx) error
	ResolveCurrentVersion(c *fiber.Ctx) error
	GetProcessVersion(c *fiber.Ctx) error
	MoveToTestScenario(c *fiber.Ctx) error
	ListProcessVersions(c *fiber.Ctx) error
	RunLoanRiskLifecycle(c *fiber.Ctx) error
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
	log := logger.GetLogger("promote_scenario")

	var req requests.PromoteScenarioRequest
	if err := c.BodyParser(&req); err != nil {
		log.Error("Failed to parse PromoteScenario request", zap.Error(err))
		return domain.ErrInvalidArgument
	}

	log.Info("Starting PromoteScenario",
		zap.Int64("process_version_id", req.ProcessVersionID),
		zap.String("comment", req.Comment),
		zap.Int64("promoted_by", req.PromotedBy),
		zap.String("client_ip", c.IP()),
	)

	if err := h.service.PromoteProcessVersion(ctx, req.ProcessVersionID, req.PromotedBy, req.Comment); err != nil {
		log.Error("PromoteScenario failed",
			zap.Int64("process_version_id", req.ProcessVersionID),
			zap.Error(err),
		)
		return err
	}

	log.Info("PromoteScenario success",
		zap.Int64("process_version_id", req.ProcessVersionID),
	)

	return responses.Success(c, "Escenario promovido a producción exitosamente", nil)
}

func (h *processLifecycleHandler) ResolveScenario(c *fiber.Ctx) error {
	ctx := c.UserContext()

	var req requests.ResolveScenarioRequest
	if err := c.BodyParser(&req); err != nil {
		return domain.ErrInvalidArgument
	}

	resolvedID, steps, err := h.service.ResolveProcessVersion(ctx, req.ProcessTypeID, req.SedeID, req.OverrideProcessVersionID, *req.Roadmap, true)
	if err != nil {

		return err
	}

	return responses.Success(c, "Escenario resuelto exitosamente", fiber.Map{
		"process_version_id": resolvedID,
		"process_steps":      steps,
	})
}

func (h *processLifecycleHandler) ResolveCurrentVersion(c *fiber.Ctx) error {
	ctx := c.UserContext()

	var req requests.ResolveScenarioRequest
	if err := c.BodyParser(&req); err != nil {
		return domain.ErrInvalidArgument
	}

	resolvedID, _, err := h.service.ResolveProcessVersion(ctx, req.ProcessTypeID, req.SedeID, req.OverrideProcessVersionID, *req.Roadmap, true)
	if err != nil {
		return err
	}

	return responses.Success(c, "Versión vigente resuelta exitosamente", fiber.Map{
		"process_version_id": resolvedID,
	})
}

func (h *processLifecycleHandler) GetProcessVersion(c *fiber.Ctx) error {
	ctx := c.UserContext()

	idUint, err := getUintID(c)
	if err != nil {
		return err
	}

	item, svcErr := h.service.GetProcessVersionByID(ctx, int64(idUint))
	if svcErr != nil {
		return svcErr
	}

	steps, svcErr := h.service.GetProcessStepsByVersionID(ctx, int64(idUint))
	if svcErr != nil {
		return svcErr
	}

	return responses.Success(c, "Versión de proceso obtenida exitosamente", fiber.Map{
		"version": item,
		"steps":   steps,
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

func (h *processLifecycleHandler) ListProcessVersions(c *fiber.Ctx) error {
	ctx := c.UserContext()

	var req dtos.PaginationRequest
	if err := c.BodyParser(&req); err != nil {
		return domain.ErrInvalidArgument
	}

	result, err := h.service.ListProcessVersions(ctx, req)
	if err != nil {
		return err
	}

	return responses.Success(c, "Versiones de proceso paginadas obtenidas exitosamente", result)
}

func (h *processLifecycleHandler) RunLoanRiskLifecycle(c *fiber.Ctx) error {
	ctx := c.UserContext()

	var req requests.RunProcessRequest
	if err := c.BodyParser(&req); err != nil {
		return domain.ErrInvalidArgument
	}

	if req.Input == nil {
		req.Input = make(map[string]any)
	}

	if _, ok := req.Input["sede_id"]; !ok {
		req.Input["sede_id"] = req.SedeID
	}

	// Extract OperatorID from JWT claims (if available)
	if token := c.Locals("user"); token != nil {
		if t, ok := token.(*jwt.Token); ok {
			if claims, ok := t.Claims.(jwt.MapClaims); ok {
				if id, ok := claims["id"].(float64); ok {
					req.OperatorID = int64(id)
				}
			}
		}
	}

	// Inyectar el roadmap al input para que esté disponible en el contexto si es necesario
	// (la nueva DTO y servicio ya lo hacen, pero mantenemos por claridad)
	req.Input["roadmap"] = req.Roadmap

	processVersionID, svcCtx, execErr := h.service.Run(ctx, req)
	output := map[string]any{
		"process_version_id": processVersionID,
		"input":              req.Input,
		"details":            map[string]any{},
	}

	if svcCtx != nil {
		if svcCtx.Results != nil {
			output["details"] = svcCtx.Results
		}
		output["result"] = svcCtx.GetAll()

		if svcCtx.Results != nil {
			type orderedResult struct {
				ServicePath string `json:"service_path"`
				StepOrder   int    `json:"step_order"`
			}

			ordered := make([]orderedResult, 0, len(svcCtx.Results))

			for path, raw := range svcCtx.Results {
				switch v := raw.(type) {
				case contracts.StepResult:
					ordered = append(ordered, orderedResult{
						ServicePath: path,
						StepOrder:   v.StepOrder,
					})
				default:
					ordered = append(ordered, orderedResult{
						ServicePath: path,
						StepOrder:   0,
					})
				}
			}

			sort.Slice(ordered, func(i, j int) bool {
				if ordered[i].StepOrder == ordered[j].StepOrder {
					return ordered[i].ServicePath < ordered[j].ServicePath
				}
				return ordered[i].StepOrder < ordered[j].StepOrder
			})

			output["execute_ordered"] = ordered

			// Inyectar métricas de rendimiento si existen (solo en modo Test)
			if svcCtx.Metrics != nil {
				output["performance"] = svcCtx.Metrics
			}
		}
	}

	if execErr != nil {
		statusCode := fiber.StatusInternalServerError

		errorPayload := map[string]any{
			"message": execErr.Error(),
		}

		switch {
		case errors.Is(execErr, domain.ErrNotFound):
			statusCode = fiber.StatusNotFound
			errorPayload["code"] = "PROCESS_VERSION_NOT_FOUND"
		case errors.Is(execErr, domain.ErrSedeNotFound):
			statusCode = fiber.StatusNotFound
			errorPayload["code"] = "SEDE_NOT_FOUND"
		case errors.Is(execErr, domain.ErrRoadmapNotFound):
			statusCode = fiber.StatusNotFound
			errorPayload["code"] = "ROADMAP_NOT_FOUND"
		case errors.Is(execErr, domain.ErrOverrideVersionNotFound):
			statusCode = fiber.StatusNotFound
			errorPayload["code"] = "OVERRIDE_VERSION_NOT_FOUND"
		case errors.Is(execErr, domain.ErrMissingRequiredKey):
			statusCode = fiber.StatusUnprocessableEntity
			errorPayload["code"] = "MISSING_REQUIRED_KEY"
		case errors.Is(execErr, domain.ErrValueOutOfRange):
			statusCode = fiber.StatusUnprocessableEntity
			errorPayload["code"] = "VALUE_OUT_OF_RANGE"
		case errors.Is(execErr, domain.ErrBusinessRuleViolation):
			statusCode = fiber.StatusUnprocessableEntity
			errorPayload["code"] = "BUSINESS_RULE_VIOLATION"
		case errors.Is(execErr, domain.ErrInvalidArgument):
			statusCode = fiber.StatusUnprocessableEntity
			errorPayload["code"] = "INVALID_ARGUMENT"
		case errors.Is(execErr, domain.ErrCritical):
			statusCode = fiber.StatusInternalServerError
			errorPayload["code"] = "CRITICAL_ERROR"
		default:
			statusCode = fiber.StatusInternalServerError
			errorPayload["code"] = "INTERNAL_ERROR"
		}

		output["error"] = errorPayload

		return responses.Error(c, statusCode, "Error ejecutando proceso lifecycle", output)
	}

	return responses.Success(c, "Loan risk lifecycle ejecutado exitosamente", output)
}
