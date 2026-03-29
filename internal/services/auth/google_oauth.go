package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

const googleUserInfoEndpoint = "https://www.googleapis.com/oauth2/v2/userinfo"

// GoogleUserInfo representa el payload devuelto por el endpoint "userinfo" de Google.
type GoogleUserInfo struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	VerifiedEmail bool   `json:"verified_email"`
	Name          string `json:"name"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Picture       string `json:"picture"`
	Locale        string `json:"locale"`
}

type googleOAuthClient struct {
	cfg         *oauth2.Config
	httpClient  *http.Client
	userInfoURL string
}

// newGoogleOAuthClientFromEnv construye el cliente OAuth2 leyendo las variables requeridas.
// Se mantiene lazy (no se inicializa al boot) para no romper despliegues que aún no habilitan Google OAuth.
func newGoogleOAuthClientFromEnv() (*googleOAuthClient, error) {
	clientID := os.Getenv("GOOGLE_CLIENT_ID")
	clientSecret := os.Getenv("GOOGLE_CLIENT_SECRET")
	redirectURL := os.Getenv("GOOGLE_REDIRECT_URL")

	if clientID == "" || clientSecret == "" || redirectURL == "" {
		return nil, errors.New("google oauth2 no está configurado (faltan GOOGLE_CLIENT_ID/GOOGLE_CLIENT_SECRET/GOOGLE_REDIRECT_URL)")
	}

	return &googleOAuthClient{
		cfg: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURL,
			Scopes: []string{
				"https://www.googleapis.com/auth/userinfo.email",
				"https://www.googleapis.com/auth/userinfo.profile",
			},
			Endpoint: google.Endpoint,
		},
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		userInfoURL: googleUserInfoEndpoint,
	}, nil
}

func (c *googleOAuthClient) AuthCodeURL(state string) (string, error) {
	if c == nil || c.cfg == nil {
		return "", errors.New("google oauth2 no inicializado")
	}
	if state == "" {
		return "", errors.New("state requerido")
	}

	hd := os.Getenv("GOOGLE_ALLOWED_DOMAIN")
	if hd != "" {
		return c.cfg.AuthCodeURL(
			state,
			oauth2.AccessTypeOnline,
			oauth2.SetAuthURLParam("include_granted_scopes", "true"),
			oauth2.SetAuthURLParam("hd", hd),
		), nil
	}

	return c.cfg.AuthCodeURL(
		state,
		oauth2.AccessTypeOnline,
		oauth2.SetAuthURLParam("include_granted_scopes", "true"),
	), nil
}

func (c *googleOAuthClient) Exchange(ctx context.Context, code string) (*oauth2.Token, error) {
	if c == nil || c.cfg == nil {
		return nil, errors.New("google oauth2 no inicializado")
	}
	if code == "" {
		return nil, errors.New("code requerido")
	}
	return c.cfg.Exchange(ctx, code)
}

func (c *googleOAuthClient) FetchUserInfo(ctx context.Context, accessToken string) (*GoogleUserInfo, error) {
	if c == nil || c.httpClient == nil {
		return nil, errors.New("http client no inicializado")
	}
	if accessToken == "" {
		return nil, errors.New("access token requerido")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.userInfoURL, nil)
	if err != nil {
		return nil, fmt.Errorf("error creando request userinfo: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error llamando a userinfo: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("error leyendo response userinfo: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, fmt.Errorf("userinfo respondió %d: %s", resp.StatusCode, string(body))
	}

	var info GoogleUserInfo
	if err := json.Unmarshal(body, &info); err != nil {
		return nil, fmt.Errorf("error parseando userinfo: %w", err)
	}
	if info.Email == "" {
		return nil, errors.New("userinfo no devolvió email")
	}

	return &info, nil
}
