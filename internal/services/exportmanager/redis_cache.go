package exportmanager

import (
	"go-fiber-core/internal/services/batchflow"

	"github.com/redis/go-redis/v9"
)

func NewRedisCache(client *redis.Client) Cache {
	return batchflow.NewRedisCache(client)
}
