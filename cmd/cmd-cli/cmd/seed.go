package cmd

import (
	"fmt"
	"go-fiber-core/internal/database/seeders"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

var onlySeeders string
var listOnly bool

var seedCmd = &cobra.Command{
	Use:   "seed",
	Short: "Ejecuta los seeders para poblar la base de datos",
	RunE: func(_ *cobra.Command, _ []string) error {
		if listOnly {
			fmt.Println("Seeders disponibles:")
			all := seeders.ListSeedersNames()
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "NOMBRE\tCOMANDO")
			for _, name := range all {
				fmt.Fprintf(w, "%s\tmake seed-one name=%s\n", name, name)
			}
			w.Flush()
			return nil
		}

		if onlySeeders == "" {
			fmt.Println("Seeders disponibles:")
			all := seeders.ListSeedersNames()
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "NOMBRE\tCOMANDO")
			for _, name := range all {
				fmt.Fprintf(w, "%s\tmake seed-one name=%s\n", name, name)
			}
			w.Flush()
		}

		fmt.Println("\nEjecutando los seeders...")

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
	seedCmd.Flags().BoolVar(&listOnly, "list", false, "Solo mostrar los seeders disponibles sin ejecutarlos")
	rootCmd.AddCommand(seedCmd)
}
