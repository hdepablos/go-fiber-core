package models

import "time"

type BulkJobOutputStatus string

const (
	BulkJobOutputStatusPending   BulkJobOutputStatus = "pending"
	BulkJobOutputStatusGenerated BulkJobOutputStatus = "generated"
	BulkJobOutputStatusFailed    BulkJobOutputStatus = "failed"
)

type BulkJobOutput struct {
	ID        int64               `gorm:"primaryKey"`
	BulkJobID int64               `gorm:"column:bulk_job_id;not null;index"`
	Type      string              `gorm:"column:type;type:varchar(50);not null;index"`
	FilePath  string              `gorm:"column:file_path;type:text;not null"`
	FileSize  *int64              `gorm:"column:file_size"`
	Status    BulkJobOutputStatus `gorm:"column:status;type:varchar(20);not null;default:pending;index"`
	Metadata  []byte              `gorm:"column:metadata;type:jsonb"`
	CreatedAt time.Time           `gorm:"column:created_at;not null"`
	UpdatedAt time.Time           `gorm:"column:updated_at;not null"`
}

func (BulkJobOutput) TableName() string { return "bulk_job_outputs" }
