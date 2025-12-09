// internal/repositories/bank/bank_interfaces.go
package bank

import (
	"context"
	"go-fiber-core/internal/dtos"
	"go-fiber-core/internal/models"

	"gorm.io/gorm"
)

type BankReader interface {
	GetByID(ctx context.Context, db *gorm.DB, id uint) (*models.Bank, error)
	GetAll(ctx context.Context, db *gorm.DB) ([]models.Bank, error)
	// --- NUEVO MÉTODO AÑADIDO ---
	// GetByRange obtiene todos los bancos cuyos IDs están dentro del rango especificado.
	GetByRange(ctx context.Context, db *gorm.DB, startID uint, endID uint) ([]models.Bank, error)
}

type BankWriter interface {
	Create(ctx context.Context, db *gorm.DB, bank *models.Bank) error
	Update(ctx context.Context, db *gorm.DB, bank *models.Bank) error
	SoftDelete(ctx context.Context, db *gorm.DB, id uint) error
	HardDelete(ctx context.Context, db *gorm.DB, id uint) error
}

type BankPagination interface {
	GetAllPaginated(ctx context.Context, db *gorm.DB, req dtos.PaginationRequest) (*dtos.PaginationResponse[models.Bank], error)
}
