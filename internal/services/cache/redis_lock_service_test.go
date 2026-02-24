package cache_test

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"go-fiber-core/internal/services/cache"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

// TestRedisRaceCondition simula un escenario de actualización crítica
// y verifica que el bloqueo impida lecturas de caché durante la ventana vulnerable.
func TestRedisRaceCondition(t *testing.T) {
	// Configuración de Redis (leyendo de variables de entorno si están disponibles, o defaults)
	// Para este test, asumiremos los valores por defecto del docker-compose o entorno local
	host := os.Getenv("REDIS_HOST")
	if host == "" {
		host = "localhost"
	}
	port := os.Getenv("REDIS_PORT")
	if port == "" {
		port = "6379"
	}
	password := os.Getenv("REDIS_PASSWORD")
	// Si no hay password en env, intentamos "redis" que es común en dev, o vacío.
	if password == "" {
		password = "redis" 
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", host, port),
		Password: password,
	})
	ctx := context.Background()

	// Limpieza previa
	key := "test:race:config"
	lockKey := "lock:test:race:config"
	rdb.Del(ctx, key, lockKey)

	// Inicializamos el servicio
	lockService := cache.NewRedisLockService(rdb)

	// 1. Estado Inicial: Caché con valor "OLD"
	err := lockService.Set(ctx, key, "OLD_VALUE", 0)
	assert.NoError(t, err)

	// Canales para sincronizar la simulación
	startUpdate := make(chan bool)
	updateDone := make(chan bool)

	var wg sync.WaitGroup

	// --- ACTOR 1: ACTUALIZADOR (ADMIN) ---
	wg.Add(1)
	go func() {
		defer wg.Done()
		<-startUpdate // Esperar señal para empezar

		fmt.Println("[ADMIN] Iniciando actualización crítica...")
		
		// Paso A: Bloquear
		err := lockService.Lock(ctx, key, 5*time.Second)
		assert.NoError(t, err)
		fmt.Println("[ADMIN] Bloqueo establecido (Lock).")

		// Simular tiempo de proceso en BD (ventana vulnerable)
		time.Sleep(200 * time.Millisecond)

		// Paso B: Actualizar "BD" (simulado, aquí no tocamos BD real)
		fmt.Println("[ADMIN] Actualización en BD completada.")

		// Paso C: Desbloquear y limpiar
		err = lockService.Unlock(ctx, key)
		assert.NoError(t, err)
		fmt.Println("[ADMIN] Desbloqueo y limpieza completados (Unlock).")
		
		updateDone <- true
	}()

	// --- ACTOR 2: LECTOR CONCURRENTE (USUARIO) ---
	wg.Add(1)
	go func() {
		defer wg.Done()
		
		// Intentar leer ANTES del bloqueo
		val, err := lockService.Get(ctx, key)
		assert.NoError(t, err)
		assert.Equal(t, "OLD_VALUE", val)
		fmt.Println("[USER] Lectura pre-bloqueo: OK (valor antiguo)")

		// Dar señal al Admin para que empiece
		startUpdate <- true

		// Esperar un poco para caer JUSTO en la ventana de bloqueo (100ms < 200ms)
		time.Sleep(100 * time.Millisecond)

		// Intentar leer DURANTE el bloqueo
		fmt.Println("[USER] Intentando leer durante bloqueo...")
		val, err = lockService.Get(ctx, key)
		
		// AQUI ESTA LA MAGIA: Debería fallar con ErrCacheLocked
		if err == cache.ErrCacheLocked {
			fmt.Println("[USER] ✅ CORRECTO: Lectura bloqueada, el sistema iría a BD.")
		} else {
			t.Errorf("[USER] ❌ FALLO: Se esperaba ErrCacheLocked, se obtuvo: %v, valor: %s", err, val)
		}

		// Esperar a que termine la actualización
		<-updateDone

		// Intentar leer DESPUES del desbloqueo
		// Debería dar Cache Miss (nil) porque Unlock borra la key vieja
		val, err = lockService.Get(ctx, key)
		if err == cache.ErrCacheMiss {
			fmt.Println("[USER] ✅ CORRECTO: Cache Miss tras actualización, el sistema iría a BD a buscar lo nuevo.")
		} else {
			t.Errorf("[USER] ❌ FALLO: Se esperaba ErrCacheMiss, se obtuvo: %v, valor: %s", err, val)
		}
	}()

	wg.Wait()
}
