// cmd/migrations.go (o el nombre que tenga el archivo)
package cmd

import (
	"errors"
	"fmt"
	"go-fiber-core/internal/database/migrations"
	"go-fiber-core/internal/dtos/config"
	"go-fiber-core/internal/utils"
	"log"

	"github.com/spf13/cobra"
)

var (
	cfgPath          string
	step             int
	migrationService *migrations.MigrationService
	appCfg           *config.AppConfig
	cleanupFunc      func()
)

// migrationsCmd es el comando padre para todas las operaciones de migración.
var migrationsCmd = &cobra.Command{
	Use:   "migrations",
	Short: "Ejecuta los comandos de migración de la base de datos",

	// Se ejecuta ANTES de cualquier subcomando de 'migrations'.
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// 1. Cargar configuración.
		// CAMBIO: Se capturan y manejan los dos valores de retorno de NewAppConfig.
		loadedCfg, err := config.NewAppConfig(cfgPath)
		if err != nil {
			return fmt.Errorf("error cargando la configuración para las migraciones: %w", err)
		}
		appCfg = loadedCfg

		// 2. Crear el servicio de migración.
		service, cleanup, err := migrations.NewMigrationService(appCfg.MultiDatabaseConfig)
		if err != nil {
			return fmt.Errorf("error creando MigrationService: %w", err)
		}
		migrationService = service
		cleanupFunc = cleanup

		return nil
	},

	// Se ejecuta DESPUÉS de cualquier subcomando para limpieza.
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		if cleanupFunc != nil {
			cleanupFunc()
		}
	},
}

var upCmd = &cobra.Command{
	Use:   "up",
	Short: "Aplica todas las migraciones disponibles",
	RunE: func(cmd *cobra.Command, args []string) error {
		log.Println("Aplicando migraciones...")
		return migrationService.Up()
	},
}

var downCmd = &cobra.Command{
	Use:   "down",
	Short: "Revierte la última migración (o múltiples con --step)",
	RunE: func(cmd *cobra.Command, args []string) error {
		if utils.IsProduction(*appCfg) {
			return errors.New("no se permite ejecutar rollback en producción")
		}
		log.Printf("Revirtiendo %d migración(es)...\n", step)
		return migrationService.Down(step)
	},
}

var createCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Crea un nuevo archivo de migración SQL",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		log.Printf("Creando archivo de migración: %s...\n", name)
		return migrationService.Create(name)
	},
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Muestra la versión y el estado de las migraciones",
	RunE: func(cmd *cobra.Command, args []string) error {
		version, err := migrationService.CurrentVersion()
		if err != nil {
			return err
		}
		fmt.Printf("Versión actual de la migración: %d\n\n", version)
		return migrationService.PrintStatus()
	},
}

var resetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Revierte todas las migraciones",
	RunE: func(cmd *cobra.Command, args []string) error {
		log.Println("Revertir todas las migraciones...")
		if utils.IsProduction(*appCfg) {
			return errors.New("no se permite ejecutar rollback en producción")
		}
		return migrationService.DownToZero()
	},
}

// init registra los comandos y sus flags en Cobra.
func init() {
	rootCmd.PersistentFlags().StringVar(&cfgPath, "config", "internal/appconfig/config.yml", "Ruta al archivo de configuración YAML")

	downCmd.Flags().IntVar(&step, "step", 1, "Número de migraciones a revertir (por defecto 1)")

	migrationsCmd.AddCommand(upCmd)
	migrationsCmd.AddCommand(downCmd)
	migrationsCmd.AddCommand(createCmd)
	migrationsCmd.AddCommand(statusCmd)
	migrationsCmd.AddCommand(resetCmd)

	rootCmd.AddCommand(migrationsCmd)
}
