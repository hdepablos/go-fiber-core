package requests

type ReplicateScenarioRequest struct {
	ProcessVersionID int64 `json:"process_version_id" validate:"required,gt=0"`
	OperatorID       int64 `json:"operator_id" validate:"required,gt=0"`
}

type PromoteScenarioRequest struct {
	ProcessVersionID int64  `json:"process_version_id" validate:"required,gt=0"`
	Comment          string `json:"comment" validate:"required,max=300"`
	PromotedBy       int64  `json:"promoted_by" validate:"required,gt=0"`
}

type ResolveScenarioRequest struct {
	ProcessTypeID            int64  `json:"process_type_id" validate:"required,gt=0"`
	SedeID                   int64  `json:"sede_id" validate:"gte=0"`
	OverrideProcessVersionID *int64 `json:"override_process_version_id,omitempty"`
	Roadmap                  *int   `json:"roadmap" validate:"required"`
}

type RunProcessRequest struct {
	ProcessTypeID            int64          `json:"process_type_id" validate:"required,gt=0"`
	SedeID                   *int64         `json:"sede_id" validate:"required,gte=0"`
	OverrideProcessVersionID *int64         `json:"override_process_version_id" validate:"required,gte=0"`
	Roadmap                  *int           `json:"roadmap" validate:"required,gte=0"`
	Input                    map[string]any `json:"input" validate:"required"`
	OperatorID               int64          `json:"-"` // Injected by controller, secure
}

type PreviewExportRequest struct {
	ProcessTypeID            int64          `json:"process_type_id" validate:"required,gt=0"`
	SedeID                   int64          `json:"sede_id" validate:"gte=0"`
	OverrideProcessVersionID int64          `json:"override_process_version_id" validate:"gte=0"`
	Roadmap                  int            `json:"roadmap" validate:"gte=0"`
	Mode                     string         `json:"mode" validate:"omitempty,oneof=prepare header body footer all"`
	Input                    map[string]any `json:"input" validate:"required"`
	BatchSize                int            `json:"batch_size,omitempty" validate:"gte=0"`
	Limit                    int            `json:"limit,omitempty" validate:"gte=0"`
	Offset                   int            `json:"offset,omitempty" validate:"gte=0"`
	ItemIDs                  []int64        `json:"item_ids,omitempty"`
	RowNumbers               []int          `json:"row_numbers,omitempty"`
}

type PreviewBatchRequest struct {
	ProcessTypeID            int64          `json:"process_type_id" validate:"required,gt=0"`
	SedeID                   int64          `json:"sede_id" validate:"gte=0"`
	OverrideProcessVersionID int64          `json:"override_process_version_id" validate:"gte=0"`
	Roadmap                  int            `json:"roadmap" validate:"gte=0"`
	Mode                     string         `json:"mode" validate:"omitempty,oneof=prepare batch all"`
	ApplyChanges             bool           `json:"apply_changes,omitempty"`
	Input                    map[string]any `json:"input" validate:"required"`
	BatchSize                int            `json:"batch_size,omitempty" validate:"gte=0"`
	Limit                    int            `json:"limit,omitempty" validate:"gte=0"`
	Offset                   int            `json:"offset,omitempty" validate:"gte=0"`
	BatchIndex               *int           `json:"batch_index,omitempty" validate:"omitempty,gte=0"`
	ItemIDs                  []int64        `json:"item_ids,omitempty"`
	RowNumbers               []int          `json:"row_numbers,omitempty"`
}

type MoveToTestScenarioRequest struct {
	ProcessVersionID int64 `json:"process_version_id" validate:"required,gt=0"`
}
