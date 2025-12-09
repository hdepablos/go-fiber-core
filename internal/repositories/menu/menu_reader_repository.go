package menu

import (
	"context"
	"errors"
	"go-fiber-core/internal/dtos/connect" // Necesario para obtener la conexión DB
	"go-fiber-core/internal/models"

	"gorm.io/gorm"
)

// MenuReaderRepository Interface (simplemente usa MenuReader)
// type MenuReaderRepository interface {
//     GetMenusByUserID(ctx context.Context, db *gorm.DB, userID uint64) ([]models.Menu, error)
// }

type menuReaderRepository struct {
	db *gorm.DB
}

// NewMenuReaderRepository es el constructor que Wire busca.
// Asumimos que recibe el ConnectDTO y utiliza la conexión de lectura.
func NewMenuReaderRepository(conn *connect.ConnectDTO) MenuReader {
	return &menuReaderRepository{db: conn.ConnectGormRead}
}

// GetMenusByUserID implementa la lógica SQL para obtener la lista plana de menús
// (ítems asignados y sus padres), filtrando por 'is_active' en la tabla pivote.
func (r *menuReaderRepository) GetMenusByUserID(ctx context.Context, db *gorm.DB, userID uint64) ([]models.Menu, error) {
	// Usamos el DB inyectado en la struct, no el pasado como argumento (si se inyecta con ConnectDTO)
	db = r.db.WithContext(ctx)

	// 1️⃣ SUB-CONSULTA: Obtener IDs de menús DIRECTAMENTE asignados y ACTIVOS
	userMenuIDs := db.
		Table("menu_user").
		Select("menu_id").
		Where("user_id = ?", userID).
		Where("is_active = ?", true)

	// 2️⃣ SUB-CONSULTA: Obtener IDs de los padres de esos menús activos
	parentIDs := db.
		Table("menus").
		Select("parent_id").
		Where("id IN (?)", userMenuIDs).
		Where("parent_id IS NOT NULL")

	// 3️⃣ CONSULTA FINAL: Obtener menús que son: A) Asignados/Activos O B) Padres necesarios
	var menus []models.Menu
	err := db.
		Table("menus").
		Where("id IN (?) OR id IN (?)", userMenuIDs, parentIDs).
		Where("menus.is_active = ?", true).
		Where("menus.deleted_at IS NULL").
		Order("menus.order_index ASC").
		Find(&menus).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return []models.Menu{}, nil
		}
		return nil, err
	}

	// Si no se encuentra nada, devolver lista vacía en lugar de un error.
	if len(menus) == 0 {
		return []models.Menu{}, nil
	}

	return menus, nil
}
