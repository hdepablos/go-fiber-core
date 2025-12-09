package serviceconfig

import (
	"errors"
	"fmt"
	"go-fiber-core/internal/domain"
	"go-fiber-core/internal/services/serviceconfig/contracts"
	"log"
	"sort"
)

// ServiceRegistryRow representa la estructura de la configuración
// que obtendríamos de nuestra base de datos.
type ServiceRegistryRow struct {
	Path  string
	Order int
}

// ExecuteServicesInOrder ejecuta una cadena completa de servicios (esto ya lo teníamos).
func ExecuteServicesInOrder(services []ServiceRegistryRow, ctx *contracts.ServiceContext) error {
	// ... (código existente, no es necesario cambiarlo)
	sort.Slice(services, func(i, j int) bool {
		return services[i].Order < services[j].Order
	})

	for _, serviceConfig := range services {
		fmt.Printf("\n▶️ Procesando servicio: %s (Orden: %d)\n", serviceConfig.Path, serviceConfig.Order)
		factory, err := GetServiceFactory(serviceConfig.Path)
		if err != nil {
			return fmt.Errorf("error al obtener la fábrica para %s: %w", serviceConfig.Path, err)
		}
		serviceInstance := factory()
		serviceInstance.Init(ctx, serviceConfig.Path)
		if err := serviceInstance.Execute(); err != nil {
			if errors.Is(err, domain.ErrCritical) {
				log.Printf("🔴 Error crítico en '%s'. Deteniendo la cadena. Error: %v", serviceConfig.Path, err)
				return err
			} else if errors.Is(err, domain.ErrTolerable) {
				log.Printf("⚠️ Error tolerable en '%s'. La ejecución continuará. Error: %v", serviceConfig.Path, err)
			} else {
				log.Printf("🛑 Error no clasificado (tratado como crítico) en '%s'. Deteniendo la cadena. Error: %v", serviceConfig.Path, err)
				return err
			}
		}
	}
	fmt.Println("\n✅ Cadena de servicios completada.")
	return nil
}

// --- ¡NUEVA FUNCIÓN! ---
// ExecuteService ejecuta un único servicio por su path.
func ExecuteService(path string, ctx *contracts.ServiceContext) error {
	fmt.Printf("▶️ Ejecutando servicio individual: %s\n", path)

	// 1. Obtiene la función constructora del registro.
	factory, err := GetServiceFactory(path)
	if err != nil {
		return fmt.Errorf("error al obtener la fábrica para %s: %w", path, err)
	}

	// 2. Ejecuta la función para crear una instancia del servicio.
	serviceInstance := factory()

	// 3. Inicializa y ejecuta el servicio.
	serviceInstance.Init(ctx, path)
	if err := serviceInstance.Execute(); err != nil {
		// A diferencia del ejecutor en cadena, aquí cualquier error es un fallo,
		// ya que solo estamos ejecutando una cosa.
		log.Printf("🚨 Error ejecutando el servicio '%s': %v", path, err)
		return err
	}

	fmt.Println("\n✅ Servicio ejecutado con éxito.")
	return nil
}
