# Instrucciones del Repositorio

## Documentacion por defecto

Cuando una solicitud implique documentar, redocumentar, reorganizar o ampliar documentacion del proyecto, la convencion por defecto es:

- Crear o actualizar documentacion para humanos en `doc/info/`.
- Crear o actualizar documentacion normativa para IA y Spec-Driven Development en `doc/specs/`.
- Mantener `README.md` como portal principal y actualizar sus enlaces cuando cambie el mapa documental.
- Actualizar `doc/info/README.md` y `doc/specs/README.md` cuando se agreguen documentos nuevos o cambie la clasificacion.

## Regla de separacion

- `doc/info/` explica contexto, uso, operacion, troubleshooting y entendimiento humano.
- `doc/specs/` define contratos, invariantes, acceptance criteria y reglas verificables.
- No duplicar la misma informacion completa en ambos lados: `info` explica y `specs` norman.

## Regla de ubicacion

- No crear `.md` documentales fuera de `doc/info/`, `doc/specs/` o `README.md` raiz.
- Excepciones permitidas: templates Markdown funcionales, o READMEs tecnicos muy locales que no formen parte del mapa global.

## Regla de trazabilidad

Toda nueva documentacion debe:

- enlazar documentos relacionados cuando corresponda,
- evitar duplicacion con documentos existentes,
- respetar la clasificacion por dominio,
- dejar claro si el archivo pertenece a humanos (`info`) o IA/SDD (`specs`).

## Convenciones tecnicas de servicios

Todo servicio nuevo o refactorizado debe estructurarse con:

- interface segregation,
- constructor injection,
- method receivers,
- implementacion concreta no exportada cuando no exista una razon fuerte para exportarla.

### Patron esperado

1. Definir una interface pequena y enfocada en el contrato del servicio.
2. Implementar una struct concreta, preferentemente no exportada.
3. Exponer un constructor `New...` que reciba dependencias explicitas.
4. Implementar la interface mediante method receivers sobre la struct concreta.

### Ejemplo de referencia

```go
type PaymentService interface {
    Process(ctx context.Context, amount float64) error
    Refund(ctx context.Context, id string) error
}

type paymentService struct {
    db     *sql.DB
    logger *slog.Logger
}

func NewPaymentService(db *sql.DB, logger *slog.Logger) PaymentService {
    return &paymentService{
        db:     db,
        logger: logger,
    }
}

func (s *paymentService) Process(ctx context.Context, amount float64) error {
    return nil
}

func (s *paymentService) Refund(ctx context.Context, id string) error {
    return nil
}
```

## Documentacion de base de datos

La base de datos debe mantener documentacion `info + specs` suficiente para:

- entender entidades, tablas y relaciones,
- identificar claves primarias, foraneas, pivotes e invariantes,
- mapear modelos GORM con migraciones SQL,
- permitir que en futuras solicitudes se pueda generar SQL a partir de la documentacion base sin relevar todo desde cero.

Cuando cambien tablas, relaciones, indices, enums o reglas de integridad, deben actualizarse:

- la documentacion humana de base de datos en `doc/info/`,
- la documentacion normativa de base de datos en `doc/specs/`,
- y sus enlaces en los indices documentales si corresponde.

## Documentacion del Makefile

El `Makefile` debe tener documentacion `info + specs` cuando no exista una cobertura canonica suficiente.

La documentacion del `Makefile` debe dejar claro:

- dominios de comandos,
- prerequisitos,
- efectos colaterales,
- comandos destructivos o sensibles,
- flujos principales de desarrollo, despliegue, datos y soporte.

## Referencias

- `doc/info/README.md`
- `doc/specs/README.md`
- `doc/specs/documentation-governance-spec.md`
- `doc/specs/documentation-defaults-spec.md`
- `doc/info/development/service-design-conventions.md`
- `doc/specs/architecture/service-design-spec.md`
