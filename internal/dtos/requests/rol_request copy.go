package requests

type CreateRoleRequest struct {
	Name       string `json:"name" validate:"required,min=3"`
}

// UpdateRoleRequest se utiliza para actualizar un rol existente.
// Solo incluye los campos que permitimos que se modifiquen a través de la API.
type UpdateRoleRequest struct {
	Name       string `json:"name" validate:"required,min=3"`
	IsActive   bool   `json:"is_active" validate:"boolean"`
}
