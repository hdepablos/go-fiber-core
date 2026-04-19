package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"unicode"
)

const configFile = "/private/var/www/go-fiber-core/internal/appconfig/config.yml"

type options struct {
	APIKey string
	Force  bool
}

func main() {
	opts := parseOptions()
	content, err := os.ReadFile(configFile)
	if err != nil {
		fatal(fmt.Errorf("leyendo config.yml: %w", err))
	}

	updated, err := upsertAPIBlock(string(content), opts)
	if err != nil {
		fatal(err)
	}

	if err := os.WriteFile(configFile, []byte(updated), 0o644); err != nil {
		fatal(fmt.Errorf("escribiendo config.yml: %w", err))
	}

	fmt.Println("Config API externa agregada correctamente")
	fmt.Printf("API Key: %s\n", opts.APIKey)
	fmt.Printf("Archivo: %s\n", configFile)
	fmt.Println("Siguientes pasos:")
	fmt.Printf("1. Resolver con appConfig.APIConfig(%q)\n", opts.APIKey)
	fmt.Printf("2. Crear adapter con: make create-external-adapter adapter_name=%s config_key=%s\n", opts.APIKey, opts.APIKey)
}

func parseOptions() options {
	var opts options
	flag.StringVar(&opts.APIKey, "api-key", "", "Clave bajo apis.xxx en config.yml")
	flag.BoolVar(&opts.Force, "force", false, "Sobrescribe la entrada si ya existe")
	flag.Parse()

	opts.APIKey = slugify(opts.APIKey)
	if opts.APIKey == "" {
		fatal(fmt.Errorf("api-key es obligatorio"))
	}
	return opts
}

func upsertAPIBlock(content string, opts options) (string, error) {
	lines := strings.Split(content, "\n")
	apiBlock := renderAPIBlock(opts.APIKey)

	apisIndex := findAPIsSection(lines)
	if apisIndex == -1 {
		if !strings.HasSuffix(content, "\n") {
			content += "\n"
		}
		if strings.TrimSpace(content) != "" {
			content += "\n"
		}
		return content + "apis:\n" + apiBlock, nil
	}

	sectionEnd := findAPIsSectionEnd(lines, apisIndex)
	entryStart, entryEnd, exists := findAPIEntry(lines, apisIndex+1, sectionEnd, opts.APIKey)
	if exists {
		if !opts.Force {
			return "", fmt.Errorf("la entrada apis.%s ya existe en %s (usa --force si quieres sobrescribir)", opts.APIKey, configFile)
		}
		replacement := strings.Split(strings.TrimRight(apiBlock, "\n"), "\n")
		lines = append(lines[:entryStart], append(replacement, lines[entryEnd:]...)...)
		return strings.Join(lines, "\n"), nil
	}

	insertLines := strings.Split(strings.TrimRight(apiBlock, "\n"), "\n")
	lines = append(lines[:sectionEnd], append(insertLines, lines[sectionEnd:]...)...)
	return strings.Join(lines, "\n"), nil
}

func renderAPIBlock(apiKey string) string {
	envBase := envBaseName(apiKey)
	return fmt.Sprintf("  %s:\n    url: ${%s_URL}\n    token: ${%s_TOKEN}\n    timeout_seconds: 10\n", apiKey, envBase, envBase)
}

func findAPIsSection(lines []string) int {
	for i, line := range lines {
		if strings.TrimSpace(line) == "apis:" {
			return i
		}
	}
	return -1
}

func findAPIsSectionEnd(lines []string, apisIndex int) int {
	for i := apisIndex + 1; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") {
			return i
		}
	}
	return len(lines)
}

func findAPIEntry(lines []string, start, end int, apiKey string) (int, int, bool) {
	entryLine := "  " + apiKey + ":"
	for i := start; i < end; i++ {
		if lines[i] != entryLine {
			continue
		}
		j := i + 1
		for ; j < end; j++ {
			line := lines[j]
			trimmed := strings.TrimSpace(line)
			if trimmed == "" {
				continue
			}
			if strings.HasPrefix(line, "  ") && !strings.HasPrefix(line, "    ") {
				break
			}
			if !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") {
				break
			}
		}
		return i, j, true
	}
	return -1, -1, false
}

func envBaseName(apiKey string) string {
	upper := strings.ToUpper(apiKey)
	var b strings.Builder
	lastUnderscore := false
	for _, r := range upper {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
			lastUnderscore = false
		default:
			if !lastUnderscore && b.Len() > 0 {
				b.WriteByte('_')
				lastUnderscore = true
			}
		}
	}
	return strings.Trim(b.String(), "_")
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

func fatal(err error) {
	fmt.Fprintf(os.Stderr, "error: %v\n", err)
	os.Exit(1)
}
