package processlifecyclemanager

import (
	"errors"

	"go-fiber-core/internal/domain"
	"gorm.io/gorm"
)

func mapGormError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return domain.ErrNotFound
	}
	return err
}

