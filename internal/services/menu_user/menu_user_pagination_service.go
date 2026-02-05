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
	GetMenusByUser(ctx context.Context, userID uint, req dtos.PaginationRequest) (*dtos.PaginationResponse[models.MenuUserResponse], error)
	GetMenusNotByUser(ctx context.Context, userID uint, req dtos.PaginationRequest) (*dtos.PaginationResponse[models.MenuUser], error)
	GetMenuAssignmentStatus(ctx context.Context, userID uint) (total uint, assigned uint, err error)
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

func (s *menuUserPaginationService) GetMenusByUser(
	ctx context.Context,
	userID uint,
	req dtos.PaginationRequest,
) (*dtos.PaginationResponse[models.MenuUserResponse], error){
	return s.paginator.GetMenusByUser(ctx, s.conn.ConnectGormRead, userID, req)
}

func (s *menuUserPaginationService) GetMenusNotByUser(
	ctx context.Context,
	userID uint,
	req dtos.PaginationRequest,
) (*dtos.PaginationResponse[models.MenuUser], error) {

	return s.paginator.GetMenusNotByUser(ctx, s.conn.ConnectGormRead, userID, req)
}


func (s *menuUserPaginationService) GetMenuAssignmentStatus(ctx context.Context, userID uint) (total uint, assigned uint, err error) {
	// Contar todos los menús
	allResp, err := s.paginator.GetAllPaginated(ctx, s.conn.ConnectGormRead, dtos.PaginationRequest{
		Page:        1,
		RowsPerPage: 1, // solo necesitamos que retorne TotalRows
	})
	if err != nil {
		return 0, 0, err
	}
	total = uint(allResp.TotalRows) // <-- TotalRows en lugar de Total

	// Contar menus asignados al usuario
	assignedResp, err := s.paginator.GetMenusByUser(ctx, s.conn.ConnectGormRead, userID, dtos.PaginationRequest{
		Page:        1,
		RowsPerPage: 1,
	})
	if err != nil {
		return total, 0, err
	}
	assigned = uint(assignedResp.TotalRows)

	return total, assigned, nil
}
