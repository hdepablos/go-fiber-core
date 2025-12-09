package contracts

// Este paquete ahora define todos los contratos y tipos de datos compartidos,
// y no depende de ningún otro paquete interno, rompiendo así el ciclo.

// ServiceContext es el objeto que transportará los datos a través de la cadena de servicios.
type ServiceContext struct {
	Age     int            `json:"age"`
	Salary  int            `json:"salary"`
	Results map[string]any `json:"results"`
}

// NewServiceContext es un constructor para inicializar el contexto cómodamente.
func NewServiceContext(age, salary int) *ServiceContext {
	return &ServiceContext{
		Age:     age,
		Salary:  salary,
		Results: make(map[string]any),
	}
}

// Service define el contrato que todo servicio debe cumplir.
type Service interface {
	// Init ahora usa el ServiceContext definido en este mismo paquete.
	Init(ctx *ServiceContext, servicePath string)
	Execute() error
}
