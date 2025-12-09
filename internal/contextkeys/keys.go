package contextkeys

import "context"

// Definimos un tipo local para la clave. Esto previene colisiones
// con otras bibliotecas que puedan usar el mismo string.
type contextKey string

// UserIDKey es la clave que usaremos para guardar y leer el ID de usuario
// del context.Context.
const UserIDKey contextKey = "userID"

// --- Helpers Opcionales (Recomendado) ---

// SetUserID enriquece un contexto con el ID de usuario.
func SetUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, UserIDKey, userID)
}

// GetUserID extrae el ID de usuario (string) del contexto.
func GetUserID(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(UserIDKey).(string)
	return userID, ok
}
