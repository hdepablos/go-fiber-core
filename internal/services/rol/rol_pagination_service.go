package rol

import (
	"context"
	"go-fiber-core/internal/dtos"
	"go-fiber-core/internal/dtos/connect"
	"go-fiber-core/internal/models"
	"go-fiber-core/internal/repositories/rol"
)

type RolPaginationService interface {
	GetAllPaginated(ctx context.Context, req dtos.PaginationRequest) (*dtos.PaginationResponse[models.Role], error)
}

type rolPaginationService struct {
	conn      *connect.ConnectDTO
	paginator rol.RolPagination
}

func NewRolPaginationService(
	conn *connect.ConnectDTO,
	paginator rol.RolPagination,
) RolPaginationService {
	return &rolPaginationService{
		conn:      conn,
		paginator: paginator,
	}
}

func (s *rolPaginationService) GetAllPaginated(ctx context.Context, req dtos.PaginationRequest) (*dtos.PaginationResponse[models.Role], error) {
	return s.paginator.GetAllPaginated(ctx, s.conn.ConnectGormRead, req)
}


