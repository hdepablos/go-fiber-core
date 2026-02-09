# 📦 Logger & Observability Package

Este paquete proporciona un sistema de logging estructurado, híbrido y orientado a la observabilidad para aplicaciones Go. Está diseñado para adaptarse automáticamente al entorno de ejecución (Local vs Cloud/AWS) y facilitar el seguimiento de métricas de rendimiento sin esfuerzo adicional.

## 🚀 Características Principales

*   **Adaptabilidad de Entorno**:
    *   **Local (`APP_ENV=local`)**: Escribe logs en archivos físicos (`pkg/logs/*.log`) con rotación automática (50MB, backups, compresión). Ideal para depuración sin ruido en consola.
    *   **Cloud (`APP_ENV=production` o cualquier otro)**: Escribe en `stdout` en formato JSON puro. Ideal para que **AWS CloudWatch**, Datadog o ELK capturen e indexen los logs automáticamente.
*   **Formato Estructurado (JSON)**: Todos los logs siguen un esquema estricto con metadatos de servicio, versión y contexto.
*   **Metrics Tracker**: Herramienta integrada para medir latencia de I/O (DB, HTTP), uso de memoria y éxito/fallo de operaciones.

## ⚙️ Configuración

El comportamiento se define mediante variables de entorno:

| Variable | Valor | Descripción |
| :--- | :--- | :--- |
| `APP_ENV` | `local` | Escribe en archivos rotativos en `./pkg/logs/`. |
| `APP_ENV` | `production`, `dev`, etc. | Escribe JSON en `os.Stdout` (Cloud Native). |
| `LOG_LEVEL` | `info` | (Opcional) Nivel mínimo de log (`debug`, `info`, `warn`, `error`). Default: `debug` en local, `info` en prod. |
| `SERVICE_NAME` | `mi-servicio` | (Opcional) Nombre del servicio inyectado en cada log. |
| `VERSION` | `1.0.0` | (Opcional) Versión del servicio inyectada en cada log. |
| `DEVELOPER` | `juan.perez` | (Opcional) Identificador del desarrollador (útil en entornos compartidos). |

---

## 📖 Guía de Implementación

### 1. Obtener una Instancia del Logger

Al inicio de tu servicio, handler o worker, obtén una instancia nombrada.
En modo local, este nombre será el prefijo del archivo de log (ej: `pkg/logs/payment-service-2026-02-09.log`).

```go
import "go-fiber-core/internal/logger"

// ... dentro de tu función o struct
log := logger.GetLogger("payment-service")
```

### 2. Logging Básico (Info, Error, Debug)

Usa los métodos estándar de `zap`. Es **CRÍTICO** usar el nivel correcto para no ensuciar CloudWatch y ahorrar costos.

*   **`log.Debug(...)`**: Solo para desarrollo. **Se ignora automáticamente en Producción**. Úsalo para trazas paso a paso ("entrando a función", "variable X vale Y").
*   **`log.Info(...)`**: Eventos de negocio importantes (Inicio de proceso, éxito de operación). Se guardan en Prod.
*   **`log.Error(...)`**: Fallos que requieren atención. Se guardan en Prod.

```go
// ✅ CORRECTO: Trazas de depuración (Solo se ven en Local)
log.Debug("Calculando impuestos...", 
    zap.Float64("subtotal", 100.50),
    zap.String("step", "init_calc"),
)

// ✅ CORRECTO: Evento de Negocio (Se ve en CloudWatch)
log.Info("Orden Creada Exitosamente", 
    zap.String("order_id", "ord-999"),
)

// ❌ INCORRECTO: No usar Info para depuración (Ensucia y cuesta dinero en AWS)
log.Info("Variable i vale 5") 
```

### 3. Seguimiento de Rendimiento (MetricsTracker)

Para operaciones críticas (endpoints API, consumidores SQS, Cron Jobs), usa `MetricsTracker` para medir automáticamente:
*   Tiempo total de ejecución.
*   Tiempo de lectura/escritura (DB, API externa).
*   Uso de memoria.
*   Conteo de registros procesados (éxito/fallo).

```go
func (s *Service) ProcesarLote(ctx context.Context) {
    // 1. Iniciar Tracker al principio de la función
    tracker := logger.NewMetricsTracker()
    
    // ... lógica de negocio ...
    
    // Medir una operación de lectura (ej. consulta DB)
    startDB := time.Now()
    users, err := db.GetUsers()
    tracker.StopRead(startDB) // Registra duración de lectura
    
    if err != nil {
        tracker.AddError("Fallo al obtener usuarios")
        tracker.RecordsFailed++
    } else {
        tracker.RecordsSuccess = len(users)
    }

    // 2. Al final, generar el log de resumen
    // Esto escribe una sola línea JSON con todas las métricas
    tracker.Log(log, "Fin de Procesamiento de Lote",
        // Helper para inyectar contexto estándar
        logger.ContextFields("req-123", "batch-001", "ProcessBatchHandler"),
    )
}
```

---

## 📊 Estructura del Log (Output)

Independientemente del entorno, el log final tendrá esta estructura JSON unificada:

```json
{
  "level": "INFO",
  "timestamp": "2026-02-09T15:04:05Z",
  "service": "go-fiber-core",
  "version": "1.0.0",
  "message": "Fin de Procesamiento de Lote",
  "caller": "services/payment.go:45",
  
  // Contexto (inyectado vía logger.ContextFields)
  "context": {
    "request_id": "req-123",
    "batch_id": "batch-001",
    "handler": "ProcessBatchHandler"
  },
  
  // Métricas de Rendimiento (generadas por MetricsTracker)
  "performance": {
    "total_duration_ms": 145,
    "db_read_ms": 12,
    "db_write_ms": 45,
    "memory_used_mb": 12.5,
    "goroutines": 8
  },
  
  // Resultados de Negocio (generados por MetricsTracker)
  "results": {
    "records_total": 50,
    "records_success": 49,
    "records_failed": 1,
    "error_details": ["Fallo al obtener usuarios"]
  }
}
```

## 🛠 Mejores Prácticas

1.  **No uses `fmt.Println`**: Siempre usa el logger. `fmt` no es estructurado y se pierde en CloudWatch.
2.  **Evita `log.Fatal`**: Usa `log.Error` y maneja el retorno de error elegantemente para no matar el servicio inesperadamente.
3.  **Contexto es Rey**: Siempre que sea posible, pasa IDs de request o batch en el campo `context` para poder trazar operaciones en todo el sistema.
