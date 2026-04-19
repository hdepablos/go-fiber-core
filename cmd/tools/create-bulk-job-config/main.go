package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"

	dbgorm "go-fiber-core/internal/database/connections/gorm"
	"go-fiber-core/internal/dtos/config"
	"go-fiber-core/internal/dtos/connect"
	"go-fiber-core/internal/dtos/requests"
	"go-fiber-core/internal/models"
	bulkjobconfigrepo "go-fiber-core/internal/repositories/bulkjobconfig"
)

const (
	defaultConfigPath = "internal/appconfig/config.yml"
	defaultStep       = int64(5)
	defaultOperatorID = uint64(1)
)

type options struct {
	ConfigPath               string
	OperatorID               uint64
	ProcessTypeID            int64
	SedeID                   int64
	OverrideProcessVersionID int64
	Roadmap                  int
}

func main() {
	opts := parseOptions()
	if err := validateOptions(opts); err != nil {
		fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	conn, cleanup, err := initializeConnections(opts.ConfigPath)
	if err != nil {
		fatal(err)
	}
	defer cleanup()

	reader := bulkjobconfigrepo.NewBulkJobConfigReaderRepo()
	writer := bulkjobconfigrepo.NewBulkJobConfigWriterRepo()

	refCode, err := reader.GetNextRefCode(ctx, conn.ConnectGormRead, defaultStep)
	if err != nil {
		fatal(fmt.Errorf("obteniendo proximo ref_code: %w", err))
	}

	rawConfig, err := buildDefaultConfig(opts)
	if err != nil {
		fatal(fmt.Errorf("armando config default: %w", err))
	}

	record := &models.BulkJobConfig{
		OperatorID: opts.OperatorID,
		RefCode:    refCode,
		IsActive:   true,
		Config:     rawConfig,
	}

	if err := writer.Create(ctx, conn.ConnectGormWrite, record); err != nil {
		fatal(fmt.Errorf("creando bulk_job_config: %w", err))
	}

	fmt.Println("Bulk job config creado correctamente")
	fmt.Println("Origen: solicitado por el comando")
	fmt.Printf("ID: %d\n", record.ID)
	fmt.Printf("Ref Code: %s\n", record.RefCode)
	fmt.Printf("Operator ID: %d\n", record.OperatorID)
	fmt.Println("Config:")
	fmt.Println(string(rawConfig))
}

func parseOptions() options {
	var opts options
	flag.StringVar(&opts.ConfigPath, "config", defaultConfigPath, "Ruta al config.yml")
	flag.Uint64Var(&opts.OperatorID, "operator-id", defaultOperatorID, "Operator ID dueño del bulk_job_config")
	flag.Int64Var(&opts.ProcessTypeID, "process-type-id", 0, "Process type ID del flujo a ejecutar")
	flag.Int64Var(&opts.SedeID, "sede-id", 0, "Sede ID para el request default")
	flag.Int64Var(&opts.OverrideProcessVersionID, "override-process-version-id", 0, "Override process version ID para el request default")
	flag.IntVar(&opts.Roadmap, "roadmap", 0, "Roadmap para el request default")
	flag.Parse()
	return opts
}

func validateOptions(opts options) error {
	if strings.TrimSpace(opts.ConfigPath) == "" {
		return fmt.Errorf("config es requerido")
	}
	if opts.ProcessTypeID <= 0 {
		return fmt.Errorf("process_type_id debe ser mayor a 0")
	}
	if opts.SedeID < 0 {
		return fmt.Errorf("sede_id no puede ser negativo")
	}
	if opts.OverrideProcessVersionID < 0 {
		return fmt.Errorf("override_process_version_id no puede ser negativo")
	}
	if opts.Roadmap < 0 {
		return fmt.Errorf("roadmap no puede ser negativo")
	}
	return nil
}

func initializeConnections(configPath string) (*connect.ConnectDTO, func(), error) {
	_ = godotenv.Load()
	appCfg, err := config.NewAppConfig(configPath)
	if err != nil {
		return nil, nil, err
	}

	gormService, cleanup, err := dbgorm.NewGormConnectService(appCfg.MultiDatabaseConfig)
	if err != nil {
		return nil, nil, err
	}

	conn := &connect.ConnectDTO{
		ConnectGormWrite: gormService.GetWriteDB(),
		ConnectGormRead:  gormService.GetReadDB(),
	}
	return conn, cleanup, nil
}

func buildDefaultConfig(opts options) ([]byte, error) {
	sedeID := opts.SedeID
	overrideProcessVersionID := opts.OverrideProcessVersionID
	roadmap := opts.Roadmap

	req := requests.RunProcessRequest{
		ProcessTypeID:            opts.ProcessTypeID,
		SedeID:                   &sedeID,
		OverrideProcessVersionID: &overrideProcessVersionID,
		Roadmap:                  &roadmap,
		Input:                    map[string]any{},
	}

	return json.MarshalIndent(req, "", "  ")
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
