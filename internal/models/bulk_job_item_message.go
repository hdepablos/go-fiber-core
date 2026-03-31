package models

import "time"

type BulkJobItemMessage struct {
	ID            int64      `gorm:"primaryKey"`
	BulkJobItemID int64      `gorm:"not null;index"`
	Severity      string     `gorm:"type:log_severity;not null"`
	Code          *string    `gorm:"type:varchar(64)"`
	DetailMessage string     `gorm:"column:detail_message;type:text;not null"`
	Meta          *[]byte    `gorm:"type:jsonb"`
	CreatedAt     time.Time  `gorm:"not null"`
}

func (BulkJobItemMessage) TableName() string { return "bulk_job_item_messages" }
