package menu

import (
	"errors"
	"fmt"
	"go-fiber-core/internal/dtos/connect"
	"go-fiber-core/internal/models"

	"context"

	"gorm.io/gorm"
	// "gorm.io/gorm/clause"
)

type menuWriterRepository struct {
	db *gorm.DB
}

func NewMenuWriterRepository(conn *connect.ConnectDTO) MenuWriter {
	return &menuWriterRepository{db: conn.ConnectGormWrite}
}

func (r *menuWriterRepository) AddBulkUsers(
	ctx context.Context,
	db *gorm.DB,
	menuIDs []uint64,
	userIDs []uint64,
) error {


	for _, mid := range menuIDs {
		for _, uid := range userIDs {
			var existing models.MenuUser

			fmt.Print("repositorio")
			err := db.WithContext(ctx).
				Unscoped().
				Where("menu_id = ? AND user_id = ?", mid, uid).
				First(&existing).Error

			fmt.Print("paso 1")
			// NO existe → crear
			if errors.Is(err, gorm.ErrRecordNotFound) {
				if err := db.Create(&models.MenuUser{
					MenuID:   uint(mid),
					UserID:   uid,
					IsActive: true,
				}).Error; err != nil {
					fmt.Print("Error de crear")
					fmt.Print(err)
					return err
				}
				continue
			}

			fmt.Print("paso 2")

			// existe soft deleted → revivir
			if existing.DeletedAt.Valid {
				if err := db.Model(&existing).
					Unscoped().
					Update("deleted_at", nil).Error; err != nil {
					fmt.Print("Error de soft")
					fmt.Print(err)
					return err
				}
			}

			fmt.Print("paso 3")
		}
	}

	return nil
}

func (r *menuWriterRepository) BulkRemoveUsers(
	ctx context.Context,
	db *gorm.DB,
	menuIDs []uint64,
	userIDs []uint64,
) error {

	return db.WithContext(ctx).
		Where("menu_id IN ? AND user_id IN ?", menuIDs, userIDs).
		Delete(&models.MenuUser{}).
		Error
}
