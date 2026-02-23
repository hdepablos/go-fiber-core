package cmd

import (
	"context"
	"errors"
	"fmt"
	"go-fiber-core/internal/database/connections/redis"
	"go-fiber-core/internal/dtos/config"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var redisClearPattern string

var redisClearKeysCmd = &cobra.Command{
	Use:   "redis-clear-keys",
	Short: "Elimina keys de Redis por nombre exacto o patrón.",
	RunE: func(cmd *cobra.Command, args []string) error {
		configPath := "internal/appconfig/config.yml"
		if os.Getenv("CONFIG_PATH") != "" {
			configPath = os.Getenv("CONFIG_PATH")
		}

		if redisClearPattern == "" {
			return errors.New("debes especificar un patrón de key con --pattern")
		}

		appConfig, err := config.NewAppConfig(configPath)
		if err != nil {
			return fmt.Errorf("error cargando configuración: %w", err)
		}

		client, cleanup, err := redis.NewRedisClient(appConfig.Redis)
		if err != nil {
			return fmt.Errorf("no se pudo conectar con Redis: %w", err)
		}
		if cleanup != nil {
			defer cleanup()
		}

		projectPrefix := appConfig.App.AppName
		if projectPrefix == "" {
			projectPrefix = "go-fiber-core"
		}

		ctx, cancel := context.WithTimeout(cmd.Context(), 5*time.Second)
		defer cancel()

		pattern := redisClearPattern
		if pattern == "*" {
			pattern = projectPrefix + ":*"
		}
		if !strings.Contains(pattern, "*") && !strings.Contains(pattern, "?") && !strings.Contains(pattern, "[") {
			deleted, err := client.Del(ctx, pattern).Result()
			if err != nil {
				return fmt.Errorf("error eliminando key %q: %w", pattern, err)
			}
			fmt.Printf("Eliminadas %d keys\n", deleted)
			return nil
		}

		var cursor uint64
		var total int64

		for {
			var keys []string
			var err error

			keys, cursor, err = client.Scan(ctx, cursor, pattern, 100).Result()
			if err != nil {
				return fmt.Errorf("error escaneando keys con patrón %q: %w", pattern, err)
			}

			if len(keys) > 0 {
				deleted, err := client.Del(ctx, keys...).Result()
				if err != nil {
					return fmt.Errorf("error eliminando keys: %w", err)
				}
				total += deleted
			}

			if cursor == 0 {
				break
			}
		}

		fmt.Printf("Eliminadas %d keys que coinciden con el patrón %q\n", total, pattern)
		return nil
	},
}

func init() {
	redisClearKeysCmd.Flags().StringVar(&redisClearPattern, "pattern", "", "Key exacta o patrón de keys a eliminar (por ejemplo go-fiber-core:process-lifecycle-process_type_id-2 o go-fiber-core:process-lifecycle-process_type_id-*)")
	rootCmd.AddCommand(redisClearKeysCmd)
}
