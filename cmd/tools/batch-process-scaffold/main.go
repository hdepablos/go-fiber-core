package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
)

type options struct {
	ProcessName       string
	ServiceSlug       string
	BatchSize         int
	ConcurrentBatches int
	RedisTTLHours     int
	BulkJobID         int64
	WithBruno         bool
	Force             bool
}

type scaffoldData struct {
	ProcessName       string
	ServiceSlug       string
	PackageName       string
	PascalName        string
	DependencyField   string
	SeedName          string
	SeederFuncName    string
	ExecutionBase     string
	StartKey          string
	ProcessBatchKey   string
	FinalizeKey       string
	BatchSize         int
	ConcurrentBatches int
	RedisTTLHours     int
	BulkJobID         int64
	ProcessTypeVar    string
	BulkJobVar        string
	RedisKeyVar       string
	ImportPath        string
}

type generatedPaths struct {
	serviceDir              string
	providerFile            string
	stepsFile               string
	dataProviderFile        string
	processorFile           string
	lifecycleFile           string
	seederFile              string
	runBrunoFile            string
	previewFolderFile       string
	previewPrepareFile      string
	previewAllFile          string
	previewBatchIndexFile   string
	previewItemIDsFile      string
	previewApplyItemIDsFile string
	previewRowNumbersFile   string
}

func main() {
	opts := parseOptions()
	if err := enrichOptions(&opts); err != nil {
		fatal(err)
	}

	data := buildScaffoldData(opts)
	paths := scaffoldPaths(data)
	if opts.WithBruno {
		fmt.Println("Aviso: batch-process ya no genera carpetas ni requests Bruno especificos; usar bruno/legacy/process-lifecycle/test-batch-process")
	}

	if err := ensurePaths(paths, opts.WithBruno, opts.Force); err != nil {
		fatal(err)
	}

	if err := os.MkdirAll(paths.serviceDir, 0o755); err != nil {
		fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(paths.seederFile), 0o755); err != nil {
		fatal(err)
	}

	files := map[string]string{
		paths.providerFile:     renderProvider(data),
		paths.stepsFile:        renderSteps(data),
		paths.dataProviderFile: renderDataProvider(data),
		paths.processorFile:    renderProcessor(data),
		paths.lifecycleFile:    renderLifecycle(data),
		paths.seederFile:       renderSeeder(data),
	}

	for path, content := range files {
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			fatal(fmt.Errorf("escribiendo %s: %w", path, err))
		}
	}

	if err := patchImportBlock("/private/var/www/go-fiber-core/cmd/api/main.go", fmt.Sprintf(`_ "go-fiber-core/internal/services/%s"`, data.ServiceSlug)); err != nil {
		fatal(err)
	}
	if err := patchImportBlock("/private/var/www/go-fiber-core/cmd/sqs-consumer/main.go", fmt.Sprintf(`_ "go-fiber-core/internal/services/%s"`, data.ServiceSlug)); err != nil {
		fatal(err)
	}
	if err := patchSeedService(data); err != nil {
		fatal(err)
	}
	if err := patchRuntimeBootstrap(data); err != nil {
		fatal(err)
	}

	fmt.Println("Scaffold batch process generado correctamente")
	fmt.Printf("Process Name: %s\n", data.ProcessName)
	fmt.Printf("Service Slug: %s\n", data.ServiceSlug)
	fmt.Println("Execution Keys:")
	fmt.Printf("- %s\n", data.StartKey)
	fmt.Printf("- %s\n", data.ProcessBatchKey)
	fmt.Printf("- %s\n", data.FinalizeKey)
	fmt.Println("Archivos creados:")
	for _, path := range sortedFileList(paths, opts.WithBruno) {
		fmt.Printf("- %s\n", path)
	}
	fmt.Println("Siguientes pasos:")
	fmt.Println("1. Ajustar el DataProvider si el origen no es bulk_job_items")
	fmt.Println("2. Reemplazar la logica de ProcessBatch/PreviewBatch")
	fmt.Println("3. Ajustar el ParentLifecycle/Finalizer si el padre no es bulk_jobs")
	fmt.Printf("4. Ejecutar el seeder: make seed-one name=%s\n", data.SeedName)
	fmt.Println("5. Usar bruno/legacy/process-lifecycle/test-batch-process con process_type_id, override_process_version_id y sede_id")
}

func parseOptions() options {
	var opts options
	flag.StringVar(&opts.ProcessName, "process-name", "", "Nombre del proceso")
	flag.StringVar(&opts.ServiceSlug, "service-slug", "", "Slug tecnico del servicio")
	flag.IntVar(&opts.BatchSize, "batch-size", 500, "Tamano del lote")
	flag.IntVar(&opts.ConcurrentBatches, "concurrent-batches", 1, "Cantidad de lotes por invocacion")
	flag.IntVar(&opts.RedisTTLHours, "redis-ttl-hours", 24, "TTL de Redis en horas")
	flag.Int64Var(&opts.BulkJobID, "bulk-job-id", 0, "BulkJobID sugerido para Bruno")
	flag.BoolVar(&opts.WithBruno, "with-bruno", false, "Genera requests Bruno base (deshabilitado por defecto: usar test-batch-process parametrizado)")
	flag.BoolVar(&opts.Force, "force", false, "Sobrescribe archivos generados si existen")
	flag.Parse()
	return opts
}

func enrichOptions(opts *options) error {
	reader := bufio.NewReader(os.Stdin)
	opts.ProcessName = strings.TrimSpace(opts.ProcessName)
	if opts.ProcessName == "" {
		opts.ProcessName = ask(reader, "process_name")
	}
	if opts.ProcessName == "" {
		return fmt.Errorf("process_name es requerido")
	}

	opts.ServiceSlug = strings.TrimSpace(opts.ServiceSlug)
	if opts.ServiceSlug == "" {
		opts.ServiceSlug = slugify(opts.ProcessName)
	}
	opts.ServiceSlug = normalizeIdentifier(opts.ServiceSlug)
	if opts.ServiceSlug == "" {
		return fmt.Errorf("service_slug invalido")
	}
	if !regexp.MustCompile(`^[a-z0-9_]+$`).MatchString(opts.ServiceSlug) {
		return fmt.Errorf("service_slug invalido: usa solo minusculas, numeros y underscore")
	}

	if opts.BatchSize <= 0 {
		opts.BatchSize = 500
	}
	if opts.ConcurrentBatches <= 0 {
		opts.ConcurrentBatches = 1
	}
	if opts.RedisTTLHours <= 0 {
		opts.RedisTTLHours = 24
	}
	return nil
}

