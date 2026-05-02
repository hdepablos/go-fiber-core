package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const repoRoot = "/private/var/www/go-fiber-core"

type options struct {
	Kind        string
	ServiceSlug string
	DryRun      bool
}

type processConfig struct {
	Kind                 string
	ServiceSlug          string
	PascalName           string
	ImportPath           string
	LegacyImportPaths    []string
	SeedName             string
	FanoutSeedName       string
	SeederFuncName       string
	FanoutSeederFuncName string
	RuntimeFieldName     string
	FilesToDelete        []string
	DirsToDelete         []string
	NeedsRuntimeWiring   bool
}

func main() {
	opts := parseOptions()
	cfg, err := buildConfig(opts)
	if err != nil {
		fatal(err)
	}

	if err := cleanupProcess(cfg, opts.DryRun); err != nil {
		fatal(err)
	}

	fmt.Println("Proceso eliminado del codigo correctamente")
	fmt.Printf("Tipo: %s\n", cfg.Kind)
	fmt.Printf("Service Slug: %s\n", cfg.ServiceSlug)
	if opts.DryRun {
		fmt.Println("Modo: dry-run")
	}
}

func parseOptions() options {
	var opts options
	flag.StringVar(&opts.Kind, "kind", "", "Tipo de proceso: batch-process o export")
	flag.StringVar(&opts.ServiceSlug, "service-slug", "", "Slug tecnico del proceso")
	flag.BoolVar(&opts.DryRun, "dry-run", false, "Muestra cambios sin escribir ni borrar")
	flag.Parse()
	return opts
}

