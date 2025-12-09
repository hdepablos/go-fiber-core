package models

import (
	"time"

	"gorm.io/gorm"
)

type MenuUser struct {
	MenuID uint `json:"menu_id" gorm:"primaryKey;column:menu_id"`
	UserID uint `json:"user_id" gorm:"primaryKey;column:user_id"`

	// Relaciones opcionales si querés acceder al detalle
	Menu      Menu           `gorm:"foreignKey:MenuID;references:ID" json:"-"`
	User      User           `gorm:"foreignKey:UserID;references:ID" json:"-"`
	IsActive  bool           `gorm:"not null;default:true" json:"is_active"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (MenuUser) TableName() string {
	return "menu_user"
}
