package rol

import (
	"context"
	"go-fiber-core/internal/dtos"
	"go-fiber-core/internal/models"
	"go-fiber-core/internal/services/pagination"
	"fmt"
	"gorm.io/gorm"
)

// --- INTERFACES, STRUCTS Y CONSTRUCTORES (sin cambios) ---
type RolReaderRepo struct{}

func NewRolReaderRepo() RolReader { return &RolReaderRepo{} }

type RolWriterRepo struct{}
func NewRolWriterRepo() RolWriter { return &RolWriterRepo{} }

type RolPaginationRepo struct {
	ps *pagination.PaginationService[models.Role]
}

func NewRolPaginationRepo(ps *pagination.PaginationService[models.Role]) RolPagination {
	return &RolPaginationRepo{ps: ps}
}

type rolCrudRepository struct {
	RolReader
	RolWriter
}

func NewRolCrudRepository(r RolReader, w RolWriter) *rolCrudRepository {
	return &rolCrudRepository{r, w}
}

// --- MÉTODOS ---

// Writer (sin cambios)
func (r *RolWriterRepo) Create(ctx context.Context, db *gorm.DB, role *models.Role) error {
	return db.WithContext(ctx).Create(role).Error
}

func (r *RolWriterRepo) Update(ctx context.Context, db *gorm.DB, role *models.Role) error {
	return db.WithContext(ctx).Save(role).Error
}

func (r *RolWriterRepo) SoftDelete(ctx context.Context, db *gorm.DB, id uint) error {
	fmt.Println("Entrando a SoftDelete del RolRepository")
	return db.WithContext(ctx).Delete(&models.Role{}, id).Error
}

func (r *RolWriterRepo) HardDelete(ctx context.Context, db *gorm.DB, id uint) error {
	return db.WithContext(ctx).Unscoped().Delete(&models.Role{}, id).Error
}

// Reader
func (r *RolReaderRepo) GetByID(ctx context.Context, db *gorm.DB, id uint) (*models.Role, error) {
	var role models.Role
	err := db.WithContext(ctx).First(&role, id).Error
	return &role, err
}

func (r *RolReaderRepo) GetAll(ctx context.Context, db *gorm.DB) ([]models.Role, error) {
	var roles []models.Role
	err := db.WithContext(ctx).Find(&roles).Error
	return roles, err
}

// Pagination (sin cambios)
func (r *RolPaginationRepo) GetAllPaginated(ctx context.Context, db *gorm.DB, req dtos.PaginationRequest) (*dtos.PaginationResponse[models.Role], error) {
	return r.ps.Execute(db.WithContext(ctx), req, nil, nil)
}
