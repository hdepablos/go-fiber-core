package pagination

import (
	"fmt"
	"go-fiber-core/internal/dtos"
	"log"
	"math"
	"reflect"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

// QueryModifier es un tipo de función para aplicar modificaciones personalizadas a la consulta.
type QueryModifier func(*gorm.DB) *gorm.DB

// ExtrasCalculator es un tipo de función para calcular datos adicionales a partir de la consulta.
type ExtrasCalculator func(query *gorm.DB) (map[string]any, error)

// PaginationService es ahora una struct genérica.
type PaginationService[T any] struct{}

// NewPaginationService es el constructor genérico.
func NewPaginationService[T any]() *PaginationService[T] {
	return &PaginationService[T]{}
}

// Execute es el método genérico principal para paginar resultados.
func (p *PaginationService[T]) Execute(db *gorm.DB, req dtos.PaginationRequest, modifier QueryModifier, extrasCalc ExtrasCalculator) (*dtos.PaginationResponse[T], error) {
	var totalRows int64
	var extras map[string]any
	var modelInstance T
	dateColumns := p.GetDateColumnsFromModel(modelInstance)

	query := db.Model(new(T))

	// 1. El modifier (con Preloads) se aplica primero
	if modifier != nil {
		query = modifier(query)
	}

	// 2. Se aplican los filtros del request
	query = p.ApplyFilters(query, req, dateColumns)

	// 3. Se calculan los extras (SUM, COUNT, etc.) sobre la consulta ya filtrada
	if extrasCalc != nil {
		var err error
		extrasQuery := query.Session(&gorm.Session{})
		extras, err = extrasCalc(extrasQuery)
		if err != nil {
			return nil, err
		}
	}

	// 4. Se cuenta el total de filas (con los mismos filtros)
	countQuery := query.Session(&gorm.Session{})
	if err := countQuery.Count(&totalRows).Error; err != nil {
		return nil, err
	}

	totalPages := 0
	if totalRows > 0 && req.RowsPerPage > 0 {
		totalPages = int(math.Ceil(float64(totalRows) / float64(req.RowsPerPage)))
	}
	if req.Page > totalPages && totalPages > 0 {
		req.Page = totalPages
	}

	if totalRows == 0 {
		return &dtos.PaginationResponse[T]{Data: []T{}, TotalRows: 0, TotalPages: 0, Page: 1, RowsPerPage: req.RowsPerPage, Extras: extras}, nil
	}

	// 5. Se decide qué estrategia de paginación usar
	const deferredJoinThreshold = 15
	if req.OptimizeWithKey != "" && req.Page > deferredJoinThreshold {
		// Paginación diferida (páginas altas)
		// Se pasa el 'modifier' para aplicar los Preloads (LA ÚLTIMA CORRECCIÓN)
		query = p.applyDeferredJoinPagination(db, query, req, modelInstance, modifier)
	} else {
		// Paginación estándar (páginas bajas)
		query = p.applyStandardPagination(query, req)
	}

	// 6. Se obtienen los datos de la página (ej: 15 registros)
	// GORM ejecutará los PRELOADS aquí, solo para estos 15 registros.
	var data []T
	if err := query.Find(&data).Error; err != nil {
		return nil, err
	}

	return &dtos.PaginationResponse[T]{Data: data, TotalRows: totalRows, TotalPages: totalPages, Page: req.Page, RowsPerPage: req.RowsPerPage, Extras: extras}, nil
}

// GetAllFiltered obtiene TODOS los registros que coinciden con los filtros, sin paginación.
// (Añadido para exportaciones o sumatorias en Go)
func (p *PaginationService[T]) GetAllFiltered(db *gorm.DB, req dtos.PaginationRequest, modifier QueryModifier) ([]T, error) {
	var modelInstance T
	dateColumns := p.GetDateColumnsFromModel(modelInstance)
	query := db.Model(new(T))
	if modifier != nil {
		query = modifier(query)
	}
	query = p.ApplyFilters(query, req, dateColumns)
	query = p.applySorting(query, req) // Usa la lógica de ordenamiento refactorizada
	var data []T
	if err := query.Find(&data).Error; err != nil {
		return nil, err
	}
	return data, nil
}

// ApplyFilters aplica los filtros de la solicitud a la consulta de GORM.
// (Sin cambios)
func (p *PaginationService[T]) ApplyFilters(db *gorm.DB, req dtos.PaginationRequest, dateColumns map[string]bool) *gorm.DB {
	if len(req.FilterBy) == 0 {
		return db
	}
	for i, filterBy := range req.FilterBy {
		if i >= len(req.FilterValues) {
			continue
		}
		filterValue := req.FilterValues[i]

		if strings.Contains(filterBy, "::") {
			parts := strings.SplitN(filterBy, "::", 2)
			finalColumnName := parts[0] + "." + parts[1]
			if values, ok := filterValue.([]any); ok {
				db = db.Where(finalColumnName+" IN ?", values)
			} else {
				db = db.Where(finalColumnName+" = ?", filterValue)
			}
			continue
		}

		if strings.Contains(filterBy, ".") && !strings.HasSuffix(filterBy, ":fuzzy") {
			lastDotIndex := strings.LastIndex(filterBy, ".")
			joinPath, finalFieldName := filterBy[:lastDotIndex], filterBy[lastDotIndex+1:]
			relations := strings.Split(joinPath, ".")

			var currentAlias string
			for _, rel := range relations {
				currentAlias = rel
			}

			gormRelations := map[string]bool{
				"profile": true,
				"offer":   true,
				"address": true,
			}

			var finalColumnName string

			if gormRelations[strings.ToLower(currentAlias)] {
				currentAlias = strings.ToUpper(currentAlias[:1]) + strings.ToLower(currentAlias[1:])
				finalColumnName = `"` + currentAlias + `".` + finalFieldName
			} else {
				finalColumnName = currentAlias + "." + finalFieldName
			}

			if values, ok := filterValue.([]any); ok {
				db = db.Where(finalColumnName+" IN ?", values)
			} else if val, ok := filterValue.(bool); ok {
				db = db.Where(finalColumnName+" = ?", val)
			} else if str, ok := filterValue.(string); ok {
				if p.isLike(str) {
					if db.Name() == "postgres" {
						db = db.Where(finalColumnName+" ILIKE ?", str)
					} else {
						db = db.Where("LOWER("+finalColumnName+") LIKE LOWER(?)", str)
					}
				} else {
					db = db.Where("LOWER("+finalColumnName+") = LOWER(?)", str)
				}
			} else if filterValue != nil {
				db = db.Where(finalColumnName+" = ?", filterValue)
			}

			continue
		}

		if _, isDate := dateColumns[filterBy]; isDate {
			if values, ok := filterValue.([]any); ok && len(values) == 2 {
				startStr, ok1 := values[0].(string)
				endStr, ok2 := values[1].(string)
				if ok1 && ok2 {
					start, err1 := time.Parse("2006-01-02", startStr)
					end, err2 := time.Parse("2006-01-02", endStr)
					if err1 == nil && err2 == nil {
						endOfDay := end.Add(24 * time.Hour)
						db = db.Where(filterBy+" >= ? AND "+filterBy+" < ?", start, endOfDay)
						continue
					}
				}
			}
		}
		if strings.HasSuffix(filterBy, ":fuzzy") {
			columnName := strings.TrimSuffix(filterBy, ":fuzzy")
			if db.Name() == "postgres" {
				db = db.Where(columnName+" % ?", filterValue)
			} else {
				if str, ok := filterValue.(string); ok {
					db = db.Where("LOWER("+columnName+") LIKE LOWER(?)", "%"+str+"%")
				}
			}
			continue
		}

		if values, ok := filterValue.([]any); ok {
			db = db.Where(filterBy+" IN ?", values)
		} else if val, ok := filterValue.(bool); ok {
			db = db.Where(filterBy+" = ?", val)
		} else if str, ok := filterValue.(string); ok {
			if p.isLike(str) {
				if db.Name() == "postgres" {
					db = db.Where(filterBy+" ILIKE ?", str)
				} else {
					db = db.Where("LOWER("+filterBy+") LIKE LOWER(?)", str)
				}
			} else {
				db = db.Where("LOWER("+filterBy+") = LOWER(?)", str)
			}
		} else if filterValue != nil {
			db = db.Where(filterBy+" = ?", filterValue)
		}
	}
	return db
}

// ApplyOrder aplica el ordenamiento a la consulta.
// (Sin cambios)
func (p *PaginationService[T]) ApplyOrder(db *gorm.DB, req dtos.PaginationRequest) *gorm.DB {
	if len(req.SortBy) > 0 {
		for i, sortBy := range req.SortBy {
			order := "ASC"
			if i < len(req.SortDesc) && req.SortDesc[i] {
				order = "DESC"
			}

			if strings.Contains(sortBy, "::") {
				db = db.Order(strings.Replace(sortBy, "::", ".", 1) + " " + order)
				continue
			}

			if strings.HasSuffix(sortBy, ":fuzzy") {
				columnName := strings.TrimSuffix(sortBy, ":fuzzy")
				if db.Name() == "postgres" {
					var matchingValue any
					for j, filterBy := range req.FilterBy {
						if filterBy == sortBy {
							if j < len(req.FilterValues) {
								matchingValue = req.FilterValues[j]
								break
							}
						}
					}
					if matchingValue != nil {
						db = db.Order(gorm.Expr(columnName+" <-> ? "+order, matchingValue))
					}
				} else {
					db = db.Order(columnName + " " + order)
				}
				continue
			}

			if strings.Contains(sortBy, ".") {
				lastDotIndex := strings.LastIndex(sortBy, ".")
				joinPath, finalFieldName := sortBy[:lastDotIndex], sortBy[lastDotIndex+1:]
				relations := strings.Split(joinPath, ".")

				var currentAlias string
				for _, rel := range relations {
					currentAlias = rel
				}

				gormRelations := map[string]bool{
					"profile": true,
					"offer":   true,
					"address": true,
				}

				if gormRelations[strings.ToLower(currentAlias)] {
					currentAlias = strings.ToUpper(currentAlias[:1]) + strings.ToLower(currentAlias[1:])
					db = db.Order(`"` + currentAlias + `".` + finalFieldName + " " + order)
				} else {
					db = db.Order(currentAlias + "." + finalFieldName + " " + order)
				}
			} else {
				db = db.Order(sortBy + " " + order)
			}
		}
	}
	return db
}

// GetDateColumnsFromModel extrae las columnas de fecha a partir de los tags del modelo.
// (Sin cambios)
func (p *PaginationService[T]) GetDateColumnsFromModel(model any) map[string]bool {
	dateColumns := make(map[string]bool)
	modelType := reflect.Indirect(reflect.ValueOf(model)).Type()
	namingStrategy := schema.NamingStrategy{}
	for i := 0; i < modelType.NumField(); i++ {
		field := modelType.Field(i)
		if tag, ok := field.Tag.Lookup("filter"); ok && tag == "date" {
			columnName := namingStrategy.ColumnName("", field.Name)
			dateColumns[columnName] = true
		}
	}
	return dateColumns
}

// applyLimitOffset es una función helper interna.
// (Sin cambios)
func (p *PaginationService[T]) applyLimitOffset(db *gorm.DB, req dtos.PaginationRequest) *gorm.DB {
	if req.Page > 0 && req.RowsPerPage > 0 {
		offset := (req.Page - 1) * req.RowsPerPage
		db = db.Limit(req.RowsPerPage).Offset(offset)
	}
	return db
}

// applySorting aplica solo el ordenamiento, incluyendo el fallback a PK.
// (Esta es la función REFACTORIZADA)
func (p *PaginationService[T]) applySorting(db *gorm.DB, req dtos.PaginationRequest) *gorm.DB {
	if len(req.SortBy) == 0 {
		var model T
		stmt := &gorm.Statement{DB: db}

		// Lógica mejorada para parsear el schema si no existe
		if stmt.Schema == nil {
			if err := stmt.Parse(db.Model(model)); err != nil {
				log.Printf("Advertencia: Falló el parseo del modelo para ordenamiento por defecto: %v", err)
				return db
			}
		}

		if stmt.Schema != nil && stmt.Schema.PrioritizedPrimaryField != nil {
			primaryKey := stmt.Schema.PrioritizedPrimaryField.DBName
			tableName := stmt.Schema.Table
			// Aplica el orden DESC por defecto (tu solicitud)
			db = db.Order(fmt.Sprintf(`"%s"."%s" DESC`, tableName, primaryKey))
		} else {
			log.Printf("Advertencia: No se pudo determinar la clave primaria para el ordenamiento por defecto del modelo %T", model)
		}
	} else {
		// Si el usuario pidió un orden, lo aplica
		db = p.ApplyOrder(db, req)
	}
	return db
}

// applyStandardPagination aplica orden, límite y offset.
// (Ahora usa el helper 'applySorting')
func (p *PaginationService[T]) applyStandardPagination(db *gorm.DB, req dtos.PaginationRequest) *gorm.DB {
	db = p.applySorting(db, req)
	return p.applyLimitOffset(db, req)
}

// applyDeferredJoinPagination optimiza la paginación buscando primero los IDs.
// (Ahora usa 'applySorting' y aplica el 'modifier' para los Preloads)
func (p *PaginationService[T]) applyDeferredJoinPagination(db *gorm.DB, query *gorm.DB, req dtos.PaginationRequest, model any, modifier QueryModifier) *gorm.DB {
	tableName := query.Statement.Table
	if tableName == "" {
		stmt := &gorm.Statement{DB: query}
		if err := stmt.Parse(model); err != nil {
			log.Printf("ERROR: Falló el parseo del modelo en paginación diferida: %v", err)
			return query.Where("1 = 0")
		}
		tableName = stmt.Schema.Table
	}
	var pageIDs []uint

	// 1. Busca solo los IDs de la página
	idQuery := query.Session(&gorm.Session{}).Select(req.OptimizeWithKey)
	idQuery = p.applyStandardPagination(idQuery, req) // Ordena y pagina los IDs
	idQuery.Find(&pageIDs)
	if len(pageIDs) == 0 {
		return query.Where("1 = 0")
	}

	whereCondition := fmt.Sprintf(`"%s"."%s" IN (?)`, tableName, req.OptimizeWithKey)

	// 2. Construye una nueva consulta limpia para buscar por esos IDs
	finalQuery := db.Model(model).Where(whereCondition, pageIDs)

	// 3. ¡LA CORRECCIÓN! Aplica el modifier (con Preloads) a esta nueva consulta
	if modifier != nil {
		finalQuery = modifier(finalQuery)
	}

	// 4. Aplica el ordenamiento (necesario para que coincida con el orden de los IDs)
	finalQuery = p.applySorting(finalQuery, req)

	return finalQuery
}

// isLike verifica si un valor es para una búsqueda con LIKE.
// (Sin cambios)
func (p *PaginationService[T]) isLike(value string) bool {
	return len(value) > 2 && strings.HasPrefix(value, "%") && strings.HasSuffix(value, "%")
}

// GetFilteredBatch usa Keyset Pagination ("seek method") para procesar grandes
// volúmenes de datos en lotes, evitando OOM y la lentitud de OFFSET.
// NOTA: Este método IGNORA el 'SortBy' del request y SIEMPRE ordena por PK ASC.
func (p *PaginationService[T]) GetFilteredBatch(
	db *gorm.DB,
	req dtos.PaginationRequest,
	modifier QueryModifier,
	batchSize int,
	lastProcessedID uint,
) ([]T, error) {

	var modelInstance T
	dateColumns := p.GetDateColumnsFromModel(modelInstance)

	query := db.Model(new(T))

	// 1. Aplicar modifier (para JOINS)
	if modifier != nil {
		query = modifier(query)
	}

	// 2. Aplicar los mismos filtros complejos de la paginación
	query = p.ApplyFilters(query, req, dateColumns)

	// 3. Obtener el nombre de la Clave Primaria (ej: "id")
	//    (Usamos la misma lógica que 'applySorting' pero simplificada)
	var pkColumn string
	stmt := &gorm.Statement{DB: db}
	if err := stmt.Parse(modelInstance); err == nil && stmt.Schema != nil && stmt.Schema.PrioritizedPrimaryField != nil {
		pkColumn = stmt.Schema.PrioritizedPrimaryField.DBName
	} else {
		// Fallback o error si no hay PK, aunque 'id' es estándar
		pkColumn = "id"
		log.Printf("Advertencia: No se pudo determinar la PK para GetFilteredBatch, usando 'id'. Modelo: %T", modelInstance)
	}

	// 4. Aplicar la lógica de Keyset Pagination (el "Seek Method")
	//    Esto es lo que te certificaron como una excelente práctica.
	if lastProcessedID > 0 {
		// Usamos el nombre de la tabla para desambiguar (ej: "products"."id")
		tableName := stmt.Schema.Table
		query = query.Where(fmt.Sprintf(`"%s"."%s" > ?`, tableName, pkColumn), lastProcessedID)
	}

	// 5. Aplicar orden y límite
	// Es OBLIGATORIO ordenar por la PK para que el Keyset funcione.
	tableName := stmt.Schema.Table
	query = query.Order(fmt.Sprintf(`"%s"."%s" ASC`, tableName, pkColumn))
	query = query.Limit(batchSize)

	// 6. Ejecutar y devolver el lote
	var data []T
	if err := query.Find(&data).Error; err != nil {
		return nil, err
	}

	return data, nil
}
