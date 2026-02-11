# Project go-fiber-core

One Paragraph of project description goes here

## Getting Started

These instructions will get you a copy of the project up and running on your local machine for development and testing purposes. See deployment for notes on how to deploy the project on a live system.

## MakeFile

Run build make command with tests
```bash
make all
```

## Configuración de Logs de Base de Datos

El sistema permite controlar el nivel de detalle de los logs de base de datos (GORM) mediante variables de entorno, sin necesidad de recompilar.

### Variables de Entorno

| Variable | Descripción | Valores Posibles | Default |
|----------|-------------|------------------|---------|
| `DB_LOG_LEVEL` | Controla qué se imprime en la consola. | `silent`, `error`, `warn`, `info` | Depende de `APP_ENV` (info en local, silent en prod) |
| `DB_SLOW_THRESHOLD_MS` | Define el umbral para considerar una query como "lenta". | Número entero (milisegundos) | `1000` (1 segundo) |

### Niveles de Log (`DB_LOG_LEVEL`)

1.  **`silent`**
    *   **Comportamiento:** No imprime nada en la consola.
    *   **Uso:** Recomendado para producción si se busca minimizar ruido y costos de ingestión de logs.

2.  **`error`**
    *   **Comportamiento:** Solo imprime errores de ejecución SQL.
    *   **Uso:** Recomendado estándar para producción. Permite ver fallos sin exponer datos sensibles de queries exitosas.

3.  **`warn`**
    *   **Comportamiento:** Imprime errores **Y** consultas lentas.
    *   **Uso:** Ideal para optimización en producción. Permite detectar cuellos de botella que superen el `DB_SLOW_THRESHOLD_MS`.

4.  **`info`**
    *   **Comportamiento:** Imprime **TODAS** las consultas SQL ejecutadas, sus parámetros y tiempo de ejecución.
    *   **Uso:** Desarrollo local. **NO** usar en producción (impacto en rendimiento y seguridad).

### Ejemplo de Configuración para Debugging de Performance

Para detectar queries que tardan más de 200ms en producción sin llenar los logs con todo el tráfico:

```bash
DB_LOG_LEVEL=warn
DB_SLOW_THRESHOLD_MS=200
```

## Infraestructura y Optimización

El proyecto utiliza configuraciones avanzadas para maximizar el rendimiento y minimizar costos en AWS Lambda:

*   **Arquitectura:** ARM64 (Graviton2)
*   **Memoria:** 1769 MB (1 vCPU completo)
*   **Logs:** Métricas de CPU y Goroutines integradas.

Para más detalles, consulta la documentación completa en: [doc/info/lambda-optimization.md](doc/info/lambda-optimization.md).

## Build the application
```bash
make build
```

Run the application
```bash
make run
```
Create DB container
```bash
make docker-run
```

Shutdown DB Container
```bash
make docker-down
```

DB Integrations Test:
```bash
make itest
```

Live reload the application:
```bash
make watch
```

Run the test suite:
```bash
make test
```

Clean up binary from the last build:
```bash
make clean
```
