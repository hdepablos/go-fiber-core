package serviceconfig

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go-fiber-core/internal/domain"
	"go-fiber-core/internal/services/dispatcher"
	"go-fiber-core/internal/services/serviceconfig/contracts"
	"log"
	"sort"
	"time"

	"golang.org/x/sync/errgroup"
)

type ServiceRegistryRow struct {
	Path           string
	Order          int
	ErrorTolerance string
	Config         []byte
	RequiredKeys   []string
	Timeout        time.Duration // Nuevo campo para timeout
}

func ExecuteServicesInOrder(ctx context.Context, services []ServiceRegistryRow, svcCtx *contracts.ServiceContext) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if svcCtx != nil {
		svcCtx.Ctx = ctx
	}
	// Sort services by order
	sort.Slice(services, func(i, j int) bool {
		return services[i].Order < services[j].Order
	})

	// Agrupar servicios por Orden
	grouped := make(map[int][]ServiceRegistryRow)
	var orders []int
	for _, s := range services {
		if _, exists := grouped[s.Order]; !exists {
			orders = append(orders, s.Order)
		}
		grouped[s.Order] = append(grouped[s.Order], s)
	}
	sort.Ints(orders) // Asegurar orden ascendente de grupos

	for _, order := range orders {
		groupRows := grouped[order]

		// Verificar cancelación del contexto antes de iniciar el grupo
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Ejecución paralela para el grupo usando errgroup
		g, groupCtx := errgroup.WithContext(ctx)

		for _, serviceConfig := range groupRows {
			sc := serviceConfig // Capturar variable para la goroutine
			g.Go(func() error {
				return executeOneService(groupCtx, sc, svcCtx)
			})
		}

		// Esperar a que todos los servicios del grupo terminen
		// Si alguno devuelve un error NO tolerable, g.Wait() lo retornará y detendrá la cadena.
		if err := g.Wait(); err != nil {
			return err
		}

		if svcCtx != nil {
			if v, ok := svcCtx.GetInputValue("__stop_chain"); ok {
				if b, ok := v.(bool); ok && b {
					return nil
				}
			}
		}
	}

	fmt.Println("\n✅ Cadena de servicios completada.")
	return nil
}

