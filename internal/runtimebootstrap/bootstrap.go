 package runtimebootstrap

import (
	"context"
	"fmt"
	"strings"

	"go-fiber-core/internal/dtos/config"
	"go-fiber-core/internal/dtos/connect"
	"go-fiber-core/internal/services/dispatcher"
	galicia "go-fiber-core/internal/services/generar_archivo_banco_galicia"
	"go-fiber-core/internal/services/queue"
	"go-fiber-core/internal/services/runtimectx"
	bulkexportv2 "go-fiber-core/internal/services/test/bulkexportV2"
	bulkexportv1 "go-fiber-core/internal/services/test/bulkexportv1"
)

type Dependencies struct {
	Dispatcher dispatcher.Dispatcher
	BulkV1     bulkexportv1.Provider
	BulkV2     bulkexportv2.Provider
	Galicia    galicia.Provider
}

func Build(ctx context.Context, appCfg *config.AppConfig, conn *connect.ConnectDTO, queueService queue.MessageQueue) (*Dependencies, error) {
	deps := &Dependencies{
		Dispatcher: dispatcher.NewProcessDispatcherService(),
	}
	if deps.Dispatcher != nil && queueService != nil {
		deps.Dispatcher.SetQueueService(queueService)
	}

	if appCfg == nil || conn == nil || conn.ConnectRedis == nil {
		return deps, nil
	}

	awsSvc, err := queue.NewAWSService(ctx)
	if err != nil {
		return deps, fmt.Errorf("runtime bootstrap aws: %w", err)
	}
	s3Client := awsSvc.NewS3Client()

	var errs []string

	if prov, err := bulkexportv1.NewProviderWithConfig(appCfg, conn, conn.ConnectRedis, s3Client); err == nil {
		deps.BulkV1 = prov
	} else {
		errs = append(errs, fmt.Sprintf("bulkexportv1: %v", err))
	}

	if prov, err := bulkexportv2.NewProviderWithConfig(appCfg, conn, conn.ConnectRedis, s3Client); err == nil {
		deps.BulkV2 = prov
	} else {
		errs = append(errs, fmt.Sprintf("bulkexportv2: %v", err))
	}

	if prov, err := galicia.NewProviderWithConfig(appCfg, conn, conn.ConnectRedis, s3Client); err == nil {
		deps.Galicia = prov
	} else {
		errs = append(errs, fmt.Sprintf("galicia: %v", err))
	}

	if len(errs) > 0 {
		return deps, fmt.Errorf("%s", strings.Join(errs, "; "))
	}
	return deps, nil
}

func (d *Dependencies) Inject(ctx context.Context) context.Context {
	if d == nil {
		return ctx
	}
	if d.Dispatcher != nil {
		ctx = runtimectx.WithDispatcher(ctx, d.Dispatcher)
	}
	if d.BulkV1 != nil {
		ctx = bulkexportv1.WithProvider(ctx, d.BulkV1)
	}
	if d.BulkV2 != nil {
		ctx = bulkexportv2.WithProvider(ctx, d.BulkV2)
	}
	if d.Galicia != nil {
		ctx = galicia.WithProvider(ctx, d.Galicia)
	}
	return ctx
}
