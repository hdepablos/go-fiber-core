package requests

type AddRolesRequest struct {
	RoleIDs []uint64 `json:"role_ids" validate:"required,min=1"`
}
