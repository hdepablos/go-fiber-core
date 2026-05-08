package bcra

import (
	"context"
	"fmt"

	"go-fiber-core/internal/dtos/config"
	"go-fiber-core/internal/dtos/connect"
	serviceData "go-fiber-core/internal/services/exports/bcra/data"
	serviceLayout "go-fiber-core/internal/services/exports/bcra/layout"
	serviceLifecycle "go-fiber-core/internal/services/exports/bcra/lifecycle"
	serviceRuntime "go-fiber-core/internal/services/exports/bcra/runtime"
	serviceSteps "go-fiber-core/internal/services/exports/bcra/steps"
	"go-fiber-core/internal/services/exportmanager"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/redis/go-redis/v9"
)

type Provider = serviceRuntime.Provider

// provider arma el entrypoint del export y expone manager, conexiones y preview.
type provider struct {
	manager    exportmanager.Manager
	conn       *connect.ConnectDTO
	components exportmanager.PreviewComponents
}

// Manager devuelve el coordinador principal del flujo de exportacion.
func (p *provider) Manager() exportmanager.Manager {
	return p.manager
}

// Connect expone las conexiones por si otro componente del proceso las necesita.
func (p *provider) Connect() *connect.ConnectDTO {
	return p.conn
}

// PreviewComponents registra las piezas necesarias para preview y debugging del export.
func (p *provider) PreviewComponents() exportmanager.PreviewComponents {
	return p.components
}

// NewProviderWithConfig construye todo el grafo del export: lifecycle, data source, layout y registro final.
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
	lifecycle := serviceLifecycle.NewParentLifecycle(conn.ConnectGormRead, conn.ConnectGormWrite)
	dataProvider := serviceData.NewDataProvider(conn.ConnectGormRead)
	headerBuilder := serviceLayout.NewHeaderBuilder()
	bodyBuilder := serviceLayout.NewBodyBuilder()
	footerBuilder := serviceLayout.NewFooterBuilder()
	outputRegistrar := serviceLifecycle.NewOutputRegistrar(conn.ConnectGormWrite)
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
		"exports/bulk_jobs/bcra",
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

// WithProvider inyecta el provider en el contexto para que lo consuman los steps.
func WithProvider(ctx context.Context, prov Provider) context.Context {
	return serviceRuntime.WithProvider(ctx, prov)
}

// ProviderFromContext recupera el provider del export desde el contexto de ejecucion.
func ProviderFromContext(ctx context.Context) (Provider, error) {
	return serviceRuntime.ProviderFromContext(ctx)
}

const processTypeName = "generar archivo BCRA"

// init registra el export en el runtime para run, preview y ejecucion administrada.
func init() {
	serviceSteps.Register()
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
		"bulk/export/bcra/start",
		"bulk/export/bcra/process_batch",
		"bulk/export/bcra/finalize",
	)
	exportmanager.RegisterManagedExportManager(func(ctx context.Context) (exportmanager.Manager, error) {
		prov, err := ProviderFromContext(ctx)
		if err != nil {
			return nil, err
		}
		return prov.Manager(), nil
	},
		"bulk/export/bcra/start",
		"bulk/export/bcra/process_batch",
		"bulk/export/bcra/finalize",
	)
}
