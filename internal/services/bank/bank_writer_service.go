// internal/services/bank/bank_writer_service.go
package bank

import (
	"context"
	"go-fiber-core/internal/dtos/connect"
	"go-fiber-core/internal/models"
	bankRepo "go-fiber-core/internal/repositories/bank"
)

type BankWriterService interface {
	Create(ctx context.Context, bank *models.Bank) error
	Update(ctx context.Context, id uint, updatedBankData *models.Bank) (*models.Bank, error)
	SoftDelete(ctx context.Context, id uint) error
	HardDelete(ctx context.Context, id uint) error
}

type bankWriterService struct {
	conn       *connect.ConnectDTO
	bankWriter bankRepo.BankWriter
	bankReader bankRepo.BankReader
}

func NewBankWriterService(
	conn *connect.ConnectDTO,
	writer bankRepo.BankWriter,
	reader bankRepo.BankReader,
) BankWriterService {
	return &bankWriterService{
		conn:       conn,
		bankWriter: writer,
		bankReader: reader,
	}
}

func (s *bankWriterService) Create(ctx context.Context, bank *models.Bank) error {
	return s.bankWriter.Create(ctx, s.conn.ConnectGormWrite, bank)
}

func (s *bankWriterService) Update(ctx context.Context, id uint, updatedBankData *models.Bank) (*models.Bank, error) {
	existingBank, err := s.bankReader.GetByID(ctx, s.conn.ConnectGormWrite, id)
	if err != nil {
		return nil, err
	}

	existingBank.Name = updatedBankData.Name
	existingBank.EntityCode = updatedBankData.EntityCode
	existingBank.Enabled = updatedBankData.Enabled

	if err := s.bankWriter.Update(ctx, s.conn.ConnectGormWrite, existingBank); err != nil {
		return nil, err
	}
	return existingBank, nil
}

func (s *bankWriterService) SoftDelete(ctx context.Context, id uint) error {
	return s.bankWriter.SoftDelete(ctx, s.conn.ConnectGormWrite, id)
}

func (s *bankWriterService) HardDelete(ctx context.Context, id uint) error {
	return s.bankWriter.HardDelete(ctx, s.conn.ConnectGormWrite, id)
}