func buildScaffoldData(opts options) scaffoldData {
	pascal := toPascal(opts.ServiceSlug)
	execBase := fmt.Sprintf("bulk/process/%s", opts.ServiceSlug)
	return scaffoldData{
		ProcessName:       opts.ProcessName,
		ServiceSlug:       opts.ServiceSlug,
		PackageName:       opts.ServiceSlug,
		PascalName:        pascal,
		DependencyField:   pascal,
		SeedName:          "batch_process_" + opts.ServiceSlug,
		SeederFuncName:    "BatchProcess" + pascal + "Seeder",
		ExecutionBase:     execBase,
		StartKey:          execBase + "/start",
		ProcessBatchKey:   execBase + "/process_batch",
		FinalizeKey:       execBase + "/finalize",
		BatchSize:         opts.BatchSize,
		ConcurrentBatches: opts.ConcurrentBatches,
		RedisTTLHours:     opts.RedisTTLHours,
		BulkJobID:         opts.BulkJobID,
		ProcessTypeVar:    "process_type_id_" + opts.ServiceSlug,
		BulkJobVar:        "bulk_job_id_" + opts.ServiceSlug,
		RedisKeyVar:       "redis_key_" + opts.ServiceSlug,
		ImportPath:        "go-fiber-core/internal/services/" + opts.ServiceSlug,
	}
}

func scaffoldPaths(data scaffoldData) generatedPaths {
	serviceDir := filepath.Join("/private/var/www/go-fiber-core/internal/services", data.ServiceSlug)
	return generatedPaths{
		serviceDir:       serviceDir,
		providerFile:     filepath.Join(serviceDir, "provider.go"),
		stepsFile:        filepath.Join(serviceDir, "steps.go"),
		dataProviderFile: filepath.Join(serviceDir, "data_provider.go"),
		processorFile:    filepath.Join(serviceDir, "processor.go"),
		lifecycleFile:    filepath.Join(serviceDir, "lifecycle.go"),
		seederFile:       filepath.Join("/private/var/www/go-fiber-core/internal/database/seeders", data.ServiceSlug+"_seeder.go"),
	}
}

func ensurePaths(paths generatedPaths, withBruno bool, force bool) error {
	_ = withBruno
	if force {
		return nil
	}
	all := []string{
		paths.providerFile,
		paths.stepsFile,
		paths.dataProviderFile,
		paths.processorFile,
		paths.lifecycleFile,
		paths.seederFile,
	}
	for _, path := range all {
		if _, err := os.Stat(path); err == nil {
			return fmt.Errorf("el archivo ya existe: %s (usa --force si quieres sobrescribir)", path)
		}
	}
	return nil
}

func patchImportBlock(filePath, importLine string) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("leyendo %s: %w", filePath, err)
	}
	fileStr := string(content)
	if strings.Contains(fileStr, importLine) {
		return nil
	}

	start := strings.Index(fileStr, "import (")
	if start == -1 {
		return fmt.Errorf("no se encontro bloque import en %s", filePath)
	}
	rest := fileStr[start:]
	end := strings.Index(rest, "\n)")
	if end == -1 {
		return fmt.Errorf("no se encontro cierre de imports en %s", filePath)
	}
	insertPos := start + end + 1
	newContent := fileStr[:insertPos] + "\n\t" + importLine + fileStr[insertPos:]
	return os.WriteFile(filePath, []byte(newContent), 0o644)
}

func patchSeedService(data scaffoldData) error {
	filePath := "/private/var/www/go-fiber-core/internal/database/seeders/seed_service.go"
	content, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}
	fileStr := string(content)

	listEntry := fmt.Sprintf("\t\t%q,", data.SeedName)
	if !strings.Contains(fileStr, listEntry) {
		anchor := "\t\t\"all_menus\","
		if !strings.Contains(fileStr, anchor) {
			return fmt.Errorf("no se encontro anchor para ListSeedersNames")
		}
		fileStr = strings.Replace(fileStr, anchor, listEntry+"\n"+anchor, 1)
	}

	funcBlock := fmt.Sprintf(`	service.AddSeeder(%q, func() error {
		return %s(pool)
	})

`, data.SeedName, data.SeederFuncName)
	if !strings.Contains(fileStr, data.SeedName) || !strings.Contains(fileStr, data.SeederFuncName) {
		anchor := `	service.AddSeeder("all_menus", func() error {`
		if !strings.Contains(fileStr, anchor) {
			return fmt.Errorf("no se encontro anchor para registerSeeders")
		}
		fileStr = strings.Replace(fileStr, anchor, funcBlock+anchor, 1)
	}

	return os.WriteFile(filePath, []byte(fileStr), 0o644)
}

