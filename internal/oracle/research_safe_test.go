package oracle

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestResearchSafeHelper(t *testing.T) {
	if os.Getenv("ORACLE_RESEARCH_SAFE_HELPER") != "1" {
		return
	}

	RunResearch(ResearchOpts{
		ConfigFile: os.Getenv("ORACLE_RESEARCH_SAFE_CONFIG"),
		OutputDir:  os.Getenv("ORACLE_RESEARCH_SAFE_OUTPUT"),
		DryRun:     os.Getenv("ORACLE_RESEARCH_SAFE_DRY_RUN") == "1",
	})
}

func TestResearchSafeEmptyConfigOutputPath(t *testing.T) {
	workDir := t.TempDir()
	configPath := filepath.Join(t.TempDir(), "stack-context.json")
	writeResearchSafeConfig(t, configPath, "")

	sentinel := filepath.Join(workDir, "keep.txt")
	if err := os.WriteFile(sentinel, []byte("keep"), 0644); err != nil {
		t.Fatal(err)
	}

	output, err := runResearchSafeSubprocess(t, workDir, configPath, "", false)
	if _, statErr := os.Stat(sentinel); statErr != nil {
		t.Fatalf("empty output path removed the subprocess cwd sentinel: %v\noutput:\n%s", statErr, output)
	}
	if err == nil {
		t.Fatalf("expected empty output path to be refused\noutput:\n%s", output)
	}
	if !strings.Contains(string(output), "output directory is required") {
		t.Fatalf("expected empty output path error, got:\n%s", output)
	}
}

func TestResearchSafeExistingDirNoForceNonTTY(t *testing.T) {
	outputDir := filepath.Join(t.TempDir(), "workspace")
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		t.Fatal(err)
	}
	sentinel := filepath.Join(outputDir, "keep.txt")
	if err := os.WriteFile(sentinel, []byte("keep"), 0644); err != nil {
		t.Fatal(err)
	}

	configPath := filepath.Join(t.TempDir(), "stack-context.json")
	writeResearchSafeConfig(t, configPath, outputDir)

	output, err := runResearchSafeSubprocess(t, t.TempDir(), configPath, "", false)
	if _, statErr := os.Stat(sentinel); statErr != nil {
		t.Fatalf("noninteractive no-force run removed existing content: %v\noutput:\n%s", statErr, output)
	}
	if err == nil {
		t.Fatalf("expected existing directory to be refused without --force\noutput:\n%s", output)
	}
	if !strings.Contains(string(output), "stdin is not a TTY") {
		t.Fatalf("expected noninteractive overwrite error, got:\n%s", output)
	}
}

func TestResearchSafeExistingDirForceNonTTY(t *testing.T) {
	outputDir := filepath.Join(t.TempDir(), "workspace")
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		t.Fatal(err)
	}
	sentinel := filepath.Join(outputDir, "keep.txt")
	if err := os.WriteFile(sentinel, []byte("keep"), 0644); err != nil {
		t.Fatal(err)
	}

	configPath := filepath.Join(t.TempDir(), "stack-context.json")
	writeResearchSafeConfig(t, configPath, outputDir)

	cmd := exec.Command("go", "run", "./cmd/oracle", "research", "--config", configPath, "--output-dir", outputDir, "--force")
	cmd.Dir = researchSafeRepoRoot(t)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("expected --force to overwrite in noninteractive mode: %v\noutput:\n%s", err, output)
	}
	if _, statErr := os.Stat(sentinel); !os.IsNotExist(statErr) {
		t.Fatalf("expected --force overwrite to remove existing content, stat error: %v\noutput:\n%s", statErr, output)
	}
	if _, statErr := os.Stat(filepath.Join(outputDir, "CLAUDE.md")); statErr != nil {
		t.Fatalf("expected --force overwrite to regenerate workspace: %v\noutput:\n%s", statErr, output)
	}
}

func TestResearchSafeRootRefused(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "stack-context.json")
	writeResearchSafeConfig(t, configPath, string(filepath.Separator))

	output, err := runResearchSafeSubprocess(t, t.TempDir(), configPath, "", true)
	if err == nil {
		t.Fatalf("expected root output path to be refused\noutput:\n%s", output)
	}
	if !strings.Contains(string(output), "protected output path") {
		t.Fatalf("expected protected output path error, got:\n%s", output)
	}
}

func runResearchSafeSubprocess(t *testing.T, dir, configPath, outputDir string, dryRun bool) ([]byte, error) {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run", "^TestResearchSafeHelper$")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"ORACLE_RESEARCH_SAFE_HELPER=1",
		"ORACLE_RESEARCH_SAFE_CONFIG="+configPath,
		"ORACLE_RESEARCH_SAFE_OUTPUT="+outputDir,
		fmt.Sprintf("ORACLE_RESEARCH_SAFE_DRY_RUN=%d", boolToInt(dryRun)),
	)
	return cmd.CombinedOutput()
}

func writeResearchSafeConfig(t *testing.T, path, outputPath string) {
	t.Helper()
	data, err := json.Marshal(StackContext{
		StackName:        "test-stack",
		FrameworkVersion: "Test Framework 1.0",
		AppType:          "api-only",
		ArchStyle:        "layered",
		StateApproach:    "n/a",
		DataLayer:        "none",
		TestingFramework: "Go testing",
		SelectedTopics:   []string{"00-core-principles"},
		TopicCount:       1,
		OutputPath:       outputPath,
		CreatedDate:      "2026-07-29",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
}

func researchSafeRepoRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	return root
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
