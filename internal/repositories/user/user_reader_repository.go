// internal/repositories/user/user_repository.go
package user

import (
	"context"
	"fmt"
	"go-fiber-core/internal/dtos"
	"go-fiber-core/internal/models"
	"go-fiber-core/internal/services/pagination"

	"gorm.io/gorm"
)

// --- INTERFACES SEGREGADAS POR ROL ---

type UserReader interface {
	GetByID(ctx context.Context, db *gorm.DB, id uint64) (*models.User, error)
	GetByEmail(ctx context.Context, db *gorm.DB, email string) (*models.User, error)
	GetByEmailWithRolesAndMenus(ctx context.Context, db *gorm.DB, email string) (*models.User, error)
	GetByEmailWithRoles(ctx context.Context, db *gorm.DB, email string) (*models.User, error)
	GetAll(ctx context.Context, db *gorm.DB) ([]models.User, error)
}

type UserWriter interface {
	Create(ctx context.Context, db *gorm.DB, user *models.User) error
	Update(ctx context.Context, db *gorm.DB, user *models.User) error
	SoftDelete(ctx context.Context, db *gorm.DB, id uint64) error
	HardDelete(ctx context.Context, db *gorm.DB, id uint) error
}

type UserPaginator interface {
	GetAllPaginated(ctx context.Context, db *gorm.DB, req dtos.PaginationRequest) (*dtos.PaginationResponse[models.User], error)
}

// --- INTERFAZ COMPUESTA (Para la API) ---
type UserRepository interface {
	UserReader
	UserWriter
	UserPaginator
}

// --- STRUCTS Y CONSTRUCTORES GRANULARES ---

// UserReaderRepo se encarga solo de la lectura. No tiene dependencias.
type UserReaderRepo struct{}

func NewUserReaderRepo() UserReader { return &UserReaderRepo{} }

// UserWriterRepo se encarga solo de la escritura. No tiene dependencias.
type UserWriterRepo struct{}

func NewUserWriterRepo() UserWriter { return &UserWriterRepo{} }

// UserPaginatorRepo se encarga solo de la paginación. SÍ tiene una dependencia.
type UserPaginatorRepo struct {
	ps *pagination.PaginationService[models.User]
}

func NewUserPaginatorRepo(ps *pagination.PaginationService[models.User]) UserPaginator {
	return &UserPaginatorRepo{ps: ps}
}

// --- STRUCT COMPUESTO (Para la API) ---

// userRepository ahora COMPONE las otras piezas.
type userRepository struct {
	UserReader
	UserWriter
	UserPaginator
}

// NewUserRepository es el constructor para el repositorio completo.
func NewUserRepository(r UserReader, w UserWriter, p UserPaginator) UserRepository {
	return &userRepository{r, w, p}
}

// --- IMPLEMENTACIONES LIGADAS A CADA STRUCT GRANULAR ---

// Métodos para UserWriterRepo
func (r *UserWriterRepo) Create(ctx context.Context, db *gorm.DB, user *models.User) error {
	return db.WithContext(ctx).Create(user).Error
}

func (r *UserWriterRepo) Update(ctx context.Context, db *gorm.DB, user *models.User) error {
	return db.WithContext(ctx).Save(user).Error
}

func (r *UserWriterRepo) SoftDelete(ctx context.Context, db *gorm.DB, id uint64) error {
	return db.WithContext(ctx).Delete(&models.User{}, id).Error
}

func (r *UserWriterRepo) HardDelete(ctx context.Context, db *gorm.DB, id uint) error {
	return db.WithContext(ctx).Unscoped().Delete(&models.User{}, id).Error
}

// Métodos para UserReaderRepo

// GetByID obtiene un usuario por ID con sus relaciones básicas
func (r *UserReaderRepo) GetByID(ctx context.Context, db *gorm.DB, id uint64) (*models.User, error) {
	var user models.User
	err := db.WithContext(ctx).
		Preload("Products").
		Preload("Roles").
		First(&user, id).Error
	return &user, err
}

// GetByEmail obtiene un usuario por email con sus roles
// Este método es usado para autenticación rápida (sin cargar menús)
func (r *UserReaderRepo) GetByEmail(ctx context.Context, db *gorm.DB, email string) (*models.User, error) {
	var user models.User
	err := db.WithContext(ctx).
		Preload("Roles").
		Where("email = ?", email).
		First(&user).Error

	fmt.Println("User found by email:", user.Roles) // Debug
	return &user, err
}

// GetByEmailWithRolesAndMenus carga el usuario junto con Roles y Menús
// IMPORTANTE: Los menús se cargan desde menu_user (permisos por usuario),
// NO desde menu_role (que ya no existe).
func (r *UserReaderRepo) GetByEmailWithRolesAndMenus(ctx context.Context, db *gorm.DB, email string) (*models.User, error) {
	var user models.User

	err := db.WithContext(ctx).
		Preload("Roles"). // ✅ Carga roles desde role_user
		Preload("Menus", func(db *gorm.DB) *gorm.DB {
			return db.
				Where("is_active = ?", true). // ✅ Solo menús activos
				Order("order_index ASC")      // ✅ Ordenados correctamente
		}). // ✅ Carga menús desde menu_user
		Where("email = ?", email).
		First(&user).Error

	if err != nil {
		return nil, err
	}

	fmt.Printf("User found: %s, Roles: %v, Menus: %d\n", user.Email, len(user.Roles), len(user.Menus))
	return &user, nil
}

// GetAll obtiene todos los usuarios con sus relaciones básicas
func (r *UserReaderRepo) GetAll(ctx context.Context, db *gorm.DB) ([]models.User, error) {
	var users []models.User
	err := db.WithContext(ctx).
		Preload("Roles").
		Find(&users).Error
	return users, err
}

// Métodos para UserPaginatorRepo
func (r *UserPaginatorRepo) GetAllPaginated(ctx context.Context, db *gorm.DB, req dtos.PaginationRequest) (*dtos.PaginationResponse[models.User], error) {
	return r.ps.Execute(db.WithContext(ctx), req, nil, nil)
}

func (r *UserReaderRepo) GetByEmailWithRoles(ctx context.Context, db *gorm.DB, email string) (*models.User, error) {
	var user models.User

	// NOTA: Usamos Preload("Roles") para cargar la relación Many-to-Many
	err := db.WithContext(ctx).
		Preload("Roles"). // Asegura que la relación Roles se cargue
		Where("email = ?", email).
		Where("deleted_at IS NULL").
		First(&user).Error

	if err != nil {
		return nil, err
	}

	return &user, nil
}
