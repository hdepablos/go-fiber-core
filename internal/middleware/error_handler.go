package middleware

import (
	"errors"
	"go-fiber-core/internal/domain" // Importamos nuestros errores
	"go-fiber-core/internal/dtos/responses"
	"log"

	fiber "github.com/gofiber/fiber/v2"
)

func ErrorHandler(c *fiber.Ctx, err error) error {
	// Revisa el tipo de error y decide qué respuesta enviar.
	switch {
	case errors.Is(err, domain.ErrNotFound):
		return responses.Error(c, fiber.StatusNotFound, err.Error())

	case errors.Is(err, domain.ErrInvalidArgument):
		return responses.Error(c, fiber.StatusBadRequest, err.Error())

	case errors.Is(err, domain.ErrAuthentication):
		return responses.Error(c, fiber.StatusUnauthorized, err.Error())

	// Y así para otros errores personalizados...

	default:
		// Si es un error que no esperamos, lo registramos para depuración
		// y devolvemos un error genérico 500.
		log.Printf("Error no manejado en la API: %v", err)
		return responses.Error(c, fiber.StatusInternalServerError, "Ha ocurrido un error inesperado en el servidor.")
	}
}
