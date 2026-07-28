package neo

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// ── buildProfileFromDetection ─────────────────────────────────────────────────

func TestBuildProfileFromDetection_GoProject(t *testing.T) {
	dir := makeTempProject(t, map[string]string{
		"go.mod": "module example.com/myapp\n\ngo 1.21\n\nrequire github.com/go-chi/chi/v5 v5.0.12\n",
	})

	detection := DetectionResult{
		Language:         "go",
		Framework:        "chi v5",
		ArchStyle:        "layered",
		TestingFramework: "ginkgo",
		DataLayer:        "bun",
		HasQueue:         true,
		Observability:    []string{"zerolog"},
		IsInternal:         false,
		ProjectType:      "single-app",
	}
	extraction := ProjectExtract{}

	profile := buildProfileFromDetection(dir, detection, extraction)

	if profile.ProjectName != filepath.Base(dir) {
		t.Errorf("ProjectName = %q, want %q", profile.ProjectName, filepath.Base(dir))
	}
	if profile.ProjectType != "single-app" {
		t.Errorf("ProjectType = %q, want %q", profile.ProjectType, "single-app")
	}
	if profile.TeamSize != "solo" {
		t.Errorf("TeamSize = %q, want %q", profile.TeamSize, "solo")
	}
	if profile.IsInternal != false {
		t.Error("IsInternal should be false")
	}
	if profile.OutputPath != dir {
		t.Errorf("OutputPath = %q, want %q", profile.OutputPath, dir)
	}
	if len(profile.Stacks) != 1 {
		t.Fatalf("expected 1 stack, got %d", len(profile.Stacks))
	}

	stack := profile.Stacks[0]
	if stack.Language != "go" {
		t.Errorf("stack.Language = %q, want %q", stack.Language, "go")
	}
	if stack.Framework != "chi v5" {
		t.Errorf("stack.Framework = %q, want %q", stack.Framework, "chi v5")
	}
	if stack.ArchStyle != "layered" {
		t.Errorf("stack.ArchStyle = %q, want %q", stack.ArchStyle, "layered")
	}
	if stack.TestingFramework != "ginkgo" {
		t.Errorf("stack.TestingFramework = %q, want %q", stack.TestingFramework, "ginkgo")
	}
	if !stack.HasQueue {
		t.Error("stack.HasQueue should be true")
	}
	if stack.DataLayer != "bun" {
		t.Errorf("stack.DataLayer = %q, want %q", stack.DataLayer, "bun")
	}
}

func TestBuildProfileFromDetection_StackName(t *testing.T) {
	tests := []struct {
		name          string
		language      string
		framework     string
		wantStackName string
	}{
		{
			name:          "go + chi => go-chi",
			language:      "go",
			framework:     "chi v5",
			wantStackName: "go-chi",
		},
		{
			name:          "nodejs + express => nodejs-express",
			language:      "nodejs",
			framework:     "express 4.18.0",
			wantStackName: "nodejs-express",
		},
		{
			name:          "nodejs + Next.js => nodejs-next.js",
			language:      "nodejs",
			framework:     "Next.js 14.0.0",
			wantStackName: "nodejs-next.js",
		},
		{
			name:          "go + unknown framework => go",
			language:      "go",
			framework:     "unknown",
			wantStackName: "go",
		},
		{
			name:          "go + empty framework => go",
			language:      "go",
			framework:     "",
			wantStackName: "go",
		},
		{
			name:          "python + FastAPI => python-fastapi",
			language:      "python",
			framework:     "FastAPI",
			wantStackName: "python-fastapi",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			detection := DetectionResult{
				Language:    tt.language,
				Framework:   tt.framework,
				ProjectType: "single-app",
			}
			profile := buildProfileFromDetection(dir, detection, ProjectExtract{})
			if len(profile.Stacks) != 1 {
				t.Fatalf("expected 1 stack, got %d", len(profile.Stacks))
			}
			if profile.Stacks[0].Name != tt.wantStackName {
				t.Errorf("stack.Name = %q, want %q", profile.Stacks[0].Name, tt.wantStackName)
			}
		})
	}
}

func TestBuildProfileFromDetection_DateFormat(t *testing.T) {
	dir := t.TempDir()
	detection := DetectionResult{Language: "go", ProjectType: "single-app"}
	profile := buildProfileFromDetection(dir, detection, ProjectExtract{})

	// CreatedDate must match YYYY-MM-DD format
	_, err := time.Parse("2006-01-02", profile.CreatedDate)
	if err != nil {
		t.Errorf("CreatedDate %q does not match YYYY-MM-DD format: %v", profile.CreatedDate, err)
	}
}

