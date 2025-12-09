// internal/dtos/requests/bulk_create_products_request.go
package requests

type BulkCreateProductsRequest struct {
	UserID   uint64                `json:"user_id" validate:"required"`
	Products []CreateProductSimple `json:"products" validate:"required,dive"`
}

type CreateProductSimple struct {
	Name  string  `json:"name" validate:"required"`
	Price float64 `json:"price" validate:"required,gt=0"`
}
