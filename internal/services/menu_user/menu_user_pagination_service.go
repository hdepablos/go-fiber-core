package menu_user

import (
	"context"
	"go-fiber-core/internal/dtos"
	"go-fiber-core/internal/dtos/connect"
	"go-fiber-core/internal/dtos/responses"
	"go-fiber-core/internal/repositories/menu_user"
	"go-fiber-core/internal/models"
)

type MenuUserPaginationService interface {
	GetAllPaginated(ctx context.Context, req dtos.PaginationRequest) (*dtos.PaginationResponse[responses.MenuUserResponse], error)
	GetMenusByUser(ctx context.Context, userID uint, req dtos.PaginationRequest) (*dtos.PaginationResponse[models.MenuUserResponse], error)
	GetMenusNotByUser(ctx context.Context, userID uint, req dtos.PaginationRequest) (*dtos.PaginationResponse[models.MenuUserResponse], error)
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

func (s *menuUserPaginationService) GetAllPaginated(ctx context.Context, req dtos.PaginationRequest) (*dtos.PaginationResponse[responses.MenuUserResponse], error) {
	result, err := s.paginator.GetAllPaginated(ctx, s.conn.ConnectGormRead, req)
	if err != nil {
		return nil, err
	}

	// Mapeo manual de models.MenuUser a responses.MenuUserResponse
	dtosData := make([]responses.MenuUserResponse, len(result.Data))
	for i, item := range result.Data {
		dtosData[i] = responses.MenuUserResponse{
			ID:         item.ID,
			MenuID:     item.MenuID,
			UserID:     item.UserID,
			OperatorID: item.OperatorID,
			IsActive:   item.IsActive,
			CreatedAt:  item.CreatedAt,
			UpdatedAt:  item.UpdatedAt,
		}

		if item.Menu.ID != 0 {
			dtosData[i].Menu = &responses.MenuSimpleResponse{
				ID:       item.Menu.ID,
				Type:     item.Menu.ItemType,
				Text:     item.Menu.ItemName,
				To:       item.Menu.ToPath,
				Icon:     item.Menu.Icon,
				IsActive: item.Menu.IsActive,
			}
		}

		if item.User.ID != 0 {
			dtosData[i].User = &responses.UserSimpleResponse{
				ID:       item.User.ID,
				Name:     item.User.Name,
				Email:    item.User.Email,
				IsActive: item.User.IsActive,
			}
		}

		if item.Operator != nil && item.Operator.ID != 0 {
			dtosData[i].Operator = &responses.UserSimpleResponse{
				ID:       item.Operator.ID,
				Name:     item.Operator.Name,
				Email:    item.Operator.Email,
				IsActive: item.Operator.IsActive,
			}
		}
	}

	return &dtos.PaginationResponse[responses.MenuUserResponse]{
		Data:        dtosData,
		TotalRows:   result.TotalRows,
		TotalPages:  result.TotalPages,
		Page:        result.Page,
		RowsPerPage: result.RowsPerPage,
		Extras:      result.Extras,
	}, nil
}

func (s *menuUserPaginationService) GetMenusByUser(ctx context.Context,userID uint,req dtos.PaginationRequest) (*dtos.PaginationResponse[models.MenuUserResponse], error){
	return s.paginator.GetMenusByUser(ctx, s.conn.ConnectGormRead, userID, req)
}

func (s *menuUserPaginationService) GetMenusNotByUser(ctx context.Context, userID uint, req dtos.PaginationRequest) (*dtos.PaginationResponse[models.MenuUserResponse], error) {
	return s.paginator.GetMenusNotByUser(ctx, s.conn.ConnectGormRead, userID, req)
}



func (s *menuUserPaginationService) GetMenuAssignmentStatus(
	ctx context.Context,
	userID uint,
) (total uint, assigned uint, err error) {

	req := dtos.PaginationRequest{
		Page:        1,
		RowsPerPage: 1, // solo necesitamos TotalRows
	}

	// =============================
	// Asignados
	// =============================
	assignedResp, err := s.paginator.GetMenusByUser(
		ctx,
		s.conn.ConnectGormRead,
		userID,
		req,
	)
	if err != nil {
		return 0, 0, err
	}

	// =============================
	// No asignados
	// =============================
	notAssignedResp, err := s.paginator.GetMenusNotByUser(
		ctx,
		s.conn.ConnectGormRead,
		userID,
		req,
	)
	if err != nil {
		return 0, 0, err
	}

	assigned = uint(assignedResp.TotalRows)
	notAssigned := uint(notAssignedResp.TotalRows)

	total = assigned + notAssigned

	return total, assigned, nil
}
