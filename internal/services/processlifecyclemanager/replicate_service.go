package processlifecyclemanager

import (
	"context"
	"fmt"

	"go-fiber-core/internal/domain"
	"go-fiber-core/internal/dtos/connect"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ReplicateService interface {
	ReplicateProcessVersion(ctx context.Context, processVersionID int64, operatorID int64) (int64, error)
}

type replicateService struct {
	conn *connect.ConnectDTO
}

func NewReplicateService(conn *connect.ConnectDTO) ReplicateService {
	return &replicateService{conn: conn}
}

func (s *replicateService) ReplicateProcessVersion(ctx context.Context, processVersionID int64, operatorID int64) (int64, error) {
	if processVersionID <= 0 || operatorID <= 0 {
		return 0, domain.ErrInvalidArgument
	}

	db := s.conn.ConnectGormWrite
	if db == nil {
		return 0, fmt.Errorf("gorm write connection is not initialized")
	}

	fmt.Println("Va a replicar con grom ...")
	var newID int64
	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var src ProcessVersion
		if err := tx.
			Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND archived_at IS NULL", processVersionID).
			First(&src).Error; err != nil {
			return mapGormError(err)
		}

		if err := tx.Exec("SELECT pg_advisory_xact_lock(?)", src.ProcessTypeID).Error; err != nil {
			return err
		}

		var nextVersionNumber int
		if err := tx.
			Model(&ProcessVersion{}).
			Where("process_type_id = ?", src.ProcessTypeID).
			Select("COALESCE(MAX(version_number), 0) + 1").
			Scan(&nextVersionNumber).Error; err != nil {
			return err
		}

		newVersion := ProcessVersion{
			ProcessTypeID: src.ProcessTypeID,
			VersionNumber: nextVersionNumber,
			SedeID:        src.SedeID,
			Status:        "DRAFT",
			OperatorID:    operatorID,
		}
		if err := tx.Create(&newVersion).Error; err != nil {
			return err
		}

		var steps []ProcessStep
		if err := tx.
			Where("process_version_id = ?", src.ID).
			Order("step_order ASC").
			Find(&steps).Error; err != nil {
			return err
		}

		if len(steps) > 0 {
			newSteps := make([]ProcessStep, 0, len(steps))
			for _, st := range steps {
				newSteps = append(newSteps, ProcessStep{
					ProcessVersionID: newVersion.ID,
					StepOrder:        st.StepOrder,
					Roadmap:          st.Roadmap,
					Name:             st.Name,
					ExecutionKey:     st.ExecutionKey,
					Config:           st.Config,
				})
			}

			if err := tx.Create(&newSteps).Error; err != nil {
				return err
			}
		}

		newID = newVersion.ID
		return nil
	})
	if err != nil {
		return 0, err
	}

	if newID <= 0 {
		return 0, domain.ErrInternal
	}
	return newID, nil
}
