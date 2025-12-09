// cmd/getreversaladapter.go (o el nombre que tenga el archivo)
package cmd

import (
	"errors"
	"fmt"
	"go-fiber-core/internal/adapters"
	"go-fiber-core/internal/dtos"
	"go-fiber-core/internal/dtos/config"
	"log/slog"
	"os"
	"time"

	resty "github.com/go-resty/resty/v2"
	"github.com/spf13/cobra"
)

// getreversaladapterCmd obtiene datos de reversiones desde el backoffice.
var getreversaladapterCmd = &cobra.Command{
	Use:   "getreversaladapter --customer-id [ID] --start-date [YYYY-MM-DD]",
	Short: "Ejecuta una consulta de reversiones al adaptador del backoffice.",
	Long: `Este comando construye una petición para obtener datos de reversiones
y la envía a través del adaptador del backoffice. Es una herramienta de
desarrollo y depuración para probar la integración.

Ejemplo:
go run main.go getreversaladapter --customer-id 564526 --start-date 2024-02-21`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// --- 1. Inicialización (Logger y Configuración) ---
		logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
		logger.Info("▶️ Ejecutando comando getreversaladapter")

		configPath, _ := cmd.Flags().GetString("config")

		// CAMBIO: Se capturan los dos valores (config y error) de NewAppConfig.
		appConfig, err := config.NewAppConfig(configPath)
		if err != nil {
			return fmt.Errorf("❌ error cargando la configuración: %w", err)
		}

		// --- 2. Lectura y Validación de Flags ---
		customerID, _ := cmd.Flags().GetInt("customer-id")
		dateStr, _ := cmd.Flags().GetString("start-date")

		if customerID == 0 {
			return errors.New("el flag --customer-id es obligatorio y no puede ser 0")
		}

		startDate, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			return fmt.Errorf("formato de fecha inválido para --start-date: %w", err)
		}

		// --- 3. Creación de Dependencias (Inyección de Dependencias) ---
		httpClient := resty.New().
			SetTimeout(10 * time.Second).
			SetBaseURL(appConfig.ApiBackoffice.Url)

		adapter := adapters.NewBackofficeAdapter(httpClient)

		// --- 4. Ejecución de la Lógica de Negocio ---
		reversalRequest := dtos.Config{
			Config: dtos.BackofficeReversal{
				CustomerID:       customerID,
				InstallmentState: []int{39, 40, 41, 42},
				Extra:            dtos.BackofficeReversalExtra{StartProduct: startDate},
				Imputation:       true,
			},
		}

		logger.Info("Enviando petición de reversión...", "customer_id", customerID, "start_date", dateStr)

		resp, err := adapter.PostReversal(cmd.Context(), reversalRequest)
		if err != nil {
			logger.Error("Error al llamar al adaptador de backoffice", "error", err)
			return fmt.Errorf("la petición de reversión falló: %w", err)
		}

		logger.Info("✅ Petición exitosa", "status_code", resp.StatusCode())
		logger.Info("Respuesta recibida", "body", string(resp.Body()))

		return nil
	},
}

func init() {
	rootCmd.AddCommand(getreversaladapterCmd)

	getreversaladapterCmd.Flags().Int("customer-id", 0, "ID del cliente para la consulta")
	getreversaladapterCmd.Flags().String("start-date", "", "Fecha de inicio del producto (formato: YYYY-MM-DD)")

	_ = getreversaladapterCmd.MarkFlagRequired("customer-id")
	_ = getreversaladapterCmd.MarkFlagRequired("start-date")
}