func buildConfig(opts options) (processConfig, error) {
	kind := normalizeKind(opts.Kind)
	if kind == "" {
		return processConfig{}, fmt.Errorf("kind invalido: usa batch-process o export")
	}
	slug := normalizeIdentifier(opts.ServiceSlug)
	if slug == "" {
		return processConfig{}, fmt.Errorf("service_slug invalido")
	}

	cfg := processConfig{
		Kind:        kind,
		ServiceSlug: slug,
		PascalName:  toPascal(slug),
		ImportPath:  "go-fiber-core/internal/services/" + slug,
	}

	switch kind {
	case "batch-process":
		cfg.ImportPath = "go-fiber-core/internal/services/batchprocess/" + slug
		cfg.LegacyImportPaths = []string{"go-fiber-core/internal/services/" + slug}
		cfg.SeedName = "batch_process_" + slug
		cfg.FanoutSeedName = "batch_process_" + slug + "_fanout"
		cfg.SeederFuncName = "BatchProcess" + cfg.PascalName + "Seeder"
		cfg.FanoutSeederFuncName = "BatchProcess" + cfg.PascalName + "FanoutSeeder"
		cfg.RuntimeFieldName = cfg.PascalName
		cfg.NeedsRuntimeWiring = true
		cfg.FilesToDelete = []string{
			filepath.Join(repoRoot, "internal/services/batchprocess", slug, "provider.go"),
			filepath.Join(repoRoot, "internal/services/batchprocess", slug, "runtime", "provider_context.go"),
			filepath.Join(repoRoot, "internal/services/batchprocess", slug, "data", "provider.go"),
			filepath.Join(repoRoot, "internal/services/batchprocess", slug, "processor", "processor.go"),
			filepath.Join(repoRoot, "internal/services/batchprocess", slug, "lifecycle", "parent.go"),
			filepath.Join(repoRoot, "internal/services/batchprocess", slug, "lifecycle", "finalizer.go"),
			filepath.Join(repoRoot, "internal/services/batchprocess", slug, "steps", "start.go"),
			filepath.Join(repoRoot, "internal/services/batchprocess", slug, "steps", "dispatch_shards.go"),
			filepath.Join(repoRoot, "internal/services/batchprocess", slug, "steps", "process_batch.go"),
			filepath.Join(repoRoot, "internal/services/batchprocess", slug, "steps", "finalize.go"),
			filepath.Join(repoRoot, "internal/services/batchprocess", slug, "steps", "input.go"),
			filepath.Join(repoRoot, "internal/services/batchprocess", slug, "steps", "failure.go"),
			filepath.Join(repoRoot, "internal/services/batchprocess", slug, "steps", "helpers.go"),
			filepath.Join(repoRoot, "internal/database/seeders", slug+"_seeder.go"),
			filepath.Join(repoRoot, "internal/database/seeders", slug+"_fanout_seeder.go"),
			filepath.Join(repoRoot, "bruno/legacy/process-lifecycle", "RunProc -> "+slug+".bru"),
		}
		cfg.DirsToDelete = []string{
			filepath.Join(repoRoot, "internal/services/batchprocess", slug),
			filepath.Join(repoRoot, "internal/services", slug),
			filepath.Join(repoRoot, "bruno/legacy/process-lifecycle/test-batch-process", slug),
		}
	case "export":
		cfg.ImportPath = "go-fiber-core/internal/services/exports/" + slug
		cfg.LegacyImportPaths = []string{"go-fiber-core/internal/services/" + slug}
		cfg.SeedName = "export_manager_" + slug
		cfg.SeederFuncName = "ExportManager" + cfg.PascalName + "Seeder"
		cfg.RuntimeFieldName = cfg.PascalName
		cfg.NeedsRuntimeWiring = true
		cfg.FilesToDelete = []string{
			filepath.Join(repoRoot, "internal/services/exports", slug, "provider.go"),
			filepath.Join(repoRoot, "internal/services/exports", slug, "steps.go"),
			filepath.Join(repoRoot, "internal/services/exports", slug, "data_provider.go"),
			filepath.Join(repoRoot, "internal/services/exports", slug, "layout.go"),
			filepath.Join(repoRoot, "internal/services/exports", slug, "lifecycle.go"),
			filepath.Join(repoRoot, "internal/database/seeders", slug+"_seeder.go"),
			filepath.Join(repoRoot, "doc/info", "exportmanager_"+slug+".md"),
			filepath.Join(repoRoot, "bruno/process-lifecycle", "RunProc -> "+slug+".bru"),
		}
		cfg.DirsToDelete = []string{
			filepath.Join(repoRoot, "internal/services/exports", slug),
			filepath.Join(repoRoot, "internal/services", slug),
		}
	default:
		return processConfig{}, fmt.Errorf("kind no soportado: %s", kind)
	}

	return cfg, nil
}

func cleanupProcess(cfg processConfig, dryRun bool) error {
	importPaths := append([]string{cfg.ImportPath}, cfg.LegacyImportPaths...)
	for _, importPath := range importPaths {
		if err := patchMainImports(filepath.Join(repoRoot, "cmd/api/main.go"), importPath, dryRun); err != nil {
			return err
		}
		if err := patchMainImports(filepath.Join(repoRoot, "cmd/sqs-consumer/main.go"), importPath, dryRun); err != nil {
			return err
		}
	}
	if cfg.NeedsRuntimeWiring {
		if err := patchRuntimeBootstrap(cfg, dryRun); err != nil {
			return err
		}
	}
	if err := patchSeedService(cfg, dryRun); err != nil {
		return err
	}

	if dryRun {
		for _, path := range append(cfg.FilesToDelete, cfg.DirsToDelete...) {
			fmt.Printf("[dry-run] delete %s\n", path)
		}
		return nil
	}

	for _, path := range cfg.FilesToDelete {
		if err := removePath(path); err != nil {
			return err
		}
	}
	for _, path := range cfg.DirsToDelete {
		if err := removePath(path); err != nil {
			return err
		}
	}
	return nil
}

func patchMainImports(filePath, importPath string, dryRun bool) error {
	pattern := regexp.MustCompile(`(?m)^\t_ "` + regexp.QuoteMeta(importPath) + `"\)?\s*$\n?`)
	return removePattern(filePath, pattern, dryRun, true)
}

