package utils

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/go-playground/locales/es"
	ut "github.com/go-playground/universal-translator"
	validator "github.com/go-playground/validator/v10"
	es_translations "github.com/go-playground/validator/v10/translations/es"
	fiber "github.com/gofiber/fiber/v2"
)

// BlacklistChecker define el contrato para chequear listas negras.
// Esto permite la inyección de dependencias y desacopla el validador.
type BlacklistChecker interface {
	IsEntityCodeBlacklisted(code string) bool
}

var (
	validate *validator.Validate
	trans    ut.Translator
)

func init() {
	validate = validator.New()
}

// SetupValidator inicializa el validador con sus dependencias y traducciones.
// Debe ser llamado una vez al arrancar la aplicación.
func SetupValidator(blacklistService BlacklistChecker) {
	validate = validator.New()

	// Le enseñamos al validador a usar los nombres de las etiquetas 'json' en los errores.
	validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return ""
		}
		return name
	})

	// Configuramos el traductor a español.
	esLocale := es.New()
	uni := ut.New(esLocale, esLocale)
	trans, _ = uni.GetTranslator("es")

	// Registramos las traducciones por defecto en español (para 'required', 'email', etc.).
	if err := es_translations.RegisterDefaultTranslations(validate, trans); err != nil {
		panic(fmt.Sprintf("failed to register default translations: %v", err))
	}

	// Registramos nuestras reglas y mensajes personalizados.
	registerCustomValidations(blacklistService)
	registerCustomMessages()
}

// registerCustomValidations registra las reglas de validación personalizadas.
func registerCustomValidations(blacklistService BlacklistChecker) {
	if err := validate.RegisterValidation("not_in_bank_blacklist", func(fl validator.FieldLevel) bool {
		return !blacklistService.IsEntityCodeBlacklisted(fl.Field().String())
	}); err != nil {
		panic(fmt.Sprintf("failed to register custom validation: %v", err))
	}
}

// registerCustomMessages registra los mensajes de error personalizados en español.
func registerCustomMessages() {
	// Mensaje para nuestra regla personalizada 'not_in_bank_blacklist'.
	_ = validate.RegisterTranslation("not_in_bank_blacklist", trans, func(ut ut.Translator) error {
		return ut.Add("not_in_bank_blacklist", "El código de entidad ingresado no está permitido.", true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		t, _ := ut.T("not_in_bank_blacklist", fe.Field())
		return t
	})

	// Puedes sobrescribir otros mensajes por defecto si lo necesitas. Por ejemplo, para 'min':
	_ = validate.RegisterTranslation("min", trans, func(ut ut.Translator) error {
		return ut.Add("min", "El campo {0} debe tener al menos {1} caracteres.", true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		t, _ := ut.T("min", fe.Field(), fe.Param())
		return t
	})
}

// validateRequest ejecuta la validación y formatea los errores.
func validateRequest(req interface{}) (map[string][]string, bool) {
	err := validate.Struct(req)
	if err != nil {
		validationErrors := make(map[string][]string)
		for _, err := range err.(validator.ValidationErrors) {
			fieldName := err.Field()
			// Usamos el traductor para obtener el mensaje de error en español.
			message := err.Translate(trans)
			validationErrors[fieldName] = append(validationErrors[fieldName], message)
		}
		return validationErrors, false
	}
	return nil, true
}

// Validate es el middleware de Fiber que se usa en las rutas.
func Validate(req interface{}) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Creamos una nueva instancia del tipo de 'req' para cada petición
		// para evitar 'race conditions' con peticiones concurrentes.
		reqPointer := reflect.New(reflect.TypeOf(req).Elem()).Interface()

		if err := c.BodyParser(reqPointer); err != nil {
			// Usamos el helper de respuesta para estandarizar el error.
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"status":  "error",
				"message": "Cuerpo de la petición inválido o malformado.",
			})
		}

		if errors, ok := validateRequest(reqPointer); !ok {
			// Usamos el helper de respuesta para estandarizar el error de validación.
			return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
				"status":  "error",
				"message": "Los datos proporcionados no son válidos.",
				"errors":  errors,
			})
		}

		c.Locals("req", reqPointer)
		return c.Next()
	}
}
