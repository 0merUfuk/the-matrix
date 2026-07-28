package export

import (
	"testing"
)

// ── Registry / Register ───────────────────────────────────────────────────────

func TestRegistryContainsAllFormats(t *testing.T) {
	expected := []string{"agents-md", "cursor", "copilot", "windsurf"}

	for _, name := range expected {
		t.Run(name+" registered", func(t *testing.T) {
			if _, ok := Registry[name]; !ok {
				t.Errorf("format %q not found in Registry — was init() called?", name)
			}
		})
	}
}

func TestFormatNamesSorted(t *testing.T) {
	names := FormatNames()

	if len(names) == 0 {
		t.Fatal("FormatNames() returned empty slice — no formats registered")
	}

	for i := 1; i < len(names); i++ {
		if names[i] < names[i-1] {
			t.Errorf("FormatNames() not sorted: %q comes before %q", names[i-1], names[i])
		}
	}
}

func TestFormatNamesContainsExpected(t *testing.T) {
	names := FormatNames()
	nameSet := make(map[string]bool, len(names))
	for _, n := range names {
		nameSet[n] = true
	}

	expected := []string{"agents-md", "cursor", "copilot", "windsurf"}
	for _, want := range expected {
		if !nameSet[want] {
			t.Errorf("expected format %q in FormatNames(), not found. got: %v", want, names)
		}
	}
}

func TestRegisterAndRetrieve(t *testing.T) {
	// Use a unique name to avoid colliding with real formats.
	const testName = "_test-mock-format"

	mock := &mockFormat{name: testName}
	Register(mock)
	defer delete(Registry, testName) // cleanup after test

	got, ok := Registry[testName]
	if !ok {
		t.Fatalf("registered format %q not found in Registry", testName)
	}
	if got.Name() != testName {
		t.Errorf("Registry[%q].Name() = %q, want %q", testName, got.Name(), testName)
	}
}

func TestFormatInterface(t *testing.T) {
	formats := []string{"agents-md", "cursor", "copilot", "windsurf"}

	for _, name := range formats {
		t.Run(name, func(t *testing.T) {
			f, ok := Registry[name]
			if !ok {
				t.Fatalf("format %q not found", name)
			}
			if f.Name() == "" {
				t.Error("Name() must not be empty")
			}
			if f.Description() == "" {
				t.Error("Description() must not be empty")
			}
			// Name must match its registry key
			if f.Name() != name {
				t.Errorf("f.Name() = %q, want %q", f.Name(), name)
			}
		})
	}
}

// mockFormat is a minimal Format implementation for registry tests.
type mockFormat struct {
	name string
}

func (m *mockFormat) Name() string        { return m.name }
func (m *mockFormat) Description() string { return "mock format for testing" }
func (m *mockFormat) Export(_ []KnowledgeDoc, _ string, _ ExportOpts) error {
	return nil
}
