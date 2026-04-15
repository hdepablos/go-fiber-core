package cache

import (
	"context"
	"errors"
	"fmt"
	"time"

	redis "github.com/redis/go-redis/v9"
)

var (
	// ErrCacheLocked se devuelve cuando el recurso está bloqueado por una operación de escritura
	ErrCacheLocked = errors.New("resource is locked for update")
	// ErrCacheMiss se devuelve cuando la key no existe
	ErrCacheMiss = redis.Nil
)

// LockService define el contrato mínimo del cache con locking.
type LockService interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Lock(ctx context.Context, key string, safetyTTL time.Duration) error
	Unlock(ctx context.Context, key string) error
	Invalidate(ctx context.Context, key string) error
}

// redisLockService encapsula la lógica de Locking Cache-Aside.
type redisLockService struct {
	client *redis.Client
}

// NewRedisLockService crea una nueva instancia del servicio
func NewRedisLockService(client *redis.Client) LockService {
	return &redisLockService{
		client: client,
	}
}

// Get intenta obtener un valor de Redis respetando el bloqueo.
// Si existe un lock para la key, retorna ErrCacheLocked (o nil, según prefieras manejarlo).
// En este caso, retornamos ErrCacheLocked para que el consumidor sepa explícitamente por qué falló,
// pero el consumidor debería tratarlo como un Cache Miss y consultar la BD.
func (s *redisLockService) Get(ctx context.Context, key string) (string, error) {
	lockKey := s.getLockKey(key)

	// 1. Verificar si existe bloqueo
	exists, err := s.client.Exists(ctx, lockKey).Result()
	if err != nil {
		return "", fmt.Errorf("error checking lock: %w", err)
	}
	if exists > 0 {
		return "", ErrCacheLocked
	}

	// 2. Si no hay bloqueo, obtener valor
	val, err := s.client.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return "", ErrCacheMiss
		}
		return "", fmt.Errorf("error getting key: %w", err)
	}

	return val, nil
}

// Set establece un valor en Redis (sin lógica de bloqueo especial, solo wrapper)
func (s *redisLockService) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	return s.client.Set(ctx, key, value, ttl).Err()
}

// Lock bloquea una key para indicar que se va a realizar una actualización crítica.
// Crea una key 'lock:original_key' con un TTL corto de seguridad.
func (s *redisLockService) Lock(ctx context.Context, key string, safetyTTL time.Duration) error {
	lockKey := s.getLockKey(key)
	// Valor arbitrario, lo importante es la existencia de la key
	return s.client.Set(ctx, lockKey, "locked", safetyTTL).Err()
}

// Unlock elimina el bloqueo Y la key original para invalidar la caché antigua.
// Debe llamarse después de que la transacción en BD haya sido exitosa.
func (s *redisLockService) Unlock(ctx context.Context, key string) error {
	lockKey := s.getLockKey(key)

	// Usamos Pipeline para borrar ambas keys en una sola ida y vuelta a Redis
	pipe := s.client.Pipeline()
	pipe.Del(ctx, key)     // Borrar dato viejo
	pipe.Del(ctx, lockKey) // Borrar lock
	_, err := pipe.Exec(ctx)

	if err != nil {
		return fmt.Errorf("error unlocking: %w", err)
	}
	return nil
}

// Invalidate simplemente borra una key (útil para casos sin lock explícito)
func (s *redisLockService) Invalidate(ctx context.Context, key string) error {
	return s.client.Del(ctx, key).Err()
}

// getLockKey genera el nombre de la key de bloqueo
func (s *redisLockService) getLockKey(key string) string {
	return fmt.Sprintf("lock:%s", key)
}
