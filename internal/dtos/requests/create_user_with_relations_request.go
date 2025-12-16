package requests

type CreateUserWithRelationsRequest struct {
	Name     string        `json:"name" validate:"required"`
	Email    string        `json:"email" validate:"required,email"`
	Password string        `json:"password" validate:"required"`
	Roles    []RoleRequest `json:"roles"`
}

type RoleRequest struct {
	ID uint64 `json:"id" validate:"required"`
}

type CreateUserWithExistingRelationsRequest struct {
	Name     string   `json:"name" validate:"required"`
	Email    string   `json:"email" validate:"required,email"`
	Password string   `json:"password" validate:"required,min=8"`
	RoleIDs  []uint64 `json:"role_ids" validate:"required,dive,gt=0"`
}
