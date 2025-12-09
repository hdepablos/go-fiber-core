package cmd

import (
	"fmt"
	"go-fiber-core/internal/database/seeders"
	"log"

	"github.com/spf13/cobra"
)

var seedCmd = &cobra.Command{
	Use:   "seed",
	Short: "Ejecuta los seeders para poblar la base de datos",
	RunE: func(_ *cobra.Command, _ []string) error {
		fmt.Println("Ejecutando los seeders...")
		if err := seeders.SeedDatabase(); err != nil {
			log.Fatalf("Error al ejecutar los seeders: %v", err)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(seedCmd)
}
