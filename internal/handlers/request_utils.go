package handlers

import (
	"context"
	"go-fiber-core/internal/contextkeys"
	"go-fiber-core/internal/domain"
	"strconv"

	fiber "github.com/gofiber/fiber/v2"
)

// --- HELPERS DE AUTENTICACIÓN ---

// getUserIDFromCtx obtiene el ID de usuario (como string) del contexto.
func getUserIDFromCtx(ctx context.Context) (string, error) {
	userID, ok := contextkeys.GetUserID(ctx) // Usa el helper del paquete contextkeys
	if !ok {
		// Esto no debería pasar si el middleware se aplicó correctamente
		return "", domain.ErrAuthentication
	}
	return userID, nil
}

// getUserIDUint64FromCtx es un helper conveniente que también convierte el ID a uint64.
func getUserIDUint64FromCtx(ctx context.Context) (uint64, error) {
	userIDStr, err := getUserIDFromCtx(ctx)
	if err != nil {
		return 0, err
	}

	userID, err := strconv.ParseUint(userIDStr, 10, 64)
	if err != nil {
		// Esto indicaría un token malformado
		return 0, domain.ErrInvalidArgument
	}
	return userID, nil
}

// --- OTROS HELPERS DE HANDLERS (Ejemplo) ---

// (Aquí es donde pusiste tu helper getUintID en bank_handler.go)
// Es mejor tenerlo aquí para que otros handlers (como user_handler, etc.)
// también puedan usarlo.

// getUintID obtiene un parámetro "id" de la URL y lo convierte a uint.
func getUintID(c *fiber.Ctx) (uint, error) {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return 0, domain.ErrInvalidArgument
	}
	return uint(id), nil
}
