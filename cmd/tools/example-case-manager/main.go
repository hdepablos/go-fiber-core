package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type exampleProcess struct {
	ProcessTypeName string
	Description     string
	RequestName     string
	RequestBody     string
}

type exampleCase struct {
	ID              string
	Title           string
	Description     string
	Seeders         []string
	ServicePackages []string
	AssetDir        string
	Processes       []exampleProcess
}

type state struct {
	ActiveCases []string `json:"active_cases"`
}

type generatedFile struct {
	Path    string
	Content string
}

type workspacePlan struct {
	Files    map[string][]byte
	Registry string
}

var allCases = []exampleCase{
	{
		ID:          "process_lifecycle_manager",
		Title:       "Process Lifecycle Manager",
		Description: "Casos base del motor: secuencial simple, batch paralelo, flujo hibrido, base order lifecycle y loan risk lifecycle.",
		Seeders:     []string{"process_lifecycle_manager"},
		ServicePackages: []string{
			"go-fiber-core/internal/services/test",
			"go-fiber-core/internal/services/test/common",
			"go-fiber-core/internal/services/test/heavy",
			"go-fiber-core/internal/services/test/loanrisk",
		},
		AssetDir: "process_lifecycle_manager",
		Processes: []exampleProcess{
			{
				ProcessTypeName: "Order process lifecycle",
				Description:     "Base process type for order lifecycle testing",
				RequestName:     "run - order-process-lifecycle",
				RequestBody: `{
  "process_type_id": {{process_type_id}},
  "sede_id": 1,
  "override_process_version_id": 0,
  "roadmap": 0,
  "input": {
    "order_id": 1001,
    "customer_id": 502,
    "amount": 125000
  }
}`,
			},
			{
				ProcessTypeName: "Case 1: Sequential Execution",
				Description:     "Validar, calcular y notificar en modo sync.",
				RequestName:     "run - case-1-sequential-execution",
				RequestBody: `{
  "process_type_id": {{process_type_id}},
  "sede_id": 1,
  "override_process_version_id": 0,
  "roadmap": 0,
  "input": {
    "age": 25,
    "email": "test@example.com"
  }
}`,
			},
			{
				ProcessTypeName: "Case 2: Parallel Batch Processing",
				Description:     "4 workers async con recursion via auto_invoke y consolidacion final.",
				RequestName:     "run - case-2-parallel-batch-processing",
				RequestBody: `{
  "process_type_id": {{process_type_id}},
  "sede_id": 1,
  "override_process_version_id": 0,
  "roadmap": 0,
  "input": {
    "partition_id": "A1",
    "last_id_processed": 0,
    "is_last_batch": false
  }
}`,
			},
			{
				ProcessTypeName: "Case 3: Hybrid Flow",
				Description:     "Validacion sync, proceso pesado async y notificacion final.",
				RequestName:     "run - case-3-hybrid-flow",
				RequestBody: `{
  "process_type_id": {{process_type_id}},
  "sede_id": 1,
  "override_process_version_id": 0,
  "roadmap": 0,
  "input": {
    "request_id": "HYB-100",
    "email": "test@example.com"
  }
}`,
			},
			{
				ProcessTypeName: "Loan risk lifecycle",
				Description:     "Pipeline de loan risk con validaciones por edad, salario y renovacion.",
				RequestName:     "run - loan-risk-lifecycle",
				RequestBody: `{
  "process_type_id": {{process_type_id}},
  "sede_id": 1,
  "override_process_version_id": 0,
  "roadmap": 0,
  "input": {
    "age": 44,
    "salary": 3100000
  }
}`,
			},
		},
	},
	{
		ID:          "test_process_scenarios",
		Title:       "Test Process Scenarios",
		Description: "Demuestra steps concurrentes y resolucion multi-sede.",
		Seeders:     []string{"test_process_scenarios"},
		ServicePackages: []string{
			"go-fiber-core/internal/services/test/steps_concurrent",
		},
		AssetDir: "test_process_scenarios",
		Processes: []exampleProcess{
			{
				ProcessTypeName: "Test Proceso de steps concurrente",
				Description:     "Cuatro steps concurrentes y uno secuencial de cierre.",
				RequestName:     "run - test-proceso-de-steps-concurrente",
				RequestBody: `{
  "process_type_id": {{process_type_id}},
  "sede_id": 1,
  "override_process_version_id": 0,
  "roadmap": 0,
  "input": {
    "trace_id": "CONCURRENT-001"
  }
}`,
			},
			{
				ProcessTypeName: "Test Multi-Sede Logic",
				Description:     "Resuelve version global o custom por sede.",
				RequestName:     "run - test-multi-sede-logic",
				RequestBody: `{
  "process_type_id": {{process_type_id}},
  "sede_id": 2,
  "override_process_version_id": 0,
  "roadmap": 0,
  "input": {
    "trace_id": "MULTISEDE-001"
  }
}`,
			},
		},
	},
	{
		ID:          "process_lifecycle_auto_invoke",
		Title:       "Process Lifecycle Auto Invoke",
		Description: "Loop simple, async y async con finalize para probar auto_invoke.",
		Seeders:     []string{"process_lifecycle_auto_invoke"},
		ServicePackages: []string{
			"go-fiber-core/internal/services/test",
		},
		AssetDir: "process_lifecycle_auto_invoke",
		Processes: []exampleProcess{
			{
				ProcessTypeName: "Test Auto Invoke Process",
				Description:     "Step unico con autoInvoke basico.",
				RequestName:     "run - test-auto-invoke-process",
				RequestBody: `{
  "process_type_id": {{process_type_id}},
  "sede_id": 1,
  "override_process_version_id": 0,
  "roadmap": 0,
  "input": {
    "last_id_processed": 0
  }
}`,
			},
			{
				ProcessTypeName: "Test Auto Invoke Process + async",
				Description:     "Auto invoke usando ASYNC y stop_condition is_last_batch.",
				RequestName:     "run - test-auto-invoke-process-async",
				RequestBody: `{
  "process_type_id": {{process_type_id}},
  "sede_id": 1,
  "override_process_version_id": 0,
  "roadmap": 0,
  "input": {
    "last_id_processed": 0
  }
}`,
			},
			{
				ProcessTypeName: "Test Auto Invoke Process + async + finalized",
				Description:     "Auto invoke async con next_step final.",
				RequestName:     "run - test-auto-invoke-process-async-finalized",
				RequestBody: `{
  "process_type_id": {{process_type_id}},
  "sede_id": 1,
  "override_process_version_id": 0,
  "roadmap": 0,
  "input": {
    "last_id_processed": 0,
    "total_processed": 0
  }
}`,
			},
		},
	},
	{
		ID:          "multi_queue_batch_one_table_process_lifecycle",
		Title:       "Multi Queue Batch One Table",
		Description: "Caso base de fan-out por lotes sobre una sola tabla con 10k registros.",
		Seeders:     []string{"multi_queue_batch_one_table_process_lifecycle"},
		ServicePackages: []string{
			"go-fiber-core/internal/services/test/mqb1t",
		},
		AssetDir: "multi_queue_batch_one_table_process_lifecycle",
		Processes: []exampleProcess{
			{
				ProcessTypeName: "MultiQueueBatchProcessorOneTableV1",
				Description:     "Organiza, despacha lotes y finaliza sobre multi_queue_batch_one_table.",
				RequestName:     "run - multi-queue-batch-one-table-v1",
				RequestBody: `{
  "process_type_id": {{process_type_id}},
  "sede_id": 1,
  "override_process_version_id": 0,
  "roadmap": 0,
  "input": {
    "batch_size": 50,
    "table": "multi_queue_batch_one_table"
  }
}`,
			},
		},
	},
	{
		ID:          "multi_queue_batch_one_table_recreate_records",
		Title:       "Multi Queue Batch One Table Recreate Records",
		Description: "Mismo proceso type que MQB1T, pero recreando 200k registros para pruebas de volumen.",
		Seeders:     []string{"multi_queue_batch_one_table_recreate_records"},
		ServicePackages: []string{
			"go-fiber-core/internal/services/test/mqb1t",
		},
		AssetDir: "multi_queue_batch_one_table_recreate_records",
		Processes: []exampleProcess{
			{
				ProcessTypeName: "MultiQueueBatchProcessorOneTableV1",
				Description:     "Recrea registros y reutiliza el proceso fan-out sobre la misma tabla.",
				RequestName:     "run - multi-queue-batch-one-table-v1-recreate-records",
				RequestBody: `{
  "process_type_id": {{process_type_id}},
  "sede_id": 1,
  "override_process_version_id": 0,
  "roadmap": 0,
  "input": {
    "batch_size": 50,
    "table": "multi_queue_batch_one_table"
  }
}`,
			},
		},
	},
}

