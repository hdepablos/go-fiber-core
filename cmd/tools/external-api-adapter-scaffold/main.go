package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

type options struct {
	AdapterName string
	ConfigKey   string
	Force       bool
}

type scaffoldData struct {
	AdapterName   string
	ConfigKey     string
	PackageName   string
	InterfaceName string
	StructName    string
	Constructor   string
	SourceName    string
}

func main() {
	opts := parseOptions()
	data := buildScaffoldData(opts)
	targetFile := filepath.Join("/private/var/www/go-fiber-core/internal/adapters", data.PackageName+"_adapter.go")

	if err := ensureTarget(targetFile, opts.Force); err != nil {
		fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(targetFile), 0o755); err != nil {
		fatal(err)
	}
	if err := os.WriteFile(targetFile, []byte(renderAdapter(data)), 0o644); err != nil {
		fatal(fmt.Errorf("escribiendo adapter: %w", err))
	}

	fmt.Println("Adapter externo generado correctamente")
	fmt.Printf("Adapter: %s\n", data.AdapterName)
	fmt.Printf("Config Key: %s\n", data.ConfigKey)
	fmt.Printf("Archivo: %s\n", targetFile)
	fmt.Println("Siguientes pasos:")
	fmt.Printf("1. Si falta, crear config con: make create-external-api-config api_key=%s\n", data.ConfigKey)
	fmt.Printf("2. Obtener config con appConfig.APIConfig(%q)\n", data.ConfigKey)
	fmt.Printf("3. Instanciar con adapters.%s(apiCfg)\n", data.Constructor)
	fmt.Println("4. Reemplazar el metodo Do por metodos de dominio si la integracion lo requiere")
}

func parseOptions() options {
	var opts options
	flag.StringVar(&opts.AdapterName, "adapter-name", "", "Nombre del adapter, por ejemplo customer_api")
	flag.StringVar(&opts.ConfigKey, "config-key", "", "Clave bajo apis.xxx en config.yml")
	flag.BoolVar(&opts.Force, "force", false, "Sobrescribe el archivo si ya existe")
	flag.Parse()

	opts.AdapterName = strings.TrimSpace(opts.AdapterName)
	opts.ConfigKey = strings.TrimSpace(opts.ConfigKey)
	if opts.AdapterName == "" {
		fatal(fmt.Errorf("adapter-name es obligatorio"))
	}
	if opts.ConfigKey == "" {
		fatal(fmt.Errorf("config-key es obligatorio"))
	}
	return opts
}

func buildScaffoldData(opts options) scaffoldData {
	pkg := slugify(opts.AdapterName)
	pascal := toPascalCase(pkg)
	return scaffoldData{
		AdapterName:   opts.AdapterName,
		ConfigKey:     opts.ConfigKey,
		PackageName:   pkg,
		InterfaceName: pascal + "Adapter",
		StructName:    lowerFirst(pascal) + "Adapter",
		Constructor:   "New" + pascal + "Adapter",
		SourceName:    opts.ConfigKey,
	}
}

func ensureTarget(targetFile string, force bool) error {
	if _, err := os.Stat(targetFile); err == nil && !force {
		return fmt.Errorf("el archivo ya existe: %s (usa --force si quieres sobrescribir)", targetFile)
	}
	return nil
}

func renderAdapter(data scaffoldData) string {
	return fmt.Sprintf(`package adapters

import (
	"context"

	"go-fiber-core/internal/dtos/config"
	"go-fiber-core/internal/services/externalhttp"

	resty "github.com/go-resty/resty/v2"
)

type %s interface {
	Do(ctx context.Context, req externalhttp.Request) (*resty.Response, error)
}

type %s struct {
	httpClient externalhttp.Client
	apiConfig  config.ApiConfig
}

func %s(apiCfg config.ApiConfig) %s {
	return %sWithService(apiCfg, externalhttp.NewClientFromAPIConfig(apiCfg, nil))
}

func %sWithService(apiCfg config.ApiConfig, httpClient externalhttp.Client) %s {
	return &%s{
		httpClient: httpClient,
		apiConfig:  apiCfg,
	}
}

func (a *%s) Do(ctx context.Context, req externalhttp.Request) (*resty.Response, error) {
	if req.Source == "" {
		req.Source = %q
	}
	return a.httpClient.Do(ctx, req)
}
`, data.InterfaceName, data.StructName, data.Constructor, data.InterfaceName, data.Constructor, data.Constructor, data.InterfaceName, data.StructName, data.StructName, data.SourceName)
}

func slugify(input string) string {
	input = strings.TrimSpace(strings.ToLower(input))
	var b strings.Builder
	lastUnderscore := false
	for _, r := range input {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
			lastUnderscore = false
		case r == '_' || r == '-' || unicode.IsSpace(r):
			if !lastUnderscore && b.Len() > 0 {
				b.WriteByte('_')
				lastUnderscore = true
			}
		}
	}
	return strings.Trim(b.String(), "_")
}

func toPascalCase(input string) string {
	parts := strings.FieldsFunc(input, func(r rune) bool {
		return r == '_' || r == '-' || unicode.IsSpace(r)
	})
	var b strings.Builder
	for _, part := range parts {
		if part == "" {
			continue
		}
		b.WriteString(strings.ToUpper(part[:1]))
		if len(part) > 1 {
			b.WriteString(part[1:])
		}
	}
	return b.String()
}

func lowerFirst(input string) string {
	if input == "" {
		return ""
	}
	return strings.ToLower(input[:1]) + input[1:]
}

func fatal(err error) {
	fmt.Fprintf(os.Stderr, "error: %v\n", err)
	os.Exit(1)
}
