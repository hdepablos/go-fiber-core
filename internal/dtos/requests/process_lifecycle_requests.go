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
	SedeID                   int64  `json:"sede_id" validate:"required"`
	OverrideProcessVersionID *int64 `json:"override_process_version_id,omitempty"`
	Roadmap                  *int   `json:"roadmap" validate:"required"`
}

type RunProcessRequest struct {
	ProcessTypeID            int64          `json:"process_type_id" validate:"required,gt=0"`
	SedeID                   int64          `json:"sede_id" validate:"required"`
	OverrideProcessVersionID *int64         `json:"override_process_version_id,omitempty"`
	Roadmap                  int            `json:"roadmap"` // Optional, default 0
	Input                    map[string]any `json:"input" validate:"required"`
	OperatorID               int64          `json:"-"` // Injected by controller, secure
}

type MoveToTestScenarioRequest struct {
	ProcessVersionID int64 `json:"process_version_id" validate:"required,gt=0"`
}
