package redis

import (
	"context"
	"fmt"
	"log"
	"time"

	redis "github.com/redis/go-redis/v9"

	"go-fiber-core/internal/dtos/config"
)

// CAMBIO: La función ahora devuelve (cliente, función_cleanup, error)
func NewRedisClient(redisConfig config.Redis) (*redis.Client, func(), error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", redisConfig.RedisHost, redisConfig.RedisPort),
		Password: redisConfig.RedisPassword,
		DB:       redisConfig.RedisDatabase,
		PoolSize: redisConfig.RedisPoolSize,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		if closeErr := rdb.Close(); closeErr != nil {
			log.Printf("Error closing Redis client after a failed ping: %v", closeErr)
		}
		// Devuelve nil para la función de limpieza si la conexión falla.
		return nil, nil, fmt.Errorf("error connecting to Redis: %w", err)
	}

	log.Println("✅ Redis connection successful.")

	// CAMBIO: Se crea la función de limpieza que cierra la conexión.
	cleanup := func() {
		if err := rdb.Close(); err != nil {
			log.Printf("Error closing Redis connection: %v", err)
		} else {
			log.Println("✅ Redis connection closed.")
		}
	}

	// CAMBIO: Se devuelve el cliente, la función de limpieza y un error nulo.
	return rdb, cleanup, nil
}
