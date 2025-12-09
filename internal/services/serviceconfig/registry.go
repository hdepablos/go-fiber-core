package serviceconfig

import (
	"fmt"
	"go-fiber-core/internal/services/serviceconfig/contracts"
	"log"
)

// ServiceFactory es un alias para un tipo de función: una que no recibe argumentos
// y devuelve un objeto que cumple con la interfaz 'contracts.Service'.
// Define la "forma" de nuestras funciones constructoras (ej. NewAgeService).
type ServiceFactory func() contracts.Service

// registry es nuestro mapa central que usará un 'string' como clave (la ruta del servicio)
// y almacenará una 'ServiceFactory' (la función constructora) como valor.
var registry = make(map[string]ServiceFactory)

// Register es la función que los servicios usarán en sus archivos 'init()'
// para inscribirse en el mapa 'registry'.
func Register(path string, factory ServiceFactory) {
	log.Printf("✒️ Registrando servicio: %s", path)
	registry[path] = factory
}

// GetServiceFactory busca en el mapa 'registry' usando la ruta.
// Si la encuentra, DEVUELVE LA FUNCIÓN que está almacenada en esa posición.
// No la ejecuta, solo la devuelve para que el executor pueda hacerlo.
func GetServiceFactory(path string) (ServiceFactory, error) {
	factory, ok := registry[path]
	if !ok {
		// Si no hay ninguna función registrada para esa ruta, devuelve un error.
		return nil, fmt.Errorf("servicio no encontrado en el registro: %s", path)
	}
	// Devuelve la función constructora encontrada.
	return factory, nil
}
