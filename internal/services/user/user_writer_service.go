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
	CreateWithRole(ctx context.Context, user *models.User, roleIDs []uint64) error
	RemoveRolesFromUsers(ctx context.Context, userIDs []uint64, roleIDs []uint64) error
	AssignRolesToUsers(ctx context.Context, userIDs []uint64, roleIDs []uint64) error
	Update(ctx context.Context, id uint64, data UpdateUserDTO) (*models.User, error)
	Activate(ctx context.Context, id uint64, operatorID uint64) error
	Deactivate(ctx context.Context, id uint64, operatorID uint64) error
	SetActiveBulk(ctx context.Context, ids []uint64, active bool, operatorID uint64) error 
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

	return s.conn.ConnectGormWrite.WithContext(ctx).Transaction(func(tx *gorm.DB) error {

		var user models.User
		if err := tx.First(&user, id).Error; err != nil {
			return err
		}

		// ❌ ANTES (esto te rompía todo)
		// tx.Model(&user).Association("Roles").Clear()
		// tx.Model(&user).Association("Menus").Clear()
		// tx.Delete(&user)

		// ✅ AHORA: solo soft lógico
		if err := tx.Model(&user).
			Update("is_active", false).Error; err != nil {
			return err
		}

		return nil
	})
}



func (s *userWriterService) HardDelete(
	ctx context.Context,
	id uint,
) error {

	return s.conn.ConnectGormWrite.WithContext(ctx).Transaction(func(tx *gorm.DB) error {

		var user models.User
		if err := tx.Unscoped().First(&user, id).Error; err != nil {
			return err
		}

		// 1️⃣ Limpiar pivotes
		if err := tx.Model(&user).Association("Roles").Clear(); err != nil {
			return err
		}
		if err := tx.Model(&user).Association("Menus").Clear(); err != nil {
			return err
		}

		// 2️⃣ Hard delete
		if err := tx.Unscoped().Delete(&user).Error; err != nil {
			return err
		}

		return nil
	})
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

func (s *userWriterService) CreateWithRole(ctx context.Context, user *models.User,roleIDs []uint64) error {

	return s.conn.ConnectGormWrite.WithContext(ctx).Transaction(func(tx *gorm.DB) error {

		// 1️⃣ Hash password
		hashedPassword, err := bcrypt.GenerateFromPassword(
			[]byte(user.Password),
			bcrypt.DefaultCost,
		)
		if err != nil {
			return err
		}

		user.Password = string(hashedPassword)
		user.IsActive = true

		// 2️⃣ Crear usuario
		if err := s.userWriter.Create(ctx, tx, user); err != nil {
			return err
		}

		// 3️⃣ Obtener roles
		var roles []models.Role
		if err := tx.Where("id IN ?", roleIDs).Find(&roles).Error; err != nil {
			return err
		}
		if len(roles) != len(roleIDs) {
			return gorm.ErrRecordNotFound
		}

		// 4️⃣ Asignar roles al usuario
		if err := tx.Model(user).Association("Roles").Append(&roles); err != nil {
			return err
		}

		// 5️⃣ Obtener menu_ids desde menu_role
		var menuRoles []models.MenuRole
		if err := tx.
			Where("role_id IN ? AND is_active = true", roleIDs).
			Find(&menuRoles).Error; err != nil {
			return err
		}

		if len(menuRoles) == 0 {
			return nil // el rol no tiene menús, válido
		}

		// 6️⃣ Deduplicar menu IDs
		menuIDMap := make(map[uint]struct{})
		for _, mr := range menuRoles {
			menuIDMap[mr.MenuID] = struct{}{}
		}

		menuIDs := make([]uint, 0, len(menuIDMap))
		for id := range menuIDMap {
			menuIDs = append(menuIDs, id)
		}

		// 7️⃣ Obtener menús
		var menus []models.Menu
		if err := tx.Where("id IN ?", menuIDs).Find(&menus).Error; err != nil {
			return err
		}

		// 8️⃣ Asignar menús al usuario
		if err := tx.Model(user).Association("Menus").Replace(&menus); err != nil {
			return err
		}

		return nil
	})
}


func (s *userWriterService) RemoveRolesFromUsers(
	ctx context.Context,
	userIDs []uint64,
	roleIDs []uint64,
) error {

	return s.conn.ConnectGormWrite.WithContext(ctx).Transaction(func(tx *gorm.DB) error {

		var users []models.User
		if err := tx.Where("id IN ?", userIDs).Find(&users).Error; err != nil {
			return err
		}

		var rolesToRemove []models.Role
		if err := tx.Where("id IN ?", roleIDs).Find(&rolesToRemove).Error; err != nil {
			return err
		}

		for _, user := range users {

			// 1️⃣ Quitar roles
			if err := tx.Model(&user).Association("Roles").Delete(&rolesToRemove); err != nil {
				return err
			}

			// 2️⃣ Roles restantes
			var remainingRoles []models.Role
			if err := tx.Model(&user).Association("Roles").Find(&remainingRoles); err != nil {
				return err
			}

			if len(remainingRoles) == 0 {
				// 🔥 Sin roles → sin menús
				if err := tx.Model(&user).Association("Menus").Clear(); err != nil {
					return err
				}
				continue
			}

			// 3️⃣ Obtener menus desde menu_role
			roleIDs := make([]uint, 0, len(remainingRoles))
			for _, r := range remainingRoles {
				roleIDs = append(roleIDs, uint(r.ID))
			}

			var menuRoles []models.MenuRole
			if err := tx.
				Where("role_id IN ? AND is_active = true", roleIDs).
				Find(&menuRoles).Error; err != nil {
				return err
			}

			menuIDMap := make(map[uint]struct{})
			for _, mr := range menuRoles {
				menuIDMap[mr.MenuID] = struct{}{}
			}

			menuIDs := make([]uint, 0, len(menuIDMap))
			for id := range menuIDMap {
				menuIDs = append(menuIDs, id)
			}

			var menus []models.Menu
			if len(menuIDs) > 0 {
				if err := tx.Where("id IN ?", menuIDs).Find(&menus).Error; err != nil {
					return err
				}
			}

			// 4️⃣ Reemplazar menús
			if err := tx.Model(&user).Association("Menus").Replace(&menus); err != nil {
				return err
			}
		}

		return nil
	})
}

func (s *userWriterService) AssignRolesToUsers(ctx context.Context, userIDs []uint64,roleIDs []uint64) error {

	return s.conn.ConnectGormWrite.WithContext(ctx).Transaction(func(tx *gorm.DB) error {

		// 1️⃣ Validar usuarios
		var users []models.User
		if err := tx.Where("id IN ?", userIDs).Find(&users).Error; err != nil {
			return err
		}
		if len(users) != len(userIDs) {
			return gorm.ErrRecordNotFound
		}

		// 2️⃣ Validar roles
		var roles []models.Role
		if err := tx.Where("id IN ?", roleIDs).Find(&roles).Error; err != nil {
			return err
		}
		if len(roles) != len(roleIDs) {
			return gorm.ErrRecordNotFound
		}

		// 3️⃣ Procesar usuario por usuario
		for _, user := range users {

			// 🔹 Roles actuales
			var currentRoles []models.Role
			if err := tx.Model(&user).Association("Roles").Find(&currentRoles); err != nil {
				return err
			}

			// 🔹 Map para toggle
			roleMap := make(map[uint]models.Role)

			for _, r := range currentRoles {
				roleMap[uint(r.ID)] = r
			}

			// 🔥 TOGGLE
			for _, r := range roles {
				if _, exists := roleMap[uint(r.ID)]; exists {
					delete(roleMap, uint(r.ID)) // estaba → se quita
				} else {
					roleMap[uint(r.ID)] = r // no estaba → se agrega
				}
			}

			// 🔹 Roles finales
			finalRoles := make([]models.Role, 0, len(roleMap))
			for _, r := range roleMap {
				finalRoles = append(finalRoles, r)
			}

			// 4️⃣ Reemplazar roles
			if err := tx.Model(&user).Association("Roles").Replace(&finalRoles); err != nil {
				return err
			}

			// 5️⃣ Recalcular menús
			if len(finalRoles) == 0 {
				// sin roles → sin menús
				if err := tx.Model(&user).Association("Menus").Clear(); err != nil {
					return err
				}
				continue
			}

			roleIDs := make([]uint, 0, len(finalRoles))
			for _, r := range finalRoles {
				roleIDs = append(roleIDs, uint(r.ID))
			}

			var menuRoles []models.MenuRole
			if err := tx.
				Where("role_id IN ? AND is_active = true", roleIDs).
				Find(&menuRoles).Error; err != nil {
				return err
			}

			menuIDMap := make(map[uint]struct{})
			for _, mr := range menuRoles {
				menuIDMap[mr.MenuID] = struct{}{}
			}

			menuIDs := make([]uint, 0, len(menuIDMap))
			for id := range menuIDMap {
				menuIDs = append(menuIDs, id)
			}

			var menus []models.Menu
			if len(menuIDs) > 0 {
				if err := tx.Where("id IN ?", menuIDs).Find(&menus).Error; err != nil {
					return err
				}
			}

			if err := tx.Model(&user).Association("Menus").Replace(&menus); err != nil {
				return err
			}
		}

		return nil
	})
}

func (s *userWriterService) Activate(ctx context.Context, id uint64, operatorID uint64) error {
	return s.conn.ConnectGormWrite.
		WithContext(ctx).
		Model(&models.User{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"is_active":   true,
			"operator_id": operatorID, // <-- guardamos quién activó
		}).Error
}

func (s *userWriterService) Deactivate(ctx context.Context, id uint64, operatorID uint64) error {
	return s.conn.ConnectGormWrite.
		WithContext(ctx).
		Model(&models.User{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"is_active":   false,
			"operator_id": operatorID, // <-- guardamos quién desactivó
		}).Error
}

func (s *userWriterService) SetActiveBulk(ctx context.Context, ids []uint64, active bool, operatorID uint64) error {
	if len(ids) == 0 {
		return nil // o return error si quieres que sea obligatorio
	}

	return s.conn.ConnectGormWrite.
		WithContext(ctx).
		Model(&models.User{}).
		Where("id IN ?", ids).
		Updates(map[string]interface{}{
			"is_active":   active,
			"operator_id": operatorID, // guardamos quién hizo la acción
		}).Error
}