func patchRuntimeBootstrap(data scaffoldData) error {
	filePath := "/private/var/www/go-fiber-core/internal/runtimebootstrap/bootstrap.go"
	content, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}
	fileStr := string(content)

	// Add proper import line (no stray parentheses).
	importLine := fmt.Sprintf("%q", data.ImportPath)
	if strings.Contains(fileStr, importLine) {
		// ok
	} else if err := patchImportBlock(filePath, importLine); err != nil {
		return err
	} else {
		// re-read after patchImportBlock write
		content, err = os.ReadFile(filePath)
		if err != nil {
			return err
		}
		fileStr = string(content)
	}

	fieldLine := fmt.Sprintf("\t%s %s.Provider", data.DependencyField, data.PackageName)
	if !strings.Contains(fileStr, fieldLine) {
		structAnchor := "type Dependencies struct {"
		start := strings.Index(fileStr, structAnchor)
		if start == -1 {
			return fmt.Errorf("no se encontro Dependencies struct en runtimebootstrap")
		}
		rest := fileStr[start:]
		end := strings.Index(rest, "\n}")
		if end == -1 {
			return fmt.Errorf("no se encontro cierre de Dependencies struct")
		}
		insertPos := start + end
		fileStr = fileStr[:insertPos] + "\n" + fieldLine + fileStr[insertPos:]
	}

	buildBlock := fmt.Sprintf(`
	if prov, err := %s.NewProviderWithConfig(appCfg, conn, conn.ConnectRedis); err == nil {
		deps.%s = prov
	} else {
		errs = append(errs, fmt.Sprintf(%q, err))
	}
`, data.PackageName, data.DependencyField, data.ServiceSlug+": %v")
	if !strings.Contains(fileStr, fmt.Sprintf("deps.%s = prov", data.DependencyField)) {
		anchor := "\tif len(errs) > 0 {"
		if !strings.Contains(fileStr, anchor) {
			return fmt.Errorf("no se encontro anchor de build en runtimebootstrap")
		}
		fileStr = strings.Replace(fileStr, anchor, buildBlock+"\n"+anchor, 1)
	}

	injectBlock := fmt.Sprintf(`
	if d.%s != nil {
		ctx = %s.WithProvider(ctx, d.%s)
	}
`, data.DependencyField, data.PackageName, data.DependencyField)
	if !strings.Contains(fileStr, fmt.Sprintf("d.%s != nil", data.DependencyField)) {
		// Prefer inserting after the last known provider injection if present.
		anchors := []string{
			"\tif d.Galicia != nil {",
			"\tif d.BulkV2 != nil {",
			"\tif d.BulkProcess != nil {",
			"\tif d.BulkV1 != nil {",
			"\tif d.Dispatcher != nil {",
		}
		inserted := false
		for _, a := range anchors {
			pos := strings.Index(fileStr, a)
			if pos == -1 {
				continue
			}
			// Insert before return ctx, but after the closing brace of that block.
			blockEnd := strings.Index(fileStr[pos:], "\n\t}")
			if blockEnd == -1 {
				continue
			}
			insertPos := pos + blockEnd + len("\n\t}")
			fileStr = fileStr[:insertPos] + injectBlock + fileStr[insertPos:]
			inserted = true
			break
		}
		if !inserted {
			anchor := "\treturn ctx"
			if !strings.Contains(fileStr, anchor) {
				return fmt.Errorf("no se encontro anchor de inject en runtimebootstrap")
			}
			fileStr = strings.Replace(fileStr, anchor, injectBlock+"\n"+anchor, 1)
		}
	}

	return os.WriteFile(filePath, []byte(fileStr), 0o644)
}

func renderProvider(data scaffoldData) string {
	return fmt.Sprintf(`package %s

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

const providerContextKey = %q

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
		return nil, fmt.Errorf("connect dto invalido")
	}
	if redisClient == nil {
		return nil, fmt.Errorf("redis client invalido")
	}

	cache := batchflow.NewRedisCache(redisClient)
	stateStore := batchflow.NewRedisStateStore(cache)
	lifecycle := NewParentLifecycle(conn.ConnectGormRead, conn.ConnectGormWrite)
	dataProvider := NewDataProvider(conn.ConnectGormRead)
	processor := NewProcessor(conn.ConnectGormWrite)
	finalizer := NewFinalizer(conn.ConnectGormRead)

	manager := batchflow.NewManager(
		lifecycle,
		dataProvider,
		processor,
		finalizer,
		stateStore,
		batchflowTTL(appCfg),
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
	return %d * time.Hour
}

func WithProvider(ctx context.Context, prov Provider) context.Context {
	return runtimectx.WithNamedValue(ctx, providerContextKey, prov)
}

func ProviderFromContext(ctx context.Context) (Provider, error) {
	if prov, ok := runtimectx.NamedValue[Provider](ctx, providerContextKey); ok && prov != nil {
		return prov, nil
	}
	return nil, fmt.Errorf("provider no disponible en contexto")
}

const processTypeName = %q

func init() {
	batchflow.RegisterPreviewProvider(processTypeName, func(ctx context.Context) (batchflow.PreviewProvider, error) {
		prov, err := ProviderFromContext(ctx)
		if err != nil {
			return nil, err
		}
		previewable, ok := prov.(batchflow.PreviewProvider)
		if !ok {
			return nil, fmt.Errorf("provider de %%s no soporta preview", processTypeName)
		}
		return previewable, nil
	},
		%q,
		%q,
		%q,
	)
}
`, data.PackageName, data.PackageName+".provider", data.RedisTTLHours, data.ProcessName, data.StartKey, data.ProcessBatchKey, data.FinalizeKey)
}