func patchSeedService(cfg processConfig, dryRun bool) error {
	filePath := filepath.Join(repoRoot, "internal/database/seeders/seed_service.go")
	listPattern := regexp.MustCompile(`(?m)^\t\t"` + regexp.QuoteMeta(cfg.SeedName) + `",\s*$\n?`)
	fanoutListPattern := regexp.MustCompile(`(?m)^\t\t"` + regexp.QuoteMeta(cfg.FanoutSeedName) + `",\s*$\n?`)
	funcPattern := regexp.MustCompile(`(?ms)\n\tservice\.AddSeeder\("` + regexp.QuoteMeta(cfg.SeedName) + `", func\(\) error \{\n\t\treturn ` + regexp.QuoteMeta(cfg.SeederFuncName) + `\(pool\)\n\t\}\)\n*`)
	fanoutFuncPattern := regexp.MustCompile(`(?ms)\n\tservice\.AddSeeder\("` + regexp.QuoteMeta(cfg.FanoutSeedName) + `", func\(\) error \{\n\t\treturn ` + regexp.QuoteMeta(cfg.FanoutSeederFuncName) + `\(pool\)\n\t\}\)\n*`)

	patterns := []*regexp.Regexp{listPattern, fanoutListPattern, funcPattern, fanoutFuncPattern}
	return removePatternsAndNormalize(filePath, patterns, dryRun, false)
}

func patchRuntimeBootstrap(cfg processConfig, dryRun bool) error {
	filePath := filepath.Join(repoRoot, "internal/runtimebootstrap/bootstrap.go")
	allImportPaths := append([]string{cfg.ImportPath}, cfg.LegacyImportPaths...)
	quotedImports := make([]string, 0, len(allImportPaths))
	for _, importPath := range allImportPaths {
		quotedImports = append(quotedImports, regexp.QuoteMeta(importPath))
	}
	importPattern := regexp.MustCompile(`(?m)^\t"(?:` + strings.Join(quotedImports, "|") + `)"\)?\s*$\n?`)
	fieldPattern := regexp.MustCompile(`(?m)^\t` + regexp.QuoteMeta(cfg.RuntimeFieldName) + `\s+` + regexp.QuoteMeta(cfg.ServiceSlug) + `\.Provider\s*$\n?`)
	legacyBuildPattern := regexp.MustCompile(`(?ms)\n\tif prov, err := ` + regexp.QuoteMeta(cfg.ServiceSlug) + `\.NewProviderWithConfig\([^)]*\); err == nil \{\n\t\tdeps\.` + regexp.QuoteMeta(cfg.RuntimeFieldName) + ` = prov\n\t\} else \{\n\t\terrs = append\(errs, fmt\.Sprintf\("` + regexp.QuoteMeta(cfg.ServiceSlug) + `: %v", err\)\)\n\t\}\n*`)
	markedBuildPattern := regexp.MustCompile(`(?ms)\n\t// scaffold:export-runtime-start:` + regexp.QuoteMeta(cfg.ServiceSlug) + `\n.*?\n\t// scaffold:export-runtime-end:` + regexp.QuoteMeta(cfg.ServiceSlug) + `\n*`)
	injectPattern := regexp.MustCompile(`(?ms)\n\tif d\.` + regexp.QuoteMeta(cfg.RuntimeFieldName) + ` != nil \{\n\t\tctx = ` + regexp.QuoteMeta(cfg.ServiceSlug) + `\.WithProvider\(ctx, d\.` + regexp.QuoteMeta(cfg.RuntimeFieldName) + `\)\n\t\}\n*`)

	patterns := []*regexp.Regexp{importPattern, fieldPattern, legacyBuildPattern, markedBuildPattern, injectPattern}
	return removePatternsAndNormalize(filePath, patterns, dryRun, true)
}

