# Arquitectura de Configuración y Secretos

Este documento describe cómo la aplicación gestiona su configuración y secretos sensibles, soportando múltiples entornos (Local, EKS, AWS Nativo) mediante el patrón **Strategy**.

## 🏗 Concepto: Secret Providers

La aplicación no asume que los secretos siempre vienen de variables de entorno. En su lugar, utiliza una interfaz `SecretProvider` que abstrae la fuente de los datos.

```go
type SecretProvider interface {
    GetSecret(ctx context.Context, key string) (string, error)
}
```

### Proveedores Implementados

1.  **`EnvSecretProvider` (Default):**
    *   Lee directamente de las variables de entorno del sistema (`os.Getenv`).
    *   Uso: Desarrollo Local, CI/CD, Despliegues en Kubernetes (donde los secretos ya se inyectan como env vars).

2.  **`AWSSecretProvider` (Producción Segura):**
    *   Se conecta a **AWS Secrets Manager** para obtener el valor en tiempo de ejecución.
    *   Uso: Entornos de producción de alta seguridad donde no se desea exponer secretos en el entorno del proceso.

## 🚀 Cómo Funciona

Al iniciar, `NewAppConfig` decide qué proveedor usar basándose en la variable `SECRET_PROVIDER`:

*   Si `SECRET_PROVIDER` está vacío o es cualquier valor distinto a `aws` -> Usa **EnvProvider**.
*   Si `SECRET_PROVIDER=aws` -> Inicializa **AWSSecretProvider**.

### Expansión de Variables

El sistema intercepta cualquier valor en `config.yaml` que tenga el formato `${NOMBRE_VARIABLE}` y lo resuelve usando el proveedor activo.

**Ejemplo:**
Si `config.yaml` tiene:
```yaml
database:
  password: "${DB_PASSWORD}"
```

*   Con **EnvProvider**: Busca la variable de entorno `DB_PASSWORD`.
*   Con **AWSSecretProvider**: Llama a Secrets Manager buscando el secreto con ID `DB_PASSWORD`.

## 🔐 Autenticación: Provider Pattern

Siguiendo la misma filosofía, la autenticación también implementa el patrón Strategy para soportar tanto autenticación local (Simple JWT) como delegada (AWS Cognito).

### AuthService & TokenService

El sistema decide qué implementación usar basándose en la variable de entorno `AUTH_PROVIDER`.

1.  **`local` (Default):**
    *   **AuthService:** Valida credenciales contra la base de datos local (PostgreSQL) usando bcrypt.
    *   **TokenService:** Genera y firma JWTs localmente usando HMAC (HS256) y un secreto compartido.
    *   **Uso:** Desarrollo, MVP, despliegues on-premise simples.

2.  **`cognito` (AWS):**
    *   **AuthService:** Delega el login a AWS Cognito (User Pools). No toca la base de datos local para credenciales.
    *   **TokenService:** Valida los tokens recibidos verificando su firma RSA contra el JWKS público de Cognito.
    *   **Uso:** Producción Enterprise, SSO, gestión centralizada de usuarios.

### Cómo cambiar de proveedor

Simplemente define la variable de entorno antes de iniciar la aplicación:

```bash
# Modo Local (Default)
export AUTH_PROVIDER=local

# Modo Cognito
export AUTH_PROVIDER=cognito
export AWS_REGION=us-east-1
export COGNITO_USER_POOL_ID=us-east-1_xxxxxxxxx
```
