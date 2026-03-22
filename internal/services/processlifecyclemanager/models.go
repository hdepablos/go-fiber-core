package processlifecyclemanager

import (
	"encoding/json"
	"time"
)

type ProcessType struct {
	ID         int64      `gorm:"column:id;primaryKey"`
	ArchivedAt *time.Time `gorm:"column:archived_at"`
}

func (ProcessType) TableName() string {
	return "process_types"
}

type ProcessVersion struct {
	ID            int64      `gorm:"column:id;primaryKey"`
	ProcessTypeID int64      `gorm:"column:process_type_id"`
	VersionNumber int        `gorm:"column:version_number"`
	SedeID        *int64     `gorm:"column:sede_id"`
	Status        string     `gorm:"column:status"`
	OperatorID    int64      `gorm:"column:operator_id"`
	ArchivedAt    *time.Time `gorm:"column:archived_at"`
	CreatedAt     time.Time  `gorm:"column:created_at"`
	UpdatedAt     time.Time  `gorm:"column:updated_at"`
}

func (ProcessVersion) TableName() string {
	return "process_versions"
}

type ProcessStep struct {
	ID               int64           `gorm:"column:id;primaryKey"`
	ProcessVersionID int64           `gorm:"column:process_version_id"`
	StepOrder        int32           `gorm:"column:step_order"`
	Roadmap          int             `gorm:"column:roadmap"`
	Name             string          `gorm:"column:name"`
	ExecutionKey     string          `gorm:"column:execution_key"`
	Config           json.RawMessage `gorm:"column:config"`
	CreatedAt        time.Time       `gorm:"column:created_at"`
}

func (ProcessStep) TableName() string {
	return "process_steps"
}

type ProcessVersionHistory struct {
	ID                 int64     `gorm:"column:id;primaryKey"`
	ProcessVersionID   int64     `gorm:"column:process_version_id"`
	ProcessTypeID      int64     `gorm:"column:process_type_id"`
	PromotedFromStatus string    `gorm:"column:promoted_from_status"`
	PromotedAt         time.Time `gorm:"column:promoted_at"`
	PromotedBy         int64     `gorm:"column:promoted_by"`
	Comment            string    `gorm:"column:comment"`
}

func (ProcessVersionHistory) TableName() string {
	return "process_version_history"
}

