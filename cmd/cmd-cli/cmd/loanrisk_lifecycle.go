package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"go-fiber-core/cmd/api/di"
	"go-fiber-core/internal/services/processlifecycle"

	"github.com/spf13/cobra"
)

var loanRiskPayload string

var loanRiskLifecycleCmd = &cobra.Command{
	Use:   "run-loanrisk-lifecycle",
	Short: "Resuelve y ejecuta el escenario 'Loan risk lifecycle' y muestra los StepResult en JSON.",
	RunE: func(cmd *cobra.Command, args []string) error {
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

		ctx := cmd.Context()
		if ctx == nil {
			ctx = context.Background()
		}

		var processTypeID int64
		err = container.Connect.ConnectPgxWrite.
			QueryRow(ctx, `SELECT id FROM process_types WHERE name = $1 AND archived_at IS NULL`, "Loan risk lifecycle").
			Scan(&processTypeID)
		if err != nil {
			return fmt.Errorf("no se pudo resolver process_type_id para 'Loan risk lifecycle': %w", err)
		}

		var overrideVersionID int64
		err = container.Connect.ConnectPgxWrite.
			QueryRow(ctx, `SELECT id FROM process_versions WHERE process_type_id = $1 AND archived_at IS NULL ORDER BY version_number DESC LIMIT 1`, processTypeID).
			Scan(&overrideVersionID)
		if err != nil {
			return fmt.Errorf("no se pudo obtener la última versión de proceso para process_type_id=%d: %w", processTypeID, err)
		}

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

		service := processlifecycle.NewService(container.Connect)

		overridePtr := &overrideVersionID
		processVersionID, svcCtx, execErr := service.RunResolvedProcess(ctx, processTypeID, input, overridePtr)
		if execErr != nil {
			log.Printf("ejecución de RunResolvedProcess terminó con error: %v", execErr)
		}

		output := map[string]any{
			"process_version_id": processVersionID,
			"input":              input,
			"results":            map[string]any{},
		}

		if svcCtx != nil && svcCtx.Results != nil {
			output["results"] = svcCtx.Results
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
