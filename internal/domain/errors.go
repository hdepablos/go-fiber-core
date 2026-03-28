package domain

import "errors"

var (
	ErrNotFound        = errors.New("el recurso solicitado no fue encontrado")
	ErrInvalidArgument = errors.New("argumento inválido")
	ErrInternal        = errors.New("ha ocurrido un error interno")
	ErrAuthentication  = errors.New("credenciales inválidas")
	ErrNoMenuAccess    = errors.New("Usted no está asociado a ningún items")

	// ErrCritical se usa para errores que deben detener inmediatamente la ejecución de la cadena.
	// Por ejemplo, una falla al procesar una transacción financiera.
	ErrCritical = errors.New("error crítico")

	// ErrTolerable se usa para errores que pueden ser registrados, pero que no deben
	// impedir que el resto de los servicios se ejecuten.
	// Por ejemplo, si un servicio opcional de enriquecimiento de datos falla.
	ErrTolerable = errors.New("error tolerable")

	// ErrMissingRequiredKey se usa cuando falta una key obligatoria en el input.
	ErrMissingRequiredKey = errors.New("falta una clave requerida en el input")

	// ErrValueOutOfRange se usa cuando un valor está fuera de los rangos permitidos.
	ErrValueOutOfRange = errors.New("valor fuera de rango permitido")

	// ErrBusinessRuleViolation se usa cuando una regla de negocio específica no se cumple.
	// Por ejemplo: "Edad insuficiente", "Saldo insuficiente", "Usuario bloqueado".
	ErrBusinessRuleViolation = errors.New("violación de regla de negocio")

	// Errores de existencia referencial
	ErrSedeNotFound            = errors.New("la sede especificada no existe")
	ErrRoadmapNotFound         = errors.New("el roadmap especificado no existe")
	ErrOverrideVersionNotFound = errors.New("la versión de proceso específica no existe")

	// ... y otros errores de negocio que necesites
)
