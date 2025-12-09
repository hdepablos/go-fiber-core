package models

import (
	"time"

	"gorm.io/gorm"
)

type Menu struct {
	ID         uint    `json:"id" gorm:"primaryKey"`
	ItemType   string  `json:"type" gorm:"column:item_type"`
	ItemName   string  `json:"text" gorm:"column:item_name"`
	ToPath     *string `json:"to,omitempty" gorm:"column:to_path"`
	Icon       *string `json:"icon,omitempty"`
	ParentID   *uint   `json:"parent_id"`
	OrderIndex int     `json:"order_index"`
	IsActive   bool    `json:"is_active"`

	// Relación con usuarios (many-to-many a través de menu_user)
	Users []User `json:"users,omitempty" gorm:"many2many:menu_user;joinForeignKey:MenuID;joinReferences:UserID"`

	// Relación jerárquica (padres e hijos)
	Children []Menu `json:"children,omitempty" gorm:"foreignKey:ParentID"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (Menu) TableName() string {
	return "menus"
}
