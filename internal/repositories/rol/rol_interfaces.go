package rol

import (
	"context"
	"go-fiber-core/internal/dtos"
	"go-fiber-core/internal/models"

	"gorm.io/gorm"
)

type RolReader interface {
	GetByID(ctx context.Context, db *gorm.DB, id uint) (*models.Role, error)
	GetAll(ctx context.Context, db *gorm.DB) ([]models.Role, error)
	// --- NUEVO MÉTODO AÑADIDO ---
	// GetByRange obtiene todos los roles cuyos IDs están dentro del rango especificado.
}

type RolWriter interface {
	Create(ctx context.Context, db *gorm.DB, role *models.Role) error
	Update(ctx context.Context, db *gorm.DB, role *models.Role) error
	SoftDelete(ctx context.Context, db *gorm.DB, id uint) error
	HardDelete(ctx context.Context, db *gorm.DB, id uint) error
}

type RolPagination interface {
	GetAllPaginated(ctx context.Context, db *gorm.DB, req dtos.PaginationRequest) (*dtos.PaginationResponse[models.Role], error)
}

type RolCrudRepository interface {
	RolReader
	RolWriter
}
