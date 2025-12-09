// internal/services/bank/deactivation_service.go
package bank

import (
	"context"
	"go-fiber-core/internal/dtos/connect"
	bankRepo "go-fiber-core/internal/repositories/bank"
	"go-fiber-core/internal/services"

	"gorm.io/gorm"
)

type DeactivationService interface {
	DeactivateBanksWithPendingDebts(ctx context.Context) error
}

type deactivationService struct {
	services.TransactionManager
	bankReader bankRepo.BankReader
	bankWriter bankRepo.BankWriter
}

func NewDeactivationService(
	conn *connect.ConnectDTO,
	reader bankRepo.BankReader,
	writer bankRepo.BankWriter,
) DeactivationService {
	return &deactivationService{
		TransactionManager: services.NewTransactionManager(conn),
		bankReader:         reader,
		bankWriter:         writer,
	}
}

func (s *deactivationService) DeactivateBanksWithPendingDebts(ctx context.Context) error {
	return s.TransactionManager.ExecuteTx(ctx, func(tx *gorm.DB) error {
		allBanks, err := s.bankReader.GetAll(ctx, tx)
		if err != nil {
			return err
		}

		for _, bank := range allBanks {
			isIndebted := true // Simulación de lógica de negocio
			if isIndebted {
				bank.Enabled = false
				if err := s.bankWriter.Update(ctx, tx, &bank); err != nil {
					return err
				}
			}
		}
		return nil
	})
}
