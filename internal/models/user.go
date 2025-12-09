package models

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	ID       uint64 `gorm:"primaryKey;autoIncrement" json:"id"`
	Name     string `gorm:"type:varchar(100);not null" json:"name"`
	Email    string `gorm:"type:varchar(255);unique;not null" json:"email"`
	Password string `gorm:"type:text;not null" json:"-"`
	IsActive bool   `gorm:"not null;default:true" json:"is_active"`

	// Relación con roles (many-to-many a través de role_user)
	Roles []Role `gorm:"many2many:role_user;joinForeignKey:UserID;joinReferences:RoleID" json:"roles,omitempty"`

	// Relación con menús (many-to-many a través de menu_user)
	Menus []Menu `gorm:"many2many:menu_user;joinForeignKey:UserID;joinReferences:MenuID" json:"menus,omitempty"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (User) TableName() string {
	return "users"
}
