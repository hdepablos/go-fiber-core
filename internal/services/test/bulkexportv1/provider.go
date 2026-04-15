package bulkexportv1

import (
	"context"
	"fmt"
	"os"
	"sync"

	gormconn "go-fiber-core/internal/database/connections/gorm"
	redisconn "go-fiber-core/internal/database/connections/redis"
	"go-fiber-core/internal/dtos/config"
	"go-fiber-core/internal/dtos/connect"
	"go-fiber-core/internal/services/queue"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/redis/go-redis/v9"
)

type Provider interface {
	Pipeline() ExportPipeline
	Connect() *connect.ConnectDTO
}

type provider struct {
	pipeline ExportPipeline
	conn     *connect.ConnectDTO
}

func (p *provider) Pipeline() ExportPipeline {
	return p.pipeline
}

func (p *provider) Connect() *connect.ConnectDTO {
	return p.conn
}

func NewProvider(conn *connect.ConnectDTO, redisClient *redis.Client, s3Client *s3.Client) (Provider, error) {
	if conn == nil || conn.ConnectGormWrite == nil {
		return nil, fmt.Errorf("connect dto inválido")
	}
	return NewProviderWithConfig(nil, conn, redisClient, s3Client)
}

func NewProviderWithConfig(appCfg *config.AppConfig, conn *connect.ConnectDTO, redisClient *redis.Client, s3Client *s3.Client) (Provider, error) {
	if conn == nil || conn.ConnectGormWrite == nil {
		return nil, fmt.Errorf("connect dto inválido")
	}
	if redisClient == nil {
		return nil, fmt.Errorf("redis client inválido")
	}
	if s3Client == nil {
		return nil, fmt.Errorf("s3 client inválido")
	}

	bulkJobs := NewGormBulkJobRepository(conn.ConnectGormRead, conn.ConnectGormWrite)
	items := NewGormBulkJobItemRepository(conn.ConnectGormRead)
	outputs := NewGormBulkJobOutputRepository(conn.ConnectGormWrite)
	cache := NewRedisCache(redisClient)
	store := NewS3Store(s3Client)
	csvBuilder := NewDefaultCSVBuilder()
	runIDProvider := NewUUIDRunIDProvider()
	defaultBucket := ""
	if appCfg != nil {
		defaultBucket = appCfg.S3.Bucket
	}

	pipeline := NewExportPipeline(bulkJobs, bulkJobs, items, outputs, cache, csvBuilder, store, runIDProvider, defaultBucket)
	return &provider{pipeline: pipeline, conn: conn}, nil
}

var (
	defaultOnce sync.Once
	defaultProv Provider
	defaultErr  error
	manualProv  Provider
	manualMu    sync.RWMutex
)

func SetDefaultProvider(prov Provider) {
	manualMu.Lock()
	defer manualMu.Unlock()
	manualProv = prov
}

func DefaultProvider(ctx context.Context) (Provider, error) {
	manualMu.RLock()
	if manualProv != nil {
		prov := manualProv
		manualMu.RUnlock()
		return prov, nil
	}
	manualMu.RUnlock()

	defaultOnce.Do(func() {
		configPath := os.Getenv("CONFIG_PATH")
		if configPath == "" {
			configPath = "internal/appconfig/config.yml"
		}
		if _, err := os.Stat(configPath); err != nil {
			if _, err2 := os.Stat("config.yml"); err2 == nil {
				configPath = "config.yml"
			}
		}

		appCfg, err := config.NewAppConfig(configPath)
		if err != nil {
			defaultErr = err
			return
		}

		gormSvc, _, err := gormconn.NewGormConnectService(appCfg.MultiDatabaseConfig)
		if err != nil {
			defaultErr = err
			return
		}

		rdb, _, err := redisconn.NewRedisClient(appCfg.Redis)
		if err != nil {
			defaultErr = err
			return
		}

		awsSvc, err := queue.NewAWSService(ctx)
		if err != nil {
			defaultErr = err
			return
		}
		s3Client := awsSvc.NewS3Client()

		conn := &connect.ConnectDTO{
			ConnectGormWrite: gormSvc.GetWriteDB(),
			ConnectGormRead:  gormSvc.GetReadDB(),
			ConnectRedis:     rdb,
		}

		prov, err := NewProviderWithConfig(appCfg, conn, rdb, s3Client)
		if err != nil {
			defaultErr = err
			return
		}
		defaultProv = prov
	})
	return defaultProv, defaultErr
}

func projectPrefix() string {
	prefix := os.Getenv("APP_NAME")
	if prefix == "" {
		prefix = "go-fiber-core"
	}
	return prefix
}
