package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"go-fiber-core/internal/services/serviceconfig"
	"go-fiber-core/internal/services/serviceconfig/contracts"
	"log"

	"github.com/spf13/cobra"

	// ¡IMPORTANTE! Esta importación en blanco asegura que el paquete loanrisk
	// se incluya en la compilación, lo que permite que sus funciones init()
	// se ejecuten y registren los servicios.
	_ "go-fiber-core/internal/services/loanrisk"
)

// fetchServiceConfigFromDB simula la obtención de la configuración desde la BD.
func fetchServiceConfigFromDB() []serviceconfig.ServiceRegistryRow {
	fmt.Println("🗄️ Obteniendo configuración de servicios desde la base de datos...")
	return []serviceconfig.ServiceRegistryRow{
		{
			Path:         "loanrisk/NewAgeService",
			Order:        1,
			Config:       []byte(`{"min_age":40,"required_keys":["age"]}`),
			RequiredKeys: []string{"age"},
		},
		{
			Path:   "loanrisk/NewValidationService",
			Order:  2,
			Config: []byte(`{}`),
		},
		{
			Path:         "loanrisk/NewSalaryService",
			Order:        3,
			Config:       []byte(`{"min_salary":2500000,"required_keys":["salary"]}`),
			RequiredKeys: []string{"salary"},
		},
		{
			Path:   "loanrisk/NewIsRenovationService",
			Order:  4,
			Config: []byte(`{}`),
		},
		{
			Path:   "loanrisk/NewRiskLevelService",
			Order:  5,
			Config: []byte(`{}`),
		},
	}
}

var serviceconfigallCmd = &cobra.Command{
	Use:   "serviceconfigall",
	Short: "Ejecuta una secuencia de servicios con manejo de errores avanzado.",
	Run: func(_ *cobra.Command, _ []string) {
		services := fetchServiceConfigFromDB()

		fmt.Println("\n=============================================")
		fmt.Println("🚀 INICIANDO CASO 1: Éxito con Error Tolerable")
		fmt.Println("=============================================")
		ctxSuccess := contracts.NewServiceContext(50, 100000)
		err := serviceconfig.ExecuteServicesInOrder(context.Background(), services, ctxSuccess)
		if err != nil {
			log.Printf("🚨 El Caso 1 finalizó con un error inesperado: %v", err)
		} else {
			jsonBytes, _ := json.MarshalIndent(ctxSuccess.Results, "", "  ")
			fmt.Println("\n✅ Resultado Final del Caso 1:")
			fmt.Println(string(jsonBytes))
		}

		fmt.Println("\n==========================================")
		fmt.Println("🚀 INICIANDO CASO 2: Fallo con Error Crítico")
		fmt.Println("==========================================")
		ctxFailure := contracts.NewServiceContext(45, 0)
		err = serviceconfig.ExecuteServicesInOrder(context.Background(), services, ctxFailure)
		if err != nil {
			log.Printf("✅ El Caso 2 finalizó correctamente con el error crítico esperado.")
		}

		fmt.Println("\n--- Fin de la simulación ---")
	},
}

// init añade nuestro nuevo comando al comando raíz.
func init() {
	rootCmd.AddCommand(serviceconfigallCmd)
}
