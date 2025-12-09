package auth

import (
	"context"
	"go-fiber-core/internal/dtos/requests"
	"go-fiber-core/internal/dtos/responses"

	jwt "github.com/golang-jwt/jwt/v5"
)

// AuthService define la interfaz para la lógica de autenticación.
type AuthService interface {
	Login(ctx context.Context, req requests.LoginRequest) (*responses.LoginResponse, error)
	Refresh(ctx context.Context, refreshTokenString string) (newAccessToken string, newRefreshToken string, err error)
	Logout(ctx context.Context, userID uint64) error
}

// TokenService define la interfaz para la generación y validación de tokens.
type TokenService interface {
	GenerateTokens(userID string) (accessToken string, refreshToken string, err error)
	ValidateToken(tokenString string) (*jwt.Token, error)
}