func renderSteps(data scaffoldData) string {
	return fmt.Sprintf(`package %s

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go-fiber-core/internal/domain"
	"go-fiber-core/internal/services/batchflow"
	"go-fiber-core/internal/services/serviceconfig"
	"go-fiber-core/internal/services/serviceconfig/contracts"
	"go-fiber-core/internal/utils"
)

type startStep struct {
	ctx         *contracts.ServiceContext
	servicePath string
	batchSize   int
	ttlHours    int
}

func NewStartStep() contracts.Service {
	return &startStep{
		batchSize: %d,
		ttlHours:  %d,
	}
}

func (s *startStep) Init(ctx *contracts.ServiceContext, servicePath string) {
	s.ctx = ctx
	s.servicePath = servicePath
	if s.ctx != nil && s.ctx.CurrentStepConfig != nil {
		if v, ok := s.ctx.CurrentStepConfig["batch_size"]; ok {
			s.batchSize = utils.ToInt(v)
		}
		if v, ok := s.ctx.CurrentStepConfig["redis_ttl_hours"]; ok {
			s.ttlHours = utils.ToInt(v)
		}
	}
}

func (s *startStep) Execute() error {
	prov, err := ProviderFromContext(s.ctx.Ctx)
	if err != nil {
		return err
	}

	input, err := buildStartInput(s.ctx)
	if err != nil {
		return err
	}
	if input.Filters != nil {
		s.ctx.SetInputValue("filters", input.Filters)
	}

	res, err := prov.Manager().Start(s.ctx.Ctx, batchflow.StartRequest{
		Input:     input,
		BatchSize: s.batchSize,
		RedisTTL:  time.Duration(s.ttlHours) * time.Hour,
	})
	if err != nil {
		markFailure(prov, s.ctx.Ctx, input, err)
		return err
	}

	s.ctx.SetInputValue("id", input.ParentID)
	s.ctx.SetInputValue("key_redis", res.RedisKey)
	s.ctx.SetInputValue("batches_list_key", res.BatchesListKey)
	s.ctx.SetInputValue("total_batches", res.TotalBatches)
	s.ctx.SetInputValue("batch_index", 0)
	s.ctx.SetInputValue("is_last_batch", false)

	s.ctx.SetResult(s.servicePath, contracts.StepResult{
		Status:  "completed",
		Message: "batchflow start completado",
		Data: map[string]any{
			"key_redis":     res.RedisKey,
			"id":            input.ParentID,
			"total_batches": res.TotalBatches,
		},
	})
	return nil
}

type processBatchStep struct {
	ctx               *contracts.ServiceContext
	servicePath       string
	concurrentBatches int
}

func NewProcessBatchStep() contracts.Service {
	return &processBatchStep{concurrentBatches: %d}
}

func (s *processBatchStep) Init(ctx *contracts.ServiceContext, servicePath string) {
	s.ctx = ctx
	s.servicePath = servicePath
	if s.ctx != nil && s.ctx.CurrentStepConfig != nil {
		if v, ok := s.ctx.CurrentStepConfig["concurrent_batches"]; ok {
			s.concurrentBatches = utils.ToInt(v)
		}
	}
	if s.concurrentBatches <= 0 {
		s.concurrentBatches = 1
	}
}

func (s *processBatchStep) Execute() error {
	prov, err := ProviderFromContext(s.ctx.Ctx)
	if err != nil {
		return err
	}
	input, err := buildInput(s.ctx)
	if err != nil {
		return err
	}

	res, err := prov.Manager().ProcessBatch(s.ctx.Ctx, batchflow.ProcessRequest{
		Input:             input,
		BatchesListKey:    fmt.Sprint(utils.MustGetInputValue(s.ctx, "batches_list_key")),
		BatchIndex:        utils.ToInt(utils.GetInputValueOrDefault(s.ctx, "batch_index", 0)),
		TotalBatches:      utils.ToInt(utils.MustGetInputValue(s.ctx, "total_batches")),
		ConcurrentBatches: s.concurrentBatches,
	})
	if err != nil {
		markFailure(prov, s.ctx.Ctx, input, err)
		return err
	}

	s.ctx.SetResult(s.servicePath, contracts.StepResult{
		Status:  "completed",
		Message: "batch procesado",
		Data: map[string]any{
			"batch_index":       res.NextBatchIndex,
			"is_last_batch":     res.IsLastBatch,
			"processed_count":   res.ProcessedCount,
			"batches_processed": res.BatchesProcessed,
		},
	})
	return nil
}

type finalizeStep struct {
	ctx         *contracts.ServiceContext
	servicePath string
}

func NewFinalizeStep() contracts.Service {
	return &finalizeStep{}
}

func (s *finalizeStep) Init(ctx *contracts.ServiceContext, servicePath string) {
	s.ctx = ctx
	s.servicePath = servicePath
}

func (s *finalizeStep) Execute() error {
	prov, err := ProviderFromContext(s.ctx.Ctx)
	if err != nil {
		return err
	}
	input, err := buildInput(s.ctx)
	if err != nil {
		return err
	}

	result, err := prov.Manager().Finalize(s.ctx.Ctx, batchflow.FinalizeRequest{
		Input:          input,
		BatchesListKey: fmt.Sprint(utils.MustGetInputValue(s.ctx, "batches_list_key")),
		TotalBatches:   utils.ToInt(utils.GetInputValueOrDefault(s.ctx, "total_batches", 0)),
	})
	if err != nil {
		markFailure(prov, s.ctx.Ctx, input, err)
		return err
	}

	s.ctx.SetResult(s.servicePath, contracts.StepResult{
		Status:  "completed",
		Message: "proceso batch finalizado",
		Data: map[string]any{
			"metadata": result.Metadata,
			"summary":  result.Summary,
		},
	})
	return nil
}

func buildInput(ctx *contracts.ServiceContext) (batchflow.Input, error) {
	input := batchflow.Input{
		RedisKey: fmt.Sprint(utils.MustGetInputValue(ctx, "key_redis")),
		ParentID: utils.ToInt64(utils.MustGetInputValue(ctx, "id")),
	}
	if input.ParentID <= 0 {
		return batchflow.Input{}, fmt.Errorf("id invalido")
	}
	if input.RedisKey == "" {
		return batchflow.Input{}, fmt.Errorf("key_redis invalida")
	}
	if rawFilters, ok := ctx.GetInputValue("filters"); ok {
		input.Filters = rawFilters
	}
	return input, nil
}

func buildStartInput(ctx *contracts.ServiceContext) (batchflow.Input, error) {
	input := batchflow.Input{
		RedisKey: fmt.Sprint(utils.GetInputValueOrDefault(ctx, "key_redis", "")),
		ParentID: utils.ToInt64(utils.MustGetInputValue(ctx, "id")),
	}
	if input.ParentID <= 0 {
		return batchflow.Input{}, fmt.Errorf("id invalido")
	}
	if rawFilters, ok := ctx.GetInputValue("filters"); ok {
		input.Filters = rawFilters
	}
	return input, nil
}

func markFailure(prov Provider, ctx context.Context, input batchflow.Input, err error) {
	if errors.Is(err, domain.ErrBusinessRuleViolation) || errors.Is(err, domain.ErrInvalidArgument) {
		return
	}
	_ = prov.Manager().Fail(ctx, input, err)
}

func init() {
	serviceconfig.Register(%q, NewStartStep)
	serviceconfig.Register(%q, NewProcessBatchStep)
	serviceconfig.Register(%q, NewFinalizeStep)
}
`, data.PackageName, data.BatchSize, data.RedisTTLHours, data.ConcurrentBatches, data.StartKey, data.ProcessBatchKey, data.FinalizeKey)
}

