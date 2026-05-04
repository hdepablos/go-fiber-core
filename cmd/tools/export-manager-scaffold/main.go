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
		paths.providerFile:                 renderProvider(data),
		paths.runtimeProviderContextFile:   renderRuntimeProviderContext(data),
		paths.dataProviderFile:             renderDataProvider(data),
		paths.layoutHeaderFile:             renderLayoutHeader(data),
		paths.layoutBodyFile:               renderLayoutBody(data),
		paths.layoutFooterFile:             renderLayoutFooter(data),
		paths.layoutHelpersFile:            renderLayoutHelpers(data),
		paths.lifecycleParentFile:          renderLifecycleParent(data),
		paths.lifecycleOutputRegistrarFile: renderLifecycleOutputRegistrar(data),
		paths.stepsStartFile:               renderStepStart(data),
		paths.stepsProcessBatchFile:        renderStepProcessBatch(data),
		paths.stepsFinalizeFile:            renderStepFinalize(data),
		paths.stepsInputFile:               renderStepInput(data),
		paths.stepsFailureFile:             renderStepFailure(data),
		paths.seederFile:                   renderSeeder(data),
	}
	if opts.WithDoc {
		files[paths.docFile] = renderDoc(data)
	}
	if opts.WithBruno {
		files[paths.brunoFile] = renderBruno(data)
	}

	for path := range files {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			fatal(err)
		}
	}
	for path, content := range files {
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			fatal(fmt.Errorf("escribiendo %s: %w", path, err))
		}
	}

	if err := patchImportBlock("/private/var/www/go-fiber-core/cmd/api/main.go", fmt.Sprintf(`_ "go-fiber-core/internal/services/exports/%s"`, data.ServiceSlug)); err != nil {
		fatal(err)
	}
	if err := patchImportBlock("/private/var/www/go-fiber-core/cmd/sqs-consumer/main.go", fmt.Sprintf(`_ "go-fiber-core/internal/services/exports/%s"`, data.ServiceSlug)); err != nil {
		fatal(err)
	}
	if err := patchSeedService(data); err != nil {
		fatal(err)
	}
	if err := patchRuntimeBootstrap(data); err != nil {
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
	fmt.Println("2. Ajustar ParentLifecycle, incluyendo Fail para auto-cancel")
	fmt.Println("3. Implementar OutputRegistrar")
	fmt.Println("4. Ajustar el layout default de header/body/footer si el archivo requiere otro formato")
	fmt.Println("5. Probar preview y run, sabiendo que ambos reutilizan BodyBuilder.renderItem(...)")
	fmt.Println("6. Verificar execution keys y registro automatico del manager")
	fmt.Printf("7. Ejecutar el seeder: make seed-one name=%s\n", data.SeedName)
}

type generatedPaths struct {
	serviceDir                   string
	providerFile                 string
	runtimeProviderContextFile   string
	dataProviderFile             string
	layoutHeaderFile             string
	layoutBodyFile               string
	layoutFooterFile             string
	layoutHelpersFile            string
	lifecycleParentFile          string
	lifecycleOutputRegistrarFile string
	stepsStartFile               string
	stepsProcessBatchFile        string
	stepsFinalizeFile            string
	stepsInputFile               string
	stepsFailureFile             string
	seederFile                   string
	docFile                      string
	brunoFile                    string
}

