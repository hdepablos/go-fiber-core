package runtimebootstrap

import (
	"context"
	"fmt"
	"strings"

	"go-fiber-core/internal/dtos/config"
	"go-fiber-core/internal/dtos/connect"
	"go-fiber-core/internal/services/bulkprocess"
	"go-fiber-core/internal/services/dispatcher"
	"go-fiber-core/internal/services/queue"
	"go-fiber-core/internal/services/runtimectx"
	"go-fiber-core/internal/services/batchprocess/punitive"
)

type Dependencies struct {
	Dispatcher  dispatcher.Dispatcher
	BulkProcess bulkprocess.Provider
	Punitive punitive.Provider
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

	var errs []string

	if prov, err := bulkprocess.NewProviderWithConfig(appCfg, conn, conn.ConnectRedis); err == nil {
		deps.BulkProcess = prov
	} else {
		errs = append(errs, fmt.Sprintf("bulkprocess: %v", err))
	}

	if prov, err := punitive.NewProviderWithConfig(appCfg, conn, conn.ConnectRedis); err == nil {
		deps.Punitive = prov
	} else {
		errs = append(errs, fmt.Sprintf("punitive: %v", err))
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
	if d.BulkProcess != nil {
		ctx = bulkprocess.WithProvider(ctx, d.BulkProcess)
	}
	if d.Punitive != nil {
		ctx = punitive.WithProvider(ctx, d.Punitive)
	}

	return ctx
}
