package cmd

import (
	"fmt"
	"go-fiber-core/internal/database/seeders"
	"strings"

	"github.com/spf13/cobra"
)

var onlySeeders string

var seedCmd = &cobra.Command{
	Use:   "seed",
	Short: "Ejecuta los seeders para poblar la base de datos",
	RunE: func(_ *cobra.Command, _ []string) error {
		fmt.Println("Ejecutando los seeders...")

		var selected []string
		if onlySeeders != "" {
			parts := strings.Split(onlySeeders, ",")
			for _, p := range parts {
				name := strings.TrimSpace(p)
				if name != "" {
					selected = append(selected, name)
				}
			}
		}

		if err := seeders.SeedDatabase(selected...); err != nil {
			return fmt.Errorf("error al ejecutar los seeders: %w", err)
		}
		return nil
	},
}

func init() {
	seedCmd.Flags().StringVar(&onlySeeders, "only", "", "Nombres de seeders a ejecutar, separados por coma")
	rootCmd.AddCommand(seedCmd)
}
