package punitive

import (
	"context"
	"fmt"
	"time"

	"go-fiber-core/internal/dtos/config"
	"go-fiber-core/internal/dtos/connect"
	serviceData "go-fiber-core/internal/services/batchprocess/punitive/data"
	serviceLifecycle "go-fiber-core/internal/services/batchprocess/punitive/lifecycle"
	serviceProcessor "go-fiber-core/internal/services/batchprocess/punitive/processor"
	serviceRuntime "go-fiber-core/internal/services/batchprocess/punitive/runtime"
	serviceSteps "go-fiber-core/internal/services/batchprocess/punitive/steps"
	"go-fiber-core/internal/services/batchflow"

	"github.com/redis/go-redis/v9"
)

type Provider = serviceRuntime.Provider

// provider arma el entrypoint del proceso y expone manager, conexiones y preview.
type provider struct {
	manager    batchflow.Manager
	conn       *connect.ConnectDTO
	components batchflow.PreviewComponents
}

// Manager devuelve el coordinador principal del flujo batch.
func (p *provider) Manager() batchflow.Manager {
	return p.manager
}

// Connect expone las conexiones por si otro componente del proceso las necesita.
func (p *provider) Connect() *connect.ConnectDTO {
	return p.conn
}

// PreviewComponents registra las piezas necesarias para preview y debugging operativo.
func (p *provider) PreviewComponents() batchflow.PreviewComponents {
	return p.components
}

// NewProviderWithConfig construye todo el grafo del proceso: lifecycle, data source, processor y finalizer.
func NewProviderWithConfig(appCfg *config.AppConfig, conn *connect.ConnectDTO, redisClient *redis.Client) (Provider, error) {
	if conn == nil || conn.ConnectGormWrite == nil || conn.ConnectGormRead == nil {
		return nil, fmt.Errorf("connect dto invalido")
	}
	if redisClient == nil {
		return nil, fmt.Errorf("redis client invalido")
	}

	cache := batchflow.NewRedisCache(redisClient)
	stateStore := batchflow.NewRedisStateStore(cache)
	runControl := batchflow.NewRunControl(cache, batchflowTTL(appCfg))
	lifecycle := serviceLifecycle.NewParentLifecycle(conn.ConnectGormRead, conn.ConnectGormWrite)
	progressRefresher, _ := lifecycle.(batchflow.BatchProgressRefresher)
	dataProvider := serviceData.NewDataProvider(conn.ConnectGormRead)
	processor := serviceProcessor.NewProcessor(conn.ConnectGormWrite)
	finalizer := serviceLifecycle.NewFinalizer(conn.ConnectGormRead)

	manager := batchflow.NewManager(
		lifecycle,
		dataProvider,
		processor,
		finalizer,
		stateStore,
		batchflowTTL(appCfg),
		runControl,
	)

	return &provider{
		manager: manager,
		conn:    conn,
		components: batchflow.PreviewComponents{
			DataProvider:      dataProvider,
			BatchProcessor:    processor,
			BatchPreviewer:    processor,
			ProgressRefresher: progressRefresher,
			StateStore:        stateStore,
		},
	}, nil
}

// batchflowTTL define por cuanto tiempo se conserva el estado temporal del run en Redis.
func batchflowTTL(appCfg *config.AppConfig) time.Duration {
	_ = appCfg
	return 24 * time.Hour
}

// WithProvider inyecta el provider en el contexto para que lo consuman los steps.
func WithProvider(ctx context.Context, prov Provider) context.Context {
	return serviceRuntime.WithProvider(ctx, prov)
}

// ProviderFromContext recupera el provider del proceso desde el contexto de ejecucion.
func ProviderFromContext(ctx context.Context) (Provider, error) {
	return serviceRuntime.ProviderFromContext(ctx)
}

const processTypeName = "Punitorios"

// init registra el proceso en el runtime para run, preview y ejecucion batch administrada.
func init() {
	serviceSteps.Register()
	batchflow.RegisterPreviewProvider(processTypeName, func(ctx context.Context) (batchflow.PreviewProvider, error) {
		prov, err := ProviderFromContext(ctx)
		if err != nil {
			return nil, err
		}
		previewable, ok := prov.(batchflow.PreviewProvider)
		if !ok {
			return nil, fmt.Errorf("provider de %s no soporta preview", processTypeName)
		}
		return previewable, nil
	},
		"bulk/process/punitive/start",
		"bulk/process/punitive/dispatch_shards",
		"bulk/process/punitive/process_batch",
		"bulk/process/punitive/finalize",
	)
	batchflow.RegisterManagedBatchManager(func(ctx context.Context) (batchflow.Manager, error) {
		prov, err := ProviderFromContext(ctx)
		if err != nil {
			return nil, err
		}
		return prov.Manager(), nil
	},
		"bulk/process/punitive/start",
		"bulk/process/punitive/dispatch_shards",
		"bulk/process/punitive/process_batch",
		"bulk/process/punitive/finalize",
	)
}