func scaffoldPaths(data scaffoldData) generatedPaths {
	serviceDir := filepath.Join("/private/var/www/go-fiber-core/internal/services/exports", data.ServiceSlug)
	return generatedPaths{
		serviceDir:                   serviceDir,
		providerFile:                 filepath.Join(serviceDir, "provider.go"),
		runtimeProviderContextFile:   filepath.Join(serviceDir, "runtime", "provider_context.go"),
		dataProviderFile:             filepath.Join(serviceDir, "data", "provider.go"),
		layoutHeaderFile:             filepath.Join(serviceDir, "layout", "header_builder.go"),
		layoutBodyFile:               filepath.Join(serviceDir, "layout", "body_builder.go"),
		layoutFooterFile:             filepath.Join(serviceDir, "layout", "footer_builder.go"),
		layoutHelpersFile:            filepath.Join(serviceDir, "layout", "layout_helpers.go"),
		lifecycleParentFile:          filepath.Join(serviceDir, "lifecycle", "parent.go"),
		lifecycleOutputRegistrarFile: filepath.Join(serviceDir, "lifecycle", "output_registrar.go"),
		stepsStartFile:               filepath.Join(serviceDir, "steps", "start.go"),
		stepsProcessBatchFile:        filepath.Join(serviceDir, "steps", "process_batch.go"),
		stepsFinalizeFile:            filepath.Join(serviceDir, "steps", "finalize.go"),
		stepsInputFile:               filepath.Join(serviceDir, "steps", "input.go"),
		stepsFailureFile:             filepath.Join(serviceDir, "steps", "failure.go"),
		seederFile:                   filepath.Join("/private/var/www/go-fiber-core/internal/database/seeders", data.ServiceSlug+"_seeder.go"),
		docFile:                      filepath.Join("/private/var/www/go-fiber-core/doc/info", data.DocFileName),
		brunoFile:                    filepath.Join("/private/var/www/go-fiber-core/bruno/process-lifecycle", data.BrunoFileName),
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
		paths.runtimeProviderContextFile,
		paths.dataProviderFile,
		paths.layoutHeaderFile,
		paths.layoutBodyFile,
		paths.layoutFooterFile,
		paths.layoutHelpersFile,
		paths.lifecycleParentFile,
		paths.lifecycleOutputRegistrarFile,
		paths.stepsStartFile,
		paths.stepsProcessBatchFile,
		paths.stepsFinalizeFile,
		paths.stepsInputFile,
		paths.stepsFailureFile,
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

	lines := strings.Split(fileStr, "\n")
	importStart := -1
	for i, line := range lines {
		if strings.TrimSpace(line) == "import (" {
			importStart = i
			break
		}
	}
	if importStart == -1 {
		return fmt.Errorf("no se encontro bloque import en %s", filePath)
	}

	insertAt := -1
	for i := importStart + 1; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		switch {
		case trimmed == ")":
			insertAt = i
			i = len(lines)
		case strings.Contains(trimmed, "\"") && strings.HasSuffix(trimmed, ")"):
			lines[i] = strings.TrimSuffix(lines[i], ")")
			insertAt = i + 1
			lines = append(lines[:insertAt], append([]string{")"}, lines[insertAt:]...)...)
			i = len(lines)
		case strings.HasPrefix(trimmed, "var ") || trimmed == "var (" || strings.HasPrefix(trimmed, "func "):
			insertAt = i
			lines = append(lines[:insertAt], append([]string{")"}, lines[insertAt:]...)...)
			i = len(lines)
		}
	}
	if insertAt == -1 {
		return fmt.Errorf("no se encontro cierre de imports en %s", filePath)
	}

	lines = append(lines[:insertAt], append([]string{"\t" + importLine}, lines[insertAt:]...)...)
	newContent := strings.Join(lines, "\n")
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

func patchRuntimeBootstrap(data scaffoldData) error {
	filePath := "/private/var/www/go-fiber-core/internal/runtimebootstrap/bootstrap.go"
	content, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}
	fileStr := string(content)

	importLine := fmt.Sprintf("%q", "go-fiber-core/internal/services/exports/"+data.ServiceSlug)
	if !strings.Contains(fileStr, importLine) {
		if patchErr := patchImportBlock(filePath, importLine); patchErr != nil {
			return patchErr
		}
		content, err = os.ReadFile(filePath)
		if err != nil {
			return err
		}
		fileStr = string(content)
	}

	fieldLine := fmt.Sprintf("\t%s %s.Provider", data.PascalName, data.PackageName)
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
	// scaffold:export-runtime-start:%s
	if awsSvc, err := queue.NewAWSService(ctx); err != nil {
		errs = append(errs, fmt.Sprintf(%q, err))
	} else {
		s3Client := awsSvc.NewS3Client()
		if prov, err := %s.NewProviderWithConfig(appCfg, conn, conn.ConnectRedis, s3Client); err == nil {
			deps.%s = prov
		} else {
			errs = append(errs, fmt.Sprintf(%q, err))
		}
	}
	// scaffold:export-runtime-end:%s
`, data.ServiceSlug, data.ServiceSlug+" aws: %v", data.PackageName, data.PascalName, data.ServiceSlug+": %v", data.ServiceSlug)
	if !strings.Contains(fileStr, fmt.Sprintf("deps.%s = prov", data.PascalName)) {
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
`, data.PascalName, data.PackageName, data.PascalName)
	if !strings.Contains(fileStr, fmt.Sprintf("d.%s != nil", data.PascalName)) {
		anchor := "\n\treturn ctx\n}"
		insertPos := strings.LastIndex(fileStr, anchor)
		if insertPos == -1 {
			return fmt.Errorf("no se encontro anchor de inject en runtimebootstrap")
		}
		fileStr = fileStr[:insertPos] + injectBlock + fileStr[insertPos:]
	}

	return os.WriteFile(filePath, []byte(fileStr), 0o644)
}

