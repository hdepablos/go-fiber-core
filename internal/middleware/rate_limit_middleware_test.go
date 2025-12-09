package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	// ✅ CORRECCIÓN: Se añaden nombres explícitos para evitar la colisión de 'v2'.
	miniredis "github.com/alicebob/miniredis/v2"
	fiber "github.com/gofiber/fiber/v2"
	redis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

// setupTest crea una instancia de Fiber y un cliente de Redis conectado a un servidor mock.
// Se añade la configuración de ProxyHeader para que Fiber lea la IP desde X-Forwarded-For.
func setupTest(t *testing.T) (*fiber.App, *redis.Client, *miniredis.Miniredis) {
	s, err := miniredis.Run()
	if err != nil {
		t.Fatalf("No se pudo iniciar miniredis: %s", err)
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})

	// Configura Fiber para que confíe en la cabecera de proxy, esencial para tests en Docker.
	app := fiber.New(fiber.Config{
		ProxyHeader: fiber.HeaderXForwardedFor,
	})

	return app, redisClient, s
}

// Test 1: Petición exitosa cuando el límite no se ha alcanzado.
func TestRateLimitMiddleware_AllowsRequest_WhenUnderLimit(t *testing.T) {
	app, redisClient, s := setupTest(t)
	defer s.Close()

	config := RateLimitConfig{
		Limit:  5,
		Window: 1 * time.Minute,
	}

	app.Use(RateLimitMiddleware(redisClient, config))
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode, "La petición debería ser exitosa")
}

// Test 2: Petición bloqueada cuando se excede el límite.
func TestRateLimitMiddleware_BlocksRequest_WhenOverLimit(t *testing.T) {
	app, redisClient, s := setupTest(t)
	defer s.Close()

	config := RateLimitConfig{
		Limit:  2,
		Window: 1 * time.Minute,
	}

	app.Use(RateLimitMiddleware(redisClient, config))
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	for i := 0; i < int(config.Limit); i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		resp, _ := app.Test(req)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusTooManyRequests, resp.StatusCode, "La petición debería ser bloqueada por exceder el límite")
}

// Test 3: Diferentes clientes (por IP) tienen contadores separados.
// Se simulan las diferentes IPs a través de la cabecera X-Forwarded-For.
func TestRateLimitMiddleware_DifferentIPsHaveDifferentCounters(t *testing.T) {
	app, redisClient, s := setupTest(t)
	defer s.Close()

	config := RateLimitConfig{
		Limit:  1,
		Window: 1 * time.Minute,
	}

	app.Use(RateLimitMiddleware(redisClient, config))
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	// Petición del Cliente 1
	req1 := httptest.NewRequest(http.MethodGet, "/", nil)
	req1.Header.Set("X-Forwarded-For", "192.168.1.1")
	resp1, _ := app.Test(req1)
	assert.Equal(t, fiber.StatusOK, resp1.StatusCode)

	// Petición del Cliente 2
	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	req2.Header.Set("X-Forwarded-For", "192.168.1.2")
	resp2, _ := app.Test(req2)
	assert.Equal(t, fiber.StatusOK, resp2.StatusCode)

	// Segunda petición del Cliente 1 (debería ser bloqueada)
	resp3, _ := app.Test(req1)
	assert.Equal(t, fiber.StatusTooManyRequests, resp3.StatusCode)
}

// Test 4: Usa la cabecera X-Client-Code como identificador prioritario.
func TestRateLimitMiddleware_UsesHeaderAsIdentifier(t *testing.T) {
	app, redisClient, s := setupTest(t)
	defer s.Close()

	config := RateLimitConfig{
		Limit:  1,
		Window: 1 * time.Minute,
	}

	app.Use(RateLimitMiddleware(redisClient, config))
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Client-Code", "client-A")
	resp, _ := app.Test(req)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	resp2, _ := app.Test(req)
	assert.Equal(t, fiber.StatusTooManyRequests, resp2.StatusCode)

	req3 := httptest.NewRequest(http.MethodGet, "/", nil)
	req3.Header.Set("X-Client-Code", "client-B")
	resp3, _ := app.Test(req3)
	assert.Equal(t, fiber.StatusOK, resp3.StatusCode)
}

// Test 5: El contador se reinicia después de que la ventana de tiempo expira.
func TestRateLimitMiddleware_CounterResetsAfterWindow(t *testing.T) {
	app, redisClient, s := setupTest(t)
	defer s.Close()

	config := RateLimitConfig{
		Limit:  1,
		Window: 2 * time.Second, // Ventana corta para el test
	}

	app.Use(RateLimitMiddleware(redisClient, config))
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	resp, _ := app.Test(req)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	resp2, _ := app.Test(req)
	assert.Equal(t, fiber.StatusTooManyRequests, resp2.StatusCode)

	// Avanzamos el tiempo del servidor mock para que la clave expire
	s.FastForward(3 * time.Second)

	resp3, _ := app.Test(req)
	assert.Equal(t, fiber.StatusOK, resp3.StatusCode, "La petición debería ser exitosa después de que la ventana expira")
}

// Test 6: Maneja correctamente un error de Redis.
func TestRateLimitMiddleware_HandlesRedisError(t *testing.T) {
	app, redisClient, s := setupTest(t)

	// Cierra el servidor mock para forzar un error de conexión
	s.Close()

	config := RateLimitConfig{
		Limit:  5,
		Window: 1 * time.Minute,
	}
	app.Use(RateLimitMiddleware(redisClient, config))
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
}