func removePattern(filePath string, pattern *regexp.Regexp, dryRun bool, normalizeImports bool) error {
	return removePatternsAndNormalize(filePath, []*regexp.Regexp{pattern}, dryRun, normalizeImports)
}

func removePatternsAndNormalize(filePath string, patterns []*regexp.Regexp, dryRun bool, normalizeImports bool) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("leyendo %s: %w", filePath, err)
	}
	updated := string(content)
	for _, pattern := range patterns {
		updated = pattern.ReplaceAllString(updated, "")
	}
	if normalizeImports {
		updated = normalizeImportBlock(updated)
	}
	if strings.Contains(filePath, "internal/database/seeders/seed_service.go") {
		updated = normalizeSeedService(updated)
	}
	if strings.Contains(filePath, "internal/runtimebootstrap/bootstrap.go") {
		updated = normalizeRuntimeBootstrap(updated)
	}
	if dryRun {
		fmt.Printf("[dry-run] patch %s\n", filePath)
		return nil
	}
	return os.WriteFile(filePath, []byte(updated), 0o644)
}

func normalizeImportBlock(fileStr string) string {
	lines := strings.Split(fileStr, "\n")
	importStart := -1
	for i, line := range lines {
		if strings.TrimSpace(line) == "import (" {
			importStart = i
			break
		}
	}
	if importStart == -1 {
		return fileStr
	}
	for i := importStart + 1; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		switch {
		case trimmed == ")":
			return strings.Join(lines, "\n")
		case strings.Contains(trimmed, "\"") && strings.HasSuffix(trimmed, ")"):
			lines[i] = strings.TrimSuffix(lines[i], ")")
			lines = append(lines[:i+1], append([]string{")"}, lines[i+1:]...)...)
			return strings.Join(lines, "\n")
		case strings.HasPrefix(trimmed, "var ") || trimmed == "var (" || strings.HasPrefix(trimmed, "func "):
			lines = append(lines[:i], append([]string{")"}, lines[i:]...)...)
			return strings.Join(lines, "\n")
		}
	}
	lines = append(lines, ")")
	return strings.Join(lines, "\n")
}

func normalizeRuntimeBootstrap(fileStr string) string {
	fileStr = strings.ReplaceAll(fileStr, "\t}\tif", "\t}\n\tif")
	fileStr = strings.ReplaceAll(fileStr, "\t}\treturn", "\t}\n\treturn")
	if !strings.Contains(fileStr, "NewProviderWithConfig(appCfg, conn, conn.ConnectRedis, s3Client)") {
		fileStr = regexp.MustCompile(`(?ms)\n\tawsSvc, err := queue\.NewAWSService\(ctx\)\n\tif err != nil \{\n\t\treturn deps, fmt\.Errorf\("runtime bootstrap aws: %w", err\)\n\t\}\n\ts3Client := awsSvc\.NewS3Client\(\)\n*`).ReplaceAllString(fileStr, "\n")
	}
	fileStr = regexp.MustCompile(`\n{3,}`).ReplaceAllString(fileStr, "\n\n")
	return fileStr
}

func normalizeSeedService(fileStr string) string {
	fileStr = regexp.MustCompile(`\}\)\s*service\.AddSeeder\(`).ReplaceAllString(fileStr, "})\n\n\tservice.AddSeeder(")
	fileStr = regexp.MustCompile(`\}\)\s*//`).ReplaceAllString(fileStr, "})\n\t//")
	fileStr = regexp.MustCompile(`\n{3,}`).ReplaceAllString(fileStr, "\n\n")
	return fileStr
}

func removePath(path string) error {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return os.RemoveAll(path)
}

func normalizeKind(kind string) string {
	switch strings.TrimSpace(strings.ToLower(kind)) {
	case "batch-process", "batch", "batchflow":
		return "batch-process"
	case "export", "export-manager", "exportmanager":
		return "export"
	default:
		return ""
	}
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

func fatal(err error) {
	fmt.Fprintf(os.Stderr, "error: %v\n", err)
	os.Exit(1)
}
