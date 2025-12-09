package requests

type CreateUserWithRelationsRequest struct {
	Name     string           `json:"name" validate:"required"`
	Email    string           `json:"email" validate:"required,email"`
	Password string           `json:"password" validate:"required"`
	Products []ProductRequest `json:"products"`
	Roles    []RoleRequest    `json:"roles"`
}

type ProductRequest struct {
	Name  string  `json:"name" validate:"required"`
	Price float64 `json:"price" validate:"required"`
}

type RoleRequest struct {
	ID uint64 `json:"id" validate:"required"`
}

type CreateUserWithExistingRelationsRequest struct {
	Name       string   `json:"name" validate:"required"`
	Email      string   `json:"email" validate:"required,email"`
	Password   string   `json:"password" validate:"required,min=8"`
	ProductIDs []uint64 `json:"product_ids" validate:"required,dive,gt=0"`
	RoleIDs    []uint64 `json:"role_ids" validate:"required,dive,gt=0"`
}
