package models

import (
	"time"

	"gorm.io/gorm"
)

type MenuRole struct {
	ID uint `json:"id" gorm:"primaryKey"`

	MenuID uint `json:"menu_id"`
	RoleID uint `json:"role_id"`
	Menu Menu `gorm:"foreignKey:MenuID"`
	Role Role `gorm:"foreignKey:RoleID"`
	OperatorID *uint `json:"operator_id"`
	Operator   *User `gorm:"foreignKey:OperatorID"`

	IsActive  bool           `json:"is_active"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"deleted_at" gorm:"index"`
}



func (MenuRole) TableName() string {
	return "menu_role"
}