func renderDataProvider(data scaffoldData) string {
	return fmt.Sprintf(`package %s

import (
	"context"
	"encoding/json"
	"fmt"

	"go-fiber-core/internal/models"
	"go-fiber-core/internal/services/batchflow"
	"go-fiber-core/internal/utils"

	"gorm.io/gorm"
)

type dataProvider struct {
	readDB *gorm.DB
}

func NewDataProvider(readDB *gorm.DB) batchflow.DataProvider {
	return &dataProvider{readDB: readDB}
}

func (p *dataProvider) LoadBatches(ctx context.Context, execCtx batchflow.ExecutionContext, batchSize int) (batchflow.LoadBatchesResult, error) {
	input := execCtx.Input
	if input.ParentID <= 0 {
		return batchflow.LoadBatchesResult{}, fmt.Errorf("id invalido")
	}
	if batchSize <= 0 {
		batchSize = %d
	}

	query := p.readDB.WithContext(ctx).
		Model(&models.BulkJobItem{}).
		Select("id", "bulk_job_id", "row_number", "reference_key", "status_code", "last_detail_message", "data", "created_at", "updated_at").
		Where("bulk_job_id = ?", input.ParentID).
		Order("id ASC")

	statusFilterApplied := false
	if input.Filters != nil {
		result, err := utils.ApplyBulkJobItemFilters(query, input.Filters)
		if err != nil {
			return batchflow.LoadBatchesResult{}, err
		}
		query = result.Query
		statusFilterApplied = result.StatusFilterApplied
	}
	if !statusFilterApplied {
		query = query.Where("status_code = ?", models.BulkJobStatusImported)
	}

	var items []models.BulkJobItem
	if err := query.Find(&items).Error; err != nil {
		return batchflow.LoadBatchesResult{}, err
	}

	batches := make([]batchflow.Batch, 0, (len(items)/batchSize)+1)
	current := batchflow.Batch{Items: make([]json.RawMessage, 0, batchSize)}
	for _, item := range items {
		payload, err := json.Marshal(map[string]any{
			"id":                  item.ID,
			"bulk_job_id":         item.BulkJobID,
			"row_number":          item.RowNumber,
			"reference_key":       item.ReferenceKey,
			"status_code":         item.StatusCode,
			"last_detail_message": item.LastDetailMessage,
			"data":                json.RawMessage(item.Data),
		})
		if err != nil {
			return batchflow.LoadBatchesResult{}, err
		}
		current.Items = append(current.Items, payload)
		if len(current.Items) == batchSize {
			batches = append(batches, current)
			current = batchflow.Batch{Items: make([]json.RawMessage, 0, batchSize)}
		}
	}
	if len(current.Items) > 0 {
		batches = append(batches, current)
	}

	return batchflow.LoadBatchesResult{
		Batches: batches,
		Summary: batchflow.Summary{
			TotalRecords: int64(len(items)),
			Metadata: map[string]any{
				"source":      "bulk_job_items",
				"bulk_job_id": input.ParentID,
			},
		},
	}, nil
}
`, data.PackageName, data.BatchSize)
}

func renderProcessor(data scaffoldData) string {
	return fmt.Sprintf(`package %s

import (
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"

	"go-fiber-core/internal/models"
	"go-fiber-core/internal/repositories/bulkjobitemmessage"
	"go-fiber-core/internal/services/batchflow"

	"gorm.io/gorm"
)

type processor struct {
	writeDB       *gorm.DB
	messageWriter bulkjobitemmessage.BulkJobItemMessageWriter
}

type batchItemPayload struct {
	ID                int64                `+"`json:\"id\"`"+`
	BulkJobID         int64                `+"`json:\"bulk_job_id\"`"+`
	RowNumber         int                  `+"`json:\"row_number\"`"+`
	ReferenceKey      string               `+"`json:\"reference_key\"`"+`
	StatusCode        models.BulkJobStatus `+"`json:\"status_code\"`"+`
	LastDetailMessage *string              `+"`json:\"last_detail_message\"`"+`
	Data              json.RawMessage      `+"`json:\"data\"`"+`
}

func NewProcessor(writeDB *gorm.DB) *processor {
	return &processor{
		writeDB:       writeDB,
		messageWriter: bulkjobitemmessage.NewBulkJobItemMessageWriterRepo(),
	}
}

func (p *processor) ProcessBatch(ctx context.Context, execCtx batchflow.ExecutionContext, batch batchflow.Batch) (batchflow.ProcessBatchResult, error) {
	items, previewItems, err := p.resolveItems(execCtx, batch)
	if err != nil {
		return batchflow.ProcessBatchResult{}, err
	}

	if err := p.writeDB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for i, item := range items {
			preview := previewItems[i]
			var lastDetail *string
			if preview.Message != "" {
				message := preview.Message
				lastDetail = &message
			}
			if err := tx.Model(&models.BulkJobItem{}).
				Where("id = ?", item.ID).
				Updates(map[string]any{
					"status_code":         preview.Status,
					"last_detail_message": lastDetail,
				}).Error; err != nil {
				return err
			}
			for _, msg := range preview.Messages {
				record := &models.BulkJobItemMessage{
					BulkJobItemID: item.ID,
					Severity:      msg.Severity,
					DetailMessage: msg.DetailMessage,
				}
				if msg.Code != "" {
					code := msg.Code
					record.Code = &code
				}
				if len(msg.Meta) > 0 {
					metaBytes, err := json.Marshal(msg.Meta)
					if err != nil {
						return err
					}
					record.Meta = &metaBytes
				}
				if err := p.messageWriter.Create(ctx, tx, record); err != nil {
					return err
				}
			}
		}
		return nil
	}); err != nil {
		return batchflow.ProcessBatchResult{}, err
	}

	return batchflow.ProcessBatchResult{
		ProcessedCount: len(items),
	}, nil
}

func (p *processor) PreviewBatch(ctx context.Context, execCtx batchflow.ExecutionContext, batch batchflow.Batch) (batchflow.PreviewBatchResult, error) {
	_, previewItems, err := p.resolveItems(execCtx, batch)
	if err != nil {
		return batchflow.PreviewBatchResult{}, err
	}
	return batchflow.PreviewBatchResult{
		Items:          previewItems,
		ProcessedCount: len(previewItems),
	}, nil
}

func (p *processor) resolveItems(execCtx batchflow.ExecutionContext, batch batchflow.Batch) ([]batchItemPayload, []batchflow.PreviewItemResult, error) {
	items := make([]batchItemPayload, 0, len(batch.Items))
	results := make([]batchflow.PreviewItemResult, 0, len(batch.Items))
	for _, raw := range batch.Items {
		var item batchItemPayload
		if err := json.Unmarshal(raw, &item); err != nil {
			return nil, nil, fmt.Errorf("unmarshal batch item: %%w", err)
		}
		items = append(items, item)
		results = append(results, buildPreviewResult(execCtx.Input.RedisKey, item))
	}
	return items, results, nil
}

// Reemplaza esta logica por la decision real del proceso.
// Se deja una version deterministica para que el scaffold sea ejecutable desde el minuto cero.
func buildPreviewResult(redisKey string, item batchItemPayload) batchflow.PreviewItemResult {
	bucket := hashBucket(redisKey, item.ID, item.RowNumber)
	status := models.BulkJobStatusProcessed
	message := ""
	messages := []batchflow.PreviewMessage{}

	switch {
	case bucket < 3:
		status = models.BulkJobStatusErrorProcess
		message = errorMessage(item, bucket)
		messages = append(messages, batchflow.PreviewMessage{
			Severity:      "ERROR",
			Code:          "ERROR_PROCESS",
			DetailMessage: message,
			Meta:          map[string]any{"bucket": bucket},
		})
	case bucket < 30:
		status = models.BulkJobStatusProcessedWithDetails
		message = detailMessage(item, bucket)
		messages = append(messages, batchflow.PreviewMessage{
			Severity:      "WARNING",
			Code:          "DETAIL_PROCESS",
			DetailMessage: message,
			Meta:          map[string]any{"bucket": bucket},
		})
	}

	return batchflow.PreviewItemResult{
		ItemID:       item.ID,
		RowNumber:    item.RowNumber,
		ReferenceKey: item.ReferenceKey,
		Status:       string(status),
		Message:      message,
		Messages:     messages,
		Metadata: map[string]any{
			"bulk_job_id": item.BulkJobID,
		},
	}
}

func hashBucket(redisKey string, itemID int64, rowNumber int) uint32 {
	h := fnv.New32a()
	_, _ = h.Write([]byte(fmt.Sprintf("%%s:%%d:%%d", redisKey, itemID, rowNumber)))
	return h.Sum32() %% 100
}

func errorMessage(item batchItemPayload, bucket uint32) string {
	options := []string{
		"Error validando el registro contra la politica del proveedor",
		"El proveedor externo rechazo el registro por datos inconsistentes",
		"No fue posible procesar el registro por una regla de negocio",
	}
	return fmt.Sprintf("%%s (item_id=%%d, row=%%d, bucket=%%d)", options[int(bucket)%%len(options)], item.ID, item.RowNumber, bucket)
}

func detailMessage(item batchItemPayload, bucket uint32) string {
	options := []string{
		"Registro procesado con observaciones",
		"Registro procesado con ajuste informado por el proveedor",
		"Registro procesado con detalle operativo",
	}
	return fmt.Sprintf("%%s (item_id=%%d, row=%%d, bucket=%%d)", options[int(bucket)%%len(options)], item.ID, item.RowNumber, bucket)
}
`, data.PackageName)
}

