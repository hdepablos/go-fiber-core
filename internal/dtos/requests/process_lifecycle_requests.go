package requests

type ReplicateScenarioRequest struct {
	ProcessVersionID int64 `json:"process_version_id" validate:"required,gt=0"`
}

type PromoteScenarioRequest struct {
	ProcessVersionID int64  `json:"process_version_id" validate:"required,gt=0"`
	Comment          string `json:"comment" validate:"required,max=300"`
}

type ResolveScenarioRequest struct {
	ProcessTypeID            int64  `json:"process_type_id" validate:"required,gt=0"`
	SedeID                   int64  `json:"sede_id" validate:"required"`
	OverrideProcessVersionID *int64 `json:"override_process_version_id,omitempty"`
}
