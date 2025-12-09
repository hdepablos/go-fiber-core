package responses

// Respuesta completa del login
type LoginResponse struct {
	Token        string             `json:"token"`
	RefreshToken string             `json:"refresh_token"`
	UserName     string             `json:"user_name"`
	RoleIDs      []uint64           `json:"role_ids"`
	Roles        []string           `json:"roles"`
	Menu         []MenuItemResponse `json:"menu"` // menú filtrado según el user
}