var caseIndex = func() map[string]exampleCase {
	out := make(map[string]exampleCase, len(allCases))
	for _, c := range allCases {
		out[c.ID] = c
	}
	return out
}()

func main() {
	var (
		action  string
		caseID  string
		repo    string
		verbose bool
	)

	flag.StringVar(&action, "action", "list", "Accion: list, create, delete o recreate")
	flag.StringVar(&caseID, "case", "all", "Case id o all")
	flag.StringVar(&repo, "repo-root", ".", "Root del repositorio")
	flag.BoolVar(&verbose, "verbose", false, "Muestra detalle adicional")
	flag.Parse()

	root, err := filepath.Abs(repo)
	if err != nil {
		exitErr(err)
	}

	switch strings.TrimSpace(strings.ToLower(action)) {
	case "list":
		if err := listCases(root, verbose); err != nil {
			exitErr(err)
		}
	case "create":
		if err := mutateCases(root, caseID, addCases); err != nil {
			exitErr(err)
		}
	case "delete":
		if err := mutateCases(root, caseID, removeCases); err != nil {
			exitErr(err)
		}
	case "recreate":
		if err := mutateCases(root, caseID, recreateCases); err != nil {
			exitErr(err)
		}
	default:
		exitErr(fmt.Errorf("accion invalida: %s", action))
	}
}

