package serviceconfig

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go-fiber-core/internal/domain"
	"go-fiber-core/internal/services/serviceconfig/contracts"
	"log"
	"sort"
)

type ServiceRegistryRow struct {
	Path           string
	Order          int
	ErrorTolerance string
	Config         []byte
	RequiredKeys   []string
}

func ExecuteServicesInOrder(ctx context.Context, services []ServiceRegistryRow, svcCtx *contracts.ServiceContext) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if svcCtx != nil {
		svcCtx.Ctx = ctx
	}
	sort.Slice(services, func(i, j int) bool {
		return services[i].Order < services[j].Order
	})

	for _, serviceConfig := range services {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		fmt.Printf("\n▶️ Procesando servicio: %s (Orden: %d)\n", serviceConfig.Path, serviceConfig.Order)
		factory, err := GetServiceFactory(serviceConfig.Path)
		if err != nil {
			return fmt.Errorf("error al obtener la fábrica para %s: %w", serviceConfig.Path, err)
		}
		serviceInstance := factory()
		if svcCtx != nil {
			svcCtx.CurrentStepConfig = nil
			if len(serviceConfig.Config) > 0 {
				var cfg map[string]any
				if err := json.Unmarshal(serviceConfig.Config, &cfg); err == nil {
					svcCtx.CurrentStepConfig = cfg
				}
			}
		}
		serviceInstance.Init(svcCtx, serviceConfig.Path)
		var execErr error
		if len(serviceConfig.RequiredKeys) > 0 && svcCtx != nil {
			for _, key := range serviceConfig.RequiredKeys {
				if _, ok := svcCtx.GetInputValue(key); !ok {
					execErr = fmt.Errorf("missing required key '%s' for service '%s': %w", key, serviceConfig.Path, domain.ErrCritical)
					break
				}
			}
		}
		if execErr == nil {
			execErr = serviceInstance.Execute()
		}
		if execErr != nil {
			if errors.Is(execErr, domain.ErrCritical) {
				log.Printf("🔴 Error crítico en '%s'. Deteniendo la cadena. Error: %v", serviceConfig.Path, execErr)
				return execErr
			}
			if errors.Is(execErr, domain.ErrTolerable) {
				switch serviceConfig.ErrorTolerance {
				case "critical":
					log.Printf("🔴 Error tolerable tratado como crítico en '%s' por configuración. Deteniendo la cadena. Error: %v", serviceConfig.Path, execErr)
					return execErr
				case "tolerable", "inherit", "":
					log.Printf("⚠️ Error tolerable en '%s'. La ejecución continuará. Error: %v", serviceConfig.Path, execErr)
					continue
				default:
					log.Printf("🛑 Error tolerable con configuración desconocida en '%s'. Deteniendo la cadena. Error: %v", serviceConfig.Path, execErr)
					return execErr
				}
			}
			switch serviceConfig.ErrorTolerance {
			case "tolerable":
				log.Printf("⚠️ Error tolerable por configuración en '%s'. La ejecución continuará. Error: %v", serviceConfig.Path, execErr)
				continue
			case "critical":
				log.Printf("🔴 Error crítico por configuración en '%s'. Deteniendo la cadena. Error: %v", serviceConfig.Path, execErr)
				return execErr
			default:
				log.Printf("🛑 Error no clasificado en '%s'. Deteniendo la cadena. Error: %v", serviceConfig.Path, execErr)
				return execErr
			}
		}
	}
	fmt.Println("\n✅ Cadena de servicios completada.")
	return nil
}

func ExecuteService(ctx context.Context, path string, svcCtx *contracts.ServiceContext) error {
	row := ServiceRegistryRow{
		Path:  path,
		Order: 1,
	}
	return ExecuteServicesInOrder(ctx, []ServiceRegistryRow{row}, svcCtx)
}
