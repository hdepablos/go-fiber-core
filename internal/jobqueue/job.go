package jobqueue

import (
	"context"
	"encoding/json"
	"fmt"

	redis "github.com/redis/go-redis/v9"
)

// JobMessage representa la estructura de un trabajo en la cola.
type JobMessage struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

// Enqueuer es un servicio para encolar trabajos en Redis.
// Se crea una vez con un cliente de Redis y se reutiliza.
type Enqueuer struct {
	redisClient *redis.Client
}

// NewEnqueuer crea una nueva instancia del servicio para encolar trabajos.
// Recibe el cliente de Redis como una dependencia.
func NewEnqueuer(client *redis.Client) *Enqueuer {
	return &Enqueuer{
		redisClient: client,
	}
}

// Enqueue serializa y encola un nuevo trabajo en la cola principal de Redis.
func (e *Enqueuer) Enqueue(ctx context.Context, jobType string, data any) error {
	// 1. Serializar los datos específicos del trabajo
	rawData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("error al serializar los datos del job: %w", err)
	}

	// 2. Crear la estructura del mensaje del job
	job := JobMessage{
		Type: jobType,
		Data: rawData,
	}

	// 3. Serializar el mensaje completo del job
	payload, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("error al serializar el payload del job: %w", err)
	}

	// 4. Encolar el trabajo en Redis usando el cliente que ya tiene el servicio
	return e.redisClient.RPush(ctx, "jobs:main", payload).Err()
}
