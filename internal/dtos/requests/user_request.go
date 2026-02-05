// internal/dtos/requests/user_requests.go
package requests

// CreateUserRequest se utiliza para la creación de un nuevo usuario.
type CreateUserRequest struct {
	Name     string `json:"name" validate:"required"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
	RoleIDs  []uint64 `json:"role_ids" validate:"required"`
}

type RemoveUserRolesRequest struct {
	UserIDs []uint64 `json:"user_ids"`
	RoleIDs []uint64 `json:"role_ids"`
}

type AssignRolesRequest struct {
	UserIDs []uint64 `json:"user_ids" validate:"required,min=1"`
	RoleIDs []uint64 `json:"role_ids" validate:"required,min=1"`
}


// UpdateUserRequest se utiliza para actualizar un usuario existente.
type UpdateUserRequest struct {
	Name  string `json:"name" validate:"required"`
	Email string `json:"email" validate:"required,email"`
	RoleIDs  *[]uint64 `json:"role_ids,omitempty"`
	// Podrías añadir más campos opcionales aquí si lo necesitas
	// Password string `json:"password,omitempty" validate:"omitempty,min=8"`
}

