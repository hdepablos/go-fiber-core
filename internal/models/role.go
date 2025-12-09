package models

import (
	"time"

	"gorm.io/gorm"
)

type Role struct {
	ID       uint64 `gorm:"primaryKey;autoIncrement" json:"id"`
	Name     string `gorm:"type:varchar(100);unique;not null" json:"name"`
	IsActive bool   `gorm:"not null;default:true" json:"is_active"`

	// Relación con usuarios (many-to-many a través de role_user)
	Users []User `gorm:"many2many:role_user;joinForeignKey:RoleID;joinReferences:UserID" json:"users,omitempty"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (Role) TableName() string {
	return "roles"
}
