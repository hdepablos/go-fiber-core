package auth

import (
	"fmt"
	"go-fiber-core/internal/dtos/config"
	"os"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
)

// NewTokenService crea una nueva instancia de TokenService.
// Decide qué implementación usar basándose en la configuración (Local vs Cognito).
func NewTokenService(cfg *config.AppConfig) TokenService {
	// Ejemplo de lógica de selección:
	// Si AUTH_PROVIDER == "cognito", retornamos la implementación de Cognito.
	if os.Getenv("AUTH_PROVIDER") == "cognito" {
		return NewCognitoTokenService(cfg)
	}

	// Por defecto, usamos la implementación local (HMAC)
	return &localTokenService{cfg: cfg.JWTConfig}
}

// localTokenService es la implementación LOCAL (HMAC) de TokenService.
type localTokenService struct {
	cfg config.JWTConfig
}

func (s *localTokenService) GenerateTokens(userID, sessionID string) (string, string, error) {
	// Config values are already time.Duration
	accessTTL := s.cfg.JwtAccessTtlMinutes
	accessToken, err := s.createToken(userID, sessionID, accessTTL, s.cfg.JwtAccessSecret, "access")
	if err != nil {
		return "", "", err
	}

	refreshTTL := s.cfg.JwtRefreshTtlDays
	refreshToken, err := s.createToken(userID, sessionID, refreshTTL, s.cfg.JwtRefreshSecret, "refresh")
	if err != nil {
		return "", "", err
	}

	return accessToken, refreshToken, nil
}

func (s *localTokenService) createToken(userID, sessionID string, ttl time.Duration, secret, tokenType string) (string, error) {
	if secret == "" {
		return "", fmt.Errorf("el secreto JWT para '%s' no está configurado", tokenType)
	}
	claims := jwt.MapClaims{
		"sub": userID,
		"sid": sessionID, // Session ID Claim
		"typ": tokenType,
		"exp": time.Now().Add(ttl).Unix(),
		"iat": time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func (s *localTokenService) ValidateToken(tokenString string) (*jwt.Token, error) {
	parser := jwt.Parser{}
	token, _, err := parser.ParseUnverified(tokenString, jwt.MapClaims{})
	if err != nil {
		return nil, fmt.Errorf("error al parsear el token: %w", err)
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("claims de token inválidos")
	}
	tokenType, _ := claims["typ"].(string)
	secret, err := s.getSecret(tokenType)
	if err != nil {
		return nil, err
	}
	if secret == "" {
		return nil, fmt.Errorf("el secreto JWT para '%s' no está configurado", tokenType)
	}
	return jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("método de firma inesperado: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})
}

func (s *localTokenService) getSecret(tokenType string) (string, error) {
	if tokenType == "access" {
		return s.cfg.JwtAccessSecret, nil
	}
	if tokenType == "refresh" {
		return s.cfg.JwtRefreshSecret, nil
	}
	return "", fmt.Errorf("tipo de token desconocido: %s", tokenType)
}