func renderLifecycle(data scaffoldData) string {
	return fmt.Sprintf(`package %s

import (
	"context"
	"fmt"

	"go-fiber-core/internal/domain"
	"go-fiber-core/internal/models"
	"go-fiber-core/internal/services/batchflow"

	"gorm.io/gorm"
)

type parentLifecycle struct {
	readDB  *gorm.DB
	writeDB *gorm.DB
}

func NewParentLifecycle(readDB, writeDB *gorm.DB) batchflow.ParentLifecycle {
	return &parentLifecycle{readDB: readDB, writeDB: writeDB}
}

func (l *parentLifecycle) Start(ctx context.Context, execCtx batchflow.ExecutionContext) error {
	var job models.BulkJob
	if err := l.readDB.WithContext(ctx).
		Select("id", "status_code").
		Where("id = ?", execCtx.Input.ParentID).
		First(&job).Error; err != nil {
		return err
	}
	if job.StatusCode != models.BulkJobStatusImported {
		return fmt.Errorf("%%w: el bulk_job %%d tiene status %%s", domain.ErrBusinessRuleViolation, execCtx.Input.ParentID, job.StatusCode)
	}
	return l.writeDB.WithContext(ctx).
		Model(&models.BulkJob{}).
		Where("id = ?", execCtx.Input.ParentID).
		Update("status_code", models.BulkJobStatusProcessing).Error
}

func (l *parentLifecycle) End(ctx context.Context, execCtx batchflow.ExecutionContext, result batchflow.FinalizeResult) error {
	status := models.BulkJobStatusProcessed
	if raw, ok := result.Metadata["bulk_job_status"].(string); ok && raw != "" {
		status = models.BulkJobStatus(raw)
	}
	updates := map[string]any{
		"status_code": status,
	}
	if raw, ok := result.Metadata["total_processed_items"]; ok {
		updates["total_processed_items"] = raw
	}
	return l.writeDB.WithContext(ctx).
		Model(&models.BulkJob{}).
		Where("id = ?", execCtx.Input.ParentID).
		Updates(updates).Error
}

func (l *parentLifecycle) Fail(ctx context.Context, execCtx batchflow.ExecutionContext, _ error) error {
	return l.writeDB.WithContext(ctx).
		Model(&models.BulkJob{}).
		Where("id = ?", execCtx.Input.ParentID).
		Update("status_code", models.BulkJobStatusErrorProcess).Error
}

type finalizer struct {
	readDB *gorm.DB
}

func NewFinalizer(readDB *gorm.DB) batchflow.Finalizer {
	return &finalizer{readDB: readDB}
}

func (f *finalizer) Finalize(ctx context.Context, execCtx batchflow.ExecutionContext, req batchflow.FinalizeRequest) (batchflow.FinalizeResult, error) {
	_ = req

	var rows []struct {
		StatusCode models.BulkJobStatus
		Total      int64
	}
	if err := f.readDB.WithContext(ctx).
		Model(&models.BulkJobItem{}).
		Select("status_code, COUNT(*) as total").
		Where("bulk_job_id = ?", execCtx.Input.ParentID).
		Group("status_code").
		Scan(&rows).Error; err != nil {
		return batchflow.FinalizeResult{}, err
	}

	counters := map[models.BulkJobStatus]int64{}
	var totalProcessed int64
	for _, row := range rows {
		counters[row.StatusCode] = row.Total
		totalProcessed += row.Total
	}

	finalStatus := models.BulkJobStatusProcessed
	errorCount := counters[models.BulkJobStatusErrorProcess]
	detailCount := counters[models.BulkJobStatusProcessedWithDetails]
	processedCount := counters[models.BulkJobStatusProcessed]

	switch {
	case errorCount > 0 && processedCount == 0 && detailCount == 0:
		finalStatus = models.BulkJobStatusErrorProcess
	case errorCount > 0 || detailCount > 0:
		finalStatus = models.BulkJobStatusProcessedWithDetails
	}

	summary := execCtx.Summary
	summary.Metadata = map[string]any{
		"status_counters": map[string]int64{
			string(models.BulkJobStatusProcessed):            processedCount,
			string(models.BulkJobStatusErrorProcess):         errorCount,
			string(models.BulkJobStatusProcessedWithDetails): detailCount,
		},
	}

	return batchflow.FinalizeResult{
		Summary: summary,
		Metadata: map[string]any{
			"bulk_job_status":       string(finalStatus),
			"total_processed_items": totalProcessed,
			"processed_count":       processedCount,
			"error_count":           errorCount,
			"detail_count":          detailCount,
		},
	}, nil
}
`, data.PackageName)
}

