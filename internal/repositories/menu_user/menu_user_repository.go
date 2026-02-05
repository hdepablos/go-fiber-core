package menu_user

import (
	"context"
	"go-fiber-core/internal/dtos"

	"go-fiber-core/internal/models"
	"go-fiber-core/internal/services/pagination"

	"gorm.io/gorm"
)

type MenuUserPaginationRepository struct {
	ps *pagination.PaginationService[models.MenuUser]
}

func NewMenuUserPaginationRepository(ps *pagination.PaginationService[models.MenuUser]) MenuUserPagination {
	return &MenuUserPaginationRepository{ps: ps}
}

func (r *MenuUserPaginationRepository) GetAllPaginated(
	ctx context.Context,
	db *gorm.DB,
	req dtos.PaginationRequest,
) (*dtos.PaginationResponse[models.MenuUser], error) {

	return r.ps.Execute(
		db.WithContext(ctx).
			Select("menu_user.id, menu_user.menu_id, menu_user.user_id, menu_user.operator_id, menu_user.is_active, menu_user.created_at, menu_user.updated_at, menu_user.deleted_at").
			Joins("LEFT JOIN menus ON menus.id = menu_user.menu_id").
			Joins("LEFT JOIN users ON users.id = menu_user.user_id").
			Joins("LEFT JOIN users AS operators ON operators.id = menu_user.operator_id"),
		req,
		func(q *gorm.DB) *gorm.DB {
			return q.
				Preload("Menu").
				Preload("User").
				Preload("Operator", func(db *gorm.DB) *gorm.DB {
					return db.Unscoped()
				})
		},
		nil,
	)
}
