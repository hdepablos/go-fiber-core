package menu_user

import (
	"context"
	"go-fiber-core/internal/dtos"
	"go-fiber-core/internal/models"
	"gorm.io/gorm"
)

type MenuUserPagination interface {
	GetAllPaginated(ctx context.Context,  db *gorm.DB, req dtos.PaginationRequest) (*dtos.PaginationResponse[models.MenuUser], error)
}