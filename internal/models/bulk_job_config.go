package models

import "time"

type BulkJobConfig struct {
	ID         int64      `gorm:"primaryKey"`
	OperatorID uint64     `gorm:"not null;index"`
	RefCode     string    `gorm:"column:ref_code;type:varchar(255);not null;index"`
	IsActive   bool       `gorm:"not null;default:true"`
	Config     []byte     `gorm:"type:jsonb;not null"`
	ArchivedAt *time.Time `gorm:""`
	CreatedAt  time.Time  `gorm:"not null"`
	UpdatedAt  time.Time  `gorm:"not null"`
}

func (BulkJobConfig) TableName() string { return "bulk_job_configs" }