func executeOneService(ctx context.Context, serviceConfig ServiceRegistryRow, svcCtx *contracts.ServiceContext) error {
	// Manejo de Timeout específico para este paso
	var cancel context.CancelFunc
	if serviceConfig.Timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, serviceConfig.Timeout)
		defer cancel()
	}

	// Verificar contexto
	select {
	case <-ctx.Done():
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			err := fmt.Errorf("timeout exceeded (%v): %w", serviceConfig.Timeout, ctx.Err())
			return handleExecutionError(serviceConfig, err)
		}
		return ctx.Err()
	default:
	}

	fmt.Printf("\n▶️ Procesando servicio: %s (Orden: %d)\n", serviceConfig.Path, serviceConfig.Order)

	factory, err := GetServiceFactory(serviceConfig.Path)
	if err != nil {
		return fmt.Errorf("error al obtener la fábrica para %s: %w", serviceConfig.Path, err)
	}

	serviceInstance := factory()

	// Preparar Configuración
	var cfg map[string]any
	if len(serviceConfig.Config) > 0 {
		if err := json.Unmarshal(serviceConfig.Config, &cfg); err != nil {
			// Error de configuración suele ser crítico
			return fmt.Errorf("error config unmarshal en '%s': %w", serviceConfig.Path, err)
		}
	}

	// Inicializar Servicio de forma segura (thread-safe)
	if svcCtx != nil {
		svcCtx.InitService(serviceInstance, serviceConfig.Path, cfg)
	} else {
		serviceInstance.Init(nil, serviceConfig.Path)
	}

	// --- LÓGICA DE EXECUTION POLICY ---
	var policy contracts.ExecutionPolicy
	if rawPolicy, ok := cfg["execution_policy"]; ok {
		policyBytes, _ := json.Marshal(rawPolicy)
		_ = json.Unmarshal(policyBytes, &policy)
	}

	// Si el modo es ASYNC, despachamos a cola y terminamos este paso
	if policy.Mode == "ASYNC" {
		// Usar el dispatcher centralizado
		if err := dispatcher.DefaultDispatcher.DispatchStep(ctx, serviceConfig.Path, serviceConfig.Order, policy, svcCtx); err != nil {
			return fmt.Errorf("dispatch failed: %w", err)
		}

		res := contracts.StepResult{
			Status:  "pending",
			Message: fmt.Sprintf("Step dispatched to queue: %s", policy.QueueTarget),
		}
		if svcCtx != nil {
			svcCtx.SetResult(serviceConfig.Path, res)
		}
		return nil // Terminamos exitosamente el despacho
	}

	// Ejecutar Servicio (Modo SYNC por defecto)
	var execErr error
	if len(serviceConfig.RequiredKeys) > 0 && svcCtx != nil {
		// ... (código existente de required keys) ...
		var missing []string
		for _, key := range serviceConfig.RequiredKeys {
			if _, ok := svcCtx.GetInputValue(key); !ok {
				missing = append(missing, key)
			}
		}
		if len(missing) > 0 {
			execErr = fmt.Errorf("%w: claves faltantes %v para el servicio '%s'", domain.ErrMissingRequiredKey, missing, serviceConfig.Path)
		}
	}

	if execErr == nil {
		// INICIO MEDICIÓN TIEMPO (Solo si hay métricas activas en contexto)
		var startTime time.Time
		var isTrackingTime bool
		if svcCtx != nil && svcCtx.Metrics != nil {
			startTime = time.Now()
			isTrackingTime = true
		}

		// Usamos un canal para respetar la cancelación del contexto si el servicio se bloquea
		done := make(chan error, 1)
		go func() {
			defer func() {
				if r := recover(); r != nil {
					done <- fmt.Errorf("panic en servicio '%s': %v", serviceConfig.Path, r)
				}
			}()
			done <- serviceInstance.Execute()
		}()

		select {
		case <-ctx.Done():
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				execErr = fmt.Errorf("timeout exceeded (%v): %w", serviceConfig.Timeout, ctx.Err())
			} else {
				execErr = ctx.Err()
			}
		case err := <-done:
			execErr = err
		}

		// FIN MEDICIÓN TIEMPO E INYECCIÓN
		if isTrackingTime && svcCtx != nil {
			// Usamos Microsegundos para mayor precisión en steps rápidos (lógica pura)
			duration := time.Since(startTime).Microseconds()

			// Recuperamos el resultado actual (que el servicio pudo haber seteado)
			// O creamos uno nuevo si falló pero queremos registrar el tiempo
			res, ok := svcCtx.GetResult(serviceConfig.Path)
			if !ok {
				// Si el servicio no guardó nada (ej: error), creamos un placeholder
				res = contracts.StepResult{
					Status: "failed", // Asumimos failed si no hay result, se sobreescribirá si execErr es nil
				}
				if execErr == nil {
					res.Status = "completed"
				}
			}

			// Inyectamos duration_us en Data
			if res.Data == nil {
				res.Data = make(map[string]any)
			}
			res.Data["duration_us"] = duration

			// Guardamos de vuelta
			svcCtx.SetResult(serviceConfig.Path, res)
		}
	}

	if execErr != nil {
		return handleExecutionError(serviceConfig, execErr)
	}

	// Éxito: Guardar Resultado
	// Nota: SetResult usa Mutex internamente, es seguro.
	if svcCtx != nil {
		if res, ok := svcCtx.GetResult(serviceConfig.Path); ok {
			res.StepOrder = serviceConfig.Order
			svcCtx.SetResult(serviceConfig.Path, res)
		}
	}
	return nil
}

func handleExecutionError(serviceConfig ServiceRegistryRow, err error) error {
	// Si está configurado explícitamente como tolerable, ignoramos cualquier error
	if serviceConfig.ErrorTolerance == "tolerable" {
		log.Printf("⚠️ Error tolerable (config) en '%s'. La ejecución continuará. Error: %v", serviceConfig.Path, err)
		return nil // Retornar nil para que errgroup no cancele el grupo
	}

	// Si el error es de dominio ErrTolerable (comportamiento legacy/interno)
	if errors.Is(err, domain.ErrTolerable) {
		// Si la config dice explícitamente critical, lo respetamos
		if serviceConfig.ErrorTolerance == "critical" {
			log.Printf("🔴 Error tolerable tratado como crítico (config) en '%s'. Deteniendo. Error: %v", serviceConfig.Path, err)
			return err
		}
		log.Printf("⚠️ Error tolerable (domain) en '%s'. La ejecución continuará. Error: %v", serviceConfig.Path, err)
		return nil
	}

	// Por defecto: Error Crítico
	log.Printf("🔴 Error en '%s'. Deteniendo la cadena. Error: %v", serviceConfig.Path, err)
	return err
}

func ExecuteService(ctx context.Context, path string, svcCtx *contracts.ServiceContext) error {
	row := ServiceRegistryRow{
		Path:  path,
		Order: 1,
	}
	return ExecuteServicesInOrder(ctx, []ServiceRegistryRow{row}, svcCtx)
}
