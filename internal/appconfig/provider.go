package appconfig

import "context"

// SecretProvider define la interfaz para obtener secretos y configuración.
// Permite desacoplar la fuente de los datos (Variables de Entorno, AWS Secrets Manager, Vault, etc.)
// de la lógica de negocio.
type SecretProvider interface {
	// GetSecret retorna el valor de un secreto dado su clave.
	// Si el secreto no existe, puede retornar error o string vacío según la implementación.
	GetSecret(ctx context.Context, key string) (string, error)
}
