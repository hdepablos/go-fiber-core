package auth

import (
	"errors"
	"fmt"
	"go-fiber-core/internal/dtos/config"

	jwt "github.com/golang-jwt/jwt/v5"
)

// cognitoTokenService implementa TokenService delegando la validación a Cognito.
// Nota: Esta es una implementación PLACEHOLDER.
// En una implementación real, aquí se usaría un JWKS (JSON Web Key Set) para verificar la firma del token.
type cognitoTokenService struct {
	cfg *config.AppConfig
}

// NewCognitoTokenService crea una nueva instancia de cognitoTokenService.
func NewCognitoTokenService(cfg *config.AppConfig) TokenService {
	return &cognitoTokenService{cfg: cfg}
}

// GenerateTokens en Cognito NO se debería usar desde el backend para crear tokens arbitrarios.
// Cognito entrega los tokens tras el login.
// Si se necesita, podría usarse para "refrescar" o intercambiar credenciales, pero el flujo es diferente.
func (s *cognitoTokenService) GenerateTokens(userID, sessionID string) (string, string, error) {
	return "", "", errors.New("operación no soportada: Cognito genera los tokens, no el backend")
}

// ValidateToken verifica la firma del token usando las claves públicas de Cognito (JWKS).
func (s *cognitoTokenService) ValidateToken(tokenString string) (*jwt.Token, error) {
	// ⚠️ TODO: Implementar validación real con JWKS.
	// 1. Descargar JWKS desde https://cognito-idp.{region}.amazonaws.com/{userPoolId}/.well-known/jwks.json
	// 2. Encontrar la clave pública correcta (kid) en el header del token.
	// 3. Verificar la firma RSA.

	// Por ahora, retornamos un error indicando que falta configuración.
	return nil, fmt.Errorf("validación Cognito no implementada completamente (requiere configuración de User Pool ID y Región)")
}
