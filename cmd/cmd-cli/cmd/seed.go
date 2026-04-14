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
			if _, err := fmt.Fprintln(w, "NOMBRE\tCOMANDO"); err != nil {
				return err
			}
			for _, name := range all {
				if _, err := fmt.Fprintf(w, "%s\tmake seed-one name=%s\n", name, name); err != nil {
					return err
				}
			}
			if err := w.Flush(); err != nil {
				return err
			}
			return nil
		}

		if onlySeeders == "" {
			fmt.Println("Seeders disponibles:")
			all := seeders.ListSeedersNames()
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			if _, err := fmt.Fprintln(w, "NOMBRE\tCOMANDO"); err != nil {
				return err
			}
			for _, name := range all {
				if _, err := fmt.Fprintf(w, "%s\tmake seed-one name=%s\n", name, name); err != nil {
					return err
				}
			}
			if err := w.Flush(); err != nil {
				return err
			}
		}

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

		if len(selected) > 0 {
			fmt.Printf("\nEjecutando %d seeder(s) seleccionado(s)...\n", len(selected))
		} else {
			fmt.Println("\nEjecutando los seeders...")
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