func renderProvider(data scaffoldData) string {
	return fmt.Sprintf(`package %s

import (
	"context"
	"fmt"

	"go-fiber-core/internal/dtos/config"
	"go-fiber-core/internal/dtos/connect"
	serviceData "go-fiber-core/internal/services/exports/%s/data"
	serviceLayout "go-fiber-core/internal/services/exports/%s/layout"
	serviceLifecycle "go-fiber-core/internal/services/exports/%s/lifecycle"
	serviceRuntime "go-fiber-core/internal/services/exports/%s/runtime"
	serviceSteps "go-fiber-core/internal/services/exports/%s/steps"
	"go-fiber-core/internal/services/exportmanager"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/redis/go-redis/v9"
)

type Provider = serviceRuntime.Provider

// provider arma el entrypoint del export y expone manager, conexiones y preview.
type provider struct {
	manager    exportmanager.Manager
	conn       *connect.ConnectDTO
	components exportmanager.PreviewComponents
}

// Manager devuelve el coordinador principal del flujo de exportacion.
func (p *provider) Manager() exportmanager.Manager {
	return p.manager
}

// Connect expone las conexiones por si otro componente del proceso las necesita.
func (p *provider) Connect() *connect.ConnectDTO {
	return p.conn
}

// PreviewComponents registra las piezas necesarias para preview y debugging del export.
func (p *provider) PreviewComponents() exportmanager.PreviewComponents {
	return p.components
}

// NewProviderWithConfig construye todo el grafo del export: lifecycle, data source, layout y registro final.
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
	lifecycle := serviceLifecycle.NewParentLifecycle(conn.ConnectGormRead, conn.ConnectGormWrite)
	dataProvider := serviceData.NewDataProvider(conn.ConnectGormRead)
	headerBuilder := serviceLayout.NewHeaderBuilder()
	bodyBuilder := serviceLayout.NewBodyBuilder()
	footerBuilder := serviceLayout.NewFooterBuilder()
	outputRegistrar := serviceLifecycle.NewOutputRegistrar(conn.ConnectGormWrite)
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

	return &provider{
		manager: manager,
		conn:    conn,
		components: exportmanager.PreviewComponents{
			DataProvider:  dataProvider,
			HeaderBuilder: headerBuilder,
			BodyBuilder:   bodyBuilder,
			FooterBuilder: footerBuilder,
			StateStore:    stateStore,
		},
	}, nil
}

// WithProvider inyecta el provider en el contexto para que lo consuman los steps.
func WithProvider(ctx context.Context, prov Provider) context.Context {
	return serviceRuntime.WithProvider(ctx, prov)
}

// ProviderFromContext recupera el provider del export desde el contexto de ejecucion.
func ProviderFromContext(ctx context.Context) (Provider, error) {
	return serviceRuntime.ProviderFromContext(ctx)
}

const processTypeName = %q

// init registra el export en el runtime para run, preview y ejecucion administrada.
func init() {
	serviceSteps.Register()
	exportmanager.RegisterPreviewProvider(processTypeName, func(ctx context.Context) (exportmanager.PreviewProvider, error) {
		prov, err := ProviderFromContext(ctx)
		if err != nil {
			return nil, err
		}
		previewable, ok := prov.(exportmanager.PreviewProvider)
		if !ok {
			return nil, fmt.Errorf("provider de %%s no soporta preview", processTypeName)
		}
		return previewable, nil
	},
		%q,
		%q,
		%q,
	)
	exportmanager.RegisterManagedExportManager(func(ctx context.Context) (exportmanager.Manager, error) {
		prov, err := ProviderFromContext(ctx)
		if err != nil {
			return nil, err
		}
		return prov.Manager(), nil
	},
		%q,
		%q,
		%q,
	)
}
`, data.PackageName, data.ServiceSlug, data.ServiceSlug, data.ServiceSlug, data.ServiceSlug, data.ServiceSlug, data.PartPrefix, data.ProcessName, data.StartKey, data.ProcessBatchKey, data.FinalizeKey, data.StartKey, data.ProcessBatchKey, data.FinalizeKey)
}

func renderRuntimeProviderContext(data scaffoldData) string {
	return fmt.Sprintf(`package runtime

import (
	"context"
	"fmt"

	"go-fiber-core/internal/dtos/connect"
	"go-fiber-core/internal/services/exportmanager"
	"go-fiber-core/internal/services/runtimectx"
)

type Provider interface {
	Manager() exportmanager.Manager
	Connect() *connect.ConnectDTO
}

const providerContextKey = %q

// WithProvider adjunta el provider del export al contexto actual.
func WithProvider(ctx context.Context, prov Provider) context.Context {
	return runtimectx.WithNamedValue(ctx, providerContextKey, prov)
}

// ProviderFromContext recupera el provider previamente inyectado para este export.
func ProviderFromContext(ctx context.Context) (Provider, error) {
	if prov, ok := runtimectx.NamedValue[Provider](ctx, providerContextKey); ok && prov != nil {
		return prov, nil
	}
	return nil, fmt.Errorf("provider no disponible en contexto")
}
`, data.PackageName+".provider")
}

