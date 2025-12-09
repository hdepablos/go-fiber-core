package auth

import (
	"fmt"
	"go-fiber-core/internal/dtos/config"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
)

// tokenService es la implementación de la interfaz TokenService.
type tokenService struct {
	cfg config.JWTConfig
}

// NewTokenService crea una nueva instancia de TokenService.
func NewTokenService(cfg *config.AppConfig) TokenService {
	return &tokenService{cfg: cfg.JWTConfig}
}

func (s *tokenService) GenerateTokens(userID string) (string, string, error) {
	accessTTL := time.Minute * time.Duration(s.cfg.JwtAccessTtlMinutes)
	accessToken, err := s.createToken(userID, accessTTL, s.cfg.JwtAccessSecret, "access")
	if err != nil {
		return "", "", err
	}

	refreshTTL := time.Hour * 24 * time.Duration(s.cfg.JwtRefreshTtlDays)
	refreshToken, err := s.createToken(userID, refreshTTL, s.cfg.JwtRefreshSecret, "refresh")
	if err != nil {
		return "", "", err
	}

	return accessToken, refreshToken, nil
}

func (s *tokenService) createToken(userID string, ttl time.Duration, secret, tokenType string) (string, error) {
	if secret == "" {
		return "", fmt.Errorf("el secreto JWT para '%s' no está configurado", tokenType)
	}
	claims := jwt.MapClaims{
		"sub": userID, "typ": tokenType,
		"exp": time.Now().Add(ttl).Unix(), "iat": time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func (s *tokenService) ValidateToken(tokenString string) (*jwt.Token, error) {
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

func (s *tokenService) getSecret(tokenType string) (string, error) {
	if tokenType == "access" {
		return s.cfg.JwtAccessSecret, nil
	}
	if tokenType == "refresh" {
		return s.cfg.JwtRefreshSecret, nil
	}
	return "", fmt.Errorf("tipo de token desconocido: %s", tokenType)
}
