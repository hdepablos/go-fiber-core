package processlifecyclemanager

import (
	"context"
	"encoding/json"
	"fmt"

	"go-fiber-core/internal/domain"
	"go-fiber-core/internal/dtos/connect"

	"gorm.io/gorm"
)

type Step struct {
	Name         string          `json:"name"`
	ExecutionKey string          `json:"execution_key"`
	Config       json.RawMessage `json:"config"`
	StepOrder    int32           `json:"step_order"`
}

type ResolveService interface {
	ResolveProcessVersion(ctx context.Context, processTypeID int64, sedeID int64, overrideProcessVersionID *int64, roadmap int) (int64, []Step, error)
}

type resolveService struct {
	conn *connect.ConnectDTO
}

func NewResolveService(conn *connect.ConnectDTO) ResolveService {
	return &resolveService{conn: conn}
}

func (s *resolveService) ResolveProcessVersion(ctx context.Context, processTypeID int64, sedeID int64, overrideProcessVersionID *int64, roadmap int) (int64, []Step, error) {
	if processTypeID <= 0 {
		return 0, nil, domain.ErrInvalidArgument
	}

	db := s.conn.ConnectGormRead
	if db == nil {
		return 0, nil, fmt.Errorf("gorm read connection is not initialized")
	}

	var pt ProcessType
	if err := db.WithContext(ctx).
		Where("id = ? AND archived_at IS NULL", processTypeID).
		First(&pt).Error; err != nil {
		return 0, nil, mapGormError(err)
	}

	var resolved ProcessVersion
	if overrideProcessVersionID != nil {
		if err := db.WithContext(ctx).
			Where("id = ? AND process_type_id = ? AND archived_at IS NULL", *overrideProcessVersionID, processTypeID).
			First(&resolved).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return 0, nil, domain.ErrOverrideVersionNotFound
			}
			return 0, nil, err
		}
	} else {
		err := db.WithContext(ctx).
			Where("process_type_id = ? AND status = ? AND sede_id = ? AND archived_at IS NULL", processTypeID, "PROD", sedeID).
			Limit(1).
			First(&resolved).Error
		if err != nil {
			if err != gorm.ErrRecordNotFound {
				return 0, nil, err
			}

			err = db.WithContext(ctx).
				Where("process_type_id = ? AND status = ? AND sede_id IS NULL AND archived_at IS NULL", processTypeID, "PROD").
				Limit(1).
				First(&resolved).Error
			if err != nil {
				return 0, nil, mapGormError(err)
			}
		}
	}

	if resolved.ID <= 0 {
		return 0, nil, domain.ErrNotFound
	}

	var steps []Step
	if err := db.WithContext(ctx).
		Table("process_steps").
		Select("name, execution_key, COALESCE(config, '{}'::jsonb) AS config, step_order").
		Where("process_version_id = ? AND roadmap = ?", resolved.ID, roadmap).
		Order("step_order ASC").
		Scan(&steps).Error; err != nil {
		return 0, nil, err
	}
	if steps == nil {
		steps = []Step{}
	}

	return resolved.ID, steps, nil
}

