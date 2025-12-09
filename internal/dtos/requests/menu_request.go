package requests

// CreateMenuRequest define los campos necesarios para crear un menú.
type CreateMenuRequest struct {
	ItemType string  `json:"item_type" validate:"required"` // Ej: "link", "submenu"
	ItemName string  `json:"item_name" validate:"required"` // Nombre visible
	ToPath   *string `json:"to,omitempty"`                  // Ruta o enlace opcional
	Icon     *string `json:"icon,omitempty"`                // Icono opcional
	ParentID *uint   `json:"parent_id,omitempty"`           // ID del padre (si es submenu)
}
