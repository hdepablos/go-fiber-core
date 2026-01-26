package middleware

import (
	"errors"
	"go-fiber-core/internal/domain"
	"go-fiber-core/internal/dtos/responses"

	"github.com/gofiber/fiber/v2"
)

// GlobalErrorHandler es el middleware para manejar errores centralizadamente.
func GlobalErrorHandler(c *fiber.Ctx, err error) error {
	// 1. Errores de Fiber (ej: 404 ruta no encontrada, 405 método no permitido)
	var fiberErr *fiber.Error
	if errors.As(err, &fiberErr) {
		return responses.Error(c, fiberErr.Code, fiberErr.Message)
	}

	// 2. Mapeo de errores de Dominio a Códigos HTTP
	switch {
	case errors.Is(err, domain.ErrNotFound):
		return responses.Error(c, fiber.StatusNotFound, err.Error())

	case errors.Is(err, domain.ErrInvalidArgument):
		return responses.Error(c, fiber.StatusBadRequest, err.Error())

	case errors.Is(err, domain.ErrAuthentication):
		return responses.Error(c, fiber.StatusUnauthorized, err.Error())

	// Agrega más casos según tus necesidades
	// case errors.Is(err, domain.ErrConflict):
	// 	return responses.Error(c, fiber.StatusConflict, err.Error())

	default:
		// 3. Error desconocido (Internal Server Error)
		// En producción, aquí podrías loguear el error original detallado (err.Error())
		// pero retornar un mensaje genérico al cliente por seguridad.
		return responses.Error(c, fiber.StatusInternalServerError, "Ha ocurrido un error interno inesperado")
	}
}
