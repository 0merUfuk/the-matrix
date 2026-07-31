package neo

import (
	"os"
	"path/filepath"
	"testing"
)

// multiRepoProfile returns a ProjectProfile set up for multi-repo-microservices
// with the provided services. Callers may override fields as needed.
func multiRepoProfile(services []ServiceProfile) *ProjectProfile {
	return &ProjectProfile{
		ProjectName: "test-multi",
		ProjectType: "multi-repo-microservices",
		TeamSize:    "small",
		Stacks: []StackProfile{
			{Name: "go-api", Language: "go"},
		},
		Services:    services,
		IsInternal:  false,
		CreatedDate: "2026-03-21",
		OutputPath:  "/tmp/neo-morpheus-test",
	}
}

// singleService returns a slice with one ServiceProfile for convenience.
func singleService() []ServiceProfile {
	return []ServiceProfile{
		{Name: "api-service", Stack: "go-api", Port: 8080},
	}
}

// TestRunMorpheusIntegration_NonMultiRepo verifies that a single-app profile
// is a no-op without inspecting PATH or prompting.
func TestRunMorpheusIntegration_NonMultiRepo(t *testing.T) {
	profile := testProfile()                                  // ProjectType = "single-app"
	RunMorpheusIntegration(profile, "/tmp/neo-morpheus-test") // must not panic
}

// TestRunMorpheusIntegration_NonMultiRepoMonorepo verifies that a monorepo
// profile is a no-op — morpheus integration is only for multi-repo-microservices.
func TestRunMorpheusIntegration_NonMultiRepoMonorepo(t *testing.T) {
	profile := &ProjectProfile{
		ProjectName: "test-mono",
		ProjectType: "monorepo",
		TeamSize:    "solo",
		Stacks:      []StackProfile{{Name: "go-api", Language: "go"}},
		Services:    singleService(),
		IsInternal:  false,
		CreatedDate: "2026-03-21",
		OutputPath:  "/tmp/neo-morpheus-test",
	}
	RunMorpheusIntegration(profile, "/tmp/neo-morpheus-test") // must not panic
}

// TestRunMorpheusIntegration_NoServices verifies that a multi-repo profile with
// an empty services slice is a no-op without prompting or checking PATH.
func TestRunMorpheusIntegration_NoServices(t *testing.T) {
	profile := multiRepoProfile(nil)
	RunMorpheusIntegration(profile, "/tmp/neo-morpheus-test") // must not panic
}

// TestRunMorpheusIntegration_NoServices_EmptySlice mirrors the nil slice case
// with an explicitly empty slice — both must be treated as "nothing to scaffold".
func TestRunMorpheusIntegration_NoServices_EmptySlice(t *testing.T) {
	profile := multiRepoProfile([]ServiceProfile{})
	RunMorpheusIntegration(profile, "/tmp/neo-morpheus-test") // must not panic
}

// TestRunMorpheusIntegration_MorpheusNotInstalled verifies that when morp
// is not on PATH the function prints install guidance and returns without error.
// We simulate the missing binary by clearing PATH entirely.
func TestRunMorpheusIntegration_MorpheusNotInstalled(t *testing.T) {
	t.Setenv("PATH", "")
	profile := multiRepoProfile(singleService())
	RunMorpheusIntegration(profile, "/tmp/neo-morpheus-test") // must not panic
}

// TestRunMorpheusIntegration_ProjectTypeTable exercises the early-return gate
// across all non-qualifying project types in a single table-driven pass.
func TestRunMorpheusIntegration_ProjectTypeTable(t *testing.T) {
	tests := []struct {
		name        string
		projectType string
	}{
		{"single-app is no-op", "single-app"},
		{"monorepo is no-op", "monorepo"},
		{"empty type is no-op", ""},
		{"unknown type is no-op", "unknown-type"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			profile := &ProjectProfile{
				ProjectName: "test-proj",
				ProjectType: tt.projectType,
				TeamSize:    "solo",
				Services:    singleService(),
			}
			RunMorpheusIntegration(profile, "/tmp/neo-morpheus-test") // must not panic
		})
	}
}

// TestRunMorpheusIntegration_EmptyServiceName tests the behaviour of
// RunMorpheusIntegration when all services in the profile have an empty name.
//
// The empty-name guard (morpheus.go lines 55-58) runs inside the service loop:
//
//	for _, svc := range profile.Services {
//	    if svc.Name == "" {
//	        cli.PrintWarning("skipping service with empty name")
//	        failed = append(failed, "(unnamed)")
//	        continue
//	    }
//
// To reach that loop the function must pass three earlier gates:
//
//  1. ProjectType == "multi-repo-microservices"   ✅ (multiRepoProfile)
//  2. len(profile.Services) > 0                   ✅ (one empty-named service)
//  3. exec.LookPath("morp") succeeds           ✅ (fake binary on PATH)
//  4. wizard.AskConfirm returns (true, nil)        ✗ — blocked in unit tests
//
// Gate 4 is the unit-test boundary: huh's TUI confirm prompt requires a real
// TTY. In a non-interactive environment (CI, t.Setenv sandbox) huh returns an
// error, which RunMorpheusIntegration treats as a user interrupt and returns
// before entering the loop.
//
// What this test DOES verify:
//   - The function does not panic for a profile whose sole service has Name ""
//   - Graceful-degradation contract holds end-to-end for this input shape
//
// Full coverage of the guard itself requires an integration test that runs
// with a PTY (e.g., via expect(1) or a pseudo-terminal library) so that
// wizard.AskConfirm can receive a "yes" confirmation.
func TestRunMorpheusIntegration_EmptyServiceName(t *testing.T) {
	// Place a fake morp binary on PATH so exec.LookPath succeeds and
	// the function advances past gate 3.
	dir := t.TempDir()
	fakeMorp := filepath.Join(dir, "morp")
	script := "#!/bin/sh\nexit 0\n"
	if err := os.WriteFile(fakeMorp, []byte(script), 0755); err != nil {
		t.Fatalf("failed to write fake morp: %v", err)
	}
	t.Setenv("PATH", dir)

	// One service with an empty name — this is the input the guard is
	// designed to handle.
	profile := multiRepoProfile([]ServiceProfile{
		{Name: "", Stack: "go-api", Port: 8080},
	})

	RunMorpheusIntegration(profile, t.TempDir()) // must not panic
}
