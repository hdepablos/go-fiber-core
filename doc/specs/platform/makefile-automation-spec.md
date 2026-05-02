---
domain: platform
summary: Reglas para el Makefile como interfaz operativa del repositorio, su descubribilidad, clasificación, contexto de ejecución y trazabilidad documental.
when_to_read:
  - cambios en Makefile
  - nuevos targets de uso humano
  - nuevos comandos Go ejecutados via Makefile
  - comandos que abren conexiones a DB, Redis u otros servicios de infraestructura
  - cambios en list-scaffolds o list-tools
code_paths:
  - Makefile
  - cmd/tools/
related_info:
  - doc/info/platform/makefile-guide.md
  - doc/info/development/process-scaffold-and-cleanup.md
related_specs:
  - doc/specs/architecture/service-runtime-bootstrap-spec.md
  - doc/specs/platform/process-scaffold-cleanup-spec.md
status: active
---

# Makefile Automation Spec

## Objetivo

Formalizar el rol del `Makefile` como interfaz operativa del repositorio y establecer cómo deben documentarse y clasificarse sus objetivos.

## Alcance

Aplica al `Makefile` raíz del proyecto y a toda evolución de comandos usada para:

- desarrollo local,
- despliegue,
- infraestructura,
- base de datos,
- observabilidad,
- AWS/S3,
- Kubernetes,
- generación de código.

## Reglas

### 1. Clasificación

Todo target nuevo debe pertenecer a uno de estos dominios, salvo justificación explícita:

- setup
- redis
- scaffolding
- build y calidad
- desarrollo local
- lambda/localstack
- eks/híbrido
- logs/observabilidad
- s3/artefactos
- base de datos

### 2. Descubribilidad

- Todo target orientado a uso humano debe aparecer en `make help` con descripción clara.
- La descripción debe incluir intención y parámetros relevantes.
- Los comandos sensibles deben advertir su impacto.
- Los comandos de scaffold o generación reutilizable deben evaluarse también para aparecer en un catálogo específico como `make list-scaffolds`.
- Las utilidades operativas de uso humano frecuente deben evaluarse para aparecer en un catálogo resumido como `make list-tools`.
- Si un scaffold soporta opciones operativas importantes como `force=true`, el catálogo debe mostrarlas.
- Si un scaffold soporta modos funcionales relevantes, por ejemplo `mode=generic` o `mode=bulk_jobs`, el catálogo debe mostrarlos.
- Si un scaffold tiene variantes técnicas relevantes, por ejemplo `sequential`, `fanout` o `dispatch_pacing`, el catálogo debe mostrarlas o referenciarlas claramente.
- Si un scaffold expone parámetros funcionales de una variante técnica, por ejemplo `pacing_messages` o `pacing_interval`, el catálogo debe mostrarlos cuando sean parte del uso recomendado.
- Si existe una herramienta operativa para versionar o parchear procesos existentes, por ejemplo `add-process-pacing`, el catálogo debe mostrarla junto con su comando base y parámetros críticos.
- Si existe un comando genérico como `clone-process-version`, el catálogo debe diferenciar claramente ese comando de los wrappers de conveniencia como `add-process-pacing`.
- Si esos comandos pertenecen conceptualmente a una familia mayor, por ejemplo `batch-process`, el catálogo debe poder presentarlos como operaciones hijas en lugar de listarlos al mismo nivel.

### 3. Seguridad operativa

Los targets destructivos o riesgosos deben dejar claro si:

- borran infraestructura,
- eliminan datos,
- remueven archivos de S3,
- resetean migraciones,
- modifican variables o entorno efectivo.

### 4. Contexto de ejecucion

- Todo target que ejecute binarios Go, scripts o utilidades que abren conexiones a DB, Redis, colas u otros servicios debe declarar implicitamente o explicitamente su contexto de ejecucion esperado.
- Si la configuracion operativa del repositorio resuelve hosts de infraestructura por DNS interno de Docker Compose, por ejemplo `postgres`, `redis` o nombres equivalentes de servicio, el target debe ejecutarse dentro de ese contexto de red, por ejemplo via `$(DC_RUN)` u otro wrapper equivalente.
- No debe asumirse que un `go run ./cmd/tools/...` ejecutado desde host puede resolver los mismos hosts que un contenedor.
- Si un comando debe soportar ejecucion desde host, la implementacion o su documentacion deben dejar claro qué configuracion o variables locales necesita para no depender de DNS interno de Docker.
- Antes de agregar un target nuevo para una utilidad operativa conectada a infraestructura, se debe revisar si el comando comparte contexto con otros targets DB-aware existentes y reutilizar el mismo patrón de ejecucion.

### 5. Trazabilidad documental

- Si el `Makefile` cambia de forma relevante, debe revisarse `doc/info/platform/makefile-guide.md`.
- Si se agrega una nueva familia de comandos, debe actualizarse esta spec o una spec relacionada.
- Los comandos vinculados a otros dominios deben enlazarse con la documentación humana correspondiente cuando aplique.
- Si se agrega un comando tipo scaffold o generador reusable, debe revisarse `make list-scaffolds` y la documentación humana de scaffolds.
- Si se agrega una utilidad operativa de uso frecuente, debe evaluarse `make list-tools` y la documentación humana del Makefile.

### 6. Scaffold y cleanup de procesos

- Los comandos de scaffold y cleanup de procesos deben estar documentados en una guía humana específica.
- Debe existir una forma simple de descubrir scaffolds vigentes, por ejemplo `make list-scaffolds`.
- Debe existir una forma simple de descubrir utilidades operativas frecuentes, por ejemplo `make list-tools`.
- Si existe un comando para crear un tipo de proceso, debe existir una estrategia documentada para revertir ese scaffold.
- Los comandos destructivos de cleanup deben ofrecer una forma segura de inspección previa cuando aplique, por ejemplo `dry_run`.
- Los cambios que afecten la convención de Bruno deben reflejarse tanto en la documentación humana como en la spec correspondiente.

## Acceptance Criteria

- El equipo puede identificar el target correcto por dominio sin leer todo el `Makefile`.
- El equipo puede descubrir scaffolds vigentes sin relevar manualmente todo el `Makefile`.
- El equipo puede descubrir utilidades operativas frecuentes sin relevar manualmente todo el `Makefile`.
- Los objetivos peligrosos son distinguibles antes de ejecutarse.
- Los comandos conectados a infraestructura se ejecutan en un contexto compatible con los hosts y credenciales definidos por la configuración operativa.
- La documentación del `Makefile` permanece sincronizada con las familias principales de targets.
- Los comandos de scaffold y cleanup de procesos quedan trazados a documentación específica.

## Trazabilidad

- `AGENTS.md`
- `Makefile`
- `cmd/tools/`
- `doc/info/platform/makefile-guide.md`
- `doc/info/development/process-scaffold-and-cleanup.md`
- `doc/specs/architecture/service-runtime-bootstrap-spec.md`
- `doc/specs/platform/process-scaffold-cleanup-spec.md`
