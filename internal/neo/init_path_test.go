package neo

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

// TestGenerateEcosystem_NeoJSONRecordsOutputPath is a regression test for
// P0-C2 (audit 2026-04-17): neo init --path <dir> updated the local
// outputDir variable but never synced profile.OutputPath before calling
// GenerateEcosystem, so .neo.json recorded the wizard's original path and
// `neo status` / `neo update` read the wrong directory.
//
// The underlying generator writes profile verbatim as .neo.json. This test
// asserts that when profile.OutputPath is set before GenerateEcosystem, it
// round-trips through .neo.json — any future refactor that serializes a
// stale path will fail here.
func TestGenerateEcosystem_NeoJSONRecordsOutputPath(t *testing.T) {
	dir := t.TempDir()

	profile := &ProjectProfile{
		ProjectName: "my-project",
		ProjectType: "single-app",
		TeamSize:    "solo",
		IsInternal:  false,
		CreatedDate: "2026-04-19",
		OutputPath:  dir,
		Stacks: []StackProfile{
			{
				Name:             "go-service",
				Language:         "go",
				Framework:        "chi",
				ArchStyle:        "layered",
				TestingFramework: "testify",
			},
		},
	}

	if _, err := GenerateEcosystem(profile, dir); err != nil {
		t.Fatalf("GenerateEcosystem: %v", err)
	}

	neoJSON := filepath.Join(dir, ".neo.json")
	data, err := os.ReadFile(neoJSON)
	if err != nil {
		t.Fatalf("reading .neo.json: %v", err)
	}

	var got ProjectProfile
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshalling .neo.json: %v", err)
	}

	if got.OutputPath != dir {
		t.Errorf(".neo.json outputPath = %q, want %q", got.OutputPath, dir)
	}
}

// TestRunInit_PathFlagSyncsProfileOutputPath verifies via source inspection
// that the --path override branch in RunInit now synchronises
// profile.OutputPath with the resolved absolute path before GenerateEcosystem
// runs. A runtime test would require driving the wizard TUI, so a targeted
// inspection guards against regression of the P0-C2 fix.
func TestRunInit_PathFlagSyncsProfileOutputPath(t *testing.T) {
	src, err := os.ReadFile("init.go")
	if err != nil {
		t.Fatalf("reading init.go: %v", err)
	}
	source := string(src)

	// After resolving opts.Path to an absolute path in the non-preset --path
	// override block, profile.OutputPath must be reassigned — otherwise
	// .neo.json is written with the stale path. (?s) allows . to span newlines.
	syncPattern := regexp.MustCompile(`(?s)outputDir = abs.{0,400}profile\.OutputPath = abs`)
	if !syncPattern.MatchString(source) {
		t.Errorf("init.go: expected profile.OutputPath = abs to appear shortly after outputDir = abs in the --path override block; P0-C2 fix appears to be missing or has regressed")
	}
}
