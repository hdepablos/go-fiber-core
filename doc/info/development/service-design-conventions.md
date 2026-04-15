# Convenciones de Diseño de Servicios

Este documento explica la convención técnica esperada para servicios nuevos o refactorizados dentro del proyecto.
Su objetivo es que el diseño sea consistente, fácil de testear y claro para mantenimiento futuro.

## Objetivo

Todo servicio debe estructurarse, por defecto, con:

- `interface segregation`
- `constructor injection`
- `method receivers`
- implementación concreta no exportada, salvo necesidad real de exportarla

## Patrón recomendado

### 1. Interface pequeña y enfocada

La interface debe exponer solo el contrato que el consumidor realmente necesita.

Buenas señales:

- pocos métodos
- responsabilidades acotadas
- nombres de método orientados al caso de uso

Malas señales:

- interfaces gigantes
- mezcla de lectura, escritura, coordinación y utilidades sin separación

### 2. Implementación concreta no exportada

La implementación concreta debería ser interna al paquete cuando no exista una razón fuerte para exponerla.

Esto permite:

- reducir acoplamiento con detalles internos
- cambiar implementación sin romper consumidores
- testear contra contrato y no contra struct concreta

### 3. Constructor `New...`

Toda dependencia relevante debe entrar por constructor:

- repositorios
- clientes externos
- logger
- config
- clock
- helpers compartidos si aplican

No conviene esconder dependencias globales si pueden inyectarse explícitamente.

### 4. Method receivers

Los métodos que implementan el contrato deben vivir sobre la struct concreta:

- `func (s *service) Execute(...) error`
- `func (s *service) Process(...) error`

Esto hace explícito el estado y las dependencias usadas por el servicio.

## Ejemplo base

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

## Beneficios

- facilita mocking y pruebas
- reduce dependencia en implementaciones concretas
- vuelve más explícita la composición de dependencias
- mejora legibilidad del paquete
- ayuda a evitar servicios “god object”

## Cuándo hacer excepciones

Puede haber excepciones cuando:

- el paquete define una implementación muy simple y sin consumidores múltiples
- existe una razón de performance o compatibilidad muy clara
- el framework o librería obliga a otra forma de construcción

Aun en esos casos, conviene preservar la intención:

- responsabilidades pequeñas
- dependencias explícitas
- separación entre contrato e implementación cuando aporte valor real

## Relación con el repositorio

Esta convención complementa:

- `AGENTS.md`
- `doc/specs/architecture/service-design-spec.md`
- `doc/info/development/create-services-steps.md`

Si se crea o refactoriza un servicio importante, esta guía debe considerarse la base humana de diseño.
