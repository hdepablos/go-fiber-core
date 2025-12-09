package bank

import (
	"context"
	"go-fiber-core/internal/dtos"
	"go-fiber-core/internal/dtos/connect"
	"go-fiber-core/internal/models"
	"go-fiber-core/internal/repositories/bank"
)

type BankPaginationService interface {
	GetAllPaginated(ctx context.Context, req dtos.PaginationRequest) (*dtos.PaginationResponse[models.Bank], error)
}

type bankPaginationService struct {
	conn      *connect.ConnectDTO
	paginator bank.BankPagination
}

func NewBankPaginationService(
	conn *connect.ConnectDTO,
	paginator bank.BankPagination,
) BankPaginationService {
	return &bankPaginationService{
		conn:      conn,
		paginator: paginator,
	}
}

func (s *bankPaginationService) GetAllPaginated(ctx context.Context, req dtos.PaginationRequest) (*dtos.PaginationResponse[models.Bank], error) {
	return s.paginator.GetAllPaginated(ctx, s.conn.ConnectGormRead, req)
}
