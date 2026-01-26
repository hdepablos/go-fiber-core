package middleware

import (
	"go-fiber-core/internal/contextkeys" // <-- Importa tu nuevo paquete
	"go-fiber-core/internal/dtos/responses"
	"go-fiber-core/internal/services/auth"
	"strings"

	"fmt"

	fiber "github.com/gofiber/fiber/v2"
	jwt "github.com/golang-jwt/jwt/v5"
	redis "github.com/redis/go-redis/v9"
)

func AuthMiddleware(tokenService auth.TokenService, redisClient *redis.Client) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return responses.Error(c, fiber.StatusUnauthorized, "falta la cabecera de autorización")
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			return responses.Error(c, fiber.StatusUnauthorized, "formato de token inválido")
		}

		tokenString := parts[1]

		token, err := tokenService.ValidateToken(tokenString)
		if err != nil || !token.Valid {
			return responses.Error(c, fiber.StatusUnauthorized, "token inválido o expirado")
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			return responses.Error(c, fiber.StatusUnauthorized, "formato de claims de token inválido")
		}

		userID, ok := claims["sub"].(string)
		if !ok {
			return responses.Error(c, fiber.StatusUnauthorized, "claim de ID de usuario inválida")
		}

		// --- VERIFICACIÓN DE SESIÓN (REVOCACIÓN INMEDIATA) ---
		if sid, ok := claims["sid"].(string); ok {
			key := fmt.Sprintf("blacklist:session:%s", sid)
			exists, err := redisClient.Exists(c.Context(), key).Result()
			if err == nil && exists > 0 {
				return responses.Error(c, fiber.StatusUnauthorized, "sesión revocada")
			}
		}

		// --- CAMBIO CLAVE ---
		// En lugar de: c.Locals("userID", userID)
		// Usamos el context.Context estándar de Go.

		// 1. Obtenemos el contexto actual
		ctx := c.UserContext()

		// 2. Creamos un nuevo contexto enriquecido usando nuestro helper
		newCtx := contextkeys.SetUserID(ctx, userID)

		// 3. Establecemos el nuevo contexto para esta solicitud
		c.SetUserContext(newCtx)
		// --- FIN DEL CAMBIO ---

		return c.Next()
	}
}
