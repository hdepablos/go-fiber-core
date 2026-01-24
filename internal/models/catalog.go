package models

type Catalogo struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	IsActive bool   `json:"is_active"`
}

// AllCatalogsResponse sigue siendo el contenedor de las listas
type AllCatalogsResponse struct {
	Bank []Catalogo `json:"bank"`
	Role []Catalogo `json:"role"`
}
