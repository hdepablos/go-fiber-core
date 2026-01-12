package menu_user

import (
	"context"
	"go-fiber-core/internal/dtos"
	
	"go-fiber-core/internal/models"
	"go-fiber-core/internal/services/pagination"
	"gorm.io/gorm"
)

type menuUserPaginationRepository struct {
	ps *pagination.PaginationService[models.MenuUser]
}

func NewMenuUserPaginationRepository(ps *pagination.PaginationService[models.MenuUser]) MenuUserPagination {
	return &menuUserPaginationRepository{ps: ps}
}

func (r *menuUserPaginationRepository) GetAllPaginated(ctx context.Context,  db *gorm.DB, req dtos.PaginationRequest) (*dtos.PaginationResponse[models.MenuUser], error) {
	return r.ps.Execute(db.WithContext(ctx), req, nil, nil)
}