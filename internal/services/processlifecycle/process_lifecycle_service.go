package processlifecycle

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"go-fiber-core/internal/domain"
	"go-fiber-core/internal/dtos"
	"go-fiber-core/internal/dtos/connect"
	"go-fiber-core/internal/models"
)

type Service interface {
	ReplicateProcessVersion(ctx context.Context, processVersionID int64, operatorID int64) (int64, error)
	PromoteProcessVersion(ctx context.Context, processVersionID int64, operatorID int64, comment string) error
	ResolveProcessVersion(ctx context.Context, processTypeID int64, sedeID int64, overrideProcessVersionID *int64) (int64, []Step, error)
	MoveProcessVersionToTest(ctx context.Context, processVersionID int64) error
	ListProcessVersions(ctx context.Context, req dtos.PaginationRequest) (*dtos.PaginationResponse[models.ProcessVersionListItem], error)
}

type Step struct {
	Name         string          `json:"name"`
	ExecutionKey string          `json:"execution_key"`
	Config       json.RawMessage `json:"config"`
	StepOrder    int32           `json:"step_order"`
}

type service struct {
	conn *connect.ConnectDTO
}

func NewService(conn *connect.ConnectDTO) Service {
	return &service{
		conn: conn,
	}
}

func (s *service) ReplicateProcessVersion(ctx context.Context, processVersionID int64, operatorID int64) (int64, error) {
	if processVersionID <= 0 || operatorID <= 0 {
		return 0, domain.ErrInvalidArgument
	}

	db := s.conn.ConnectPgxWrite
	if db == nil {
		return 0, fmt.Errorf("pgx write connection is not initialized")
	}

	var newVersionID int64
	err := db.
		QueryRow(ctx, `SELECT replicate_process_version($1, $2)`, processVersionID, operatorID).
		Scan(&newVersionID)
	if err != nil {
		return 0, mapPgxError(err)
	}

	return newVersionID, nil
}

func (s *service) PromoteProcessVersion(ctx context.Context, processVersionID int64, operatorID int64, comment string) error {
	if processVersionID <= 0 || operatorID <= 0 {
		return domain.ErrInvalidArgument
	}

	db := s.conn.ConnectPgxWrite
	if db == nil {
		return fmt.Errorf("pgx write connection is not initialized")
	}

	_, err := db.Exec(ctx, `SELECT promote_process_version($1, $2, $3)`, processVersionID, operatorID, comment)
	if err != nil {
		return mapPgxError(err)
	}
	return nil
}

func (s *service) ResolveProcessVersion(ctx context.Context, processTypeID int64, sedeID int64, overrideProcessVersionID *int64) (int64, []Step, error) {
	if processTypeID <= 0 {
		return 0, nil, domain.ErrInvalidArgument
	}

	db := s.conn.ConnectPgxWrite
	if db == nil {
		return 0, nil, fmt.Errorf("pgx write connection is not initialized")
	}

	var resolvedID int64
	var stepsJSON []byte

	if overrideProcessVersionID != nil {
		err := db.
			QueryRow(ctx, `SELECT process_version_id, process_steps FROM resolve_process_version($1, $2, $3)`, processTypeID, sedeID, *overrideProcessVersionID).
			Scan(&resolvedID, &stepsJSON)
		if err != nil {
			return 0, nil, mapPgxError(err)
		}
	} else {
		err := db.
			QueryRow(ctx, `SELECT process_version_id, process_steps FROM resolve_process_version($1, $2, NULL)`, processTypeID, sedeID).
			Scan(&resolvedID, &stepsJSON)
		if err != nil {
			return 0, nil, mapPgxError(err)
		}
	}

	var steps []Step
	if len(stepsJSON) == 0 {
		steps = []Step{}
	} else {
		if err := json.Unmarshal(stepsJSON, &steps); err != nil {
			return 0, nil, domain.ErrInternal
		}
	}

	return resolvedID, steps, nil
}

func (s *service) MoveProcessVersionToTest(ctx context.Context, processVersionID int64) error {
	if processVersionID <= 0 {
		return domain.ErrInvalidArgument
	}

	db := s.conn.ConnectPgxWrite
	if db == nil {
		return fmt.Errorf("pgx write connection is not initialized")
	}

	_, err := db.Exec(ctx, `SELECT move_process_version_to_test($1)`, processVersionID)
	if err != nil {
		return mapPgxError(err)
	}

	return nil
}

