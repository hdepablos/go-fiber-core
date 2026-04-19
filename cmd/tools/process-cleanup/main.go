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
		cfg.SeedName = "batch_process_" + slug
		cfg.FanoutSeedName = "batch_process_" + slug + "_fanout"
		cfg.SeederFuncName = "BatchProcess" + cfg.PascalName + "Seeder"
		cfg.FanoutSeederFuncName = "BatchProcess" + cfg.PascalName + "FanoutSeeder"
		cfg.RuntimeFieldName = cfg.PascalName
		cfg.NeedsRuntimeWiring = true
		cfg.FilesToDelete = []string{
			filepath.Join(repoRoot, "internal/services", slug, "provider.go"),
			filepath.Join(repoRoot, "internal/services", slug, "steps.go"),
			filepath.Join(repoRoot, "internal/services", slug, "data_provider.go"),
			filepath.Join(repoRoot, "internal/services", slug, "processor.go"),
			filepath.Join(repoRoot, "internal/services", slug, "lifecycle.go"),
			filepath.Join(repoRoot, "internal/database/seeders", slug+"_seeder.go"),
			filepath.Join(repoRoot, "internal/database/seeders", slug+"_fanout_seeder.go"),
			filepath.Join(repoRoot, "bruno/legacy/process-lifecycle", "RunProc -> "+slug+".bru"),
		}
		cfg.DirsToDelete = []string{
			filepath.Join(repoRoot, "internal/services", slug),
			filepath.Join(repoRoot, "bruno/legacy/process-lifecycle/test-batch-process", slug),
		}
	case "export":
		cfg.SeedName = "export_manager_" + slug
		cfg.SeederFuncName = "ExportManager" + cfg.PascalName + "Seeder"
		cfg.RuntimeFieldName = cfg.PascalName
		cfg.NeedsRuntimeWiring = true
		cfg.FilesToDelete = []string{
			filepath.Join(repoRoot, "internal/services", slug, "provider.go"),
			filepath.Join(repoRoot, "internal/services", slug, "steps.go"),
			filepath.Join(repoRoot, "internal/services", slug, "data_provider.go"),
			filepath.Join(repoRoot, "internal/services", slug, "layout.go"),
			filepath.Join(repoRoot, "internal/services", slug, "lifecycle.go"),
			filepath.Join(repoRoot, "internal/database/seeders", slug+"_seeder.go"),
			filepath.Join(repoRoot, "doc/info", "exportmanager_"+slug+".md"),
			filepath.Join(repoRoot, "bruno/process-lifecycle", "RunProc -> "+slug+".bru"),
		}
		cfg.DirsToDelete = []string{
			filepath.Join(repoRoot, "internal/services", slug),
		}
	default:
		return processConfig{}, fmt.Errorf("kind no soportado: %s", kind)
	}

	return cfg, nil
}

func cleanupProcess(cfg processConfig, dryRun bool) error {
	if err := patchMainImports(filepath.Join(repoRoot, "cmd/api/main.go"), cfg.ImportPath, dryRun); err != nil {
		return err
	}
	if err := patchMainImports(filepath.Join(repoRoot, "cmd/sqs-consumer/main.go"), cfg.ImportPath, dryRun); err != nil {
		return err
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
	line := "\n\t_ " + fmt.Sprintf("%q", importPath)
	return removeLiteral(filePath, line, dryRun)
}

func patchSeedService(cfg processConfig, dryRun bool) error {
	filePath := filepath.Join(repoRoot, "internal/database/seeders/seed_service.go")
	listEntry := fmt.Sprintf("\n\t\t%q,", cfg.SeedName)
	if err := removeLiteral(filePath, listEntry, dryRun); err != nil {
		return err
	}
	fanoutListEntry := fmt.Sprintf("\n\t\t%q,", cfg.FanoutSeedName)
	if err := removeLiteral(filePath, fanoutListEntry, dryRun); err != nil {
		return err
	}

	funcBlock := fmt.Sprintf(`
	service.AddSeeder(%q, func() error {
		return %s(pool)
	})
`, cfg.SeedName, cfg.SeederFuncName)
	if err := removeLiteral(filePath, funcBlock, dryRun); err != nil {
		return err
	}
	fanoutFuncBlock := fmt.Sprintf(`
	service.AddSeeder(%q, func() error {
		return %s(pool)
	})
`, cfg.FanoutSeedName, cfg.FanoutSeederFuncName)
	return removeLiteral(filePath, fanoutFuncBlock, dryRun)
}

func patchRuntimeBootstrap(cfg processConfig, dryRun bool) error {
	filePath := filepath.Join(repoRoot, "internal/runtimebootstrap/bootstrap.go")
	importLine := "\n\t" + fmt.Sprintf("%q", cfg.ImportPath)
	if err := removeLiteral(filePath, importLine, dryRun); err != nil {
		return err
	}

	fieldLine := fmt.Sprintf("\n\t%s  %s.Provider", cfg.RuntimeFieldName, cfg.ServiceSlug)
	if err := removeLiteral(filePath, fieldLine, dryRun); err != nil {
		altFieldLine := fmt.Sprintf("\n\t%s %s.Provider", cfg.RuntimeFieldName, cfg.ServiceSlug)
		if err2 := removeLiteral(filePath, altFieldLine, dryRun); err2 != nil {
			return err2
		}
	}

	buildCall := fmt.Sprintf("%s.NewProviderWithConfig(appCfg, conn, conn.ConnectRedis)", cfg.ServiceSlug)
	if cfg.Kind == "export" {
		buildCall = fmt.Sprintf("%s.NewProviderWithConfig(appCfg, conn, conn.ConnectRedis, s3Client)", cfg.ServiceSlug)
	}
	buildBlock := fmt.Sprintf(`
	if prov, err := %s; err == nil {
		deps.%s = prov
	} else {
		errs = append(errs, fmt.Sprintf(%q, err))
	}
`, buildCall, cfg.RuntimeFieldName, cfg.ServiceSlug+": %v")
	if err := removeLiteral(filePath, buildBlock, dryRun); err != nil {
		return err
	}

	injectBlock := fmt.Sprintf(`
	if d.%s != nil {
		ctx = %s.WithProvider(ctx, d.%s)
	}
`, cfg.RuntimeFieldName, cfg.ServiceSlug, cfg.RuntimeFieldName)
	return removeLiteral(filePath, injectBlock, dryRun)
}

func removeLiteral(filePath, literal string, dryRun bool) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("leyendo %s: %w", filePath, err)
	}
	fileStr := string(content)
	if !strings.Contains(fileStr, literal) {
		return nil
	}
	updated := strings.Replace(fileStr, literal, "", 1)
	if dryRun {
		fmt.Printf("[dry-run] patch %s\n", filePath)
		return nil
	}
	return os.WriteFile(filePath, []byte(updated), 0o644)
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
