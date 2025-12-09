package responses

type MenuItemResponse struct {
	Type     string             `json:"type"`
	Text     string             `json:"text"`
	Icon     *string            `json:"icon,omitempty"`
	To       *string            `json:"to,omitempty"`
	Children []MenuItemResponse `json:"children,omitempty"`
}
