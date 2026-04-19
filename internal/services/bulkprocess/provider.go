package bulkprocess

import (
	"context"
	"fmt"
	"time"

	"go-fiber-core/internal/dtos/config"
	"go-fiber-core/internal/dtos/connect"
	"go-fiber-core/internal/services/batchflow"
	"go-fiber-core/internal/services/runtimectx"

	"github.com/redis/go-redis/v9"
)

type Provider interface {
	Manager() batchflow.Manager
	Connect() *connect.ConnectDTO
}

type provider struct {
	manager    batchflow.Manager
	conn       *connect.ConnectDTO
	components batchflow.PreviewComponents
}

const providerContextKey = "bulkprocess.provider"

func (p *provider) Manager() batchflow.Manager {
	return p.manager
}

func (p *provider) Connect() *connect.ConnectDTO {
	return p.conn
}

func (p *provider) PreviewComponents() batchflow.PreviewComponents {
	return p.components
}

func NewProviderWithConfig(appCfg *config.AppConfig, conn *connect.ConnectDTO, redisClient *redis.Client) (Provider, error) {
	if conn == nil || conn.ConnectGormWrite == nil || conn.ConnectGormRead == nil {
		return nil, fmt.Errorf("connect dto inválido")
	}
	if redisClient == nil {
		return nil, fmt.Errorf("redis client inválido")
	}

	cache := batchflow.NewRedisCache(redisClient)
	stateStore := batchflow.NewRedisStateStore(cache)
	runControl := batchflow.NewRunControl(cache, batchflowTTL(appCfg))
	lifecycle := NewParentLifecycle(conn.ConnectGormRead, conn.ConnectGormWrite)
	dataProvider := NewDataProvider(conn.ConnectGormRead)
	processor := NewBulkJobProcessor(conn.ConnectGormWrite)
	finalizer := NewBulkJobFinalizer(conn.ConnectGormRead)

	defaultTTL := batchflowTTL(appCfg)
	manager := batchflow.NewManager(
		lifecycle,
		dataProvider,
		processor,
		finalizer,
		stateStore,
		defaultTTL,
		runControl,
	)

	return &provider{
		manager: manager,
		conn:    conn,
		components: batchflow.PreviewComponents{
			DataProvider:   dataProvider,
			BatchProcessor: processor,
			BatchPreviewer: processor,
			StateStore:     stateStore,
		},
	}, nil
}

func batchflowTTL(appCfg *config.AppConfig) time.Duration {
	_ = appCfg
	return 24 * time.Hour
}

func WithProvider(ctx context.Context, prov Provider) context.Context {
	return runtimectx.WithNamedValue(ctx, providerContextKey, prov)
}

func ProviderFromContext(ctx context.Context) (Provider, error) {
	if prov, ok := runtimectx.NamedValue[Provider](ctx, providerContextKey); ok && prov != nil {
		return prov, nil
	}
	return nil, fmt.Errorf("bulkprocess provider no disponible en contexto")
}

const processTypeName = "procesar lote generico"

func init() {
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
		"bulk/process/generic/start",
		"bulk/process/generic/dispatch_shards",
		"bulk/process/generic/process_batch",
		"bulk/process/generic/finalize",
	)
	batchflow.RegisterManagedBatchManager(func(ctx context.Context) (batchflow.Manager, error) {
		prov, err := ProviderFromContext(ctx)
		if err != nil {
			return nil, err
		}
		return prov.Manager(), nil
	},
		"bulk/process/generic/start",
		"bulk/process/generic/dispatch_shards",
		"bulk/process/generic/process_batch",
		"bulk/process/generic/finalize",
	)
}
