package processlifecyclemanager

import (
	"context"
	"fmt"

	"go-fiber-core/internal/domain"
	"go-fiber-core/internal/dtos/connect"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type MoveToTestService interface {
	MoveProcessVersionToTest(ctx context.Context, processVersionID int64) error
}

type moveToTestService struct {
	conn *connect.ConnectDTO
}

func NewMoveToTestService(conn *connect.ConnectDTO) MoveToTestService {
	return &moveToTestService{conn: conn}
}

func (s *moveToTestService) MoveProcessVersionToTest(ctx context.Context, processVersionID int64) error {
	if processVersionID <= 0 {
		return domain.ErrInvalidArgument
	}

	db := s.conn.ConnectGormWrite
	if db == nil {
		return fmt.Errorf("gorm write connection is not initialized")
	}

	fmt.Println("Va a mover la version de proceso con ID", processVersionID, "a TEST con grom ...")
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var pv ProcessVersion
		if err := tx.
			Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND archived_at IS NULL", processVersionID).
			First(&pv).Error; err != nil {
			return mapGormError(err)
		}

		if pv.Status != "DRAFT" {
			return domain.ErrInvalidArgument
		}

		return tx.Model(&ProcessVersion{}).
			Where("id = ?", pv.ID).
			Updates(map[string]any{
				"status":     "TEST",
				"updated_at": gorm.Expr("NOW()"),
			}).Error
	})
}
