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
	"go-fiber-core/internal/services/processlifecycle"
	processmanager "go-fiber-core/internal/services/processlifecyclemanager"
)

const defaultConfigPath = "internal/appconfig/config.yml"

type options struct {
	ConfigPath       string
	SourceVersionID  int64
	OperatorID       int64
	WithPacing       bool
	PacingMessages   int
	PacingInterval   int
	ProcessBatchStep string
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

	service := processlifecycle.NewService(conn)
	newVersionID, err := service.ReplicateProcessVersion(ctx, opts.SourceVersionID, opts.OperatorID)
	if err != nil {
		fatal(fmt.Errorf("replicando process_version %d: %w", opts.SourceVersionID, err))
	}

	fmt.Println("Version clonada correctamente")
	fmt.Printf("Source Version ID: %d\n", opts.SourceVersionID)
	fmt.Printf("New Version ID: %d\n", newVersionID)

	if !opts.WithPacing {
		return
	}

	steps, err := service.GetProcessStepsByVersionID(ctx, newVersionID)
	if err != nil {
		fatal(fmt.Errorf("obteniendo steps de version clonada %d: %w", newVersionID, err))
	}

	targetStep, updatedConfig, err := patchReplicatedSteps(steps, opts)
	if err != nil {
		fatal(err)
	}

	if err := conn.ConnectGormWrite.WithContext(ctx).
		Model(&processmanager.ProcessStep{}).
		Where("process_version_id = ? AND step_order = ?", newVersionID, targetStep.StepOrder).
		Update("config", updatedConfig).Error; err != nil {
		fatal(fmt.Errorf("actualizando process_batch en version %d: %w", newVersionID, err))
	}

	fmt.Printf("Process Batch Step: %s\n", targetStep.ExecutionKey)
	fmt.Printf("Dispatch Pacing: enabled=true messages_per_interval=%d interval_seconds=%d\n", opts.PacingMessages, opts.PacingInterval)
}

func parseOptions() options {
	var opts options
	flag.StringVar(&opts.ConfigPath, "config", defaultConfigPath, "Ruta al config.yml")
	flag.Int64Var(&opts.SourceVersionID, "source-version-id", 0, "Version origen a clonar")
	flag.Int64Var(&opts.OperatorID, "operator-id", 0, "Operator ID que crea la nueva version")
	flag.BoolVar(&opts.WithPacing, "with-pacing", false, "Aplica dispatch_pacing al process_batch de la version clonada")
	flag.IntVar(&opts.PacingMessages, "pacing-messages", 100, "Cantidad de items por invocacion cuando with_pacing=true")
	flag.IntVar(&opts.PacingInterval, "pacing-interval", 2, "Delay entre auto_invoke (1..10) cuando with_pacing=true")
	flag.StringVar(&opts.ProcessBatchStep, "process-batch-step", "", "Execution key exacta del step process_batch si quieres forzarla")
	flag.Parse()
	return opts
}

func validateOptions(opts options) error {
	if strings.TrimSpace(opts.ConfigPath) == "" {
		return fmt.Errorf("config es requerido")
	}
	if opts.SourceVersionID <= 0 {
		return fmt.Errorf("source_version_id debe ser mayor a 0")
	}
	if opts.OperatorID <= 0 {
		return fmt.Errorf("operator_id debe ser mayor a 0")
	}
	if !opts.WithPacing {
		return nil
	}
	if opts.PacingMessages <= 0 {
		return fmt.Errorf("pacing_messages debe ser mayor a 0 cuando with_pacing=true")
	}
	if opts.PacingInterval < 1 || opts.PacingInterval > 10 {
		return fmt.Errorf("pacing_interval debe estar entre 1 y 10 cuando with_pacing=true")
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

func patchReplicatedSteps(steps []processlifecycle.Step, opts options) (processlifecycle.Step, json.RawMessage, error) {
	step, err := selectProcessBatchStep(steps, opts.ProcessBatchStep)
	if err != nil {
		return processlifecycle.Step{}, nil, err
	}
	updated, err := patchProcessBatchConfig(step.Config, opts.PacingMessages, opts.PacingInterval)
	if err != nil {
		return processlifecycle.Step{}, nil, fmt.Errorf("parchando config de %s: %w", step.ExecutionKey, err)
	}
	return step, updated, nil
}

func selectProcessBatchStep(steps []processlifecycle.Step, forcedExecutionKey string) (processlifecycle.Step, error) {
	forcedExecutionKey = strings.TrimSpace(forcedExecutionKey)
	matches := make([]processlifecycle.Step, 0, 1)
	for _, step := range steps {
		if forcedExecutionKey != "" {
			if step.ExecutionKey == forcedExecutionKey {
				return step, nil
			}
			continue
		}
		if isProcessBatchExecutionKey(step.ExecutionKey) {
			matches = append(matches, step)
		}
	}
	if forcedExecutionKey != "" {
		return processlifecycle.Step{}, fmt.Errorf("no se encontro process_batch_step=%q en la version clonada", forcedExecutionKey)
	}
	if len(matches) == 0 {
		return processlifecycle.Step{}, fmt.Errorf("la version clonada no tiene un step process_batch")
	}
	if len(matches) > 1 {
		return processlifecycle.Step{}, fmt.Errorf("se encontraron multiples steps process_batch; usa process_batch_step para elegir uno")
	}
	return matches[0], nil
}

func isProcessBatchExecutionKey(key string) bool {
	key = strings.TrimSpace(strings.ToLower(key))
	return strings.HasSuffix(key, "/process_batch") || key == "process_batch" || strings.Contains(key, "process_batch")
}

func patchProcessBatchConfig(raw json.RawMessage, pacingMessages, pacingInterval int) (json.RawMessage, error) {
	cfg := make(map[string]any)
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &cfg); err != nil {
			return nil, fmt.Errorf("config json invalido: %w", err)
		}
	}

	cfg["dispatch_pacing"] = map[string]any{
		"enabled":               true,
		"messages_per_interval": pacingMessages,
		"interval_seconds":      pacingInterval,
	}

	policy := ensureMap(cfg, "execution_policy")
	policy["mode"] = "ASYNC"
	autoInvoke := ensureMap(policy, "auto_invoke")
	autoInvoke["enabled"] = true
	autoInvoke["cursor_field"] = firstNonEmptyString(autoInvoke["cursor_field"], "batch_index")
	autoInvoke["stop_condition"] = firstNonEmptyString(autoInvoke["stop_condition"], inferStopCondition(cfg))
	autoInvoke["delay_seconds"] = pacingInterval

	updated, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return nil, err
	}
	return json.RawMessage(updated), nil
}

func ensureMap(parent map[string]any, key string) map[string]any {
	if current, ok := parent[key]; ok {
		if typed, ok := current.(map[string]any); ok {
			return typed
		}
	}
	next := make(map[string]any)
	parent[key] = next
	return next
}

func inferStopCondition(cfg map[string]any) string {
	executionMode, ok := cfg["execution_mode"].(map[string]any)
	if !ok {
		return "is_last_batch"
	}
	if strings.EqualFold(strings.TrimSpace(stringValue(executionMode["type"])), "fanout") {
		return "is_shard_complete"
	}
	return "is_last_batch"
}

func firstNonEmptyString(value any, fallback string) string {
	if s := strings.TrimSpace(stringValue(value)); s != "" {
		return s
	}
	return fallback
}

func stringValue(value any) string {
	if value == nil {
		return ""
	}
	switch v := value.(type) {
	case string:
		return v
	default:
		return fmt.Sprintf("%v", v)
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "Error:", err)
	os.Exit(1)
}
