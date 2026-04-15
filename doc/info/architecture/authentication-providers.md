# Arquitectura de Autenticación: Provider Pattern

Este documento detalla el sistema de autenticación modular del proyecto, diseñado para soportar múltiples estrategias de identidad (Local vs Cloud/SaaS) sin modificar la lógica de negocio.

## 🎯 Objetivo

Permitir que la aplicación cambie su mecanismo de autenticación simplemente modificando una variable de entorno, facilitando la transición de un MVP (Auth Local) a un producto Enterprise (AWS Cognito, Auth0, etc.).

## 🏗 Diseño: Interfaces y Providers

El sistema se basa en dos interfaces principales definidas en `internal/services/auth/interfaces.go`:

1.  **`AuthService`**: Maneja las operaciones de identidad (Login, Logout, Gestión de Sesiones).
2.  **`TokenService`**: Maneja la emisión y validación de tokens (JWT).

### Implementaciones Disponibles

| Característica | Local (Default) | AWS Cognito |
| :--- | :--- | :--- |
| **Provider ID** | `local` (o vacío) | `cognito` |
| **AuthService** | `localAuthService` | `cognitoAuthService` |
| **TokenService** | `localTokenService` | `cognitoTokenService` |
| **Validación Credenciales** | `bcrypt` contra DB PostgreSQL local | Delegada a AWS User Pools (SRP/Password) |
| **Firma de Tokens** | HMAC (HS256) con `JWT_ACCESS_SECRET` | RSA (RS256) verificado contra JWKS público de AWS |
| **Gestión de Sesiones** | Tabla `sessions` en PostgreSQL | Gestionada por AWS (Stateless en backend) |

## 🚀 Cómo Configurar

La selección del proveedor se realiza mediante la variable de entorno `AUTH_PROVIDER`.

### 1. Modo Local (Desarrollo / On-Premise)

Es el modo por defecto. No requiere configuración especial más allá de las variables de base de datos y secretos JWT.

```bash
# .env
AUTH_PROVIDER=local  # Opcional, es el default
JWT_ACCESS_SECRET=mi_secreto_super_seguro
JWT_REFRESH_SECRET=otro_secreto
```

**Flujo:**
1.  Cliente envía `email` y `password`.
2.  Backend busca usuario en DB y verifica hash `bcrypt`.
3.  Backend genera JWT firmado con `JWT_ACCESS_SECRET`.

### 2. Modo AWS Cognito (Producción / Enterprise)

Delega la autenticación a AWS. Ideal para entornos Serverless o cuando se requiere SSO/MFA.

```bash
# .env
AUTH_PROVIDER=cognito
AWS_REGION=us-east-1
COGNITO_USER_POOL_ID=us-east-1_xxxxxxxxx
COGNITO_CLIENT_ID=xxxxxxxxxxxxxx
```

**Flujo:**
1.  Cliente envía credenciales.
2.  Backend (o Frontend) se autentica contra Cognito.
3.  Cognito devuelve Access/ID Tokens firmados por AWS.
4.  Backend recibe el token en los requests y valida su firma usando el JWKS público de Cognito (sin necesidad de secreto compartido).

## 🛠 Implementación Técnica

### Factory Pattern (`internal/services/auth/auth_service.go`)

El constructor `NewAuthService` actúa como una fábrica que instancia la implementación correcta:

```go
func NewAuthService(...) AuthService {
    if os.Getenv("AUTH_PROVIDER") == "cognito" {
        return NewCognitoAuthService()
    }
    return &localAuthService{...} // Default
}
```

### Pasos para Completar la Integración con Cognito

Actualmente, la estructura `cognitoAuthService` y `cognitoTokenService` están creadas como **placeholders**. Para activarlas completamente:

1.  **Implementar `ValidateToken` en `cognito_token_service.go`:**
    *   Usar una librería como `github.com/MicahParks/keyfunc` para descargar y cachear el JWKS de Cognito.
    *   Configurar el parser de JWT para usar la clave pública correspondiente al `kid` del header del token.

2.  **Implementar `Login` en `cognito_auth_service.go`:**
    *   Usar el SDK de AWS (`github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider`) para llamar a `InitiateAuth`.
    *   Mapear la respuesta de AWS a nuestro `LoginResponse`.

## 📚 Referencias

*   [Código: Auth Service Factory](file:///internal/services/auth/auth_service.go)
*   [Código: Cognito Placeholder](file:///internal/services/auth/cognito_auth_service.go)
*   [Documentación: Gestión de Secretos](configuration-secrets.md)
