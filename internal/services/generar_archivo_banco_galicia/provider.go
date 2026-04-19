package generar_archivo_banco_galicia

import (
	"context"
	"fmt"

	"go-fiber-core/internal/dtos/config"
	"go-fiber-core/internal/dtos/connect"
	"go-fiber-core/internal/services/exportmanager"
	"go-fiber-core/internal/services/runtimectx"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/redis/go-redis/v9"
)

type Provider interface {
	Manager() exportmanager.Manager
	Connect() *connect.ConnectDTO
}

type provider struct {
	manager    exportmanager.Manager
	conn       *connect.ConnectDTO
	components exportmanager.PreviewComponents
}

const providerContextKey = "generar_archivo_banco_galicia.provider"

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

func WithProvider(ctx context.Context, prov Provider) context.Context {
	return runtimectx.WithNamedValue(ctx, providerContextKey, prov)
}

func ProviderFromContext(ctx context.Context) (Provider, error) {
	if prov, ok := runtimectx.NamedValue[Provider](ctx, providerContextKey); ok && prov != nil {
		return prov, nil
	}
	return nil, fmt.Errorf("provider de banco galicia no disponible en contexto")
}

const processTypeName = "generar archivo banco galicia"

func init() {
	exportmanager.RegisterPreviewProvider(processTypeName, func(ctx context.Context) (exportmanager.PreviewProvider, error) {
		prov, err := ProviderFromContext(ctx)
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
	exportmanager.RegisterManagedExportManager(func(ctx context.Context) (exportmanager.Manager, error) {
		prov, err := ProviderFromContext(ctx)
		if err != nil {
			return nil, err
		}
		return prov.Manager(), nil
	},
		"bulk/export/generar_archivo_banco_galicia/start",
		"bulk/export/generar_archivo_banco_galicia/process_batch",
		"bulk/export/generar_archivo_banco_galicia/finalize",
	)
}
