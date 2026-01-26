package menu_user

import (
	"context"
	"go-fiber-core/internal/dtos"
	"go-fiber-core/internal/dtos/connect"
	"go-fiber-core/internal/models"
	"go-fiber-core/internal/repositories/menu_user"
)

type MenuUserPaginationService interface {
	GetAllPaginated(ctx context.Context, req dtos.PaginationRequest) (*dtos.PaginationResponse[models.MenuUser], error)
}

type menuUserPaginationService struct {
	conn      *connect.ConnectDTO
	paginator menu_user.MenuUserPagination
}

func NewMenuUserPaginationService(
	conn *connect.ConnectDTO,
	paginator menu_user.MenuUserPagination,
) MenuUserPaginationService {
	return &menuUserPaginationService{
		conn:      conn,
		paginator: paginator,
	}
}

func (s *menuUserPaginationService) GetAllPaginated(ctx context.Context, req dtos.PaginationRequest) (*dtos.PaginationResponse[models.MenuUser], error) {
	return s.paginator.GetAllPaginated(ctx, s.conn.ConnectGormRead, req)
}