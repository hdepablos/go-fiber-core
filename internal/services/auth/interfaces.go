package auth

import (
	"context"
	"go-fiber-core/internal/dtos"
	"go-fiber-core/internal/dtos/requests"
	"go-fiber-core/internal/dtos/responses"
	"go-fiber-core/internal/models"

	jwt "github.com/golang-jwt/jwt/v5"
)

// AuthService define la interfaz para la lógica de autenticación.
type AuthService interface {
	// Login ahora acepta userAgent e IP para registrar la sesión
	Login(ctx context.Context, req requests.LoginRequest, userAgent, clientIP, origin, requestID string) (*responses.LoginResponse, error)
	Refresh(ctx context.Context, refreshTokenString string) (newAccessToken string, newRefreshToken string, err error)
	Logout(ctx context.Context, userID uint64, userAgent, clientIP, origin, requestID string) error
	RevokeSession(ctx context.Context, sessionID string) error
	RevokeUserSessions(ctx context.Context, userID uint64) error
	RevokeAllSessions(ctx context.Context) error
	GetActiveSessions(ctx context.Context, req dtos.PaginationRequest) (*dtos.PaginationResponse[models.Session], error)

	// Google OAuth2
	GoogleAuthURL(state string) (string, error)
	GoogleCallbackLogin(ctx context.Context, code, userAgent, clientIP, origin, requestID string) (*responses.GoogleOAuthLoginResponse, error)
	SaveGoogleOAuthState(ctx context.Context, state string) error
	ConsumeGoogleOAuthState(ctx context.Context, state string) (bool, error)
	SaveGoogleOAuthLoginResult(ctx context.Context, code string, result *responses.GoogleOAuthLoginResponse) error
	ConsumeGoogleOAuthLoginResult(ctx context.Context, code string) (*responses.GoogleOAuthLoginResponse, error)
}

// TokenService define la interfaz para la generación y validación de tokens.
type TokenService interface {
	// GenerateTokens ahora acepta sessionID para vincular el token a una sesión específica
	GenerateTokens(userID, sessionID string) (accessToken string, refreshToken string, err error)
	ValidateToken(tokenString string) (*jwt.Token, error)
}
