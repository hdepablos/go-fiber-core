package models

import "time"

type MenuResponse struct {
	ID       uint   `json:"id"`
	Type     string `json:"type"`
	Name     string `json:"name"`
	Icon     string `json:"icon"`
}

type UserResponse struct {
	ID    uint   `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email,omitempty"`
}

type MenuUserResponse struct {
	ID         uint   `json:"id"`
	MenuID     uint   `json:"menu_id"`
	UserID     uint   `json:"user_id"`
	OperatorID *uint  `json:"operator_id"`

	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	Menu     MenuResponse `json:"Menu"`
	User     UserResponse `json:"User"`
	Operator *UserResponse `json:"Operator,omitempty"`
}

type MenuUserRow struct {
	ID         uint
	MenuID     uint
	UserID     uint
	OperatorID *uint
	IsActive   bool
	CreatedAt  time.Time
	UpdatedAt  time.Time

	MenuIDRef uint
	MenuType  string
	MenuName  string
	MenuIcon  string

	UserIDRef uint
	UserName  string
	UserEmail string

	OperatorIDRef *uint
	OperatorName  *string
}