func renderSeeder(data scaffoldData) string {
	bt := "`"
	return fmt.Sprintf(`package seeders

import (
	"context"
	"fmt"
	"log/slog"

	pgx "github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func %s(pool *pgxpool.Pool) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultSeederTimeout)
	defer cancel()

	logger := slog.Default().With("seeder", %q)
	logger.Info("iniciando seeder")

	return executeInTransaction(ctx, pool, func(ctx context.Context, tx pgx.Tx) error {
		const processTypeName = %q
		const processDescription = "Scaffold de batchflow generado automaticamente"

		var processTypeID int64
		err := tx.QueryRow(ctx, "SELECT id FROM process_types WHERE name = $1 AND archived_at IS NULL", processTypeName).Scan(&processTypeID)
		if err != nil {
			if err == pgx.ErrNoRows {
				if err := tx.QueryRow(ctx,
					%sINSERT INTO process_types (name, description, is_visible)
					VALUES ($1, $2, $3)
					RETURNING id%s,
					processTypeName,
					processDescription,
					true,
				).Scan(&processTypeID); err != nil {
					return fmt.Errorf("insert process_types '%%s': %%w", processTypeName, err)
				}
			} else {
				return fmt.Errorf("select process_types '%%s': %%w", processTypeName, err)
			}
		}

		var versionID int64
		err = tx.QueryRow(ctx,
			%sSELECT id
			 FROM process_versions
			 WHERE process_type_id = $1 AND version_number = 1 AND sede_id IS NULL AND archived_at IS NULL%s,
			processTypeID,
		).Scan(&versionID)
		if err != nil {
			if err == pgx.ErrNoRows {
				if err := tx.QueryRow(ctx,
					%sINSERT INTO process_versions (process_type_id, version_number, status, operator_id, sede_id)
					VALUES ($1, $2, $3, $4, NULL)
					RETURNING id%s,
					processTypeID,
					1,
					"TEST",
					1,
				).Scan(&versionID); err != nil {
					return fmt.Errorf("insert process_versions '%%s': %%w", processTypeName, err)
				}
			} else {
				return fmt.Errorf("select process_versions '%%s': %%w", processTypeName, err)
			}
		}

		steps := []struct {
			Order        int
			Name         string
			ExecutionKey string
			Config       string
		}{
			{
				Order:        1,
				Name:         "Step 1: Preparar lotes",
				ExecutionKey: %q,
				Config:       %s{"batch_size":%d,"redis_ttl_hours":%d}%s,
			},
			{
				Order:        2,
				Name:         "Step 2: Procesar lotes",
				ExecutionKey: %q,
				Config: %s{
					"concurrent_batches": %d,
					"execution_policy": {
						"mode": "ASYNC",
						"label": %q,
						"auto_invoke": {
							"enabled": true,
							"cursor_field": "batch_index",
							"stop_condition": "is_last_batch"
						},
						"next_step": %q
					}
				}%s,
			},
			{
				Order:        3,
				Name:         "Step 3: Finalizar",
				ExecutionKey: %q,
				Config:       "{}",
			},
		}

		validExecutionKeys := make([]string, 0, len(steps))
		for _, s := range steps {
			validExecutionKeys = append(validExecutionKeys, s.ExecutionKey)
		}

		if _, err := tx.Exec(ctx,
			%sDELETE FROM process_steps
			 WHERE process_version_id = $1
			   AND NOT (execution_key = ANY($2))%s,
			versionID,
			validExecutionKeys,
		); err != nil {
			return fmt.Errorf("delete obsolete process_steps for '%%s': %%w", processTypeName, err)
		}

		for _, s := range steps {
			if err := UpsertStep(ctx, tx, versionID, s.Order, s.Name, s.ExecutionKey, s.Config); err != nil {
				return fmt.Errorf("upsert step failed (%%s): %%w", s.ExecutionKey, err)
			}
		}

		logger.Info("seeder completado exitosamente", "process_type_id", processTypeID, "version_id", versionID)
		return nil
	})
}
`, data.SeederFuncName, data.SeedName, data.ProcessName, bt, bt, bt, bt, bt, bt, data.StartKey, bt, data.BatchSize, data.RedisTTLHours, bt, data.ProcessBatchKey, bt, data.ConcurrentBatches, data.ProcessName, data.FinalizeKey, bt, data.FinalizeKey, bt, bt)
}

func renderRunBruno(data scaffoldData) string {
	idValue := 1
	if data.BulkJobID > 0 {
		idValue = int(data.BulkJobID)
	}
	return fmt.Sprintf(`meta {
  name: RunProc -> %s
  type: http
}

vars {
  %s: 0
  %s: %d
}

post {
  url: {{urlBase}}api/v1/process-lifecycle/run
  body: json
  auth: bearer
}

auth:bearer {
  token: {{access_token}}
}

headers {
  X-Client-Code: bruno
}

body:json {
  {
    "process_type_id": {{%s}},
    "sede_id": 0,
    "override_process_version_id": 0,
    "roadmap": 0,
    "input": {
      "id": {{%s}}
    }
  }
}
`, data.ServiceSlug, data.ProcessTypeVar, data.BulkJobVar, idValue, data.ProcessTypeVar, data.BulkJobVar)
}

func renderPreviewFolder(data scaffoldData) string {
	return fmt.Sprintf(`meta {
  name: %s
}
`, data.ServiceSlug)
}