func renderLayoutHeader(data scaffoldData) string {
	_ = data
	return `package layout

import (
	"context"

	"go-fiber-core/internal/services/exportmanager"
)

type headerBuilder struct{}

// NewHeaderBuilder crea la pieza que construye el encabezado del archivo final.
func NewHeaderBuilder() exportmanager.HeaderBuilder {
	return &headerBuilder{}
}

// BuildHeader arma las primeras lineas del archivo usando el contexto global del export.
func (b *headerBuilder) BuildHeader(ctx context.Context, execCtx exportmanager.ExecutionContext) ([]string, error) {
	_ = execCtx

	line, err := buildHeaderLine(ctx)
	if err != nil {
		return nil, err
	}
	return []string{line}, nil
}
`
}

func renderLayoutBody(data scaffoldData) string {
	_ = data
	return `package layout

import (
	"context"
	"encoding/json"

	"go-fiber-core/internal/services/exportmanager"
)

type bodyBuilder struct{}

// NewBodyBuilder crea la pieza que transforma cada item en lineas del cuerpo del archivo.
func NewBodyBuilder() exportmanager.BodyBuilder {
	return &bodyBuilder{}
}

// BuildBodyLines conserva el contrato del framework y delega la logica real del item en renderItem.
func (b *bodyBuilder) BuildBodyLines(ctx context.Context, execCtx exportmanager.ExecutionContext, item json.RawMessage) ([]string, error) {
	return b.renderItem(ctx, execCtx, item)
}

// renderItem es el punto de extension estandar del export.
// Aqui el developer transforma un registro del batch en una o varias lineas del archivo.
func (b *bodyBuilder) renderItem(ctx context.Context, execCtx exportmanager.ExecutionContext, item json.RawMessage) ([]string, error) {
	_ = execCtx

	line, err := buildBodyLine(ctx, item)
	if err != nil {
		return nil, err
	}
	return []string{line}, nil
}
`
}

func renderLayoutFooter(data scaffoldData) string {
	_ = data
	return `package layout

import (
	"context"

	"go-fiber-core/internal/services/exportmanager"
)

type footerBuilder struct{}

// NewFooterBuilder crea la pieza que construye el cierre del archivo.
func NewFooterBuilder() exportmanager.FooterBuilder {
	return &footerBuilder{}
}

// BuildFooter arma las lineas finales del archivo usando el Summary y el runtime del export.
func (b *footerBuilder) BuildFooter(ctx context.Context, execCtx exportmanager.ExecutionContext) ([]string, error) {
	_ = execCtx

	line, err := buildFooterLine(ctx)
	if err != nil {
		return nil, err
	}
	return []string{line}, nil
}
`
}

