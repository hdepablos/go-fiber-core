// internal/dtos/requests/create_product_request.go
package requests

type CreateProductRequest struct {
	Name     string  `json:"name" validate:"required,min=3"`
	Price    float64 `json:"price" validate:"required,gt=100"`
	UserID   uint64  `json:"user_id" validate:"required"`
	BankID   *uint64 `json:"bank_id" validate:"required"`
	IsActive bool    `json:"is_active" validate:"omitempty"`
}

type UpdateProductRequest struct {
	Name  string  `json:"name" validate:"required,min=3"`
	Price float64 `json:"price" validate:"required,gt=0"`
}
