# Core Architecture Spec

## Objetivo

Formalizar las reglas base de arquitectura transversal del proyecto: autenticacion por provider, resolucion de secretos y estrategia de consistencia con Redis.

## Alcance

Aplica a los modulos y decisiones descritas en:

- `doc/info/architecture/authentication-providers.md`
- `doc/info/architecture/configuration-secrets.md`
- `doc/info/architecture/redis-locking-strategy.md`
- `doc/info/architecture/process-architecture-evolution.md`

## Reglas

### 1. Autenticacion por provider

- La seleccion del proveedor debe depender de configuracion, no de cambios de logica de negocio.
- Debe existir una implementacion default segura para entorno local.
- Las implementaciones cloud deben respetar los mismos contratos funcionales expuestos al resto del sistema.

### 2. Configuracion y secretos

- Toda resolucion de secretos debe abstraerse mediante provider.
- La aplicacion no debe asumir una unica fuente de secretos.
- Los valores dinamicos en configuracion deben poder resolverse sin acoplarse a un proveedor concreto.

### 3. Consistencia de cache

- Las lecturas concurrentes no deben repoblar datos obsoletos durante una escritura critica.
- Las escrituras criticas deben forzar bypass o invalidacion controlada de cache.
- El naming de keys de lock y data debe ser consistente y predecible.

## Acceptance Criteria

- Los cambios de provider no requieren alterar casos de uso consumidores.
- La estrategia de secretos soporta al menos entorno local y proveedor remoto.
- La estrategia de locking evita race conditions de lectura/escritura sobre cache.
