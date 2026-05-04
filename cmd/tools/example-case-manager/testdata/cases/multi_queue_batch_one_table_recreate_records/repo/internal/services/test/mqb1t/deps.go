package mqb1t

import (
	"context"
	"fmt"
	"os"
	"sync"

	gormconn "go-fiber-core/internal/database/connections/gorm"
	redisconn "go-fiber-core/internal/database/connections/redis"
	"go-fiber-core/internal/dtos/config"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

var (
	// depsOnce garantiza que la inicialización ocurra una sola vez por proceso (por pod).
	// Esto evita reconectar DB/Redis en cada step (organize/process_batch/finalize).
	depsOnce sync.Once

	// depsErr guarda el primer error de inicialización (si falla config/DB/Redis).
	// Las siguientes llamadas a getDeps devuelven el mismo error.
	depsErr error

	// dbWrite es la conexión de escritura a la base de datos (GORM).
	// En este flujo se usa para actualizar el estado de los registros del run.
	dbWrite *gorm.DB

	// rdb es el cliente Redis compartido.
	// Se usa para coordinar batches (total/done/finalize/started_at) entre steps.
	rdb *redis.Client
)

// getDeps devuelve las "dependencias" del flujo (deps = dependencies):
// - DB (write) para actualizar estados de registros
// - Redis para coordinación entre lotes
//
// Nota: se inicializa con sync.Once, por lo que si cambia el env/config en runtime,
// este helper NO recarga automáticamente.
func getDeps(ctx context.Context) (*gorm.DB, *redis.Client, error) {
	_ = ctx
	depsOnce.Do(func() {
		// Resuelve el path de config. Prioriza CONFIG_PATH y tiene un fallback para local.
		configPath := os.Getenv("CONFIG_PATH")
		if configPath == "" {
			configPath = "internal/appconfig/config.yml"
		}
		if _, err := os.Stat(configPath); err != nil {
			if _, err2 := os.Stat("config.yml"); err2 == nil {
				configPath = "config.yml"
			}
		}

		// Carga configuración (DB/Redis) desde YAML.
		appCfg, err := config.NewAppConfig(configPath)
		if err != nil {
			depsErr = err
			return
		}

		// Crea el servicio de conexión GORM y obtiene el DB de escritura.
		gormSvc, _, err := gormconn.NewGormConnectService(appCfg.MultiDatabaseConfig)
		if err != nil {
			depsErr = err
			return
		}

		// Crea el cliente Redis (usado para locks/contadores del proceso).
		client, _, err := redisconn.NewRedisClient(appCfg.Redis)
		if err != nil {
			depsErr = err
			return
		}

		// Cachea dependencias para próximos steps.
		dbWrite = gormSvc.GetWriteDB()
		rdb = client

		// Validaciones básicas: si algo viene nil, fallar temprano con error explicativo.
		if dbWrite == nil {
			depsErr = fmt.Errorf("gorm write db is nil")
			return
		}
		if rdb == nil {
			depsErr = fmt.Errorf("redis client is nil")
			return
		}
	})

	return dbWrite, rdb, depsErr
}
