package user

import (
	"context"
	"go-fiber-core/internal/dtos/connect"
	"go-fiber-core/internal/models"
	userRepo "go-fiber-core/internal/repositories/user"
	"go-fiber-core/internal/services"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type UpdateUserDTO struct {
	Name     *string
	Email    *string
	Password *string
	IsActive *bool
}

type UserWriterService interface {
	Create(ctx context.Context, user *models.User) error
	Update(ctx context.Context, id uint64, data UpdateUserDTO) (*models.User, error)
	SoftDelete(ctx context.Context, id uint64) error
	HardDelete(ctx context.Context, id uint) error
}

type userWriterService struct {
	services.TransactionManager
	conn       connect.ConnectDTO
	userWriter userRepo.UserWriter
	userReader userRepo.UserReader
}

func NewUserWriterService(
	conn *connect.ConnectDTO,
	writer userRepo.UserWriter,
	reader userRepo.UserReader,
) UserWriterService {
	return &userWriterService{
		TransactionManager: services.NewTransactionManager(conn),
		conn:               *conn,
		userWriter:         writer,
		userReader:         reader,
	}
}

func (s *userWriterService) Create(ctx context.Context, user *models.User) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	user.Password = string(hashedPassword)
	user.IsActive = true
	return s.userWriter.Create(ctx, s.conn.ConnectGormWrite, user)
}

func (s *userWriterService) Update(ctx context.Context, id uint64, data UpdateUserDTO) (*models.User, error) {
	existingUser, err := s.userReader.GetByID(ctx, s.conn.ConnectGormWrite, id)
	if err != nil {
		return nil, err
	}
	if data.Name != nil {
		existingUser.Name = *data.Name
	}
	if data.Email != nil {
		existingUser.Email = *data.Email
	}
	if data.IsActive != nil {
		existingUser.IsActive = *data.IsActive
	}
	if data.Password != nil && *data.Password != "" {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(*data.Password), bcrypt.DefaultCost)
		if err != nil {
			return nil, err
		}
		existingUser.Password = string(hashedPassword)
	}
	if err := s.userWriter.Update(ctx, s.conn.ConnectGormWrite, existingUser); err != nil {
		return nil, err
	}
	return existingUser, nil
}

func (s *userWriterService) SoftDelete(ctx context.Context, id uint64) error {
	return s.userWriter.SoftDelete(ctx, s.conn.ConnectGormWrite, id)
}

func (s *userWriterService) HardDelete(ctx context.Context, id uint) error {
	return s.userWriter.HardDelete(ctx, s.conn.ConnectGormWrite, id)
}
func (s *userWriterService) CreateWithProductsAndRoles(ctx context.Context, user *models.User, roleIDs []uint64) error {
	db := s.conn.ConnectGormWrite // ✅ usa Conn (viene del TransactionManager)

	// Hash de contraseña
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	user.Password = string(hashedPassword)
	user.IsActive = true

	// Transacción: usuario + productos + roles
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Crear usuario (y sus productos si vienen en user.Products)
		if err := s.userWriter.Create(ctx, tx, user); err != nil { // ✅ usa userWriter
			return err
		}

		// Asociar roles existentes
		if len(roleIDs) > 0 {
			var roles []models.Role
			if err := tx.Find(&roles, roleIDs).Error; err != nil {
				return err
			}
			if err := tx.Model(user).Association("Roles").Replace(roles); err != nil {
				return err
			}
		}

		return nil
	})
}