func listCases(root string, verbose bool) error {
	st, err := readState(root)
	if err != nil {
		return err
	}
	active := make(map[string]bool, len(st.ActiveCases))
	for _, id := range st.ActiveCases {
		active[id] = true
	}

	fmt.Println("Example cases disponibles:")
	for _, c := range allCases {
		status := "inactive"
		if active[c.ID] {
			status = "active"
		}
		fmt.Printf("- %s [%s]\n", c.ID, status)
		fmt.Printf("  titulo: %s\n", c.Title)
		fmt.Printf("  seeder: %s\n", strings.Join(c.Seeders, ", "))
		if verbose {
			var names []string
			for _, p := range c.Processes {
				names = append(names, p.ProcessTypeName)
			}
			fmt.Printf("  process_types: %s\n", strings.Join(names, " | "))
			fmt.Printf("  descripcion: %s\n", c.Description)
			fmt.Printf("  create: make create-example-case case=%s\n", c.ID)
			fmt.Printf("  delete: make delete-example-case case=%s\n", c.ID)
			fmt.Printf("  recreate: make recreate-example-case case=%s\n", c.ID)
			fmt.Printf("  seed: make seed-example-case case=%s\n", c.ID)
		}
	}
	return nil
}

type mutator func(current map[string]bool, targets []string) error

func mutateCases(root, rawCase string, fn mutator) error {
	targets, err := resolveCases(rawCase)
	if err != nil {
		return err
	}
	st, err := readState(root)
	if err != nil {
		return err
	}
	current := make(map[string]bool, len(st.ActiveCases))
	for _, id := range st.ActiveCases {
		current[id] = true
	}
	if applyErr := fn(current, targets); applyErr != nil {
		return applyErr
	}
	next := sortedEnabledCases(current)
	plan, err := buildWorkspacePlan(root, next)
	if err != nil {
		return err
	}
	if err := applyWorkspacePlan(root, plan); err != nil {
		return err
	}
	if err := writeState(root, state{ActiveCases: next}); err != nil {
		return err
	}
	fmt.Printf("Casos activos: %s\n", strings.Join(next, ", "))
	return nil
}

