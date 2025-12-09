package bank

import (
	"context"
	"go-fiber-core/internal/dtos/connect"
	"go-fiber-core/internal/models"
	bankRepo "go-fiber-core/internal/repositories/bank"
)

type BankReaderService interface {
	GetByID(ctx context.Context, id uint) (*models.Bank, error)
	GetAll(ctx context.Context) ([]models.Bank, error)
}

type bankReaderService struct {
	conn       *connect.ConnectDTO
	bankReader bankRepo.BankReader
}

func NewBankReaderService(conn *connect.ConnectDTO, reader bankRepo.BankReader) BankReaderService {
	return &bankReaderService{
		conn:       conn,
		bankReader: reader,
	}
}

func (s *bankReaderService) GetByID(ctx context.Context, id uint) (*models.Bank, error) {
	return s.bankReader.GetByID(ctx, s.conn.ConnectGormRead, id)
}

func (s *bankReaderService) GetAll(ctx context.Context) ([]models.Bank, error) {
	return s.bankReader.GetAll(ctx, s.conn.ConnectGormRead)
}
