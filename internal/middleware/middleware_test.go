package middleware

import (
	jwt "github.com/golang-jwt/jwt/v5" // ✅ Esta línea soluciona el error
)

// --- Definiciones de Mocks para todo el paquete 'middleware' ---

// Definimos la interfaz que usan nuestros mocks.
// Esta debe coincidir con la interfaz real de tu paquete 'services'.
type TokenService interface {
	ValidateToken(tokenString string) (*jwt.Token, error)
	GenerateTokens(userID string) (string, string, error)
}

// Definimos el mock que implementa la interfaz TokenService.
type mockTokenService struct {
	tokenToReturn *jwt.Token
	errorToReturn error
}

// Implementamos los métodos para nuestro mock.
func (m *mockTokenService) ValidateToken(tokenString string) (*jwt.Token, error) {
	return m.tokenToReturn, m.errorToReturn
}

func (m *mockTokenService) GenerateTokens(userID string) (string, string, error) {
	return "access_token_mock", "refresh_token_mock", nil
}
