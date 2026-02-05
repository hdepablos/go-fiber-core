package menu_user

import (
	"context"
	"go-fiber-core/internal/dtos"
	"go-fiber-core/internal/dtos/connect"
	"go-fiber-core/internal/dtos/responses"
	"go-fiber-core/internal/repositories/menu_user"
)

type MenuUserPaginationService interface {
	GetAllPaginated(ctx context.Context, req dtos.PaginationRequest) (*dtos.PaginationResponse[responses.MenuUserResponse], error)
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
