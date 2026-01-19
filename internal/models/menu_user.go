package models

import (
	"time"

	"gorm.io/gorm"
)

type MenuUser struct {
	ID uint `json:"id" gorm:"primaryKey"`

	MenuID uint `json:"menu_id"`
	UserID uint `json:"user_id"`

	Menu Menu `gorm:"foreignKey:MenuID"`
	User User `gorm:"foreignKey:UserID"`

	IsActive  bool           `json:"is_active"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"deleted_at" gorm:"index"`
}



func (MenuUser) TableName() string {
	return "menu_user"
}