func addCases(current map[string]bool, targets []string) error {
	for _, id := range targets {
		current[id] = true
	}
	return nil
}

func removeCases(current map[string]bool, targets []string) error {
	for _, id := range targets {
		delete(current, id)
	}
	return nil
}

func recreateCases(current map[string]bool, targets []string) error {
	for _, id := range targets {
		current[id] = true
	}
	return nil
}

func resolveCases(raw string) ([]string, error) {
	value := strings.TrimSpace(strings.ToLower(raw))
	if value == "" || value == "all" {
		out := make([]string, 0, len(allCases))
		for _, c := range allCases {
			out = append(out, c.ID)
		}
		return out, nil
	}
	if _, ok := caseIndex[value]; !ok {
		return nil, fmt.Errorf("case invalido: %s", raw)
	}
	return []string{value}, nil
}

func sortedEnabledCases(current map[string]bool) []string {
	var out []string
	for _, c := range allCases {
		if current[c.ID] {
			out = append(out, c.ID)
		}
	}
	return out
}

func buildWorkspacePlan(root string, activeCases []string) (workspacePlan, error) {
	files := make(map[string][]byte)
	for _, id := range activeCases {
		c := caseIndex[id]
		assetRoot := filepath.Join(root, "cmd/tools/example-case-manager/testdata/cases", c.AssetDir, "repo")
		if err := filepath.WalkDir(assetRoot, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			rel, err := filepath.Rel(assetRoot, path)
			if err != nil {
				return err
			}
			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			target := filepath.Join(root, rel)
			if prev, ok := files[target]; ok && string(prev) != string(content) {
				return fmt.Errorf("archivo compartido con contenido distinto: %s", rel)
			}
			files[target] = content
			return nil
		}); err != nil {
			return workspacePlan{}, fmt.Errorf("leer assets de %s: %w", id, err)
		}
		for _, f := range renderBrunoFiles(root, c) {
			files[f.Path] = []byte(f.Content)
		}
	}

	return workspacePlan{
		Files:    files,
		Registry: renderRegistry(activeCases),
	}, nil
}

func applyWorkspacePlan(root string, plan workspacePlan) error {
	managed, err := managedPaths(root)
	if err != nil {
		return err
	}
	desired := make(map[string]bool, len(plan.Files))
	for path, content := range plan.Files {
		desired[path] = true
		if err := writeFile(path, content); err != nil {
			return err
		}
	}
	for _, path := range managed {
		if desired[path] {
			continue
		}
		if err := removeManagedFile(root, path); err != nil {
			return err
		}
	}
	registryPath := filepath.Join(root, "internal/services/examplesregistry/zz_generated_imports.go")
	if err := writeFile(registryPath, []byte(plan.Registry)); err != nil {
		return err
	}
	return syncBrunoCaseFolders(root, plan.Files)
}

func managedPaths(root string) ([]string, error) {
	out := map[string]bool{}
	for _, c := range allCases {
		assetRoot := filepath.Join(root, "cmd/tools/example-case-manager/testdata/cases", c.AssetDir, "repo")
		err := filepath.WalkDir(assetRoot, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			rel, err := filepath.Rel(assetRoot, path)
			if err != nil {
				return err
			}
			out[filepath.Join(root, rel)] = true
			return nil
		})
		if err != nil {
			return nil, err
		}
		for _, f := range renderBrunoFiles(root, c) {
			out[f.Path] = true
		}
	}
	var paths []string
	for path := range out {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return paths, nil
}

func renderRegistry(activeCases []string) string {
	packages := map[string]bool{}
	for _, id := range activeCases {
		for _, pkg := range caseIndex[id].ServicePackages {
			packages[pkg] = true
		}
	}
	var imports []string
	for pkg := range packages {
		imports = append(imports, pkg)
	}
	sort.Strings(imports)

	var b strings.Builder
	b.WriteString("package examplesregistry\n\n")
	b.WriteString("// Code generated by example-case-manager. DO NOT EDIT.\n")
	if len(imports) == 0 {
		b.WriteString("// No active example cases.\n")
		return b.String()
	}
	b.WriteString("import (\n")
	for _, pkg := range imports {
		fmt.Fprintf(&b, "\t_ %q\n", pkg)
	}
	b.WriteString(")\n")
	return b.String()
}

