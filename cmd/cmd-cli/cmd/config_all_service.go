package cmd

import (
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
		{Path: "loanrisk/NewAgeService", Order: 1},
		{Path: "loanrisk/NewValidationService", Order: 2},
		{Path: "loanrisk/NewSalaryService", Order: 3},
		{Path: "loanrisk/NewIsRenovationService", Order: 4},
		{Path: "loanrisk/NewRiskLevelService", Order: 5},
	}
}

var serviceconfigallCmd = &cobra.Command{
	Use:   "serviceconfigall",
	Short: "Ejecuta una secuencia de servicios con manejo de errores avanzado.",
	Run: func(_ *cobra.Command, _ []string) {
		services := fetchServiceConfigFromDB()

		// --- CASO 1: ÉXITO (con error tolerable) ---
		fmt.Println("\n=============================================")
		fmt.Println("🚀 INICIANDO CASO 1: Éxito con Error Tolerable")
		fmt.Println("=============================================")
		// Usamos una edad de 50 para activar el error tolerable
		ctxSuccess := contracts.NewServiceContext(50, 100000)
		err := serviceconfig.ExecuteServicesInOrder(services, ctxSuccess)
		if err != nil {
			// Este bloque no debería ejecutarse si solo hay errores tolerables.
			log.Printf("🚨 El Caso 1 finalizó con un error inesperado: %v", err)
		} else {
			jsonBytes, _ := json.MarshalIndent(ctxSuccess.Results, "", "  ")
			fmt.Println("\n✅ Resultado Final del Caso 1:")
			fmt.Println(string(jsonBytes))
		}

		// --- CASO 2: FALLO (con error crítico) ---
		fmt.Println("\n==========================================")
		fmt.Println("🚀 INICIANDO CASO 2: Fallo con Error Crítico")
		fmt.Println("==========================================")
		// Usamos un salario de 0 para activar el error crítico
		ctxFailure := contracts.NewServiceContext(45, 0)
		err = serviceconfig.ExecuteServicesInOrder(services, ctxFailure)
		if err != nil {
			// Este bloque SÍ debería ejecutarse.
			log.Printf("✅ El Caso 2 finalizó correctamente con el error crítico esperado.")
			// Observa que los servicios posteriores al de Salario no se ejecutaron.
		}

		fmt.Println("\n--- Fin de la simulación ---")
	},
}

// init añade nuestro nuevo comando al comando raíz.
func init() {
	rootCmd.AddCommand(serviceconfigallCmd)
}
