package generar_archivo_banco_galicia

import (
	"context"
	"fmt"
	"os"
	"sync"

	gormconn "go-fiber-core/internal/database/connections/gorm"
	redisconn "go-fiber-core/internal/database/connections/redis"
	"go-fiber-core/internal/dtos/config"
	"go-fiber-core/internal/dtos/connect"
	"go-fiber-core/internal/services/exportmanager"
	"go-fiber-core/internal/services/queue"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/redis/go-redis/v9"
)

type Provider interface {
	Manager() exportmanager.Manager
	Connect() *connect.ConnectDTO
}

type provider struct {
	manager   exportmanager.Manager
	conn      *connect.ConnectDTO
	components exportmanager.PreviewComponents
}

func (p *provider) Manager() exportmanager.Manager {
	return p.manager
}

func (p *provider) Connect() *connect.ConnectDTO {
	return p.conn
}

func (p *provider) PreviewComponents() exportmanager.PreviewComponents {
	return p.components
}

func NewProviderWithConfig(appCfg *config.AppConfig, conn *connect.ConnectDTO, redisClient *redis.Client, s3Client *s3.Client) (Provider, error) {
	if conn == nil || conn.ConnectGormWrite == nil || conn.ConnectGormRead == nil {
		return nil, fmt.Errorf("connect dto invalido")
	}
	if redisClient == nil {
		return nil, fmt.Errorf("redis client invalido")
	}
	if s3Client == nil {
		return nil, fmt.Errorf("s3 client invalido")
	}

	cache := exportmanager.NewRedisCache(redisClient)
	stateStore := exportmanager.NewRedisStateStore(cache)
	lifecycle := NewParentLifecycle(conn.ConnectGormRead, conn.ConnectGormWrite)
	dataProvider := NewDataProvider(conn.ConnectGormRead)
	headerBuilder := NewHeaderBuilder()
	bodyBuilder := NewBodyBuilder()
	footerBuilder := NewFooterBuilder()
	outputRegistrar := NewOutputRegistrar(conn.ConnectGormWrite)
	store := exportmanager.NewS3Store(s3Client)

	defaultBucket := ""
	if appCfg != nil {
		defaultBucket = appCfg.S3.Bucket
	}

	manager := exportmanager.NewManager(
		lifecycle,
		dataProvider,
		headerBuilder,
		bodyBuilder,
		footerBuilder,
		outputRegistrar,
		stateStore,
		store,
		defaultBucket,
		"exports/bulk_jobs/generar_archivo_banco_galicia",
	)

	return &provider{
		manager: manager,
		conn:    conn,
		components: exportmanager.PreviewComponents{
			DataProvider:  dataProvider,
			HeaderBuilder: headerBuilder,
			BodyBuilder:   bodyBuilder,
			FooterBuilder: footerBuilder,
			StateStore:    stateStore,
		},
	}, nil
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

		conn := &connect.ConnectDTO{
			ConnectGormWrite: gormSvc.GetWriteDB(),
			ConnectGormRead:  gormSvc.GetReadDB(),
			ConnectRedis:     rdb,
		}

		prov, err := NewProviderWithConfig(appCfg, conn, rdb, awsSvc.NewS3Client())
		if err != nil {
			defaultErr = err
			return
		}
		defaultProv = prov
	})

	return defaultProv, defaultErr
}

const processTypeName = "generar archivo banco galicia"

func init() {
	exportmanager.RegisterPreviewProvider(processTypeName, func(ctx context.Context) (exportmanager.PreviewProvider, error) {
		prov, err := DefaultProvider(ctx)
		if err != nil {
			return nil, err
		}
		previewable, ok := prov.(exportmanager.PreviewProvider)
		if !ok {
			return nil, fmt.Errorf("provider de %s no soporta preview", processTypeName)
		}
		return previewable, nil
	},
		"bulk/export/generar_archivo_banco_galicia/start",
		"bulk/export/generar_archivo_banco_galicia/process_batch",
		"bulk/export/generar_archivo_banco_galicia/finalize",
	)
}
