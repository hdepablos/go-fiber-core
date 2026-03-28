package responses

type GoogleOAuthUser struct {
	GoogleID      string `json:"google_id"`
	Email         string `json:"email"`
	VerifiedEmail bool   `json:"verified_email"`
	Name          string `json:"name"`
	GivenName     string `json:"given_name,omitempty"`
	FamilyName    string `json:"family_name,omitempty"`
	Picture       string `json:"picture,omitempty"`
	Locale        string `json:"locale,omitempty"`
}

type GoogleOAuthLoginResponse struct {
	Provider     string             `json:"provider"`
	AccessToken  string             `json:"access_token"`
	RefreshToken string             `json:"refresh_token"`
	UserID       uint64             `json:"user_id"`
	UserName     string             `json:"user_name"`
	RoleIDs      []uint64           `json:"role_ids"`
	Roles        []string           `json:"roles"`
	Menu         []MenuItemResponse `json:"menu"`
	User         GoogleOAuthUser    `json:"user"`
}
