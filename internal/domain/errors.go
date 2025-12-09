package domain

import "errors"

var (
	ErrNotFound        = errors.New("el recurso solicitado no fue encontrado")
	ErrInvalidArgument = errors.New("argumento inválido")
	ErrInternal        = errors.New("ha ocurrido un error interno")
	ErrAuthentication  = errors.New("credenciales inválidas")

	// ErrCritical se usa para errores que deben detener inmediatamente la ejecución de la cadena.
	// Por ejemplo, una falla al procesar una transacción financiera.
	ErrCritical = errors.New("error crítico")

	// ErrTolerable se usa para errores que pueden ser registrados, pero que no deben
	// impedir que el resto de los servicios se ejecuten.
	// Por ejemplo, si un servicio opcional de enriquecimiento de datos falla.
	ErrTolerable = errors.New("error tolerable")

	// ... y otros errores de negocio que necesites
)
