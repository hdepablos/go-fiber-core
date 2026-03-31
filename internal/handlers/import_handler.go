package handlers

import (
	"path/filepath"
	"strconv"
	"strings"

	"go-fiber-core/internal/domain"
	importsService "go-fiber-core/internal/services/imports"

	fiber "github.com/gofiber/fiber/v2"
)

type ImportHandler interface {
	Process(c *fiber.Ctx) error
}

type importHandler struct {
	service importsService.Service
}

func NewImportHandler(service importsService.Service) ImportHandler {
	return &importHandler{service: service}
}

func (h *importHandler) Process(c *fiber.Ctx) error {
	ctx := c.UserContext()

	operatorID, err := getUserIDUint64FromCtx(ctx)
	if err != nil {
		return err
	}

	branchID, err := strconv.ParseInt(c.Params("branchId"), 10, 64)
	if err != nil {
		return domain.ErrInvalidArgument
	}

	refCode := strings.TrimSpace(c.Params("refCode"))
	if refCode == "" {
		return domain.ErrInvalidArgument
	}

	total, err := strconv.Atoi(c.Params("total"))
	if err != nil || total < 0 {
		return domain.ErrInvalidArgument
	}

	keyCode := strings.TrimSpace(c.Params("key"))
	if keyCode == "" {
		return domain.ErrInvalidArgument
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

	if err := h.service.Process(ctx, operatorID, branchID, refCode, total, keyCode, fh.Filename, f); err != nil {
		return err
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"processing": true,
	})
}

