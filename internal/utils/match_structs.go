package utils

import (
	"reflect"
)

// MatchStructs copia los valores de los campos coincidentes entre dos structs
func MatchStructs(src, dest any) {
	srcVal := reflect.ValueOf(src)
	destVal := reflect.ValueOf(dest)

	// Verificamos que ambos sean punteros a structs
	if srcVal.Kind() != reflect.Ptr || destVal.Kind() != reflect.Ptr {
		panic("Ambos parámetros deben ser punteros a structs")
	}

	srcVal = srcVal.Elem()
	destVal = destVal.Elem()

	// Iteramos sobre los campos del struct destino
	for i := 0; i < destVal.NumField(); i++ {
		fieldDest := destVal.Type().Field(i) // Información del campo
		fieldSrc := srcVal.FieldByName(fieldDest.Name)

		// Verificamos si el campo existe en el struct fuente y tienen el mismo tipo
		if fieldSrc.IsValid() && fieldSrc.Type() == fieldDest.Type {
			// Seteamos el valor del campo en el struct destino
			destVal.Field(i).Set(fieldSrc)
		}
	}
}
