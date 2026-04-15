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

### 3. Seguridad operativa

Los targets destructivos o riesgosos deben dejar claro si:

- borran infraestructura,
- eliminan datos,
- remueven archivos de S3,
- resetean migraciones,
- modifican variables o entorno efectivo.

### 4. Trazabilidad documental

- Si el `Makefile` cambia de forma relevante, debe revisarse `doc/info/platform/makefile-guide.md`.
- Si se agrega una nueva familia de comandos, debe actualizarse esta spec o una spec relacionada.
- Los comandos vinculados a otros dominios deben enlazarse con la documentación humana correspondiente cuando aplique.

## Acceptance Criteria

- El equipo puede identificar el target correcto por dominio sin leer todo el `Makefile`.
- Los objetivos peligrosos son distinguibles antes de ejecutarse.
- La documentación del `Makefile` permanece sincronizada con las familias principales de targets.

## Trazabilidad

- `AGENTS.md`
- `Makefile`
- `doc/info/platform/makefile-guide.md`
