package utils

import (
	"errors"

	"gorm.io/gorm"
)

// GetRandomID recibe un modelo y devuelve un ID aleatorio existente en la tabla.
// Si no hay registros o hay un error, retorna 1.
func GetRandomID(db *gorm.DB, model any) int {
	// Obtenemos el nombre de la tabla del modelo usando reflexión
	stmt := &gorm.Statement{DB: db}
	if err := stmt.Parse(model); err != nil {
		// Si no se puede parsear el modelo, retornamos 1
		return 1
	}
	tableName := stmt.Schema.Table

	// Consulta SQL para obtener un ID aleatorio
	var randomID int
	query := db.Table(tableName).Select("id").Order("RANDOM()").Limit(1).Scan(&randomID)

	// Manejamos los posibles errores
	if query.Error != nil {
		if errors.Is(query.Error, gorm.ErrRecordNotFound) {
			// Si no se encontraron registros, retornamos 1
			return 1
		}
		// Si hay otro tipo de error, también retornamos 1
		return 1
	}

	// Si se obtuvo un ID aleatorio, lo retornamos
	if randomID > 0 {
		return randomID
	}

	// En caso de que algo falle inesperadamente, retornamos 1
	return 1
}
