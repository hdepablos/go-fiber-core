package processlifecyclemanager

import (
	"context"
	"errors"
	"fmt"

	"go-fiber-core/internal/domain"
	"go-fiber-core/internal/dtos/connect"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type PromoteService interface {
	PromoteProcessVersion(ctx context.Context, processVersionID int64, operatorID int64, comment string) error
}

type promoteService struct {
	conn *connect.ConnectDTO
}

func NewPromoteService(conn *connect.ConnectDTO) PromoteService {
	return &promoteService{conn: conn}
}

func (s *promoteService) PromoteProcessVersion(ctx context.Context, processVersionID int64, operatorID int64, comment string) error {
	if processVersionID <= 0 || operatorID <= 0 {
		return domain.ErrInvalidArgument
	}
	if len(comment) > 300 {
		return domain.ErrInvalidArgument
	}

	db := s.conn.ConnectGormWrite
	if db == nil {
		return fmt.Errorf("gorm write connection is not initialized")
	}

	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var pv ProcessVersion
		if err := tx.
			Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND archived_at IS NULL", processVersionID).
			First(&pv).Error; err != nil {
			return mapGormError(err)
		}

		switch pv.Status {
		case "TEST", "HISTORY":
		default:
			return domain.ErrInvalidArgument
		}

		var stepCount int64
		if err := tx.Model(&ProcessStep{}).
			Where("process_version_id = ?", processVersionID).
			Count(&stepCount).Error; err != nil {
			return err
		}
		if stepCount == 0 {
			return domain.ErrInvalidArgument
		}

		var currentProd ProcessVersion
		err := tx.
			Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("process_type_id = ? AND status = ? AND archived_at IS NULL", pv.ProcessTypeID, "PROD").
			Where("sede_id IS NOT DISTINCT FROM ?", pv.SedeID).
			First(&currentProd).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		if err == nil && currentProd.ID > 0 {
			if err := tx.
				Model(&ProcessVersion{}).
				Where("id = ?", currentProd.ID).
				Updates(map[string]any{
					"status":     "HISTORY",
					"updated_at": gorm.Expr("NOW()"),
				}).Error; err != nil {
				return err
			}
		}

		if err := tx.
			Model(&ProcessVersion{}).
			Where("id = ?", pv.ID).
			Updates(map[string]any{
				"status":     "PROD",
				"updated_at": gorm.Expr("NOW()"),
			}).Error; err != nil {
			return err
		}

		h := ProcessVersionHistory{
			ProcessVersionID:   pv.ID,
			ProcessTypeID:      pv.ProcessTypeID,
			PromotedFromStatus: pv.Status,
			PromotedBy:         operatorID,
			Comment:            comment,
		}
		if err := tx.Create(&h).Error; err != nil {
			return err
		}

		return nil
	})
}

