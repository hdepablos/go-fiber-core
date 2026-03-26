package mqb1t

import (
	"context"
	"fmt"
	"os"
	"sync"

	gormconn "go-fiber-core/internal/database/connections/gorm"
	redisconn "go-fiber-core/internal/database/connections/redis"
	"go-fiber-core/internal/dtos/config"
	"go-fiber-core/internal/logger"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var (
	depsOnce sync.Once
	depsErr  error
	dbWrite  *gorm.DB
	rdb      *redis.Client
)

func getDeps(ctx context.Context) (*gorm.DB, *redis.Client, error) {
	_ = ctx
	depsOnce.Do(func() {
		log := logger.GetLoggerToFile("MultiQueueBatchProcessorOneTable", "pkg/logs/MultiQueueBatchProcessorOneTable.log").With(
			zap.String("component", "mqb1t"),
			zap.String("scope", "deps"),
		)

		configPath := os.Getenv("CONFIG_PATH")
		if configPath == "" {
			configPath = "internal/appconfig/config.yml"
		}
		if _, err := os.Stat(configPath); err != nil {
			if _, err2 := os.Stat("config.yml"); err2 == nil {
				configPath = "config.yml"
			}
		}

		log.Info("loading config", zap.String("config_path", configPath))
		appCfg, err := config.NewAppConfig(configPath)
		if err != nil {
			log.Error("load config failed", zap.Error(err))
			depsErr = err
			return
		}

		gormSvc, _, err := gormconn.NewGormConnectService(appCfg.MultiDatabaseConfig)
		if err != nil {
			log.Error("gorm connect failed", zap.Error(err))
			depsErr = err
			return
		}

		client, _, err := redisconn.NewRedisClient(appCfg.Redis)
		if err != nil {
			log.Error("redis connect failed",
				zap.String("host", appCfg.Redis.RedisHost),
				zap.String("port", appCfg.Redis.RedisPort),
				zap.Int("db", appCfg.Redis.RedisDatabase),
				zap.Error(err),
			)
			depsErr = err
			return
		}

		dbWrite = gormSvc.GetWriteDB()
		rdb = client

		if dbWrite == nil {
			log.Error("gorm write db is nil")
			depsErr = fmt.Errorf("gorm write db is nil")
			return
		}
		if rdb == nil {
			log.Error("redis client is nil")
			depsErr = fmt.Errorf("redis client is nil")
			return
		}
		log.Info("deps ready",
			zap.String("redis_host", appCfg.Redis.RedisHost),
			zap.String("redis_port", appCfg.Redis.RedisPort),
			zap.Int("redis_db", appCfg.Redis.RedisDatabase),
		)
	})

	return dbWrite, rdb, depsErr
}
