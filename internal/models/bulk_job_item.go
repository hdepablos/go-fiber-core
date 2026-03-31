package models

import "time"

type BulkJobItem struct {
	ID                int64         `gorm:"primaryKey"`
	BulkJobID          int64         `gorm:"not null;index"`
	RowNumber          int           `gorm:"not null"`
	ReferenceKey       string        `gorm:"type:varchar(255);not null;index"`
	Data               []byte        `gorm:"type:jsonb;not null"`
	StatusCode         BulkJobStatus `gorm:"column:status_code;type:bulk_job_status;not null;index"`
	LastDetailMessage  *string       `gorm:"type:text"`
	CreatedAt          time.Time     `gorm:"not null"`
	UpdatedAt          time.Time     `gorm:"not null"`
}

func (BulkJobItem) TableName() string { return "bulk_job_items" }
