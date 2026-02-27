package appconfig

import (
	"context"
	"os"
)

// EnvSecretProvider implementa SecretProvider leyendo variables de entorno del sistema operativo.
// Es la implementación por defecto para desarrollo local y entornos contenerizados simples (Docker/K8s).
type EnvSecretProvider struct{}

func NewEnvSecretProvider() *EnvSecretProvider {
	return &EnvSecretProvider{}
}

func (p *EnvSecretProvider) GetSecret(ctx context.Context, key string) (string, error) {
	// os.Getenv no retorna error si la clave no existe, solo string vacío.
	return os.Getenv(key), nil
}