func (s *service) ListProcessVersions(ctx context.Context, req dtos.PaginationRequest) (*dtos.PaginationResponse[models.ProcessVersionListItem], error) {
	db := s.conn.ConnectGormRead
	if db == nil {
		return nil, fmt.Errorf("gorm read connection is not initialized")
	}

	if req.Page <= 0 {
		req.Page = 1
	}
	if req.RowsPerPage <= 0 {
		req.RowsPerPage = 15
	}

	base := db.WithContext(ctx).
		Table("process_versions pv").
		Joins("JOIN process_types pt ON pt.id = pv.process_type_id").
		Joins("LEFT JOIN users u ON u.id = pv.operator_id").
		Joins(`LEFT JOIN (
			SELECT
				h.process_version_id,
				h.promoted_at AS valid_from,
				LEAD(h.promoted_at) OVER (
					PARTITION BY v.process_type_id, v.sede_id
					ORDER BY h.promoted_at
				) AS valid_to
			FROM process_version_history h
			JOIN process_versions v ON v.id = h.process_version_id
		) hv ON hv.process_version_id = pv.id`).
		Where("pt.is_visible = ? AND pt.archived_at IS NULL AND pv.archived_at IS NULL", true)

	for i, f := range req.FilterBy {
		if i >= len(req.FilterValues) {
			continue
		}

		val := req.FilterValues[i]

		switch f {
		case "status":
			base = base.Where("pv.status = ?", val)
		case "sede_id":
			base = base.Where("pv.sede_id = ?", val)
		case "process_type_id":
			base = base.Where("pv.process_type_id = ?", val)
		case "name", "process_type_name":
			base = base.Where("pt.name ILIKE ?", fmt.Sprintf("%%%v%%", val))
		case "operator_email":
			base = base.Where("u.email ILIKE ?", fmt.Sprintf("%%%v%%", val))
		}
	}

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, err
	}

	if total == 0 {
		return &dtos.PaginationResponse[models.ProcessVersionListItem]{
			Data:        []models.ProcessVersionListItem{},
			TotalRows:   0,
			TotalPages:  0,
			Page:        req.Page,
			RowsPerPage: req.RowsPerPage,
			Extras:      map[string]any{},
		}, nil
	}

	if len(req.SortBy) > 0 {
		for i, field := range req.SortBy {
			desc := false
			if i < len(req.SortDesc) {
				desc = req.SortDesc[i]
			}

			column := ""
			switch field {
			case "id":
				column = "pv.id"
			case "version_number":
				column = "pv.version_number"
			case "status":
				column = "pv.status"
			case "sede_id":
				column = "pv.sede_id"
			case "name", "process_type_name":
				column = "pt.name"
			case "operator_email":
				column = "u.email"
			default:
				continue
			}

			if desc {
				base = base.Order(column + " DESC")
			} else {
				base = base.Order(column + " ASC")
			}
		}
	} else {
		base = base.Order("pv.id DESC")
	}

	offset := (req.Page - 1) * req.RowsPerPage

	var rows []models.ProcessVersionRow
	if err := base.
		Select(`
			pv.id AS id,
			pt.name AS process_type_name,
			pt.is_visible AS process_type_is_visible,
			pv.version_number,
			pv.sede_id,
			pv.status,
			u.email AS operator_email,
			hv.valid_from,
			hv.valid_to`,
		).
		Limit(req.RowsPerPage).
		Offset(offset).
		Scan(&rows).Error; err != nil {
		return nil, err
	}

	items := make([]models.ProcessVersionListItem, 0, len(rows))
	for _, r := range rows {
		item := models.ProcessVersionListItem{
			ID:                   r.ID,
			ProcessTypeName:      r.ProcessTypeName,
			ProcessTypeIsVisible: r.ProcessTypeIsVisible,
			VersionNumber:        r.VersionNumber,
			SedeID:               r.SedeID,
			Status:               r.Status,
			OperatorEmail:        r.OperatorEmail,
			ValidFrom:            r.ValidFrom,
			ValidTo:              r.ValidTo,
		}
		items = append(items, item)
	}

	totalPages := 0
	if req.RowsPerPage > 0 {
		totalPages = int((total + int64(req.RowsPerPage) - 1) / int64(req.RowsPerPage))
	}

	return &dtos.PaginationResponse[models.ProcessVersionListItem]{
		Data:        items,
		TotalRows:   total,
		TotalPages:  totalPages,
		Page:        req.Page,
		RowsPerPage: req.RowsPerPage,
		Extras:      map[string]any{},
	}, nil
}

func mapPgxError(err error) error {
	if err == nil {
		return nil
	}

	msg := err.Error()

	switch {
	case strings.Contains(msg, "Process version not found or archived"):
		return domain.ErrNotFound
	case strings.Contains(msg, "Process type does not exist or is archived"):
		return domain.ErrNotFound
	case strings.Contains(msg, "No active version found"):
		return domain.ErrNotFound
	case strings.Contains(msg, "Override version invalid"):
		return domain.ErrInvalidArgument
	case strings.Contains(msg, "Only DRAFT versions can be moved to TEST"):
		return domain.ErrInvalidArgument
	case strings.Contains(msg, "Cannot promote version without steps"),
		strings.Contains(msg, "Promotion comment exceeds 300 characters"),
		strings.Contains(msg, "Only TEST or HISTORY versions can be promoted to PROD"):
		return domain.ErrInvalidArgument
	default:
		return domain.ErrInternal
	}
}
