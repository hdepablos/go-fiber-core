package models

import "time"

type BulkJobListItem struct {
	ID                          int64         `json:"id"`
	OperatorID                  uint64        `json:"operator_id"`
	BranchID                    int64         `json:"branch_id"`
	KeyCode                     *string       `json:"key_code"`
	RefCode                     string        `json:"ref_code"`
	StatusCode                  BulkJobStatus `json:"status_code"`
	DeclaredTotalRecords        int           `json:"declared_total_records"`
	FileName                    *string       `json:"file_name"`
	CreatedAt                   time.Time     `json:"created_at"`
	UpdatedAt                   time.Time     `json:"updated_at"`
	TotalRecords                int64         `json:"total_records"`
	PendingRecords              int64         `json:"pending_records"`
	ProcessedRecords            int64         `json:"processed_records"`
	ErrorRecords                int64         `json:"error_records"`
	ProcessedWithDetailsRecords int64         `json:"processed_with_details_records"`
}
