// internal/repositories/bank/bank_repository.go
package bank

import (
	"context"
	"go-fiber-core/internal/dtos"
	"go-fiber-core/internal/models"
	"go-fiber-core/internal/services/pagination"

	"gorm.io/gorm"
)

// --- INTERFACES, STRUCTS Y CONSTRUCTORES (sin cambios) ---
type BankReaderRepo struct{}

func NewBankReaderRepo() BankReader { return &BankReaderRepo{} }

type BankWriterRepo struct{}

func NewBankWriterRepo() BankWriter { return &BankWriterRepo{} }

type BankPaginationRepo struct {
	ps *pagination.PaginationService[models.Bank]
}

func NewBankPaginationRepo(ps *pagination.PaginationService[models.Bank]) BankPagination {
	return &BankPaginationRepo{ps: ps}
}

type bankCrudRepository struct {
	BankReader
	BankWriter
}

func NewBankCrudRepository(r BankReader, w BankWriter) *bankCrudRepository {
	return &bankCrudRepository{r, w}
}

// --- MÉTODOS ---

// Writer (sin cambios)
func (r *BankWriterRepo) Create(ctx context.Context, db *gorm.DB, bank *models.Bank) error {
	return db.WithContext(ctx).Create(bank).Error
}

func (r *BankWriterRepo) Update(ctx context.Context, db *gorm.DB, bank *models.Bank) error {
	return db.WithContext(ctx).Save(bank).Error
}

func (r *BankWriterRepo) SoftDelete(ctx context.Context, db *gorm.DB, id uint) error {
	return db.WithContext(ctx).Delete(&models.Bank{}, id).Error
}

func (r *BankWriterRepo) HardDelete(ctx context.Context, db *gorm.DB, id uint) error {
	return db.WithContext(ctx).Unscoped().Delete(&models.Bank{}, id).Error
}

// Reader
func (r *BankReaderRepo) GetByID(ctx context.Context, db *gorm.DB, id uint) (*models.Bank, error) {
	var bank models.Bank
	err := db.WithContext(ctx).First(&bank, id).Error
	return &bank, err
}

func (r *BankReaderRepo) GetAll(ctx context.Context, db *gorm.DB) ([]models.Bank, error) {
	var banks []models.Bank
	err := db.WithContext(ctx).Find(&banks).Error
	return banks, err
}

// --- IMPLEMENTACIÓN DEL NUEVO MÉTODO ---
func (r *BankReaderRepo) GetByRange(ctx context.Context, db *gorm.DB, startID uint, endID uint) ([]models.Bank, error) {
	var banks []models.Bank
	// Usamos una cláusula Where para filtrar por el rango de IDs
	err := db.WithContext(ctx).Where("id >= ? AND id <= ?", startID, endID).Find(&banks).Error
	return banks, err
}

// Pagination (sin cambios)
func (r *BankPaginationRepo) GetAllPaginated(ctx context.Context, db *gorm.DB, req dtos.PaginationRequest) (*dtos.PaginationResponse[models.Bank], error) {
	return r.ps.Execute(db.WithContext(ctx), req, nil, nil)
}
