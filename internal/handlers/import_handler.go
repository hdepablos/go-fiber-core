package handlers

import (
	"path/filepath"
	"strconv"
	"strings"

	"go-fiber-core/internal/domain"
	"go-fiber-core/internal/dtos"
	"go-fiber-core/internal/dtos/requests"
	"go-fiber-core/internal/dtos/responses"
	importsService "go-fiber-core/internal/services/imports"

	fiber "github.com/gofiber/fiber/v2"
)

type ImportHandler interface {
	Create(c *fiber.Ctx) error
	CreateLegacy(c *fiber.Ctx) error
	GetBulkJobsPaginated(c *fiber.Ctx) error
}

type importHandler struct {
	service importsService.Service
}

func NewImportHandler(service importsService.Service) ImportHandler {
	return &importHandler{service: service}
}

func (h *importHandler) Create(c *fiber.Ctx) error {
	ctx := c.UserContext()

	operatorID, err := getUserIDUint64FromCtx(ctx)
	if err != nil {
		return err
	}

	req, err := parseCreateImportRequest(c)
	if err != nil {
		return err
	}

	fh, err := c.FormFile("file")
	if err != nil || fh == nil {
		return domain.ErrInvalidArgument
	}

	ext := strings.ToLower(filepath.Ext(fh.Filename))
	if ext != ".csv" && ext != ".txt" {
		return domain.ErrInvalidArgument
	}

	f, err := fh.Open()
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	if err := h.service.Process(ctx, operatorID, req.BranchID, req.RefCode, req.Total, req.KeyCode, fh.Filename, f); err != nil {
		return err
	}

	return responses.Success(c, "Import iniciado exitosamente", fiber.Map{
		"processing": true,
	})
}

func (h *importHandler) CreateLegacy(c *fiber.Ctx) error {
	ctx := c.UserContext()

	operatorID, err := getUserIDUint64FromCtx(ctx)
	if err != nil {
		return err
	}

	req, err := parseLegacyImportRequest(c)
	if err != nil {
		return err
	}

	fh, err := c.FormFile("file")
	if err != nil || fh == nil {
		return domain.ErrInvalidArgument
	}

	ext := strings.ToLower(filepath.Ext(fh.Filename))
	if ext != ".csv" && ext != ".txt" {
		return domain.ErrInvalidArgument
	}

	f, err := fh.Open()
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	if err := h.service.Process(ctx, operatorID, req.BranchID, req.RefCode, req.Total, req.KeyCode, fh.Filename, f); err != nil {
		return err
	}

	return responses.Success(c, "Import iniciado exitosamente", fiber.Map{
		"processing": true,
		"legacy":     true,
	})
}

func (h *importHandler) GetBulkJobsPaginated(c *fiber.Ctx) error {
	ctx := c.UserContext()

	if _, err := getUserIDUint64FromCtx(ctx); err != nil {
		return responses.Error(c, fiber.StatusUnauthorized, "Error de autenticación", err)
	}

	var req dtos.PaginationRequest
	if err := c.BodyParser(&req); err != nil {
		return domain.ErrInvalidArgument
	}

	result, err := h.service.GetBulkJobsPaginated(ctx, req)
	if err != nil {
		return err
	}

	return responses.Success(c, "Bulk jobs paginados obtenidos exitosamente", result)
}

func parseCreateImportRequest(c *fiber.Ctx) (requests.CreateImportRequest, error) {
	req := requests.CreateImportRequest{
		RefCode: strings.TrimSpace(c.FormValue("ref_code")),
		KeyCode: strings.TrimSpace(c.FormValue("key_code")),
	}

	branchID, err := strconv.ParseInt(strings.TrimSpace(c.FormValue("branch_id")), 10, 64)
	if err != nil {
		return requests.CreateImportRequest{}, domain.ErrInvalidArgument
	}
	total, err := strconv.Atoi(strings.TrimSpace(c.FormValue("total")))
	if err != nil {
		return requests.CreateImportRequest{}, domain.ErrInvalidArgument
	}

	req.BranchID = branchID
	req.Total = total
	return normalizeCreateImportRequest(req)
}

func parseLegacyImportRequest(c *fiber.Ctx) (requests.CreateImportRequest, error) {
	branchID, err := strconv.ParseInt(c.Params("branchId"), 10, 64)
	if err != nil {
		return requests.CreateImportRequest{}, domain.ErrInvalidArgument
	}
	total, err := strconv.Atoi(c.Params("total"))
	if err != nil {
		return requests.CreateImportRequest{}, domain.ErrInvalidArgument
	}

	req := requests.CreateImportRequest{
		BranchID: branchID,
		RefCode:  c.Params("refCode"),
		Total:    total,
		KeyCode:  c.Params("key"),
	}
	return normalizeCreateImportRequest(req)
}

func normalizeCreateImportRequest(req requests.CreateImportRequest) (requests.CreateImportRequest, error) {
	req.RefCode = strings.TrimSpace(req.RefCode)
	req.KeyCode = strings.TrimSpace(req.KeyCode)
	if req.BranchID < 0 || req.RefCode == "" || req.KeyCode == "" || req.Total < 0 {
		return requests.CreateImportRequest{}, domain.ErrInvalidArgument
	}
	return req, nil
}
