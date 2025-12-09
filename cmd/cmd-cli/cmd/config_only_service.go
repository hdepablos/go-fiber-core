package cmd

import (
	"encoding/json"
	"fmt"
	"go-fiber-core/internal/services/serviceconfig"
	"go-fiber-core/internal/services/serviceconfig/contracts"

	"github.com/spf13/cobra"
)

var serviceconfigonlyCmd = &cobra.Command{
	// Definimos que el comando recibe un argumento: el path del servicio.
	Use:   "serviceconfigonly [path]",
	Short: "Ejecuta un único servicio por su path de registro.",
	Long: `Este comando permite ejecutar un servicio específico de forma aislada.
Debes proporcionar el path completo con el que fue registrado.
Ejemplo: go run . serviceconfigonly loanrisk/NewAgeService`,
	// Validamos que se reciba exactamente un argumento.
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// El path del servicio es el primer argumento que recibimos.
		pathService := args[0]

		fmt.Printf("Solicitud para ejecutar el servicio: %s\n", pathService)

		// Creamos nuestro contexto de datos, igual que antes.
		ctx := contracts.NewServiceContext(45, 100000)

		// Llamamos a nuestra nueva función ExecuteService.
		err := serviceconfig.ExecuteService(pathService, ctx)
		if err != nil {
			// El error ya fue logueado dentro de ExecuteService, aquí solo salimos.
			return
		}

		// Mostramos el mapa de resultados del contexto.
		jsonBytes, _ := json.MarshalIndent(ctx.Results, "", "  ")
		fmt.Println("✅ Resultado Final:")
		fmt.Println(string(jsonBytes))
	},
}

func init() {
	rootCmd.AddCommand(serviceconfigonlyCmd)
}