func renderBrunoFiles(root string, c exampleCase) []generatedFile {
	baseDir := filepath.Join(root, "bruno/legacy/process-lifecycle/example-cases", c.ID)
	var out []generatedFile
	out = append(out, generatedFile{
		Path: filepath.Join(baseDir, "folder.bru"),
		Content: fmt.Sprintf(`meta {
  name: %s
  type: http
  seq: 1
}

auth {
  mode: inherit
}
`, c.ID),
	})
	for i, p := range c.Processes {
		out = append(out, generatedFile{
			Path: filepath.Join(baseDir, sanitizeBrunoName(p.RequestName)+".bru"),
			Content: fmt.Sprintf(`meta {
  name: %s
  type: http
  seq: %d
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
%s
}
`, p.RequestName, i+2, indentBody(p.RequestBody)),
		})
	}
	return out
}

func syncBrunoCaseFolders(root string, desired map[string][]byte) error {
	baseDir := filepath.Join(root, "bruno/legacy/process-lifecycle/example-cases")
	entries, err := os.ReadDir(baseDir)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		caseDir := filepath.Join(baseDir, entry.Name())
		keep := false
		for path := range desired {
			if strings.HasPrefix(path, caseDir+string(os.PathSeparator)) {
				keep = true
				break
			}
		}
		if keep {
			continue
		}
		if err := os.RemoveAll(caseDir); err != nil {
			return err
		}
	}
	return nil
}

func indentBody(body string) string {
	lines := strings.Split(body, "\n")
	for i, line := range lines {
		lines[i] = "  " + line
	}
	return strings.Join(lines, "\n")
}

func sanitizeBrunoName(name string) string {
	raw := strings.ToLower(name)
	var b strings.Builder
	lastHyphen := false
	for _, r := range raw {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			lastHyphen = false
		case r == '+':
			if !lastHyphen && b.Len() > 0 {
				b.WriteByte('-')
			}
			b.WriteString("plus")
			lastHyphen = false
		default:
			if !lastHyphen && b.Len() > 0 {
				b.WriteByte('-')
				lastHyphen = true
			}
		}
	}
	return strings.Trim(b.String(), "-")
}

func removeManagedFile(root, path string) error {
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	pruneEmptyDirs(root, filepath.Dir(path))
	return nil
}

func pruneEmptyDirs(root, dir string) {
	stops := map[string]bool{
		root: true,
	}
	for _, stop := range []string{
		filepath.Join(root, "internal/services/test"),
		filepath.Join(root, "bruno/legacy/process-lifecycle/example-cases"),
	} {
		stops[stop] = true
	}
	current := dir
	for current != "." && current != "/" {
		if stops[current] {
			return
		}
		entries, err := os.ReadDir(current)
		if err != nil || len(entries) > 0 {
			return
		}
		_ = os.Remove(current)
		current = filepath.Dir(current)
	}
}

func writeFile(path string, content []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, content, 0o644)
}

func readState(root string) (state, error) {
	path := filepath.Join(root, ".example-cases-state.json")
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return state{}, nil
	}
	if err != nil {
		return state{}, err
	}
	var st state
	if err := json.Unmarshal(raw, &st); err != nil {
		return state{}, err
	}
	filtered := st.ActiveCases[:0]
	for _, id := range st.ActiveCases {
		if _, ok := caseIndex[id]; ok {
			filtered = append(filtered, id)
		}
	}
	st.ActiveCases = filtered
	return st, nil
}

func writeState(root string, st state) error {
	if st.ActiveCases == nil {
		st.ActiveCases = []string{}
	}
	path := filepath.Join(root, ".example-cases-state.json")
	data, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return writeFile(path, data)
}

func exitErr(err error) {
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(1)
}
