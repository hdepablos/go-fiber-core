package database

import "sort"

// ServiceRegistryRow representa cada fila de la tabla que contiene los servicios y su orden.
type ServiceRegistryRow struct {
	Path  string
	Order int
}

// Datos simulados de la base de datos
var mockData = []ServiceRegistryRow{
	{Path: "services/NewPadronService", Order: 1},
	{Path: "services/NewReverService", Order: 2},
	{Path: "services/NewReturnService", Order: 3},
	{Path: "services/NewMyService", Order: 4},
}

func GetServiceRegistry() ([]ServiceRegistryRow, error) {
	// Ordenar los registros por el campo `Order`
	sort.Slice(mockData, func(i, j int) bool {
		return mockData[i].Order < mockData[j].Order
	})
	return mockData, nil
}
