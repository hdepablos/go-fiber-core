package menu

import (
	"go-fiber-core/internal/dtos/connect"
	"go-fiber-core/internal/models"

	"context"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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

	var relations []models.MenuUser

	for _, mid := range menuIDs {
		for _, uid := range userIDs {
			relations = append(relations, models.MenuUser{
				MenuID:   uint(mid),
				UserID:   uint(uid),
				IsActive: true,
			})
		}
	}

	return db.WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}). // evita duplicados
		Create(&relations).Error
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
