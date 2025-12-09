package menu

import (
	"context"
	"go-fiber-core/internal/models"

	"gorm.io/gorm"
)

// type MenuWriter interface {
// 	Create(ctx context.Context, db *gorm.DB, menu *models.Menu) error
// 	SoftDelete(ctx context.Context, db *gorm.DB, id uint) error
// 	AddRoles(ctx context.Context, db *gorm.DB, menuID uint, roleIDs []uint64) error
// 	RemoveRoles(ctx context.Context, db *gorm.DB, menuID uint, roleIDs []uint64) error
// 	BulkAddRoles(ctx context.Context, db *gorm.DB, menuIDs []uint64, roleIDs []uint64) error
// 	BulkRemoveRoles(ctx context.Context, db *gorm.DB, menuIDs []uint64, roleIDs []uint64) error
// }

type MenuReader interface {
	GetMenusByUserID(ctx context.Context, db *gorm.DB, userID uint64) ([]models.Menu, error)
}
