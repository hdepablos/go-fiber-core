package jobqueue

import (
	"context"
	"encoding/json"
	"log"
	"time"

	redis "github.com/redis/go-redis/v9"
)

func StartWorker(ctx context.Context, rdb *redis.Client, queueName string) {
	log.Println("[WORKER] Iniciando escucha en:", queueName)

	for {
		select {
		case <-ctx.Done():
			log.Println("[WORKER] Finalizando por contexto cancelado.")
			return
		default:
			result, err := rdb.BRPop(ctx, 5*time.Second, queueName).Result()
			if err == redis.Nil {
				continue
			} else if err != nil {
				log.Println("error al leer de redis:", err)
				continue
			}

			if len(result) < 2 {
				continue
			}

			payload := result[1]
			var job JobMessage
			if err := json.Unmarshal([]byte(payload), &job); err != nil {
				log.Println("error al parsear job:", err)
				continue
			}

			log.Printf("[WORKER] Procesando job: %s\n", job.Type)
			if err := HandleJob(job); err != nil {
				log.Printf("[WORKER] Error ejecutando job: %v\n", err)
			} else {
				log.Printf("[WORKER] Job ejecutado correctamente.")
			}
		}
	}
}
