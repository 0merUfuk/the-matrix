package neo

import (
	"strings"
	"testing"
	"time"
)

func TestGetPreset_ValidNames(t *testing.T) {
	presetNames := []string{"internal-go-service", "go-service", "nextjs-solo"}
	for _, name := range presetNames {
		t.Run(name, func(t *testing.T) {
			profile, ok := GetPreset(name)
			if !ok {
				t.Fatalf("GetPreset(%q) returned false, want true", name)
			}
			if profile.ProjectType == "" {
				t.Errorf("ProjectType is empty")
			}
			if profile.TeamSize == "" {
				t.Errorf("TeamSize is empty")
			}
			if len(profile.Stacks) == 0 {
				t.Errorf("Stacks is empty")
			}
			// ProjectName and OutputPath must NOT be set by preset (caller fills them)
			if profile.ProjectName != "" {
				t.Errorf("ProjectName should be empty in preset template, got %q", profile.ProjectName)
			}
			if profile.OutputPath != "" {
				t.Errorf("OutputPath should be empty in preset template, got %q", profile.OutputPath)
			}
			// CreatedDate must be set to today
			today := time.Now().Format("2006-01-02")
			if profile.CreatedDate != today {
				t.Errorf("CreatedDate = %q, want %q", profile.CreatedDate, today)
			}
		})
	}
}

func TestGetPreset_InvalidName(t *testing.T) {
	_, ok := GetPreset("nonexistent-preset")
	if ok {
		t.Fatal("GetPreset(nonexistent) returned true, want false")
	}
}

func TestListPresets(t *testing.T) {
	names := ListPresets()
	if len(names) == 0 {
		t.Fatal("ListPresets() returned empty slice")
	}
	// Should include our 3 built-ins
	expected := map[string]bool{
		"internal-go-service": true,
		"go-service":          true,
		"nextjs-solo":         true,
	}
	for _, n := range names {
		if !expected[n] {
			t.Errorf("unexpected preset %q in ListPresets()", n)
		}
		delete(expected, n)
	}
	for missing := range expected {
		t.Errorf("preset %q missing from ListPresets()", missing)
	}
}

func TestListPresets_Sorted(t *testing.T) {
	names := ListPresets()
	for i := 1; i < len(names); i++ {
		if names[i] < names[i-1] {
			t.Errorf("ListPresets() not sorted: %q > %q", names[i-1], names[i])
		}
	}
}

func TestGetPreset_InternalGoService_Fields(t *testing.T) {
	profile, ok := GetPreset("internal-go-service")
	if !ok {
		t.Fatal("preset not found")
	}
	if !profile.IsInternal {
		t.Error("internal-go-service: IsInternal should be true")
	}
	if len(profile.Stacks) == 0 {
		t.Fatal("no stacks")
	}
	s := profile.Stacks[0]
	if s.Language != "go" {
		t.Errorf("Language = %q, want %q", s.Language, "go")
	}
	if !s.HasQueue {
		t.Error("internal-go-service: HasQueue should be true")
	}
	if s.DataLayer != "bun" {
		t.Errorf("DataLayer = %q, want %q", s.DataLayer, "bun")
	}
}

func TestGetPreset_NextjsSolo_Fields(t *testing.T) {
	profile, ok := GetPreset("nextjs-solo")
	if !ok {
		t.Fatal("preset not found")
	}
	if profile.IsInternal {
		t.Error("nextjs-solo: IsInternal should be false")
	}
	if profile.TeamSize != "solo" {
		t.Errorf("TeamSize = %q, want %q", profile.TeamSize, "solo")
	}
	if len(profile.Stacks) == 0 {
		t.Fatal("no stacks")
	}
	s := profile.Stacks[0]
	if s.Language != "nodejs" {
		t.Errorf("Language = %q, want %q", s.Language, "nodejs")
	}
	if s.HasQueue {
		t.Error("nextjs-solo: HasQueue should be false")
	}
}

func TestGetPreset_UnknownError(t *testing.T) {
	// runPresetFlow error message should mention valid presets
	// Test that GetPreset("bad") returns false so callers can format the error
	_, ok := GetPreset("bad-name")
	if ok {
		t.Fatal("GetPreset(bad-name) returned true")
	}
	// Verify the valid preset names are exposed via ListPresets for error messages
	names := strings.Join(ListPresets(), ", ")
	if !strings.Contains(names, "go-service") {
		t.Errorf("ListPresets() output doesn't contain 'go-service': %q", names)
	}
}

func TestGetPreset_ReturnsCopy(t *testing.T) {
	// Modifying the returned profile should not affect subsequent calls
	p1, _ := GetPreset("go-service")
	p1.ProjectName = "mutated"
	p1.Stacks[0].Name = "mutated-stack"

	p2, _ := GetPreset("go-service")
	if p2.ProjectName != "" {
		t.Error("mutation of first profile affected second call")
	}
	if p2.Stacks[0].Name != "go-service" {
		t.Errorf("stack name mutation leaked: got %q", p2.Stacks[0].Name)
	}
}

