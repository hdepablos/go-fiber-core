package middleware

import (
	"log"
	"time"

	fiber "github.com/gofiber/fiber/v2"
	redis "github.com/redis/go-redis/v9"
)

const rateLimitKeyPrefix = "rate_limit:"

// RateLimitConfig permite configurar el middleware
type RateLimitConfig struct {
	Limit  int64
	Window time.Duration
}

// RateLimitMiddleware aplica un rate limit por IP o cabecera de forma atómica.
func RateLimitMiddleware(redisClient *redis.Client, config RateLimitConfig) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// 1. Usa un identificador fiable: cabecera para usuarios autenticados o IP para anónimos.
		clientIdentifier := c.Get("X-Client-Code") // Ideal para una API Key
		if clientIdentifier == "" {
			clientIdentifier = c.IP()
		}

		key := rateLimitKeyPrefix + clientIdentifier

		// 2. Utiliza el contexto de la petición de Fiber.
		ctx := c.Context()

		// 3. Usa una pipeline de Redis para ejecutar comandos de forma atómica.
		var countCmd *redis.IntCmd
		_, err := redisClient.Pipelined(ctx, func(pipe redis.Pipeliner) error {
			// Incrementa el contador. Si la clave no existe, la crea con valor 1.
			countCmd = pipe.Incr(ctx, key)
			// Establece la expiración en cada petición para implementar una ventana deslizante.
			pipe.Expire(ctx, key, config.Window)
			return nil
		})

		if err != nil {
			log.Printf("Error al ejecutar la pipeline de Redis: %v", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Internal server error"})
		}

		// Obtenemos el valor actual del contador
		count, err := countCmd.Result()
		if err != nil {
			log.Printf("Error al obtener el resultado del contador: %v", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Internal server error"})
		}

		// 4. Comprueba si se ha excedido el límite.
		if count > config.Limit {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error": "Rate limit exceeded",
			})
		}

		// Si todo está bien, pasa a la siguiente ruta.
		return c.Next()
	}
}
