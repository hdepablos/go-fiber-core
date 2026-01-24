package models

import (
	"time"

	"github.com/google/uuid"
)

// Session representa la sesión de un usuario logueado.
// Permite controlar accesos y revocar permisos inmediatamente.
type Session struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;"` // ID único de la sesión
	UserID    uint64    `gorm:"not null;index"`         // Usuario dueño de la sesión
	UserAgent string    `gorm:"type:varchar(512)"`      // Navegador/Dispositivo
	ClientIP  string    `gorm:"type:varchar(45)"`       // IP del cliente
	IsBlocked bool      `gorm:"default:false;not null"` // Si la sesión fue invalidada manualmente
	ExpiresAt time.Time `gorm:"not null"`               // Cuándo expira esta sesión (usualmente == refresh token expiration)
	CreatedAt time.Time
	UpdatedAt time.Time

	User User `gorm:"foreignKey:UserID"`
}

// TableName especifica el nombre de la tabla en la base de datos
func (Session) TableName() string {
	return "sessions"
}
