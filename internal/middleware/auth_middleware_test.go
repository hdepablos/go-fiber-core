package middleware

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	fiber "github.com/gofiber/fiber/v2"
	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
)

// Este archivo solo contiene las funciones de test.
// Las definiciones de 'mockTokenService' las toma del archivo 'middleware_test.go'.

func TestAuthMiddleware_Success(t *testing.T) {
	app := fiber.New()
	expectedUserID := "user-123"

	mockService := &mockTokenService{ // <- Go encontrará la definición en el otro archivo
		tokenToReturn: &jwt.Token{
			Valid: true,
			Claims: jwt.MapClaims{
				"sub": expectedUserID,
			},
		},
		errorToReturn: nil,
	}

	app.Use(AuthMiddleware(mockService))
	app.Get("/protected", func(c *fiber.Ctx) error {
		userID := c.Locals("userID")
		return c.Status(fiber.StatusOK).JSON(fiber.Map{"userID": userID})
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer un-token-valido")
	resp, _ := app.Test(req)

	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	assert.JSONEq(t, fmt.Sprintf(`{"userID": "%s"}`, expectedUserID), string(body))
}

func TestAuthMiddleware_MissingHeader(t *testing.T) {
	app := fiber.New()
	mockService := &mockTokenService{}

	app.Use(AuthMiddleware(mockService))
	app.Get("/protected", func(c *fiber.Ctx) error { return c.SendStatus(200) })

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	resp, _ := app.Test(req)

	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(body), "falta la cabecera de autorización")
}

func TestAuthMiddleware_InvalidTokenFormat(t *testing.T) {
	app := fiber.New()
	mockService := &mockTokenService{}

	app.Use(AuthMiddleware(mockService))
	app.Get("/protected", func(c *fiber.Ctx) error { return c.SendStatus(200) })

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "un-token-sin-bearer")
	resp, _ := app.Test(req)

	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(body), "formato de token inválido")
}

func TestAuthMiddleware_TokenServiceValidationError(t *testing.T) {
	app := fiber.New()
	mockService := &mockTokenService{
		tokenToReturn: nil,
		errorToReturn: errors.New("error de validación"),
	}

	app.Use(AuthMiddleware(mockService))
	app.Get("/protected", func(c *fiber.Ctx) error { return c.SendStatus(200) })

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer token-que-falla")
	resp, _ := app.Test(req)

	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(body), "token inválido o expirado")
}

func TestAuthMiddleware_TokenNotValid(t *testing.T) {
	app := fiber.New()
	mockService := &mockTokenService{
		tokenToReturn: &jwt.Token{Valid: false},
		errorToReturn: nil,
	}

	app.Use(AuthMiddleware(mockService))
	app.Get("/protected", func(c *fiber.Ctx) error { return c.SendStatus(200) })

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer token-invalido")
	resp, _ := app.Test(req)

	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(body), "token inválido o expirado")
}

func TestAuthMiddleware_InvalidClaimsFormat(t *testing.T) {
	app := fiber.New()
	mockService := &mockTokenService{
		tokenToReturn: &jwt.Token{
			Valid:  true,
			Claims: &jwt.RegisteredClaims{},
		},
	}

	app.Use(AuthMiddleware(mockService))
	app.Get("/protected", func(c *fiber.Ctx) error { return c.SendStatus(200) })

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer token-claims-raras")
	resp, _ := app.Test(req)

	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(body), "formato de claims de token inválido")
}

func TestAuthMiddleware_MissingSubClaim(t *testing.T) {
	app := fiber.New()
	mockService := &mockTokenService{
		tokenToReturn: &jwt.Token{
			Valid: true,
			Claims: jwt.MapClaims{
				"role": "admin",
			},
		},
	}

	app.Use(AuthMiddleware(mockService))
	app.Get("/protected", func(c *fiber.Ctx) error { return c.SendStatus(200) })

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer token-sin-sub")
	resp, _ := app.Test(req)

	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(body), "claim de ID de usuario inválida")
}

func TestAuthMiddleware_SubClaimNotAString(t *testing.T) {
	app := fiber.New()
	mockService := &mockTokenService{
		tokenToReturn: &jwt.Token{
			Valid: true,
			Claims: jwt.MapClaims{
				"sub": 12345,
			},
		},
	}

	app.Use(AuthMiddleware(mockService))
	app.Get("/protected", func(c *fiber.Ctx) error { return c.SendStatus(200) })

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer token-sub-invalido")
	resp, _ := app.Test(req)

	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(body), "claim de ID de usuario inválida")
}
