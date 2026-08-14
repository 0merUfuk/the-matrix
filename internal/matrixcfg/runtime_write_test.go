package matrixcfg

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestWriteRuntime_NewFile(t *testing.T) {
	dir := t.TempDir()

	if err := WriteRuntime(dir, "codex"); err != nil {
		t.Fatalf("WriteRuntime() error = %v", err)
	}

	path := filepath.Join(dir, ".matrix.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(data), "runtime: codex\n"; got != want {
		t.Fatalf("config = %q, want %q", got, want)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := info.Mode().Perm(), os.FileMode(0644); got != want {
		t.Errorf("mode = %o, want %o", got, want)
	}
}

func TestWriteRuntime_UpdatesExistingYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".matrix.yaml")
	original := "# project config\nmodels:\n  default: terra # keep this comment\n"
	if err := os.WriteFile(path, []byte(original), 0600); err != nil {
		t.Fatal(err)
	}

	if err := WriteRuntime(dir, " codex "); err != nil {
		t.Fatalf("WriteRuntime() error = %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	for _, preserved := range []string{"# project config", "models:", "default: terra # keep this comment", "runtime: codex"} {
		if !strings.Contains(content, preserved) {
			t.Errorf("config does not preserve %q:\n%s", preserved, content)
		}
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := info.Mode().Perm(), os.FileMode(0600); got != want {
		t.Errorf("mode = %o, want %o", got, want)
	}
}

func TestWriteRuntime_UpdatesExistingYMLWithoutShadowing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".matrix.yml")
	if err := os.WriteFile(path, []byte("neo:\n  team-size: small\n"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := WriteRuntime(dir, "claude"); err != nil {
		t.Fatalf("WriteRuntime() error = %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, ".matrix.yaml")); !os.IsNotExist(err) {
		t.Fatalf(".matrix.yaml should not be created, stat error = %v", err)
	}
	assertRuntimeValue(t, path, "claude")
}

func TestWriteRuntime_PrefersExistingYAML(t *testing.T) {
	dir := t.TempDir()
	yamlPath := filepath.Join(dir, ".matrix.yaml")
	ymlPath := filepath.Join(dir, ".matrix.yml")
	if err := os.WriteFile(yamlPath, []byte("runtime: claude\n"), 0644); err != nil {
		t.Fatal(err)
	}
	ymlOriginal := []byte("runtime: untouched\n")
	if err := os.WriteFile(ymlPath, ymlOriginal, 0644); err != nil {
		t.Fatal(err)
	}

	if err := WriteRuntime(dir, "codex"); err != nil {
		t.Fatalf("WriteRuntime() error = %v", err)
	}

	assertRuntimeValue(t, yamlPath, "codex")
	ymlData, err := os.ReadFile(ymlPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(ymlData) != string(ymlOriginal) {
		t.Fatalf(".matrix.yml changed: got %q, want %q", ymlData, ymlOriginal)
	}
}

func TestWriteRuntime_ReplacesRuntime(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".matrix.yaml")
	if err := os.WriteFile(path, []byte("runtime: claude\nneo:\n  team-size: small\n"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := WriteRuntime(dir, "codex"); err != nil {
		t.Fatalf("WriteRuntime() error = %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(data), "runtime:") != 1 {
		t.Fatalf("runtime key was duplicated:\n%s", data)
	}
	assertRuntimeValue(t, path, "codex")
}

func TestWriteRuntime_PreservesUnknownNestedKeys(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".matrix.yaml")
	original := "future-section:\n  nested:\n    enabled: true\n    values:\n      - one\n      - two\nmodels:\n  future-model: custom\n"
	if err := os.WriteFile(path, []byte(original), 0644); err != nil {
		t.Fatal(err)
	}

	if err := WriteRuntime(dir, "codex"); err != nil {
		t.Fatalf("WriteRuntime() error = %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := yaml.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	future, ok := got["future-section"].(map[string]any)
	if !ok {
		t.Fatalf("future-section was not preserved: %#v", got)
	}
	nested, ok := future["nested"].(map[string]any)
	if !ok || nested["enabled"] != true {
		t.Fatalf("nested unknown keys were not preserved: %#v", future)
	}
	values, ok := nested["values"].([]any)
	if !ok || len(values) != 2 || values[0] != "one" || values[1] != "two" {
		t.Fatalf("nested unknown list was not preserved: %#v", nested)
	}
	models, ok := got["models"].(map[string]any)
	if !ok || models["future-model"] != "custom" {
		t.Fatalf("unknown model key was not preserved: %#v", got["models"])
	}
	if got["runtime"] != "codex" {
		t.Fatalf("runtime = %#v, want codex", got["runtime"])
	}
}

func TestWriteRuntime_EmptyIsNoOp(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".matrix.yml")
	original := []byte("runtime: claude\n# unchanged\n")
	if err := os.WriteFile(path, original, 0600); err != nil {
		t.Fatal(err)
	}

	for _, runtimeName := range []string{"", "  \t\n"} {
		if err := WriteRuntime(dir, runtimeName); err != nil {
			t.Fatalf("WriteRuntime(%q) error = %v", runtimeName, err)
		}
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != string(original) {
		t.Fatalf("config changed on empty runtime: got %q, want %q", data, original)
	}
	if _, err := os.Stat(filepath.Join(dir, ".matrix.yaml")); !os.IsNotExist(err) {
		t.Fatalf(".matrix.yaml should not be created, stat error = %v", err)
	}
}

func assertRuntimeValue(t *testing.T, path, want string) {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var config map[string]any
	if err := yaml.Unmarshal(data, &config); err != nil {
		t.Fatal(err)
	}
	if got := config["runtime"]; got != want {
		t.Errorf("runtime = %#v, want %q", got, want)
	}
}
