package models

type Catalog struct {
	ID       uint64 `json:"id"`
	Name     string `json:"name"`
	IsActive bool   `json:"is_active"`
}

// AllCatalogsResponse sigue siendo el contenedor de las listas
type AllCatalogsResponse struct {
	Banks []Catalog `json:"banks"`
	Roles []Catalog `json:"roles"`
}
