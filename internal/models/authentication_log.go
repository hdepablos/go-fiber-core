package models

import "time"

type AuthenticationLog struct {
	ID              int64      `gorm:"primaryKey"`
	UserID          *uint64    `gorm:"index"`
	EmailSnapshot   *string    `gorm:"type:varchar(255)"`
	EventType       string     `gorm:"type:varchar(32);not null;index"`
	FailureReason   *string    `gorm:"type:varchar(64)"`
	IPAddress       string     `gorm:"type:varchar(45);not null;index"`
	UserAgent       string     `gorm:"type:text;not null"`
	Browser         string     `gorm:"type:varchar(64);not null"`
	OperatingSystem string     `gorm:"type:varchar(64);not null"`
	DeviceType      string     `gorm:"type:varchar(16);not null"`
	Country         *string    `gorm:"type:varchar(64)"`
	City            *string    `gorm:"type:varchar(64)"`
	RequestID       *string    `gorm:"type:varchar(64)"`
	Origin          *string    `gorm:"type:varchar(32)"`
	CreatedAt       time.Time  `gorm:"not null"`
}

func (AuthenticationLog) TableName() string {
	return "authentication_logs"
}
