package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
)

func main() {
	namePtr := flag.String("name", "", "Nombre del servicio (formato: carpeta/servicio)")
	flag.Parse()

	if *namePtr == "" {
		fmt.Println("❌ Debes especificar un nombre: -name carpeta/servicio")
		os.Exit(1)
	}

	parts := strings.Split(*namePtr, "/")
	if len(parts) < 2 {
		fmt.Println("❌ El formato debe ser 'carpeta/servicio' o 'ruta/a/servicio' (ej: loanrisk/verify_income, test/concurrent/step1)")
		os.Exit(1)
	}

	// El último elemento es el nombre del servicio (archivo)
	serviceName := parts[len(parts)-1]

	// Todo lo anterior es la ruta de carpetas
	folderPath := strings.Join(parts[:len(parts)-1], "/")

	// El nombre del paquete será el nombre de la última carpeta contenedora
	packageName := parts[len(parts)-2]

	// Validar nombres
	for _, part := range parts {
		if !isValidName(part) {
			fmt.Printf("❌ Nombre inválido: '%s'. Usa solo letras minúsculas, números y guiones bajos.\n", part)
			os.Exit(1)
		}
	}

	// Rutas
	baseDir := "internal/services"
	targetDir := filepath.Join(baseDir, folderPath)
	targetFile := filepath.Join(targetDir, serviceName+".go")

	// Crear directorio si no existe
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		fmt.Printf("❌ Error creando directorio %s: %v\n", targetDir, err)
		os.Exit(1)
	}

	// Verificar si el archivo ya existe
	if _, err := os.Stat(targetFile); err == nil {
		fmt.Printf("⚠️ El archivo %s ya existe. No se sobrescribirá.\n", targetFile)
		os.Exit(1)
	}

	// Generar contenido
	content := generateContent(folderPath, packageName, serviceName)

	// Escribir archivo
	if err := os.WriteFile(targetFile, []byte(content), 0644); err != nil {
		fmt.Printf("❌ Error escribiendo archivo %s: %v\n", targetFile, err)
		os.Exit(1)
	}

	fmt.Printf("✅ Servicio creado: %s\n", targetFile)

	// Inyectar imports
	injectImport("cmd/api/main.go", folderPath)
	injectImport("cmd/cmd-cli/main.go", folderPath)
}

func isValidName(s string) bool {
	match, _ := regexp.MatchString("^[a-z0-9_]+$", s)
	return match
}

func toCamelCase(s string) string {
	parts := strings.Split(s, "_")
	for i, part := range parts {
		if len(part) > 0 {
			runes := []rune(part)
			runes[0] = unicode.ToUpper(runes[0])
			parts[i] = string(runes)
		}
	}
	return strings.Join(parts, "")
}

func generateContent(folderPath, packageName, serviceName string) string {
	structName := toCamelCase(serviceName)

	return fmt.Sprintf(`package %s

import (
	"fmt"
	"go-fiber-core/internal/services/serviceconfig"
	"go-fiber-core/internal/services/serviceconfig/contracts"
)

// %s es la implementación del servicio.
type %s struct {
	ctx         *contracts.ServiceContext
	servicePath string
}

// New%sService es el constructor.
func New%sService() contracts.Service {
	return &%s{}
}

// Init inicializa el servicio con su contexto.
func (s *%s) Init(ctx *contracts.ServiceContext, servicePath string) {
	s.ctx = ctx
	s.servicePath = servicePath
}

// Execute contiene la lógica de negocio.
func (s *%s) Execute() error {
	// TODO: Implementar lógica aquí
	fmt.Printf("🚀 Ejecutando servicio: %%s\n", s.servicePath)

	// Ejemplo de lectura de input
	// val, ok := s.ctx.GetInputValue("some_key")

	// Ejemplo de resultado
	result := contracts.StepResult{
		Status: "ok",
		Data: map[string]any{
			"executed": true,
		},
	}

	s.ctx.SetResult(s.servicePath, result)
	return nil
}

func init() {
	serviceconfig.Register("%s/%s", New%sService)
}
`, packageName, structName, structName, structName, structName, structName, structName, structName, folderPath, serviceName, structName)
}

func injectImport(filePath, folder string) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Printf("⚠️ No se pudo leer %s para inyectar import: %v\n", filePath, err)
		return
	}

	importPath := fmt.Sprintf(`"go-fiber-core/internal/services/%s"`, folder)
	fileStr := string(content)

	if strings.Contains(fileStr, importPath) {
		fmt.Printf("ℹ️ Import ya existe en %s\n", filePath)
		return
	}

	// Buscar bloque de imports con blank identifiers
	// Estrategia simple: buscar el último import y añadirlo después
	// O mejor, buscar un marcador conocido o el final del bloque import

	// Vamos a buscar la línea `_ "go-fiber-core/internal/services/` existente para añadirlo cerca
	// Si no, lo metemos en el bloque import general

	lines := strings.Split(fileStr, "\n")
	newLines := make([]string, 0, len(lines)+1)
	injected := false

	// Regex para detectar imports de servicios
	serviceImportRegex := regexp.MustCompile(`_ "go-fiber-core/internal/services/.*"`)

	lastImportIdx := -1

	for i, line := range lines {
		if serviceImportRegex.MatchString(line) {
			lastImportIdx = i
		}
	}

	if lastImportIdx != -1 {
		// Insertar después del último import de servicio encontrado
		for i, line := range lines {
			newLines = append(newLines, line)
			if i == lastImportIdx && !injected {
				newLines = append(newLines, fmt.Sprintf("\t_ %s", importPath))
				injected = true
			}
		}
	} else {
		// Fallback: intentar encontrar el cierre del import `)`
		for _, line := range lines {
			if strings.TrimSpace(line) == ")" && !injected {
				// Mirar si estamos dentro de un bloque import (simplificado)
				// Asumimos que el primer `)` que encontramos tras `import (` es el cierre
				// Pero para ser más seguros, simplemente lo añadimos al final de los imports si no encontramos otros servicios
				newLines = append(newLines, fmt.Sprintf("\t_ %s", importPath))
				injected = true
			}
			newLines = append(newLines, line)
		}
	}

	if injected {
		if err := os.WriteFile(filePath, []byte(strings.Join(newLines, "\n")), 0644); err != nil {
			fmt.Printf("❌ Error actualizando %s: %v\n", filePath, err)
		} else {
			fmt.Printf("✅ Import inyectado en %s\n", filePath)
		}
	} else {
		fmt.Printf("⚠️ No se pudo inyectar el import automáticamente en %s. Por favor agrégalo manualmente.\n", filePath)
	}
}
