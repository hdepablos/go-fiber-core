package requests

// BulkAddRolesToMenusRequest permite asociar múltiples roles a múltiples menús.
type BulkAddRolesToMenusRequest struct {
	MenuIDs []uint64 `json:"menu_ids" validate:"required,min=1"`
	RoleIDs []uint64 `json:"role_ids" validate:"required,min=1"`
}