func TestBuildProfileFromDetection_IsInternalPropagated(t *testing.T) {
	dir := t.TempDir()
	detection := DetectionResult{
		Language:    "go",
		ProjectType: "single-app",
		IsInternal:    true,
	}
	profile := buildProfileFromDetection(dir, detection, ProjectExtract{})
	if !profile.IsInternal {
		t.Error("IsInternal should be true when detection.IsInternal is true")
	}
}

// ── padRight ──────────────────────────────────────────────────────────────────

func TestPadRight(t *testing.T) {
	tests := []struct {
		input string
		width int
		want  string
	}{
		{"foo", 6, "foo   "},
		{"foo", 3, "foo"},       // exact width — no padding
		{"foobar", 3, "foobar"}, // over width — returned as-is
		{"", 4, "    "},
		{"hello world", 5, "hello world"}, // already longer
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := padRight(tt.input, tt.width)
			if got != tt.want {
				t.Errorf("padRight(%q, %d) = %q, want %q", tt.input, tt.width, got, tt.want)
			}
		})
	}
}

// ── Integration: detect → profile → GenerateEcosystem ────────────────────────
//
// These tests exercise the full analyze pipeline without invoking RunAnalyze
// (which calls os.Exit). They verify that detection feeds correctly into the
// ecosystem generator and that actual files are produced.

func TestAnalyzePipeline_GoProject(t *testing.T) {
	// Build a realistic Go project fixture
	srcDir := makeTempProject(t, map[string]string{
		"go.mod": "module example.com/myapp\n\ngo 1.21\n\n" +
			"require (\n" +
			"\tgithub.com/go-chi/chi/v5 v5.0.12\n" +
			"\tgithub.com/uptrace/bun v1.2.0\n" +
			"\tgithub.com/onsi/ginkgo/v2 v2.15.0\n" +
			")\n",
		"Makefile": "dev:\n\tgo run ./cmd/...\n\ntest:\n\tgo test ./...\n\nbuild:\n\tgo build ./...\n",
	})
	if err := os.MkdirAll(filepath.Join(srcDir, ".github", "workflows"), 0755); err != nil {
		t.Fatal(err)
	}
	makeTempDirs(t, srcDir, "cmd", "internal")

	detection := RunDetection(srcDir)
	if detection.Language != "go" {
		t.Fatalf("expected language=go, got %q", detection.Language)
	}

	extraction := RunExtraction(srcDir)

	outDir := t.TempDir()
	profile := buildProfileFromDetection(srcDir, detection, extraction)
	profile.OutputPath = outDir

	written, err := GenerateEcosystem(&profile, outDir)
	if err != nil {
		t.Fatalf("GenerateEcosystem failed: %v", err)
	}

	if len(written) == 0 {
		t.Fatal("no files generated")
	}

	// Core files must exist
	mustExist := []string{
		"CLAUDE.md",
		".claude/SERVICE_CONTEXT.md",
		".claude/NEXT_STEPS.md",
		".claude/DECISIONS.md",
		".claude/KNOWN_ISSUES.md",
		".claude/agents/manager.md",
		".claude/agents/developer.md",
		".claude/agents/tester.md",
		".claude/agents/reviewer.md",
		".claude/agents/strategist.md",
	}
	for _, rel := range mustExist {
		if _, err := os.Stat(filepath.Join(outDir, rel)); err != nil {
			t.Errorf("expected file %q to exist: %v", rel, err)
		}
	}

	// CLAUDE.md must contain the project name derived from srcDir
	claudeContent, _ := os.ReadFile(filepath.Join(outDir, "CLAUDE.md"))
	if !strings.Contains(string(claudeContent), filepath.Base(srcDir)) {
		t.Errorf("CLAUDE.md should contain project name %q", filepath.Base(srcDir))
	}
}

func TestAnalyzePipeline_NodeJSProject(t *testing.T) {
	srcDir := makeTempProject(t, map[string]string{
		"package.json": `{
			"name": "my-api",
			"version": "1.0.0",
			"dependencies": {
				"express": "^4.18.0",
				"prisma": "^5.0.0"
			},
			"devDependencies": {
				"jest": "^29.0.0"
			},
			"scripts": {
				"dev": "node server.js",
				"test": "jest",
				"build": "tsc"
			}
		}`,
	})

	detection := RunDetection(srcDir)
	if detection.Language != "nodejs" {
		t.Fatalf("expected language=nodejs, got %q", detection.Language)
	}
	if detection.TestingFramework != "jest" {
		t.Errorf("TestingFramework = %q, want jest", detection.TestingFramework)
	}
	if detection.DataLayer != "prisma" {
		t.Errorf("DataLayer = %q, want prisma", detection.DataLayer)
	}

	extraction := RunExtraction(srcDir)
	if _, ok := extraction.BuildCommands["dev"]; !ok {
		t.Error("expected 'dev' in BuildCommands")
	}

	outDir := t.TempDir()
	profile := buildProfileFromDetection(srcDir, detection, extraction)
	profile.OutputPath = outDir

	written, err := GenerateEcosystem(&profile, outDir)
	if err != nil {
		t.Fatalf("GenerateEcosystem failed: %v", err)
	}
	if len(written) == 0 {
		t.Fatal("no files generated")
	}
}

