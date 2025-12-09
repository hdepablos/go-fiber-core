package responses

import (
	fiber "github.com/gofiber/fiber/v2"
)

// Response es la estructura base para todas las respuestas de la API.
type Response struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
	Errors  any    `json:"errors,omitempty"`
}

// Success genera una respuesta exitosa estándar (HTTP 200).
func Success(c *fiber.Ctx, message string, data any) error {
	response := Response{
		Status:  "success",
		Message: message,
	}
	if data != nil {
		response.Data = data
	}
	return c.JSON(response)
}

// Error genera una respuesta de error genérica con un código de estado específico.
func Error(c *fiber.Ctx, status int, message string, data ...any) error {
	response := Response{
		Status:  "error",
		Message: message,
	}
	if len(data) > 0 {
		response.Data = data[0]
	}
	return c.Status(status).JSON(response)
}

// ValidationError genera una respuesta de error de validación (HTTP 422)
// con un formato estructurado de errores por campo, similar a Laravel.
func ValidationError(c *fiber.Ctx, validationErrors map[string][]string) error {
	response := Response{
		Status:  "error",
		Message: "Los datos proporcionados no son válidos.",
		Errors:  validationErrors,
	}
	return c.Status(fiber.StatusUnprocessableEntity).JSON(response)
}
