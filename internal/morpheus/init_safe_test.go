package morpheus

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"
)

func TestRunInitSafeHelper(t *testing.T) {
	if os.Getenv("MORPHEUS_INIT_SAFE_HELPER") != "1" {
		return
	}

	opts := InitOpts{
		OutputDir:   os.Getenv("MORPHEUS_INIT_SAFE_OUTPUT_DIR"),
		ContextPath: os.Getenv("MORPHEUS_INIT_SAFE_CONTEXT_PATH"),
	}
	setInitForce(&opts, os.Getenv("MORPHEUS_INIT_SAFE_FORCE") == "1")
	RunInit(opts)
}

func TestInitSafeOptsExposesForce(t *testing.T) {
	if _, ok := reflect.TypeOf(InitOpts{}).FieldByName("Force"); !ok {
		t.Fatal("InitOpts must expose Force so noninteractive callers can explicitly authorize overwrite")
	}
}

func TestInitSafeRefusesExistingDirectoryWithoutForce(t *testing.T) {
	root := t.TempDir()
	outputDir := filepath.Join(root, "existing-output")
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		t.Fatalf("create output directory: %v", err)
	}
	sentinel := filepath.Join(outputDir, "keep.txt")
	if err := os.WriteFile(sentinel, []byte("keep"), 0644); err != nil {
		t.Fatalf("write sentinel: %v", err)
	}

	output, err := runInitSafeSubprocess(t, outputDir, writeInitSafeContext(t, root), false)
	if err == nil {
		t.Fatalf("RunInit exited successfully without --force for an existing directory:\n%s", output)
	}
	if _, err := os.Stat(sentinel); err != nil {
		t.Fatalf("existing directory contents were changed without --force: %v\nsubprocess output:\n%s", err, output)
	}
	if !strings.Contains(string(output), "--force") {
		t.Fatalf("RunInit did not explain how to authorize overwrite:\n%s", output)
	}
}

func TestInitSafeForceOverwritesExistingDirectoryWithoutTTY(t *testing.T) {
	root := t.TempDir()
	outputDir := filepath.Join(root, "existing-output")
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		t.Fatalf("create output directory: %v", err)
	}
	sentinel := filepath.Join(outputDir, "replace.txt")
	if err := os.WriteFile(sentinel, []byte("replace"), 0644); err != nil {
		t.Fatalf("write sentinel: %v", err)
	}

	output, err := runInitSafeSubprocess(t, outputDir, writeInitSafeContext(t, root), true)
	// ContextProfile cannot represent entities, so the existing no-TTY context
	// path reaches the known late TARGET_STATE template failure. This assertion
	// proves --force passed the overwrite gate and regenerated early artifacts.
	if err != nil && !strings.Contains(string(output), "template render failed for go/tasks/TARGET_STATE.md.tmpl") {
		t.Fatalf("RunInit with --force did not reach generation: %v\n%s", err, output)
	}
	if !strings.Contains(string(output), "Overwriting "+outputDir) {
		t.Fatalf("RunInit with --force did not authorize overwrite:\n%s", output)
	}
	if _, err := os.Stat(sentinel); !os.IsNotExist(err) {
		t.Fatalf("forced overwrite left prior file in place: %v", err)
	}
	if _, err := os.Stat(filepath.Join(outputDir, "CLAUDE.md")); err != nil {
		t.Fatalf("forced overwrite did not regenerate files: %v\n%s", err, output)
	}
}

func TestInitSafeRefusesProtectedPathsEvenWithForce(t *testing.T) {
	source, err := os.ReadFile("init.go")
	if err != nil {
		t.Fatalf("read init.go: %v", err)
	}
	guardIndex := strings.Index(string(source), "safewrite.ConfirmOverwrite")
	removeIndex := strings.Index(string(source), "os.RemoveAll(outputDir)")
	if guardIndex < 0 || guardIndex > removeIndex {
		t.Fatal("init.go lacks a safe overwrite guard before os.RemoveAll; protected-path subprocess is intentionally not invoked")
	}

	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("get home directory: %v", err)
	}
	contextPath := writeInitSafeContext(t, t.TempDir())
	for _, outputDir := range []string{"/", home} {
		t.Run(outputDir, func(t *testing.T) {
			output, err := runInitSafeSubprocess(t, outputDir, contextPath, true)
			if err == nil {
				t.Fatalf("RunInit accepted protected output path %q:\n%s", outputDir, output)
			}
			if !strings.Contains(string(output), "protected output path") {
				t.Fatalf("RunInit did not report protected output path %q:\n%s", outputDir, output)
			}
		})
	}
}

func TestInitSafeCommandWiresForceFlag(t *testing.T) {
	source, err := os.ReadFile(filepath.Join("..", "..", "cmd", "morpheus", "main.go"))
	if err != nil {
		t.Fatalf("read morpheus command: %v", err)
	}
	if !strings.Contains(string(source), "BoolVar(&force, \"force\", false") {
		t.Fatal("morpheus init must expose a --force flag")
	}
	if !regexp.MustCompile(`Force:\s+force`).Match(source) {
		t.Fatal("morpheus init must wire --force into InitOpts")
	}
}

func runInitSafeSubprocess(t *testing.T, outputDir, contextPath string, force bool) ([]byte, error) {
	t.Helper()

	cmd := exec.Command(os.Args[0], "-test.run", "^TestRunInitSafeHelper$")
	emptyPath := filepath.Join(t.TempDir(), "empty-path")
	cmd.Env = append(environmentWithoutPath(),
		"MORPHEUS_INIT_SAFE_HELPER=1",
		"MORPHEUS_INIT_SAFE_OUTPUT_DIR="+outputDir,
		"MORPHEUS_INIT_SAFE_CONTEXT_PATH="+contextPath,
		"MORPHEUS_INIT_SAFE_FORCE="+mapInitSafeForce(force),
		"PATH="+emptyPath,
	)
	return cmd.CombinedOutput()
}

func environmentWithoutPath() []string {
	env := make([]string, 0, len(os.Environ()))
	for _, entry := range os.Environ() {
		if !strings.HasPrefix(entry, "PATH=") {
			env = append(env, entry)
		}
	}
	return env
}

func mapInitSafeForce(force bool) string {
	if force {
		return "1"
	}
	return "0"
}

func setInitForce(opts *InitOpts, force bool) {
	field := reflect.ValueOf(opts).Elem().FieldByName("Force")
	if field.IsValid() && field.CanSet() && field.Kind() == reflect.Bool {
		field.SetBool(force)
	}
}

func writeInitSafeContext(t *testing.T, root string) string {
	t.Helper()

	path := filepath.Join(root, "context.json")
	content := []byte(`{"projectName":"safe-test-service","isInternal":false,"stacks":[{"language":"go"}]}`)
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatalf("write context profile: %v", err)
	}
	return path
}