// TestGetPreset_GoService_Fields verifies that the generic go-service preset
// has the expected field values: IsInternal=false, HasQueue=false, DataLayer="none",
// Language="go", TeamSize="solo", TestingFramework="testing".
func TestGetPreset_GoService_Fields(t *testing.T) {
	profile, ok := GetPreset("go-service")
	if !ok {
		t.Fatal("go-service preset not found")
	}
	if profile.IsInternal {
		t.Error("go-service: IsInternal should be false")
	}
	if profile.TeamSize != "solo" {
		t.Errorf("TeamSize = %q, want %q", profile.TeamSize, "solo")
	}
	if profile.ProjectType != "single-app" {
		t.Errorf("ProjectType = %q, want %q", profile.ProjectType, "single-app")
	}
	if len(profile.Stacks) == 0 {
		t.Fatal("go-service: no stacks")
	}
	s := profile.Stacks[0]
	if s.Language != "go" {
		t.Errorf("Language = %q, want %q", s.Language, "go")
	}
	if s.HasQueue {
		t.Error("go-service: HasQueue should be false")
	}
	if s.DataLayer != "none" {
		t.Errorf("DataLayer = %q, want %q", s.DataLayer, "none")
	}
	if s.TestingFramework != "testing" {
		t.Errorf("TestingFramework = %q, want %q", s.TestingFramework, "testing")
	}
}

// TestListPresets_ExactCount guards against silent addition of new presets
// without corresponding test coverage. If a new preset is added to builtinPresets
// the test fails explicitly, prompting the author to update both the count and
// the field-validation test for the new preset.
func TestListPresets_ExactCount(t *testing.T) {
	const wantCount = 3
	names := ListPresets()
	if len(names) != wantCount {
		t.Errorf("ListPresets() returned %d presets, want %d — add a field-validation test for any new preset",
			len(names), wantCount)
	}
}

// TestGetPreset_CaseSensitive confirms that preset lookup is case-sensitive.
// Map keys in builtinPresets are lowercase-kebab; uppercase variants must not match.
func TestGetPreset_CaseSensitive(t *testing.T) {
	caseVariants := []string{
		"GO-SERVICE",
		"Go-Service",
		"originating project-GO-SERVICE",
		"Internal-Go-Service",
		"NEXTJS-SOLO",
	}
	for _, name := range caseVariants {
		t.Run(name, func(t *testing.T) {
			_, ok := GetPreset(name)
			if ok {
				t.Errorf("GetPreset(%q) returned true — lookup must be case-sensitive", name)
			}
		})
	}
}

// TestGetPreset_WhitespaceAndEmpty confirms that names with leading/trailing
// whitespace and the empty string do not match any preset.
func TestGetPreset_WhitespaceAndEmpty(t *testing.T) {
	names := []string{
		"",
		" go-service",
		"go-service ",
		" go-service ",
		"\tgo-service",
	}
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			_, ok := GetPreset(name)
			if ok {
				t.Errorf("GetPreset(%q) returned true — should only match exact keys", name)
			}
		})
	}
}

// TestGetPreset_CreatedDateFormat confirms that the CreatedDate returned by
// GetPreset is a valid YYYY-MM-DD string (not just today's raw string).
// This guards against accidental format changes in the time.Format call.
func TestGetPreset_CreatedDateFormat(t *testing.T) {
	for _, name := range ListPresets() {
		t.Run(name, func(t *testing.T) {
			profile, ok := GetPreset(name)
			if !ok {
				t.Fatalf("GetPreset(%q) returned false", name)
			}
			_, err := time.Parse("2006-01-02", profile.CreatedDate)
			if err != nil {
				t.Errorf("CreatedDate %q does not parse as YYYY-MM-DD: %v", profile.CreatedDate, err)
			}
		})
	}
}

// TestGetPreset_AllPresetsHaveNonEmptyStackName verifies that every preset's
// stack has a non-empty Name field. An empty Name would silently produce a
// malformed rule file path (e.g., "-services.md").
func TestGetPreset_AllPresetsHaveNonEmptyStackName(t *testing.T) {
	for _, name := range ListPresets() {
		t.Run(name, func(t *testing.T) {
			profile, ok := GetPreset(name)
			if !ok {
				t.Fatalf("GetPreset(%q) returned false", name)
			}
			for i, s := range profile.Stacks {
				if s.Name == "" {
					t.Errorf("stack[%d] has empty Name in preset %q", i, name)
				}
				if s.Language == "" {
					t.Errorf("stack[%d] has empty Language in preset %q", i, name)
				}
			}
		})
	}
}

// runPresetFlow is directly unit-tested in init_test.go. Tests here cover
// preset map correctness and stack resolution only.
