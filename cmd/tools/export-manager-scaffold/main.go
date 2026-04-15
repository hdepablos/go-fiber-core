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
	ProcessName   string
	ServiceSlug   string
	BatchSize     int
	PartPrefix    string
	RedisTTLHours int
	FileBase      string
	BulkJobID     int64
	WithDoc       bool
	WithBruno     bool
	Force         bool
}

type scaffoldData struct {
	ProcessName     string
	ServiceSlug     string
	PackageName     string
	PascalName      string
	SeedName        string
	SeederFuncName  string
	ExecutionBase   string
	StartKey        string
	ProcessBatchKey string
	FinalizeKey     string
	BatchSize       int
	PartPrefix      string
	RedisTTLHours   int
	FileBase        string
	BulkJobID       int64
	UseBulkJobMode  bool
	DocFileName     string
	BrunoFileName   string
}

func main() {
	opts := parseOptions()
	if err := enrichOptions(&opts); err != nil {
		fatal(err)
	}

	data := buildScaffoldData(opts)
	paths := scaffoldPaths(data)

	if err := ensurePaths(paths, opts.Force); err != nil {
		fatal(err)
	}

	if err := os.MkdirAll(paths.serviceDir, 0o755); err != nil {
		fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(paths.seederFile), 0o755); err != nil {
		fatal(err)
	}
	if opts.WithDoc {
		if err := os.MkdirAll(filepath.Dir(paths.docFile), 0o755); err != nil {
			fatal(err)
		}
	}
	if opts.WithBruno {
		if err := os.MkdirAll(filepath.Dir(paths.brunoFile), 0o755); err != nil {
			fatal(err)
		}
	}

	files := map[string]string{
		paths.providerFile:     renderProvider(data),
		paths.stepsFile:        renderSteps(data),
		paths.dataProviderFile: renderDataProvider(data),
		paths.layoutFile:       renderLayout(data),
		paths.lifecycleFile:    renderLifecycle(data),
		paths.seederFile:       renderSeeder(data),
	}
	if opts.WithDoc {
		files[paths.docFile] = renderDoc(data)
	}
	if opts.WithBruno {
		files[paths.brunoFile] = renderBruno(data)
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

	fmt.Println("Scaffold generado correctamente")
	fmt.Printf("Process Name: %s\n", data.ProcessName)
	fmt.Printf("Service Slug: %s\n", data.ServiceSlug)
	fmt.Println("Execution Keys:")
	fmt.Printf("- %s\n", data.StartKey)
	fmt.Printf("- %s\n", data.ProcessBatchKey)
	fmt.Printf("- %s\n", data.FinalizeKey)
	fmt.Println("Archivos creados:")
	for _, path := range sortedFileList(paths, opts.WithDoc, opts.WithBruno) {
		fmt.Printf("- %s\n", path)
	}
	fmt.Println("Siguientes pasos:")
	fmt.Println("1. Implementar DataProvider")
	fmt.Println("2. Implementar ParentLifecycle")
	fmt.Println("3. Implementar OutputRegistrar")
	fmt.Printf("4. Ejecutar el seeder: make seed-one name=%s\n", data.SeedName)
}

type generatedPaths struct {
	serviceDir       string
	providerFile     string
	stepsFile        string
	dataProviderFile string
	layoutFile       string
	lifecycleFile    string
	seederFile       string
	docFile          string
	brunoFile        string
}

func scaffoldPaths(data scaffoldData) generatedPaths {
	serviceDir := filepath.Join("/private/var/www/go-fiber-core/internal/services", data.ServiceSlug)
	return generatedPaths{
		serviceDir:       serviceDir,
		providerFile:     filepath.Join(serviceDir, "provider.go"),
		stepsFile:        filepath.Join(serviceDir, "steps.go"),
		dataProviderFile: filepath.Join(serviceDir, "data_provider.go"),
		layoutFile:       filepath.Join(serviceDir, "layout.go"),
		lifecycleFile:    filepath.Join(serviceDir, "lifecycle.go"),
		seederFile:       filepath.Join("/private/var/www/go-fiber-core/internal/database/seeders", data.ServiceSlug+"_seeder.go"),
		docFile:          filepath.Join("/private/var/www/go-fiber-core/doc/info", data.DocFileName),
		brunoFile:        filepath.Join("/private/var/www/go-fiber-core/bruno/process-lifecycle", data.BrunoFileName),
	}
}

func parseOptions() options {
	var opts options
	flag.StringVar(&opts.ProcessName, "process-name", "", "Nombre del proceso")
	flag.StringVar(&opts.ServiceSlug, "service-slug", "", "Slug tecnico del servicio")
	flag.IntVar(&opts.BatchSize, "batch-size", 5000, "Tamano del lote")
	flag.StringVar(&opts.PartPrefix, "part-prefix", "", "Prefijo temporal de partes en S3")
	flag.IntVar(&opts.RedisTTLHours, "redis-ttl-hours", 24, "TTL de Redis en horas")
	flag.StringVar(&opts.FileBase, "file", "", "Base de la ruta final del archivo")
	flag.Int64Var(&opts.BulkJobID, "bulk-job-id", 0, "Genera scaffold funcional basado en bulk_jobs")
	flag.BoolVar(&opts.WithDoc, "with-doc", true, "Genera documentacion base")
	flag.BoolVar(&opts.WithBruno, "with-bruno", true, "Genera request Bruno base")
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

	if opts.BatchSize <= 0 {
		opts.BatchSize = 5000
	}
	if opts.PartPrefix == "" {
		opts.PartPrefix = fmt.Sprintf("exports/bulk_jobs/%s", opts.ServiceSlug)
	}
	if opts.RedisTTLHours <= 0 {
		opts.RedisTTLHours = 24
	}

	opts.FileBase = strings.TrimSpace(opts.FileBase)
	if opts.FileBase == "" {
		opts.FileBase = ask(reader, "file")
	}
	if opts.FileBase == "" {
		return fmt.Errorf("file es requerido")
	}

	if !regexp.MustCompile(`^[a-z0-9_]+$`).MatchString(opts.ServiceSlug) {
		return fmt.Errorf("service_slug invalido: usa solo minusculas, numeros y underscore")
	}

	return nil
}

func buildScaffoldData(opts options) scaffoldData {
	pascal := toPascal(opts.ServiceSlug)
	seedName := "export_manager_" + opts.ServiceSlug
	execBase := fmt.Sprintf("bulk/export/%s", opts.ServiceSlug)
	return scaffoldData{
		ProcessName:     opts.ProcessName,
		ServiceSlug:     opts.ServiceSlug,
		PackageName:     opts.ServiceSlug,
		PascalName:      pascal,
		SeedName:        seedName,
		SeederFuncName:  "ExportManager" + pascal + "Seeder",
		ExecutionBase:   execBase,
		StartKey:        execBase + "/start",
		ProcessBatchKey: execBase + "/process_batch",
		FinalizeKey:     execBase + "/finalize",
		BatchSize:       opts.BatchSize,
		PartPrefix:      opts.PartPrefix,
		RedisTTLHours:   opts.RedisTTLHours,
		FileBase:        opts.FileBase,
		BulkJobID:       opts.BulkJobID,
		UseBulkJobMode:  opts.BulkJobID > 0,
		DocFileName:     "exportmanager_" + opts.ServiceSlug + ".md",
		BrunoFileName:   "RunProc -> " + opts.ServiceSlug + ".bru",
	}
}

func ensurePaths(paths generatedPaths, force bool) error {
	all := []string{
		paths.providerFile,
		paths.stepsFile,
		paths.dataProviderFile,
		paths.layoutFile,
		paths.lifecycleFile,
		paths.seederFile,
	}
	all = append(all, paths.docFile, paths.brunoFile)
	if force {
		return nil
	}
	for _, path := range all {
		if path == "" {
			continue
		}
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

	listEntry := fmt.Sprintf("\t\t\"%s\",", data.SeedName)
	if !strings.Contains(fileStr, listEntry) {
		anchor := "\t\t\"all_menus\","
		fileStr = strings.Replace(fileStr, anchor, listEntry+"\n"+anchor, 1)
	}

	funcBlock := fmt.Sprintf(`	service.AddSeeder("%s", func() error {
		return %s(pool)
	})

`, data.SeedName, data.SeederFuncName)
	if !strings.Contains(fileStr, data.SeedName) || !strings.Contains(fileStr, data.SeederFuncName) {
		anchor := `	service.AddSeeder("all_menus", func() error {`
		fileStr = strings.Replace(fileStr, anchor, funcBlock+anchor, 1)
	}

	return os.WriteFile(filePath, []byte(fileStr), 0o644)
}

func renderProvider(data scaffoldData) string {
	return fmt.Sprintf(`package %s

import (
	"context"
	"fmt"
	"os"
	"sync"

	gormconn "go-fiber-core/internal/database/connections/gorm"
	redisconn "go-fiber-core/internal/database/connections/redis"
	"go-fiber-core/internal/dtos/config"
	"go-fiber-core/internal/dtos/connect"
	"go-fiber-core/internal/services/exportmanager"
	"go-fiber-core/internal/services/queue"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/redis/go-redis/v9"
)

type Provider interface {
	Manager() exportmanager.Manager
	Connect() *connect.ConnectDTO
}

type provider struct {
	manager exportmanager.Manager
	conn    *connect.ConnectDTO
}

func (p *provider) Manager() exportmanager.Manager {
	return p.manager
}

func (p *provider) Connect() *connect.ConnectDTO {
	return p.conn
}

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
	lifecycle := NewParentLifecycle(conn.ConnectGormRead, conn.ConnectGormWrite)
	dataProvider := NewDataProvider(conn.ConnectGormRead)
	headerBuilder := NewHeaderBuilder()
	bodyBuilder := NewBodyBuilder()
	footerBuilder := NewFooterBuilder()
	outputRegistrar := NewOutputRegistrar(conn.ConnectGormWrite)
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
		%q,
	)

	return &provider{manager: manager, conn: conn}, nil
}

var (
	defaultOnce sync.Once
	defaultProv Provider
	defaultErr  error
	manualProv  Provider
	manualMu    sync.RWMutex
)

func SetDefaultProvider(prov Provider) {
	manualMu.Lock()
	defer manualMu.Unlock()
	manualProv = prov
}

func DefaultProvider(ctx context.Context) (Provider, error) {
	manualMu.RLock()
	if manualProv != nil {
		prov := manualProv
		manualMu.RUnlock()
		return prov, nil
	}
	manualMu.RUnlock()

	defaultOnce.Do(func() {
		configPath := os.Getenv("CONFIG_PATH")
		if configPath == "" {
			configPath = "internal/appconfig/config.yml"
		}
		if _, err := os.Stat(configPath); err != nil {
			if _, err2 := os.Stat("config.yml"); err2 == nil {
				configPath = "config.yml"
			}
		}

		appCfg, err := config.NewAppConfig(configPath)
		if err != nil {
			defaultErr = err
			return
		}

		gormSvc, _, err := gormconn.NewGormConnectService(appCfg.MultiDatabaseConfig)
		if err != nil {
			defaultErr = err
			return
		}
		rdb, _, err := redisconn.NewRedisClient(appCfg.Redis)
		if err != nil {
			defaultErr = err
			return
		}
		awsSvc, err := queue.NewAWSService(ctx)
		if err != nil {
			defaultErr = err
			return
		}

		conn := &connect.ConnectDTO{
			ConnectGormWrite: gormSvc.GetWriteDB(),
			ConnectGormRead:  gormSvc.GetReadDB(),
			ConnectRedis:     rdb,
		}

		prov, err := NewProviderWithConfig(appCfg, conn, rdb, awsSvc.NewS3Client())
		if err != nil {
			defaultErr = err
			return
		}
		defaultProv = prov
	})

	return defaultProv, defaultErr
}
`, data.PackageName, data.PartPrefix)
}

func renderSteps(data scaffoldData) string {
	return fmt.Sprintf(`package %s

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go-fiber-core/internal/domain"
	"go-fiber-core/internal/services/exportmanager"
	"go-fiber-core/internal/services/serviceconfig"
	"go-fiber-core/internal/services/serviceconfig/contracts"
	"go-fiber-core/internal/utils"
)

type StartStep struct {
	ctx         *contracts.ServiceContext
	servicePath string
	batchSize   int
	ttlHours    int
	partPrefix  string
}

func NewStartStep() contracts.Service {
	return &StartStep{
		batchSize: %d,
		ttlHours:  %d,
	}
}

func (s *StartStep) Init(ctx *contracts.ServiceContext, servicePath string) {
	s.ctx = ctx
	s.servicePath = servicePath
	if s.ctx != nil && s.ctx.CurrentStepConfig != nil {
		if v, ok := s.ctx.CurrentStepConfig["batch_size"]; ok {
			s.batchSize = utils.ToInt(v)
		}
		if v, ok := s.ctx.CurrentStepConfig["redis_ttl_hours"]; ok {
			s.ttlHours = utils.ToInt(v)
		}
		if v, ok := s.ctx.CurrentStepConfig["part_prefix"]; ok {
			if str, ok := v.(string); ok {
				s.partPrefix = str
			}
		}
	}
}

func (s *StartStep) Execute() error {
	prov, err := DefaultProvider(s.ctx.Ctx)
	if err != nil {
		return err
	}

	input, err := buildStartInput(s.ctx)
	if err != nil {
		return err
	}

	res, err := prov.Manager().Start(s.ctx.Ctx, exportmanager.StartRequest{
		Input:      input,
		BatchSize:  s.batchSize,
		RedisTTL:   time.Duration(s.ttlHours) * time.Hour,
		S3Bucket:   fmt.Sprint(utils.GetInputValueOrDefault(s.ctx, "s3_bucket", "")),
		PartPrefix: s.partPrefix,
	})
	if err != nil {
		markFailure(prov, s.ctx.Ctx, input, err)
		return err
	}

	s.ctx.SetInputValue("id", input.ParentID)
	s.ctx.SetInputValue("key_redis", res.RedisKey)
	s.ctx.SetInputValue("batches_list_key", res.BatchesListKey)
	s.ctx.SetInputValue("parts_list_key", res.PartsListKey)
	s.ctx.SetInputValue("total_batches", res.TotalBatches)
	s.ctx.SetInputValue("batch_index", 0)
	s.ctx.SetInputValue("is_last_batch", false)
	s.ctx.SetInputValue("s3_bucket", res.S3Bucket)
	s.ctx.SetInputValue("part_prefix", res.PartPrefix)

	s.ctx.SetResult(s.servicePath, contracts.StepResult{
		Status:  "completed",
		Message: "export manager start completado",
		Data: map[string]any{
			"key_redis":     res.RedisKey,
			"id":            input.ParentID,
			"total_batches": res.TotalBatches,
		},
	})
	return nil
}

type ProcessBatchStep struct {
	ctx         *contracts.ServiceContext
	servicePath string
}

func NewProcessBatchStep() contracts.Service {
	return &ProcessBatchStep{}
}

func (s *ProcessBatchStep) Init(ctx *contracts.ServiceContext, servicePath string) {
	s.ctx = ctx
	s.servicePath = servicePath
}

func (s *ProcessBatchStep) Execute() error {
	prov, err := DefaultProvider(s.ctx.Ctx)
	if err != nil {
		return err
	}
	input, err := buildInput(s.ctx)
	if err != nil {
		return err
	}

	res, err := prov.Manager().ProcessBatch(s.ctx.Ctx, exportmanager.ProcessBatchRequest{
		Input:          input,
		BatchesListKey: fmt.Sprint(utils.MustGetInputValue(s.ctx, "batches_list_key")),
		PartsListKey:   fmt.Sprint(utils.MustGetInputValue(s.ctx, "parts_list_key")),
		S3Bucket:       fmt.Sprint(utils.GetInputValueOrDefault(s.ctx, "s3_bucket", "")),
		PartPrefix:     fmt.Sprint(utils.GetInputValueOrDefault(s.ctx, "part_prefix", "")),
		BatchIndex:     utils.ToInt(utils.GetInputValueOrDefault(s.ctx, "batch_index", 0)),
		TotalBatches:   utils.ToInt(utils.MustGetInputValue(s.ctx, "total_batches")),
	})
	if err != nil {
		markFailure(prov, s.ctx.Ctx, input, err)
		return err
	}

	s.ctx.SetResult(s.servicePath, contracts.StepResult{
		Status:  "completed",
		Message: "batch procesado",
		Data: map[string]any{
			"batch_index":     res.NextBatchIndex,
			"is_last_batch":   res.IsLastBatch,
			"processed_count": res.ProcessedCount,
			"s3_part_key":     res.S3PartKey,
		},
	})
	return nil
}

type FinalizeStep struct {
	ctx         *contracts.ServiceContext
	servicePath string
	fileBase    string
}

func NewFinalizeStep() contracts.Service {
	return &FinalizeStep{}
}

func (s *FinalizeStep) Init(ctx *contracts.ServiceContext, servicePath string) {
	s.ctx = ctx
	s.servicePath = servicePath
	if s.ctx != nil && s.ctx.CurrentStepConfig != nil {
		if v, ok := s.ctx.CurrentStepConfig["file"]; ok {
			if str, ok := v.(string); ok {
				s.fileBase = str
			}
		}
	}
}

func (s *FinalizeStep) Execute() error {
	prov, err := DefaultProvider(s.ctx.Ctx)
	if err != nil {
		return err
	}
	input, err := buildInput(s.ctx)
	if err != nil {
		return err
	}

	output, err := prov.Manager().Finalize(s.ctx.Ctx, exportmanager.FinalizeRequest{
		Input:        input,
		PartsListKey: fmt.Sprint(utils.MustGetInputValue(s.ctx, "parts_list_key")),
		S3Bucket:     fmt.Sprint(utils.GetInputValueOrDefault(s.ctx, "s3_bucket", "")),
		FileBase:     s.fileBase,
		TotalParts:   utils.ToInt(utils.GetInputValueOrDefault(s.ctx, "total_batches", 0)),
	})
	if err != nil {
		markFailure(prov, s.ctx.Ctx, input, err)
		return err
	}

	s.ctx.SetResult(s.servicePath, contracts.StepResult{
		Status:  "completed",
		Message: "archivo final generado",
		Data: map[string]any{
			"s3_final_key": output.Key,
			"s3_file_path": output.Path,
			"file_size":    output.FileSize,
			"parts":        output.Parts,
		},
	})
	return nil
}

func buildInput(ctx *contracts.ServiceContext) (exportmanager.Input, error) {
	input := exportmanager.Input{
		RedisKey: fmt.Sprint(utils.MustGetInputValue(ctx, "key_redis")),
		ParentID: utils.ToInt64(utils.MustGetInputValue(ctx, "id")),
	}
	if input.ParentID <= 0 {
		return exportmanager.Input{}, fmt.Errorf("id invalido")
	}
	if input.RedisKey == "" {
		return exportmanager.Input{}, fmt.Errorf("key_redis invalida")
	}
	if rawFilters, ok := ctx.GetInputValue("filters"); ok {
		if filters, ok := rawFilters.(map[string]any); ok {
			input.Filters = filters
		}
	}
	return input, nil
}

func buildStartInput(ctx *contracts.ServiceContext) (exportmanager.Input, error) {
	input := exportmanager.Input{
		RedisKey: fmt.Sprint(utils.GetInputValueOrDefault(ctx, "key_redis", "")),
		ParentID: utils.ToInt64(utils.MustGetInputValue(ctx, "id")),
	}
	if input.ParentID <= 0 {
		return exportmanager.Input{}, fmt.Errorf("id invalido")
	}
	if rawFilters, ok := ctx.GetInputValue("filters"); ok {
		if filters, ok := rawFilters.(map[string]any); ok {
			input.Filters = filters
		}
	}
	return input, nil
}

func markFailure(prov Provider, ctx context.Context, input exportmanager.Input, err error) {
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
`, data.PackageName, data.BatchSize, data.RedisTTLHours, data.StartKey, data.ProcessBatchKey, data.FinalizeKey)
}

func renderDataProvider(data scaffoldData) string {
	if data.UseBulkJobMode {
		return fmt.Sprintf(`package %s

import (
	"context"
	"encoding/json"
	"fmt"

	"go-fiber-core/internal/models"
	"go-fiber-core/internal/services/exportmanager"
	"go-fiber-core/internal/utils"

	"gorm.io/gorm"
)

type DataProvider struct {
	readDB *gorm.DB
}

func NewDataProvider(readDB *gorm.DB) *DataProvider {
	return &DataProvider{readDB: readDB}
}

func (p *DataProvider) LoadBatches(ctx context.Context, execCtx exportmanager.ExecutionContext, batchSize int) (exportmanager.LoadBatchesResult, error) {
	input := execCtx.Input
	if input.ParentID <= 0 {
		return exportmanager.LoadBatchesResult{}, fmt.Errorf("id invalido")
	}
	if batchSize <= 0 {
		batchSize = 5000
	}

	query := p.readDB.WithContext(ctx).
		Model(&models.BulkJobItem{}).
		Select("id", "row_number", "reference_key", "data", "created_at", "updated_at").
		Where("bulk_job_id = ?", input.ParentID).
		Order("id ASC")

	statusFilterApplied := false
	if input.Filters != nil {
		result, err := utils.ApplyBulkJobItemFilters(query, input.Filters)
		if err != nil {
			return exportmanager.LoadBatchesResult{}, err
		}
		query = result.Query
		statusFilterApplied = result.StatusFilterApplied
	}
	if !statusFilterApplied {
		query = query.Where("status_code = ?", models.BulkJobStatusImported)
	}

	var items []models.BulkJobItem
	if err := query.Find(&items).Error; err != nil {
		return exportmanager.LoadBatchesResult{}, err
	}

	batches := make([]exportmanager.Batch, 0, (len(items)/batchSize)+1)
	var current exportmanager.Batch
	current.Items = make([]json.RawMessage, 0, batchSize)

	totalAmount := 0.0
	for _, item := range items {
		payload, err := json.Marshal(map[string]any{
			"id":            item.ID,
			"row_number":    item.RowNumber,
			"reference_key": item.ReferenceKey,
			"data":          json.RawMessage(item.Data),
		})
		if err != nil {
			return exportmanager.LoadBatchesResult{}, err
		}
		current.Items = append(current.Items, payload)
		totalAmount += utils.ExtractAmount(item.Data)

		if len(current.Items) == batchSize {
			batches = append(batches, current)
			current = exportmanager.Batch{Items: make([]json.RawMessage, 0, batchSize)}
		}
	}
	if len(current.Items) > 0 {
		batches = append(batches, current)
	}

	if execCtx.Runtime != nil {
		_ = execCtx.Runtime.Set(ctx, "total_records", len(items))
		_ = execCtx.Runtime.Set(ctx, "total_amount", totalAmount)
	}

	return exportmanager.LoadBatchesResult{
		Batches: batches,
		Summary: exportmanager.Summary{
			TotalRecords: int64(len(items)),
			TotalAmount:  totalAmount,
			Defaults: map[string]any{
				"total_records": len(items),
				"total_amount":  totalAmount,
			},
			Metadata: map[string]any{
				"source":      "bulk_job_items",
				"bulk_job_id": input.ParentID,
			},
		},
	}, nil
}
`, data.PackageName)
	}
	return fmt.Sprintf(`package %s

import (
	"context"
	"fmt"

	"go-fiber-core/internal/services/exportmanager"

	"gorm.io/gorm"
)

type DataProvider struct {
	readDB *gorm.DB
}

func NewDataProvider(readDB *gorm.DB) *DataProvider {
	return &DataProvider{readDB: readDB}
}

func (p *DataProvider) LoadBatches(ctx context.Context, execCtx exportmanager.ExecutionContext, batchSize int) (exportmanager.LoadBatchesResult, error) {
	_ = ctx
	_ = execCtx
	_ = batchSize
	_ = p.readDB

	// TODO: Implementar la carga de lotes reales para este proceso.
	// El developer recibe:
	// - execCtx.Input.ParentID: id de la tabla padre
	// - execCtx.Input.RedisKey: key unica de la corrida
	// - execCtx.Input.Filters: filtros opcionales
	//
	// Ejemplo de uso de runtime compartido:
	// _ = execCtx.Runtime.Set(ctx, "total_amount", 123.45)
	//
	// Retorna batches + summary.
	return exportmanager.LoadBatchesResult{}, fmt.Errorf("TODO: implement LoadBatches for process %%q", %q)
}
`, data.PackageName, data.ProcessName)
}

func renderLayout(data scaffoldData) string {
	return fmt.Sprintf(`package %s

import (
	"context"
	"encoding/json"

	"go-fiber-core/internal/services/exportmanager"
	"go-fiber-core/internal/utils"
)

type HeaderBuilder struct{}

func NewHeaderBuilder() *HeaderBuilder {
	return &HeaderBuilder{}
}

func (b *HeaderBuilder) BuildHeader(ctx context.Context, execCtx exportmanager.ExecutionContext) ([]string, error) {
	_ = ctx
	_ = execCtx

	// TODO: personalizar el header del archivo.
	// Aqui puedes usar execCtx.Input.ParentID y execCtx.Input.RedisKey.
	line, err := utils.BuildCSVLine([]string{"header"}, 0)
	if err != nil {
		return nil, err
	}
	return []string{line}, nil
}

type BodyBuilder struct{}

func NewBodyBuilder() *BodyBuilder {
	return &BodyBuilder{}
}

func (b *BodyBuilder) BuildBodyLines(ctx context.Context, execCtx exportmanager.ExecutionContext, item json.RawMessage) ([]string, error) {
	_ = ctx
	_ = execCtx

	// TODO: transformar cada item en una o varias lineas del archivo.
	line, err := utils.BuildCSVLine([]string{string(item)}, 0)
	if err != nil {
		return nil, err
	}
	return []string{line}, nil
}

type FooterBuilder struct{}

func NewFooterBuilder() *FooterBuilder {
	return &FooterBuilder{}
}

func (b *FooterBuilder) BuildFooter(ctx context.Context, execCtx exportmanager.ExecutionContext) ([]string, error) {
	_ = ctx
	_ = execCtx

	// TODO: personalizar el footer.
	// Si no deseas footer, reemplaza esto por: return []string{}, nil
	line, err := utils.BuildCSVLine([]string{"footer"}, 0)
	if err != nil {
		return nil, err
	}
	return []string{line}, nil
}
`, data.PackageName)
}

func renderLifecycle(data scaffoldData) string {
	if data.UseBulkJobMode {
		return fmt.Sprintf(`package %s

import (
	"context"
	"encoding/json"
	"fmt"

	"go-fiber-core/internal/domain"
	"go-fiber-core/internal/models"
	"go-fiber-core/internal/services/exportmanager"

	"gorm.io/gorm"
)

type ParentLifecycle struct {
	readDB  *gorm.DB
	writeDB *gorm.DB
}

func NewParentLifecycle(readDB, writeDB *gorm.DB) *ParentLifecycle {
	return &ParentLifecycle{readDB: readDB, writeDB: writeDB}
}

func (l *ParentLifecycle) Start(ctx context.Context, execCtx exportmanager.ExecutionContext) error {
	input := execCtx.Input
	var job models.BulkJob
	if err := l.readDB.WithContext(ctx).
		Select("id", "status_code").
		Where("id = ?", input.ParentID).
		First(&job).Error; err != nil {
		return err
	}
	if job.StatusCode != models.BulkJobStatusImported {
		return fmt.Errorf("%%w: Verifique el proceso con el id %%d ya fue procesado actualmente con el status %%s de la tabla bulk_jobs", domain.ErrBusinessRuleViolation, input.ParentID, job.StatusCode)
	}
	return l.writeDB.WithContext(ctx).
		Model(&models.BulkJob{}).
		Where("id = ?", input.ParentID).
		Update("status_code", models.BulkJobStatusProcessing).Error
}

func (l *ParentLifecycle) End(ctx context.Context, execCtx exportmanager.ExecutionContext, _ exportmanager.OutputResult) error {
	return l.writeDB.WithContext(ctx).
		Model(&models.BulkJob{}).
		Where("id = ?", execCtx.Input.ParentID).
		Update("status_code", models.BulkJobStatusProcessed).Error
}

func (l *ParentLifecycle) Fail(ctx context.Context, execCtx exportmanager.ExecutionContext, _ error) error {
	return l.writeDB.WithContext(ctx).
		Model(&models.BulkJob{}).
		Where("id = ?", execCtx.Input.ParentID).
		Update("status_code", models.BulkJobStatusErrorProcess).Error
}

type OutputRegistrar struct {
	writeDB *gorm.DB
}

func NewOutputRegistrar(writeDB *gorm.DB) *OutputRegistrar {
	return &OutputRegistrar{writeDB: writeDB}
}

func (r *OutputRegistrar) Register(ctx context.Context, execCtx exportmanager.ExecutionContext, output exportmanager.OutputResult) error {
	metadata, err := json.Marshal(map[string]any{
		"bucket":        output.Bucket,
		"key":           output.Key,
		"content_type":  output.ContentType,
		"parts":         output.Parts,
		"total_records": execCtx.Summary.TotalRecords,
		"total_amount":  execCtx.Summary.TotalAmount,
		"redis_key":     execCtx.Input.RedisKey,
	})
	if err != nil {
		return err
	}

	fileSize := output.FileSize
	record := &models.BulkJobOutput{
		BulkJobID: execCtx.Input.ParentID,
		Type:      "csv",
		FilePath:  output.Path,
		FileSize:  &fileSize,
		Status:    models.BulkJobOutputStatusGenerated,
		Metadata:  metadata,
	}
	return r.writeDB.WithContext(ctx).Create(record).Error
}
`, data.PackageName)
	}
	return fmt.Sprintf(`package %s

import (
	"context"

	"go-fiber-core/internal/services/exportmanager"

	"gorm.io/gorm"
)

type ParentLifecycle struct {
	readDB  *gorm.DB
	writeDB *gorm.DB
}

func NewParentLifecycle(readDB, writeDB *gorm.DB) *ParentLifecycle {
	return &ParentLifecycle{readDB: readDB, writeDB: writeDB}
}

func (l *ParentLifecycle) Start(ctx context.Context, execCtx exportmanager.ExecutionContext) error {
	_ = ctx
	_ = execCtx
	_ = l.readDB
	_ = l.writeDB

	// TODO: cambiar el status del padre a PROCESSING.
	return nil
}

func (l *ParentLifecycle) End(ctx context.Context, execCtx exportmanager.ExecutionContext, output exportmanager.OutputResult) error {
	_ = ctx
	_ = execCtx
	_ = output

	// TODO: cambiar el status del padre a PROCESSED.
	return nil
}

func (l *ParentLifecycle) Fail(ctx context.Context, execCtx exportmanager.ExecutionContext, cause error) error {
	_ = ctx
	_ = execCtx
	_ = cause

	// TODO: cambiar el status del padre a ERROR_PROCESS.
	return nil
}

type OutputRegistrar struct {
	writeDB *gorm.DB
}

func NewOutputRegistrar(writeDB *gorm.DB) *OutputRegistrar {
	return &OutputRegistrar{writeDB: writeDB}
}

func (r *OutputRegistrar) Register(ctx context.Context, execCtx exportmanager.ExecutionContext, output exportmanager.OutputResult) error {
	_ = ctx
	_ = execCtx
	_ = output
	_ = r.writeDB

	// TODO: persistir el output final en la tabla o repositorio que corresponda.
	return nil
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
		const processDescription = "Scaffold de exportmanager generado automaticamente"

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
				Name:         "Step 1: Start + organize batches con ExportManager",
				ExecutionKey: %q,
				Config:       %s{"batch_size":%d,"part_prefix":%q,"redis_ttl_hours":%d}%s,
			},
			{
				Order:        2,
				Name:         "Step 2: Procesar batch con header/body/footer",
				ExecutionKey: %q,
				Config: %s{
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
				Name:         "Step 3: Merge final y cierre del proceso",
				ExecutionKey: %q,
				Config:       %s{"file":%q}%s,
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
`, data.SeederFuncName, data.SeedName, data.ProcessName, bt, bt, bt, bt, bt, bt, data.StartKey, bt, data.BatchSize, data.PartPrefix, data.RedisTTLHours, bt, data.ProcessBatchKey, bt, data.ProcessName, data.FinalizeKey, bt, data.FinalizeKey, bt, data.FileBase, bt, bt, bt)
}

func renderDoc(data scaffoldData) string {
	mode := "generico"
	nextSteps := "1. Implementar `DataProvider`\n2. Implementar `ParentLifecycle`\n3. Implementar `OutputRegistrar`\n4. Personalizar `HeaderBuilder`, `BodyBuilder` y `FooterBuilder`\n"
	if data.UseBulkJobMode {
		mode = fmt.Sprintf("bulk_jobs funcional (bulk_job_id=%d)", data.BulkJobID)
		nextSteps = "1. Ajustar DataProvider si necesitas filtros adicionales\n2. Personalizar `HeaderBuilder`, `BodyBuilder` y `FooterBuilder`\n3. Ejecutar el seeder y probar con Bruno\n"
	}
	return fmt.Sprintf("# %s\n\n## Objetivo\n\nScaffold base generado automaticamente para montar un flujo sobre `exportmanager`.\n\n## Modo\n\n- `%s`\n\n## Execution Keys\n\n- `%s`\n- `%s`\n- `%s`\n\n## Config base\n\n- batch_size: %d\n- part_prefix: `%s`\n- redis_ttl_hours: %d\n- file: `%s`\n\n## Variables que recibe el developer\n\nTodos los servicios funcionales reciben:\n\n- `id`: id del padre de negocio\n- `key_redis`: key unica de la corrida\n\n## Runtime en Redis\n\nEl runtime permite compartir datos entre data/header/body/footer/end.\n\nEjemplo:\n\n- guardar `total_amount` en header\n- leer `total_amount` en footer\n\n## Siguientes pasos\n\n%s", data.ProcessName, mode, data.StartKey, data.ProcessBatchKey, data.FinalizeKey, data.BatchSize, data.PartPrefix, data.RedisTTLHours, data.FileBase, nextSteps)
}

func renderBruno(data scaffoldData) string {
	idValue := 1
	if data.UseBulkJobMode && data.BulkJobID > 0 {
		idValue = int(data.BulkJobID)
	}
	return fmt.Sprintf(`meta {
  name: RunProc -> %s
  type: http
}

vars {
  process_type_id_%s: 0
}

post {
  url: {{urlBase}}api/v1/process-lifecycle/run
  body: json
  auth: bearer
}

auth:bearer {
  token: {{access_token}}
}

body:json {
  {
    "process_type_id": {{process_type_id_%s}},
    "sede_id": 0,
    "override_process_version_id": 0,
    "roadmap": 0,
    "input": {
      "id": %d
    }
  }
}
`, data.ServiceSlug, data.ServiceSlug, data.ServiceSlug, idValue)
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
			b.WriteRune('_')
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

func sortedFileList(paths generatedPaths, withDoc, withBruno bool) []string {
	out := []string{
		paths.providerFile,
		paths.stepsFile,
		paths.dataProviderFile,
		paths.layoutFile,
		paths.lifecycleFile,
		paths.seederFile,
	}
	if withDoc {
		out = append(out, paths.docFile)
	}
	if withBruno {
		out = append(out, paths.brunoFile)
	}
	return out
}

func fatal(err error) {
	fmt.Fprintf(os.Stderr, "error: %v\n", err)
	os.Exit(1)
}