func TestAnalyzePipeline_DryRun_NoFiles(t *testing.T) {
	// Verify that BuildNeoManifest produces entries but GenerateEcosystem
	// is NOT called when DryRun=true (simulate dry-run by checking manifest
	// without writing).
	srcDir := makeTempProject(t, map[string]string{
		"go.mod": "module example.com/app\n\ngo 1.21\n",
	})

	detection := RunDetection(srcDir)
	extraction := RunExtraction(srcDir)

	outDir := t.TempDir()
	profile := buildProfileFromDetection(srcDir, detection, extraction)
	profile.OutputPath = outDir

	// Build manifest only — dry run simulation
	manifest := BuildNeoManifest(&profile, outDir)
	if len(manifest) == 0 {
		t.Error("manifest should not be empty for a valid project")
	}

	// Verify nothing was written to outDir
	entries, err := os.ReadDir(outDir)
	if err != nil {
		t.Fatalf("reading outDir: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("dry-run simulation: expected empty outDir, found %d entries", len(entries))
	}
}

func TestAnalyzePipeline_ExistingClaudeDir_Force(t *testing.T) {
	// Simulate --force: GenerateEcosystem should overwrite existing .claude/
	srcDir := makeTempProject(t, map[string]string{
		"go.mod": "module example.com/app\n\ngo 1.21\n",
	})

	outDir := t.TempDir()

	// Pre-create a .claude/ dir with a sentinel file
	claudeDir := filepath.Join(outDir, ".claude")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatal(err)
	}
	sentinelPath := filepath.Join(claudeDir, "old-file.txt")
	if err := os.WriteFile(sentinelPath, []byte("old content"), 0644); err != nil {
		t.Fatal(err)
	}

	detection := RunDetection(srcDir)
	extraction := RunExtraction(srcDir)
	profile := buildProfileFromDetection(srcDir, detection, extraction)
	profile.OutputPath = outDir

	// GenerateEcosystem writes new files (force mode — no gate in this function)
	written, err := GenerateEcosystem(&profile, outDir)
	if err != nil {
		t.Fatalf("GenerateEcosystem failed: %v", err)
	}
	if len(written) == 0 {
		t.Fatal("no files generated")
	}

	// New files must exist
	if _, err := os.Stat(filepath.Join(outDir, "CLAUDE.md")); err != nil {
		t.Error("CLAUDE.md should exist after generation")
	}
}

func TestAnalyzePipeline_InternalProject(t *testing.T) {
	srcDir := makeTempProject(t, map[string]string{
		"go.mod":   "module example.com/svc\n\ngo 1.21\n",
		"CLAUDE.md": "# originating project Service\n\nThis is a originating project microservice.\n",
	})

	detection := RunDetection(srcDir)
	if !detection.IsInternal {
		t.Error("IsInternal should be true for originating project project")
	}

	outDir := t.TempDir()
	profile := buildProfileFromDetection(srcDir, detection, ProjectExtract{})
	profile.OutputPath = outDir

	_, err := GenerateEcosystem(&profile, outDir)
	if err != nil {
		t.Fatalf("GenerateEcosystem failed: %v", err)
	}

	// CLAUDE.md content should reflect originating project project
	claudeContent, _ := os.ReadFile(filepath.Join(outDir, "CLAUDE.md"))
	if len(claudeContent) == 0 {
		t.Error("CLAUDE.md should not be empty")
	}
}

func TestAnalyzePipeline_MonorepoProject(t *testing.T) {
	// Create a monorepo with two Go services
	srcDir := t.TempDir()
	for _, svc := range []string{"service-a", "service-b"} {
		svcDir := filepath.Join(srcDir, svc)
		if err := os.MkdirAll(svcDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(svcDir, "go.mod"), []byte("module "+svc+"\n"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	// Root has no go.mod — language will be unknown but project type is monorepo
	detection := RunDetection(srcDir)
	if detection.ProjectType != "monorepo" {
		t.Errorf("ProjectType = %q, want monorepo", detection.ProjectType)
	}
}