func renderPreviewPrepareBruno(data scaffoldData) string {
	return fmt.Sprintf(`meta {
  name: preview - prepare
  type: http
  seq: 1
}

vars {
  %s: 0
  %s: %d
  %s: preview-%s
}

post {
  url: {{urlBase}}api/v1/process-lifecycle/batch-preview
  body: json
  auth: bearer
}

auth:bearer {
  token: {{access_token}}
}

headers {
  X-Client-Code: bruno
}

body:json {
  {
    "process_type_id": {{%s}},
    "sede_id": 0,
    "override_process_version_id": 0,
    "roadmap": 0,
    "mode": "prepare",
    "input": {
      "id": {{%s}},
      "key_redis": "{{%s}}"
    }
  }
}
`, data.ProcessTypeVar, data.BulkJobVar, brunoBulkJobID(data), data.RedisKeyVar, data.ServiceSlug, data.ProcessTypeVar, data.BulkJobVar, data.RedisKeyVar)
}

func renderPreviewAllBruno(data scaffoldData) string {
	return fmt.Sprintf(`meta {
  name: preview - all
  type: http
  seq: 2
}

post {
  url: {{urlBase}}api/v1/process-lifecycle/batch-preview
  body: json
  auth: bearer
}

auth:bearer {
  token: {{access_token}}
}

headers {
  X-Client-Code: bruno
}

body:json {
  {
    "process_type_id": {{%s}},
    "sede_id": 0,
    "override_process_version_id": 0,
    "roadmap": 0,
    "mode": "all",
    "limit": 10,
    "offset": 0,
    "input": {
      "id": {{%s}},
      "key_redis": "{{%s}}"
    }
  }
}
`, data.ProcessTypeVar, data.BulkJobVar, data.RedisKeyVar)
}

func renderPreviewBatchIndexBruno(data scaffoldData) string {
	return fmt.Sprintf(`meta {
  name: preview - batch - batch_index
  type: http
  seq: 3
}

post {
  url: {{urlBase}}api/v1/process-lifecycle/batch-preview
  body: json
  auth: bearer
}

auth:bearer {
  token: {{access_token}}
}

headers {
  X-Client-Code: bruno
}

body:json {
  {
    "process_type_id": {{%s}},
    "sede_id": 0,
    "override_process_version_id": 0,
    "roadmap": 0,
    "mode": "batch",
    "batch_index": 0,
    "limit": 10,
    "input": {
      "id": {{%s}},
      "key_redis": "{{%s}}"
    }
  }
}
`, data.ProcessTypeVar, data.BulkJobVar, data.RedisKeyVar)
}

func renderPreviewItemIDsBruno(data scaffoldData) string {
	return fmt.Sprintf(`meta {
  name: preview - batch - item_ids
  type: http
  seq: 4
}

post {
  url: {{urlBase}}api/v1/process-lifecycle/batch-preview
  body: json
  auth: bearer
}

auth:bearer {
  token: {{access_token}}
}

headers {
  X-Client-Code: bruno
}

body:json {
  {
    "process_type_id": {{%s}},
    "sede_id": 0,
    "override_process_version_id": 0,
    "roadmap": 0,
    "mode": "batch",
    "limit": 10,
    "item_ids": [1],
    "input": {
      "id": {{%s}},
      "key_redis": "{{%s}}"
    }
  }
}
`, data.ProcessTypeVar, data.BulkJobVar, data.RedisKeyVar)
}

func renderPreviewApplyItemIDsBruno(data scaffoldData) string {
	return fmt.Sprintf(`meta {
  name: preview - batch - item_ids - apply_changes
  type: http
  seq: 5
}

post {
  url: {{urlBase}}api/v1/process-lifecycle/batch-preview
  body: json
  auth: bearer
}

auth:bearer {
  token: {{access_token}}
}

headers {
  X-Client-Code: bruno
}

body:json {
  {
    "process_type_id": {{%s}},
    "sede_id": 0,
    "override_process_version_id": 0,
    "roadmap": 0,
    "mode": "batch",
    "apply_changes": true,
    "limit": 10,
    "item_ids": [1],
    "input": {
      "id": {{%s}},
      "key_redis": "{{%s}}"
    }
  }
}
`, data.ProcessTypeVar, data.BulkJobVar, data.RedisKeyVar)
}

func renderPreviewRowNumbersBruno(data scaffoldData) string {
	return fmt.Sprintf(`meta {
  name: preview - batch - row_numbers
  type: http
  seq: 5
}

post {
  url: {{urlBase}}api/v1/process-lifecycle/batch-preview
  body: json
  auth: bearer
}

auth:bearer {
  token: {{access_token}}
}

headers {
  X-Client-Code: bruno
}

body:json {
  {
    "process_type_id": {{%s}},
    "sede_id": 0,
    "override_process_version_id": 0,
    "roadmap": 0,
    "mode": "batch",
    "limit": 10,
    "row_numbers": [1],
    "input": {
      "id": {{%s}},
      "key_redis": "{{%s}}"
    }
  }
}
`, data.ProcessTypeVar, data.BulkJobVar, data.RedisKeyVar)
}

func brunoBulkJobID(data scaffoldData) int64 {
	if data.BulkJobID > 0 {
		return data.BulkJobID
	}
	return 1
}

func ask(reader *bufio.Reader, label string) string {
	fmt.Printf("%s: ", label)
	text, _ := reader.ReadString('\n')
	return strings.TrimSpace(text)
}

func slugify(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	var b strings.Builder
	prevUnderscore := false
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			prevUnderscore = false
			continue
		}
		if !prevUnderscore {
			b.WriteRune(rune('_'))
			prevUnderscore = true
		}
	}
	return strings.Trim(b.String(), "_")
}

func normalizeIdentifier(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, "-", "_")
	s = regexp.MustCompile(`[^a-z0-9_]+`).ReplaceAllString(s, "_")
	s = regexp.MustCompile(`_+`).ReplaceAllString(s, "_")
	return strings.Trim(s, "_")
}

func toPascal(s string) string {
	parts := strings.Split(s, "_")
	for i, part := range parts {
		if part == "" {
			continue
		}
		parts[i] = strings.ToUpper(part[:1]) + strings.ToLower(part[1:])
	}
	return strings.Join(parts, "")
}

func sortedFileList(paths generatedPaths, withBruno bool) []string {
	_ = withBruno
	out := []string{
		paths.providerFile,
		paths.stepsFile,
		paths.dataProviderFile,
		paths.processorFile,
		paths.lifecycleFile,
		paths.seederFile,
	}
	return out
}

func fatal(err error) {
	fmt.Fprintf(os.Stderr, "error: %v\n", err)
	os.Exit(1)
}
