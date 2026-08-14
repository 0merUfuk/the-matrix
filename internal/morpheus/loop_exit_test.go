package morpheus

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoopExitClassification(t *testing.T) {
	tests := []struct {
		name        string
		runtimeName string
		want        string
	}{
		{name: "claude continues after max turns", runtimeName: "claude", want: "1"},
		{name: "codex stops on exit one", runtimeName: "codex", want: "2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			if _, err := GenerateMorpheusFiles(testProjectContext(), root, tt.runtimeName); err != nil {
				t.Fatalf("GenerateMorpheusFiles(%s): %v", tt.runtimeName, err)
			}

			configPath := filepath.Join(root, ".autonomous", "config.sh")
			cmd := exec.Command("bash", "-c", `source "$1"; set +e; classify_exit 1; printf '%s' "$?"`, "bash", configPath)
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("classify exit for %s: %v\n%s", tt.runtimeName, err, output)
			}
			if got := string(output); got != tt.want {
				t.Fatalf("classify_exit(1) for %s = %q, want %q", tt.runtimeName, got, tt.want)
			}
		})
	}
}

func TestLoopExitUsesRuntimeClassifier(t *testing.T) {
	root := t.TempDir()
	if _, err := GenerateMorpheusFiles(testProjectContext(), root, "codex"); err != nil {
		t.Fatalf("GenerateMorpheusFiles(codex): %v", err)
	}

	content, err := os.ReadFile(filepath.Join(root, ".autonomous", "loop.sh"))
	if err != nil {
		t.Fatalf("read loop.sh: %v", err)
	}
	for _, want := range []string{`classify_exit "$exit_code"`, `exit_reason="RUNTIME_ERROR"`, `return 1`} {
		if !strings.Contains(string(content), want) {
			t.Errorf("loop.sh does not contain %q", want)
		}
	}
	assertFailureReturnsBeforeTag(t, string(content))
}

func assertFailureReturnsBeforeTag(t *testing.T, content string) {
	t.Helper()
	failureIndex := strings.Index(content, `FAILED with code ${exit_code}`)
	if failureIndex < 0 {
		t.Fatal("loop.sh does not contain the runtime failure branch")
	}
	failureTail := content[failureIndex:]
	returnIndex := strings.Index(failureTail, "return 1")
	tagIndex := strings.Index(failureTail, `tag_cycle "$cycle" "post-${role}"`)
	if returnIndex < 0 || tagIndex < 0 || returnIndex > tagIndex {
		t.Fatalf("loop.sh must return on runtime failure before tagging the phase")
	}
}
