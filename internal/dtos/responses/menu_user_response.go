package responses

import "time"

// MenuUserResponse representa la respuesta para un elemento de MenuUser.
type MenuUserResponse struct {
	ID         uint          `json:"id"`
	MenuID     uint          `json:"menu_id"`
	UserID     uint64        `json:"user_id"`
	OperatorID *uint64       `json:"operator_id"`
	IsActive   bool          `json:"is_active"`
	CreatedAt  time.Time     `json:"created_at"`
	UpdatedAt  time.Time     `json:"updated_at"`
	
	// Objetos anidados simplificados o completos según necesidad
	Menu       *MenuSimpleResponse `json:"Menu"`
	User       *UserSimpleResponse `json:"User"`
	Operator   *UserSimpleResponse `json:"Operator"`
}

type MenuSimpleResponse struct {
	ID       uint    `json:"id"`
	Type     string  `json:"type"`
	Text     string  `json:"text"`
	To       *string `json:"to,omitempty"`
	Icon     *string `json:"icon,omitempty"`
	IsActive bool    `json:"is_active"`
}

type UserSimpleResponse struct {
	ID       uint64 `json:"id"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	IsActive bool   `json:"is_active"`
}
