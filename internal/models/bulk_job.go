package models

import "time"

type BulkJobStatus string

const (
	BulkJobStatusImporting            BulkJobStatus = "IMPORTING"
	BulkJobStatusImported             BulkJobStatus = "IMPORTED"
	BulkJobStatusProcessing           BulkJobStatus = "PROCESSING"
	BulkJobStatusProcessed            BulkJobStatus = "PROCESSED"
	BulkJobStatusErrorImporting       BulkJobStatus = "ERROR_IMPORTING"
	BulkJobStatusErrorProcess         BulkJobStatus = "ERROR_PROCESS"
	BulkJobStatusProcessedWithDetails BulkJobStatus = "PROCESSED_WITH_DETAILS"
)

type BulkJob struct {
	ID               int64         `gorm:"primaryKey"`
	OperatorID       uint64        `gorm:"not null;index"`
	BranchID         int64         `gorm:"not null;default:0;index"`
	KeyCode          *string       `gorm:"type:varchar(20)"`
	RefCode          string        `gorm:"column:ref_code;type:varchar(255);not null;index"`
	StatusCode       BulkJobStatus `gorm:"column:status_code;type:bulk_job_status;not null;index"`
	TotalDetailItems int           `gorm:"not null;default:0"`
	FileName         *string       `gorm:"type:varchar(255)"`
	CreatedAt        time.Time     `gorm:"not null"`
	UpdatedAt        time.Time     `gorm:"not null"`
}

func (BulkJob) TableName() string { return "bulk_jobs" }
