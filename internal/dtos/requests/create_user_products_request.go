package requests

type RoleNewRequest struct {
	Name string `json:"name" validate:"required"`
}

type CreateUserWithNewProductsAndRolesRequest struct {
	Name     string           `json:"name" validate:"required"`
	Email    string           `json:"email" validate:"required,email"`
	Password string           `json:"password" validate:"required"`
	Roles    []RoleNewRequest `json:"roles"` // ⚡ roles nuevos
}
