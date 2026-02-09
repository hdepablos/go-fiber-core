package cmd

import (
	"fmt"
	"go-fiber-core/internal/database/seeders"

	"github.com/spf13/cobra"
)

var seedCmd = &cobra.Command{
	Use:   "seed",
	Short: "Ejecuta los seeders para poblar la base de datos",
	RunE: func(_ *cobra.Command, _ []string) error {
		fmt.Println("Ejecutando los seeders...")
		if err := seeders.SeedDatabase(); err != nil {
			// Usamos fmt.Printf o log.Printf en lugar de log.Fatalf para permitir que el CLI maneje el error
			// Cobra capturará el error retornado y lo mostrará adecuadamente sin panic
			return fmt.Errorf("error al ejecutar los seeders: %w", err)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(seedCmd)
}
