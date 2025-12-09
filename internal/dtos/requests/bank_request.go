package requests

type CreateBankRequest struct {
	Name       string `json:"name" validate:"required,min=3"`
	EntityCode string `json:"entity_code" validate:"required,len=3"`
}

// UpdateBankRequest se utiliza para actualizar un banco existente.
// Solo incluye los campos que permitimos que se modifiquen a través de la API.
type UpdateBankRequest struct {
	Name       string `json:"name" validate:"required,min=3"`
	EntityCode string `json:"entity_code" validate:"required,len=3"`
	Enabled    bool   `json:"enabled" validate:"boolean"`
}
