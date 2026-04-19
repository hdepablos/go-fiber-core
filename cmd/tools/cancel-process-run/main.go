package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"

	redisconn "go-fiber-core/internal/database/connections/redis"
	"go-fiber-core/internal/dtos/config"
	"go-fiber-core/internal/services/batchflow"
)

const defaultConfigPath = "internal/appconfig/config.yml"

type options struct {
	ConfigPath string
	RunKey     string
	BulkJobID  int64
	Reason     string
}

func main() {
	opts := parseOptions()
	if err := validateOptions(opts); err != nil {
		fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	control, cleanup, err := initializeRunControl(opts.ConfigPath)
	if err != nil {
		fatal(err)
	}
	defer cleanup()

	status, err := control.Cancel(ctx, batchflow.RunCancelRequest{
		RunKey:      strings.TrimSpace(opts.RunKey),
		ParentID:    opts.BulkJobID,
		Reason:      strings.TrimSpace(opts.Reason),
		RequestedBy: "command:cancel-process-run",
		Source:      "cmd_tools",
	})
	if err != nil {
		fatal(err)
	}

	fmt.Println("Corrida cancelada correctamente")
	fmt.Printf("Run Key: %s\n", status.RunKey)
	fmt.Printf("Bulk Job ID: %d\n", status.ParentID)
	fmt.Printf("State: %s\n", status.State)
	fmt.Printf("Cancelled: %t\n", status.Cancelled)
	fmt.Printf("Reason: %s\n", status.Reason)
}

func parseOptions() options {
	var opts options
	flag.StringVar(&opts.ConfigPath, "config", defaultConfigPath, "Ruta al config.yml")
	flag.StringVar(&opts.RunKey, "run-key", "", "Run key / key_redis de la corrida")
	flag.Int64Var(&opts.BulkJobID, "bulk-job-id", 0, "Bulk job ID para resolver la corrida activa")
	flag.StringVar(&opts.Reason, "reason", "manual_cancel", "Motivo de la cancelación")
	flag.Parse()
	return opts
}

func validateOptions(opts options) error {
	if strings.TrimSpace(opts.ConfigPath) == "" {
		return fmt.Errorf("config es requerido")
	}
	if strings.TrimSpace(opts.RunKey) == "" && opts.BulkJobID <= 0 {
		return fmt.Errorf("debes especificar run_key o bulk_job_id")
	}
	return nil
}

func initializeRunControl(configPath string) (*batchflow.RunControl, func(), error) {
	_ = godotenv.Load()

	appCfg, err := config.NewAppConfig(configPath)
	if err != nil {
		return nil, nil, err
	}

	client, cleanup, err := redisconn.NewRedisClient(appCfg.Redis)
	if err != nil {
		return nil, nil, err
	}

	control := batchflow.NewRunControl(batchflow.NewRedisCache(client), 24*time.Hour)
	return control, cleanup, nil
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
