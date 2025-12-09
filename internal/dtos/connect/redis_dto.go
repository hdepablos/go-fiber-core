package connect

import "encoding/json"

type RedisDTO struct {
	RedisHost             string      `json:"redis_host"`
	RedisPort             string      `json:"redis_port"`
	RedisPassword         string      `json:"redis_password"`
	RedisDatabase         json.Number `json:"redis_database"`
	RedisExpiresInSeconds json.Number `json:"redis_expires_in_seconds"`
	RedisPoolSize         json.Number `json:"redis_pool_size"`
}
