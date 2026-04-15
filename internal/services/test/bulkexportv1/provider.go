package bulkexportv1

import (
	"context"
	"fmt"
	"os"

	"go-fiber-core/internal/dtos/config"
	"go-fiber-core/internal/dtos/connect"
	"go-fiber-core/internal/services/runtimectx"

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

const providerContextKey = "bulkexportv1.provider"

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

func WithProvider(ctx context.Context, prov Provider) context.Context {
	return runtimectx.WithNamedValue(ctx, providerContextKey, prov)
}

func ProviderFromContext(ctx context.Context) (Provider, error) {
	if prov, ok := runtimectx.NamedValue[Provider](ctx, providerContextKey); ok && prov != nil {
		return prov, nil
	}
	return nil, fmt.Errorf("bulkexportv1 provider no disponible en contexto")
}

func projectPrefix() string {
	prefix := os.Getenv("APP_NAME")
	if prefix == "" {
		prefix = "go-fiber-core"
	}
	return prefix
}
