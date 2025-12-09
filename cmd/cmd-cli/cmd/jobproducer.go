// cmd/jobproducer.go (o el nombre que tenga el archivo)
package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go-fiber-core/internal/database/connections/redis"
	"go-fiber-core/internal/dtos/config"
	"go-fiber-core/internal/jobqueue"
	"log/slog"
	"os"
	"strconv"

	redisClient "github.com/redis/go-redis/v9"
	"github.com/spf13/cobra"
)

// jobproducerCmd representa el comando jobproducer
var jobproducerCmd = &cobra.Command{
	Use:   "jobproducer [total_jobs]",
	Short: "Encola un número específico de trabajos de prueba en Redis.",
	Long: `Este comando genera y encola trabajos simulados en una cola de Redis.
Es útil para probar y demostrar el sistema de workers distribuidos.

Argumentos:
  total_jobs    El número de trabajos a generar y encolar.

Ejemplo:
  go run main.go jobproducer 100 --config path/to/your/config.yml`,
	RunE: func(cmd *cobra.Command, args []string) error {
		logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

		totalJobs, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("el argumento [total_jobs] no es un número válido: %w", err)
		}
		if totalJobs <= 0 {
			return errors.New("el número de trabajos debe ser un entero positivo")
		}

		logger.Info("▶️ Iniciando el productor de trabajos", "total_jobs", totalJobs)

		configPath, _ := cmd.Flags().GetString("config")
		logger.Debug("Cargando configuración", "path", configPath)

		// CAMBIO: Se capturan los dos valores (config y error) de NewAppConfig.
		appConfig, err := config.NewAppConfig(configPath)
		if err != nil {
			return fmt.Errorf("❌ error cargando la configuración: %w", err)
		}

		redisClient, cleanupRedis, err := redis.NewRedisClient(appConfig.Redis)
		if err != nil {
			return fmt.Errorf("❌ no se pudo conectar con Redis: %w", err)
		}
		defer cleanupRedis()

		logger.Info("✅ Conectado a Redis exitosamente.")

		err = enqueueJobs(cmd.Context(), redisClient, logger, totalJobs)
		if err != nil {
			return err
		}

		logger.Info("🎉 Proceso de encolado finalizado exitosamente.", "total_jobs_enqueued", totalJobs)
		return nil
	},
	Args: cobra.ExactArgs(1),
}

// enqueueJobs contiene la lógica de negocio para crear y encolar los trabajos.
func enqueueJobs(ctx context.Context, redisClient *redisClient.Client, logger *slog.Logger, totalJobs int) error {
	logger.Info("📬 Encolando trabajos...")

	for i := 1; i <= totalJobs; i++ {
		data := map[string]any{
			"to":      fmt.Sprintf("user%d@example.com", i),
			"subject": fmt.Sprintf("Notificación #%d", i),
			"body":    fmt.Sprintf("Hola User %d, este es un mensaje automatizado.", i),
		}

		rawData, err := json.Marshal(data)
		if err != nil {
			logger.Error("Error serializando los datos del trabajo", "job_id", i, "error", err)
			continue
		}

		job := jobqueue.JobMessage{
			Type: "send_email",
			Data: rawData,
		}

		payload, err := json.Marshal(job)
		if err != nil {
			logger.Error("Error serializando el mensaje del job", "job_id", i, "error", err)
			continue
		}

		if err := redisClient.LPush(ctx, "job_queue", payload).Err(); err != nil {
			logger.Error("Error enviando el trabajo a Redis", "job_id", i, "error", err)
		} else {
			logger.Debug("✉️ Trabajo encolado correctamente", "job_id", i)
		}
	}

	return nil
}

func init() {
	rootCmd.AddCommand(jobproducerCmd)
	jobproducerCmd.Flags().String("config", "internal/appconfig/config.yml", "Ruta al archivo de configuración YAML")
}