func renderLayoutHelpers(data scaffoldData) string {
	_ = data
	bt := "`"
	return fmt.Sprintf(`package layout

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"go-fiber-core/internal/models"
	"go-fiber-core/internal/utils"
)

var exportColumns = [5]string{
	"amount",
	"descount1",
	"descount2",
	"sweep_days",
	"collection_file_id",
}

var csvHeader = append(
	[]string{
		"id",
		"bulk_job_id",
		"row_number",
		"reference_key",
		"status_code",
		"last_detail_message",
		"created_at",
		"updated_at",
	},
	append(exportColumns[:], "new_importe")...,
)

type exportData struct {
	fields     [5]string
	newImporte string
}

type bulkItemPayload struct {
	ID                int64                %sjson:"ID"%s
	BulkJobID         int64                %sjson:"BulkJobID"%s
	RowNumber         int                  %sjson:"RowNumber"%s
	ReferenceKey      string               %sjson:"ReferenceKey"%s
	StatusCode        models.BulkJobStatus %sjson:"StatusCode"%s
	LastDetailMessage *string              %sjson:"LastDetailMessage"%s
	CreatedAt         time.Time            %sjson:"CreatedAt"%s
	UpdatedAt         time.Time            %sjson:"UpdatedAt"%s
	Data              json.RawMessage      %sjson:"Data"%s
}

func buildHeaderLine(_ context.Context) (string, error) {
	return utils.BuildCSVLine(csvHeader, ';')
}

func buildBodyLine(_ context.Context, item json.RawMessage) (string, error) {
	var payload bulkItemPayload
	if err := json.Unmarshal(item, &payload); err != nil {
		return "", fmt.Errorf("unmarshal bulk item: %%w", err)
	}

	data, err := extractExportData(payload.Data)
	if err != nil {
		return "", fmt.Errorf("extract export data: %%w", err)
	}

	row, err := payload.toRow(data)
	if err != nil {
		return "", fmt.Errorf("build row: %%w", err)
	}

	line, err := utils.BuildCSVLine(row, ';')
	if err != nil {
		return "", fmt.Errorf("build body line: %%w", err)
	}
	return line, nil
}

func buildFooterLine(_ context.Context) (string, error) {
	return utils.BuildCSVLine([]string{"footer"}, ';')
}

func (p *bulkItemPayload) lastDetail() string {
	if p.LastDetailMessage != nil {
		return *p.LastDetailMessage
	}
	return ""
}

func (p *bulkItemPayload) toRow(data exportData) ([]string, error) {
	createdAt, err := utils.FormatDate(p.CreatedAt.Format(time.RFC3339), "YYYY-MM-DD HH:mm:ss")
	if err != nil {
		return nil, fmt.Errorf("format createdAt: %%w", err)
	}

	updatedAt, err := utils.FormatDate(p.UpdatedAt.Format(time.RFC3339), "DDMMYYYY")
	if err != nil {
		return nil, fmt.Errorf("format updatedAt: %%w", err)
	}

	return []string{
		fmt.Sprintf("%%d", p.ID),
		fmt.Sprintf("%%d", p.BulkJobID),
		fmt.Sprintf("%%d", p.RowNumber),
		p.ReferenceKey,
		string(p.StatusCode),
		p.lastDetail(),
		createdAt,
		updatedAt,
		data.fields[0],
		data.fields[1],
		data.fields[2],
		data.fields[3],
		data.fields[4],
		data.newImporte,
	}, nil
}

func extractExportData(raw json.RawMessage) (exportData, error) {
	fields, err := extractDataFields(raw)
	if err != nil {
		return exportData{}, err
	}

	newImporte, err := calcNewImporte(fields[0], fields[1], fields[2])
	if err != nil {
		return exportData{}, fmt.Errorf("calcular new_importe: %%w", err)
	}

	return exportData{
		fields:     fields,
		newImporte: newImporte,
	}, nil
}

func calcNewImporte(amount, descount1, descount2 string) (string, error) {
	amt, err := utils.ParseDecimal(amount)
	if err != nil {
		return "", fmt.Errorf("parsear amount %%q: %%w", amount, err)
	}

	result := amt - (utils.ParseDecimalOrZero(descount1) + utils.ParseDecimalOrZero(descount2))
	return strconv.FormatFloat(result, 'f', -1, 64), nil
}

func extractDataFields(raw json.RawMessage) ([5]string, error) {
	var fields [5]string

	if len(raw) == 0 || string(raw) == "null" {
		return fields, nil
	}

	if raw[0] == '"' {
		var encoded string
		if err := json.Unmarshal(raw, &encoded); err != nil {
			return fields, fmt.Errorf("unmarshal string data: %%w", err)
		}
		if encoded == "" {
			return fields, nil
		}
		if decoded, err := base64.StdEncoding.DecodeString(encoded); err == nil && len(decoded) > 0 {
			return extractDataFields(json.RawMessage(decoded))
		}
		return extractDataFields(json.RawMessage(encoded))
	}

	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()

	var data map[string]any
	if err := dec.Decode(&data); err != nil {
		return fields, fmt.Errorf("decode data object: %%w", err)
	}

	for i, key := range exportColumns {
		fields[i] = utils.StringifyValue(data[key])
	}
	return fields, nil
}
`, bt, bt, bt, bt, bt, bt, bt, bt, bt, bt, bt, bt, bt, bt, bt, bt, bt, bt)
}

