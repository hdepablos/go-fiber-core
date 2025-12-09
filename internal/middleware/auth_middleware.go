package middleware

import (
	"go-fiber-core/internal/contextkeys" // <-- Importa tu nuevo paquete
	"go-fiber-core/internal/services/auth"
	"strings"

	fiber "github.com/gofiber/fiber/v2"
	jwt "github.com/golang-jwt/jwt/v5"
)

func AuthMiddleware(tokenService auth.TokenService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "falta la cabecera de autorización"})
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "formato de token inválido"})
		}

		tokenString := parts[1]

		token, err := tokenService.ValidateToken(tokenString)
		if err != nil || !token.Valid {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "token inválido o expirado"})
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "formato de claims de token inválido"})
		}

		userID, ok := claims["sub"].(string)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "claim de ID de usuario inválida"})
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
