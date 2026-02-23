package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"sort"

	"go-fiber-core/cmd/api/di"
	"go-fiber-core/internal/domain"
	"go-fiber-core/internal/services/processlifecycle"
	"go-fiber-core/internal/services/serviceconfig/contracts"

	"github.com/spf13/cobra"
)

const (
	loanRiskProcessTypeID int64 = 2
)

var loanRiskPayload string

var loanRiskLifecycleCmd = &cobra.Command{
	Use:   "run-loanrisk-lifecycle",
	Short: "Resuelve y ejecuta el escenario 'Loan risk lifecycle' y muestra los StepResult en JSON.",
	RunE: func(cmd *cobra.Command, args []string) error {
		// 1) Cargar configuración de la app y construir el contenedor DI
		configPath := "internal/appconfig/config.yml"
		if os.Getenv("CONFIG_PATH") != "" {
			configPath = os.Getenv("CONFIG_PATH")
		}

		container, cleanup, err := di.InitializeAppContainer(configPath)
		if err != nil {
			return fmt.Errorf("error inicializando contenedor de aplicación: %w", err)
		}
		defer cleanup()

		if container.Connect == nil || container.Connect.ConnectPgxWrite == nil {
			return fmt.Errorf("conexión pgx write no inicializada en el contenedor")
		}

		// 2) Preparar contexto base para toda la ejecución
		ctx := cmd.Context()
		if ctx == nil {
			ctx = context.Background()
		}

		// 3) Definir process_type_id de forma explícita (tipo de proceso conocido)
		processTypeID := loanRiskProcessTypeID

		// 4) Construir el payload de entrada (process bag) para todo el flujo
		input := make(map[string]any)
		if loanRiskPayload == "" {
			input["age"] = 50
			input["salary"] = 100000
			input["sede_id"] = int64(1)
		} else {
			if err := json.Unmarshal([]byte(loanRiskPayload), &input); err != nil {
				return fmt.Errorf("payload JSON inválido: %w", err)
			}
		}

		// 5) Configurar la resolución de versión:
		//    - process_type_id: el que acabamos de resolver por nombre
		//    - sede_id: se tomará de input["sede_id"] (o 1 por defecto)
		//    - override_process_version_id: nil → equivalente a NULL, dejar que
		//      la función resolve_process_version decida la versión vigente PROD.
		var overrideProcessVersionID *int64

		// 6) Crear el servicio de lifecycle que orquesta:
		//    - resolve_process_version (PG)
		//    - construcción del registry de servicios a partir de los steps
		//    - ejecución secuencial de los servicios configurados
		service := processlifecycle.NewService(container.Connect)

		// 7) Ejecutar todo el flujo:
		//    - Resuelve la versión y obtiene los steps desde la BD
		//    - Registra los steps en el ServiceRegistry
		//    - Ejecuta todos los servicios con el process bag (input)
		processVersionID, svcCtx, execErr := service.RunResolvedProcess(ctx, processTypeID, input, overrideProcessVersionID)
		if execErr != nil {
			log.Printf("ejecución de RunResolvedProcess terminó con error: %v", execErr)
		}

		// 8) Construir la respuesta de salida para inspección / debugging
		//    Incluye siempre el input y, si hubo error, una estructura de error legible.
		output := map[string]any{
			"process_version_id": processVersionID,
			"input":              input,
			"results":            map[string]any{},
		}

		if svcCtx != nil && svcCtx.Results != nil {
			output["results"] = svcCtx.Results

			type orderedResult struct {
				ServicePath string `json:"service_path"`
				StepOrder   int    `json:"step_order"`
			}

			ordered := make([]orderedResult, 0, len(svcCtx.Results))

			for path, raw := range svcCtx.Results {
				switch v := raw.(type) {
				case contracts.StepResult:
					ordered = append(ordered, orderedResult{
						ServicePath: path,
						StepOrder:   v.StepOrder,
					})
				default:
					ordered = append(ordered, orderedResult{
						ServicePath: path,
						StepOrder:   0,
					})
				}
			}

			sort.Slice(ordered, func(i, j int) bool {
				if ordered[i].StepOrder == ordered[j].StepOrder {
					return ordered[i].ServicePath < ordered[j].ServicePath
				}
				return ordered[i].StepOrder < ordered[j].StepOrder
			})

			output["execute_ordered"] = ordered
		}

		if execErr != nil {
			errorPayload := map[string]any{
				"message": execErr.Error(),
			}

			switch {
			case errors.Is(execErr, domain.ErrNotFound):
				errorPayload["code"] = "PROCESS_VERSION_NOT_FOUND"
			case errors.Is(execErr, domain.ErrInvalidArgument):
				errorPayload["code"] = "INVALID_ARGUMENT"
			default:
				errorPayload["code"] = "INTERNAL_ERROR"
			}

			output["error"] = errorPayload
		}

		jsonBytes, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			return fmt.Errorf("no se pudo serializar resultados a JSON: %w", err)
		}

		fmt.Println(string(jsonBytes))

		if execErr != nil {
			return execErr
		}

		return nil
	},
}

func init() {
	loanRiskLifecycleCmd.Flags().StringVar(&loanRiskPayload, "payload", "", "JSON con los datos de entrada para el escenario de Loan risk")

	rootCmd.AddCommand(loanRiskLifecycleCmd)
}