func renderLifecycleParent(data scaffoldData) string {
	if data.UseBulkJobMode {
		return `package lifecycle

import (
	"context"
	"fmt"

	"go-fiber-core/internal/domain"
	"go-fiber-core/internal/models"
	"go-fiber-core/internal/services/exportmanager"

	"gorm.io/gorm"
)

type parentLifecycle struct {
	readDB  *gorm.DB
	writeDB *gorm.DB
}

// NewParentLifecycle concentra los cambios de status del padre durante todo el export.
func NewParentLifecycle(readDB, writeDB *gorm.DB) exportmanager.ParentLifecycle {
	return &parentLifecycle{readDB: readDB, writeDB: writeDB}
}

// Start valida el padre y lo mueve a PROCESSING antes de iniciar el export.
func (l *parentLifecycle) Start(ctx context.Context, execCtx exportmanager.ExecutionContext) error {
	input := execCtx.Input
	var job models.BulkJob
	if err := l.readDB.WithContext(ctx).
		Select("id", "status_code").
		Where("id = ?", input.ParentID).
		First(&job).Error; err != nil {
		return err
	}
	if job.StatusCode != models.BulkJobStatusImported {
		return fmt.Errorf("%w: Verifique el proceso con el id %d ya fue procesado actualmente con el status %s de la tabla bulk_jobs", domain.ErrBusinessRuleViolation, input.ParentID, job.StatusCode)
	}
	return l.writeDB.WithContext(ctx).
		Model(&models.BulkJob{}).
		Where("id = ?", input.ParentID).
		Update("status_code", models.BulkJobStatusProcessing).Error
}

// End persiste el status final exitoso del padre cuando el archivo ya fue generado.
func (l *parentLifecycle) End(ctx context.Context, execCtx exportmanager.ExecutionContext, _ exportmanager.OutputResult) error {
	return l.writeDB.WithContext(ctx).
		Model(&models.BulkJob{}).
		Where("id = ?", execCtx.Input.ParentID).
		Update("status_code", models.BulkJobStatusProcessed).Error
}

// Fail marca error de proceso en el padre cuando el export no puede completarse.
func (l *parentLifecycle) Fail(ctx context.Context, execCtx exportmanager.ExecutionContext, _ error) error {
	return l.writeDB.WithContext(ctx).
		Model(&models.BulkJob{}).
		Where("id = ?", execCtx.Input.ParentID).
		Update("status_code", models.BulkJobStatusErrorProcess).Error
}
`
	}
	return `package lifecycle

import (
	"context"

	"go-fiber-core/internal/services/exportmanager"

	"gorm.io/gorm"
)

type parentLifecycle struct {
	readDB  *gorm.DB
	writeDB *gorm.DB
}

// NewParentLifecycle concentra los cambios de status del padre durante todo el export.
func NewParentLifecycle(readDB, writeDB *gorm.DB) exportmanager.ParentLifecycle {
	return &parentLifecycle{readDB: readDB, writeDB: writeDB}
}

// Start es el punto para validar la entidad padre y cambiar su status al iniciar.
func (l *parentLifecycle) Start(ctx context.Context, execCtx exportmanager.ExecutionContext) error {
	_ = ctx
	_ = execCtx
	_ = l.readDB
	_ = l.writeDB

	// TODO: cambiar el status del padre a PROCESSING.
	return nil
}

// End persiste el status final exitoso del padre al cerrar el export.
func (l *parentLifecycle) End(ctx context.Context, execCtx exportmanager.ExecutionContext, output exportmanager.OutputResult) error {
	_ = ctx
	_ = execCtx
	_ = output

	// TODO: cambiar el status del padre a PROCESSED.
	return nil
}

// Fail persiste el status de error del padre cuando el export no puede continuar.
func (l *parentLifecycle) Fail(ctx context.Context, execCtx exportmanager.ExecutionContext, cause error) error {
	_ = ctx
	_ = execCtx
	_ = cause

	// TODO: cambiar el status del padre a ERROR_PROCESS.
	return nil
}
`
}

func renderLifecycleOutputRegistrar(data scaffoldData) string {
	if data.UseBulkJobMode {
		return `package lifecycle

import (
	"context"
	"encoding/json"

	"go-fiber-core/internal/models"
	"go-fiber-core/internal/services/exportmanager"

	"gorm.io/gorm"
)

type outputRegistrar struct {
	writeDB *gorm.DB
}

// NewOutputRegistrar crea la pieza que persiste el archivo final generado por el export.
func NewOutputRegistrar(writeDB *gorm.DB) exportmanager.OutputRegistrar {
	return &outputRegistrar{writeDB: writeDB}
}

// Register guarda la referencia final del archivo y su metadata operativa en la tabla destino.
func (r *outputRegistrar) Register(ctx context.Context, execCtx exportmanager.ExecutionContext, output exportmanager.OutputResult) error {
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
`
	}
	return `package lifecycle

import (
	"context"

	"go-fiber-core/internal/services/exportmanager"

	"gorm.io/gorm"
)

type outputRegistrar struct {
	writeDB *gorm.DB
}

// NewOutputRegistrar crea la pieza que persiste el archivo final generado por el export.
func NewOutputRegistrar(writeDB *gorm.DB) exportmanager.OutputRegistrar {
	return &outputRegistrar{writeDB: writeDB}
}

// Register es el punto donde se debe persistir el archivo final en la tabla o repositorio elegido.
func (r *outputRegistrar) Register(ctx context.Context, execCtx exportmanager.ExecutionContext, output exportmanager.OutputResult) error {
	_ = ctx
	_ = execCtx
	_ = output
	_ = r.writeDB

	// TODO: persistir el output final en la tabla o repositorio que corresponda.
	return nil
}
`
}

