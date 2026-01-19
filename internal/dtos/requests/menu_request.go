package requests

type CreateMenuRequest struct {
	ItemType   string  `json:"item_type" validate:"required,oneof=link separator group line"`
	ItemName   string  `json:"item_name" validate:"required"`
	ToPath     *string `json:"to_path,omitempty"`
	Icon       *string `json:"icon,omitempty"`
	ParentID   *uint   `json:"parent_id"`
	OrderIndex int     `json:"order_index"`
}
