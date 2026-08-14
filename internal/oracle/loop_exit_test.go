package oracle

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoopExitCodexFailureStopsBeforeCompletion(t *testing.T) {
	root := t.TempDir()
	if _, err := GenerateFiles(testStackContext(), root, "codex"); err != nil {
		t.Fatalf("GenerateFiles(codex): %v", err)
	}

	binDir := t.TempDir()
	codexPath := filepath.Join(binDir, "codex")
	if err := os.WriteFile(codexPath, []byte("#!/bin/sh\nexit 1\n"), 0755); err != nil {
		t.Fatalf("write fake codex: %v", err)
	}

	cmd := exec.Command("bash", filepath.Join(root, ".autonomous", "loop.sh"))
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "CODEX_API_KEY=test", "PATH="+binDir+":"+os.Getenv("PATH"))
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("loop succeeded after Codex failure\n%s", output)
	}
	if !strings.Contains(string(output), "FAILED with code 1") {
		t.Fatalf("loop did not report runtime failure\n%s", output)
	}

	status, readErr := os.ReadFile(filepath.Join(root, "tasks", "status.txt"))
	if readErr != nil {
		t.Fatalf("read status: %v", readErr)
	}
	if got := strings.TrimSpace(string(status)); got != "RUNTIME_ERROR" {
		t.Fatalf("status = %q, want RUNTIME_ERROR", got)
	}
}

func TestLoopExitOracleScriptsGuardCompletion(t *testing.T) {
	for _, name := range []string{"loop.sh", "update.sh"} {
		content, err := templateFS.ReadFile("templates/" + name)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		for _, want := range []string{`classify_exit "$exit_code"`, `echo "RUNTIME_ERROR"`, `if ! run_phase`} {
			if !strings.Contains(string(content), want) {
				t.Errorf("%s does not contain %q", name, want)
			}
		}
		if name == "loop.sh" {
			assertOracleFailureReturnsBeforeTag(t, string(content))
		}
	}
}

func assertOracleFailureReturnsBeforeTag(t *testing.T, content string) {
	t.Helper()
	failureIndex := strings.Index(content, `FAILED with code ${exit_code}`)
	if failureIndex < 0 {
		t.Fatal("loop.sh does not contain the runtime failure branch")
	}
	failureTail := content[failureIndex:]
	returnIndex := strings.Index(failureTail, "return 1")
	tagIndex := strings.Index(failureTail, `tag_cycle "$cycle" "post-${role}"`)
	if returnIndex < 0 || tagIndex < 0 || returnIndex > tagIndex {
		t.Fatal("loop.sh must return on runtime failure before tagging the phase")
	}
}
