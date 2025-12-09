package models

import (
	"time"
)

// RefreshToken almacena los tokens de refresco para poder invalidarlos.
type RefreshToken struct {
	ID        uint      `gorm:"primarykey"`
	UserID    uint64    `gorm:"not null;index"` // Coincide con el tipo de ID del usuario
	Token     string    `gorm:"type:varchar(512);unique;not null"`
	ExpiresAt time.Time `gorm:"not null"`

	User      User `gorm:"foreignKey:UserID"`
	CreatedAt time.Time
}

// TableName especifica el nombre de la tabla en la base de datos
func (RefreshToken) TableName() string {
	return "refresh_tokens"
}