func renderStepStart(data scaffoldData) string {
	return fmt.Sprintf(`package steps

import (
	"fmt"
	"time"

	serviceRuntime "go-fiber-core/internal/services/exports/%s/runtime"
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

// Register publica los steps del export en el runtime de serviceconfig.
func Register() {
	serviceconfig.Register(%q, NewStartStep)
	serviceconfig.Register(%q, NewProcessBatchStep)
	serviceconfig.Register(%q, NewFinalizeStep)
}

// NewStartStep crea el step que prepara la corrida y reserva el estado temporal del export.
func NewStartStep() contracts.Service {
	return &StartStep{
		batchSize: %d,
		ttlHours:  %d,
	}
}

// Init absorbe la configuracion del step definida en el seeder o version del proceso.
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

// Execute inicia el export, genera las claves runtime y deja listo el primer batch para procesar.
func (s *StartStep) Execute() error {
	prov, err := serviceRuntime.ProviderFromContext(s.ctx.Ctx)
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
`, data.ServiceSlug, data.StartKey, data.ProcessBatchKey, data.FinalizeKey, data.BatchSize, data.RedisTTLHours)
}

func renderStepProcessBatch(data scaffoldData) string {
	return fmt.Sprintf(`package steps

import (
	"fmt"

	serviceRuntime "go-fiber-core/internal/services/exports/%s/runtime"
	"go-fiber-core/internal/services/exportmanager"
	"go-fiber-core/internal/services/serviceconfig/contracts"
	"go-fiber-core/internal/utils"
)

type ProcessBatchStep struct {
	ctx         *contracts.ServiceContext
	servicePath string
}

// NewProcessBatchStep crea el step que procesa y sube cada parte temporal del archivo.
func NewProcessBatchStep() contracts.Service {
	return &ProcessBatchStep{}
}

// Init conserva el contexto necesario para resolver input y publicar el resultado del batch.
func (s *ProcessBatchStep) Init(ctx *contracts.ServiceContext, servicePath string) {
	s.ctx = ctx
	s.servicePath = servicePath
}

// Execute entrega el batch actual al manager para construir y almacenar una parte temporal del archivo.
func (s *ProcessBatchStep) Execute() error {
	prov, err := serviceRuntime.ProviderFromContext(s.ctx.Ctx)
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
`, data.ServiceSlug)
}

func renderStepFinalize(data scaffoldData) string {
	return fmt.Sprintf(`package steps

import (
	"fmt"

	serviceRuntime "go-fiber-core/internal/services/exports/%s/runtime"
	"go-fiber-core/internal/services/exportmanager"
	"go-fiber-core/internal/services/serviceconfig/contracts"
	"go-fiber-core/internal/utils"
)

type FinalizeStep struct {
	ctx         *contracts.ServiceContext
	servicePath string
	fileBase    string
}

// NewFinalizeStep crea el step encargado de unir partes y publicar el archivo final.
func NewFinalizeStep() contracts.Service {
	return &FinalizeStep{}
}

// Init absorbe la configuracion final del nombre base del archivo a generar.
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

// Execute llama al manager.Finalize para ensamblar el archivo final y registrar su salida.
func (s *FinalizeStep) Execute() error {
	prov, err := serviceRuntime.ProviderFromContext(s.ctx.Ctx)
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
`, data.ServiceSlug)
}

func renderStepInput(data scaffoldData) string {
	_ = data
	return `package steps

import (
	"fmt"

	"go-fiber-core/internal/services/exportmanager"
	"go-fiber-core/internal/services/serviceconfig/contracts"
	"go-fiber-core/internal/utils"
)

// buildInput reconstruye el input comun que usan process_batch y finalize desde el contexto runtime.
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

// buildStartInput arma el input inicial del run antes de que exista key_redis obligatoria.
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
`
}

func renderStepFailure(data scaffoldData) string {
	return fmt.Sprintf(`package steps

import (
	"context"
	"errors"

	"go-fiber-core/internal/domain"
	serviceRuntime "go-fiber-core/internal/services/exports/%s/runtime"
	"go-fiber-core/internal/services/exportmanager"
)

// markFailure centraliza el fallback de error para que todos los steps cambien el status del padre igual.
func markFailure(prov serviceRuntime.Provider, ctx context.Context, input exportmanager.Input, err error) {
	if errors.Is(err, domain.ErrBusinessRuleViolation) || errors.Is(err, domain.ErrInvalidArgument) {
		return
	}
	_ = prov.Manager().Fail(ctx, input, err)
}
`, data.ServiceSlug)
}

