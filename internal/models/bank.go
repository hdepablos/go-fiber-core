package models

import (
	"time"

	"gorm.io/gorm"
)

func (Bank) TableName() string {
	return "banks"
}

type Bank struct {
	ID         uint64         `gorm:"primaryKey" json:"id"`
	Name       string         `gorm:"size:255;not null" json:"name" validate:"required,min=3,max=255"`
	EntityCode string         `gorm:"size:50;not null;unique" json:"entity_code" validate:"required,alphanum,max=50"`
	Enabled    bool           `gorm:"default:true" json:"enabled"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-" filter:"date"`
	CreatedAt  time.Time      `gorm:"index" json:"created_at" filter:"date"`
	UpdatedAt  time.Time      `json:"updated_at" filter:"date"`
}