func renderDataProvider(data scaffoldData) string {
	if data.UseBulkJobMode {
		return `package data

import (
	"context"
	"encoding/json"
	"fmt"

	"go-fiber-core/internal/models"
	"go-fiber-core/internal/services/exportmanager"
	"go-fiber-core/internal/utils"

	"gorm.io/gorm"
)

type dataProvider struct {
	readDB *gorm.DB
}

// NewDataProvider crea el origen de datos que alimenta al manager con batches del export.
func NewDataProvider(readDB *gorm.DB) exportmanager.DataProvider {
	return &dataProvider{readDB: readDB}
}

// LoadBatches define la fuente de datos, corta los lotes y prepara el Summary global del archivo.
func (p *dataProvider) LoadBatches(ctx context.Context, execCtx exportmanager.ExecutionContext, batchSize int) (exportmanager.LoadBatchesResult, error) {
	input := execCtx.Input
	if input.ParentID <= 0 {
		return exportmanager.LoadBatchesResult{}, fmt.Errorf("id invalido")
	}
	if batchSize <= 0 {
		batchSize = 5000
	}

	// La consulta base define la fuente de datos del export.
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

	// Se carga el universo completo y luego se parte en batches en memoria.
	var items []models.BulkJobItem
	if err := query.Find(&items).Error; err != nil {
		return exportmanager.LoadBatchesResult{}, err
	}

	batches := make([]exportmanager.Batch, 0, (len(items)/batchSize)+1)
	var current exportmanager.Batch
	current.Items = make([]json.RawMessage, 0, batchSize)

	totalAmount := 0.0
	for _, item := range items {
		// Cada item se serializa para que el body builder lo consuma sin acoplarse a GORM.
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

	// Runtime y Summary dejan los totales disponibles para header, footer y registro final.
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
`
	}
	return fmt.Sprintf(`package data

import (
	"context"
	"fmt"

	"go-fiber-core/internal/services/exportmanager"

	"gorm.io/gorm"
)

type dataProvider struct {
	readDB *gorm.DB
}

// NewDataProvider crea el origen de datos que alimenta al manager con batches del export.
func NewDataProvider(readDB *gorm.DB) exportmanager.DataProvider {
	return &dataProvider{readDB: readDB}
}

// LoadBatches es el punto para consultar la fuente real del export y devolver batches + summary.
func (p *dataProvider) LoadBatches(ctx context.Context, execCtx exportmanager.ExecutionContext, batchSize int) (exportmanager.LoadBatchesResult, error) {
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
`, data.ProcessName)
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
	nextSteps := "1. Implementar `DataProvider`\n2. Implementar `ParentLifecycle`\n3. Implementar `OutputRegistrar`\n4. Ajustar `layout/header_builder.go`, `layout/body_builder.go`, `layout/footer_builder.go` y `layout/layout_helpers.go` si el formato final difiere del default\n"
	if data.UseBulkJobMode {
		mode = fmt.Sprintf("bulk_jobs funcional (bulk_job_id=%d)", data.BulkJobID)
		nextSteps = "1. Ajustar DataProvider si necesitas filtros adicionales\n2. Ajustar `layout/header_builder.go`, `layout/body_builder.go`, `layout/footer_builder.go` y `layout/layout_helpers.go` si el formato final difiere del default\n3. Ejecutar el seeder y probar con Bruno\n"
	}
	return fmt.Sprintf("# %s\n\n## Objetivo\n\nScaffold base generado automaticamente para montar un flujo sobre `exportmanager`.\n\n## Modo\n\n- `%s`\n- `item-oriented` por contrato: cada registro del batch se transforma desde `BodyBuilder.renderItem(...)`\n\n## Execution Keys\n\n- `%s`\n- `%s`\n- `%s`\n\n## Config base\n\n- batch_size: %d\n- part_prefix: `%s`\n- redis_ttl_hours: %d\n- file: `%s`\n\n## Variables que recibe el developer\n\nTodos los servicios funcionales reciben:\n\n- `id`: id del padre de negocio\n- `key_redis`: key unica de la corrida\n\n## Runtime en Redis\n\nEl runtime permite compartir datos entre data/header/body/footer/end.\n\nEjemplo:\n\n- guardar `total_amount` en header\n- leer `total_amount` en footer\n\n## Layout por defecto\n\n- `header`, `body` y `footer` salen funcionales desde el scaffold\n- usa CSV con separador `;`\n- incluye columnas históricas y cálculo de `new_importe`\n- concentra helpers compartidos en `layout/layout_helpers.go`\n\n## BodyBuilder\n\n- `BuildBodyLines(...)` mantiene el contrato del framework\n- `renderItem(...)` es el punto de extension recomendado para el developer\n- el preview reutiliza este mismo camino de render por item\n\n## Siguientes pasos\n\n%s", data.ProcessName, mode, data.StartKey, data.ProcessBatchKey, data.FinalizeKey, data.BatchSize, data.PartPrefix, data.RedisTTLHours, data.FileBase, nextSteps)
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
		paths.runtimeProviderContextFile,
		paths.dataProviderFile,
		paths.layoutHeaderFile,
		paths.layoutBodyFile,
		paths.layoutFooterFile,
		paths.layoutHelpersFile,
		paths.lifecycleParentFile,
		paths.lifecycleOutputRegistrarFile,
		paths.stepsStartFile,
		paths.stepsProcessBatchFile,
		paths.stepsFinalizeFile,
		paths.stepsInputFile,
		paths.stepsFailureFile,
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
