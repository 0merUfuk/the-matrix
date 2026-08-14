package morpheus

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCodexRoleToolsSelectSandboxAndSearch(t *testing.T) {
	root := t.TempDir()
	if _, err := GenerateMorpheusFiles(testProjectContext(), root, "codex"); err != nil {
		t.Fatalf("GenerateMorpheusFiles(codex): %v", err)
	}

	binDir := t.TempDir()
	codexPath := filepath.Join(binDir, "codex")
	if err := os.WriteFile(codexPath, []byte("#!/bin/sh\nprintf '%s\\n' \"$@\"\n"), 0o755); err != nil {
		t.Fatalf("write codex stub: %v", err)
	}

	tests := []struct {
		name        string
		tools       string
		wantSandbox string
		wantSearch  bool
	}{
		{name: "reviewer is read only", tools: "REVIEWER_TOOLS", wantSandbox: "read-only"},
		{name: "security is read only", tools: "SECURITY_TOOLS", wantSandbox: "read-only"},
		{name: "developer can write", tools: "DEVELOPER_TOOLS", wantSandbox: "workspace-write"},
		{name: "strategist is read only with search", tools: "STRATEGIST_TOOLS", wantSandbox: "read-only", wantSearch: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outputPath := filepath.Join(t.TempDir(), "args.txt")
			script := `source "$CONFIG"; run_agent "prompt" "${` + tt.tools + `}" 1 "$OUTPUT"`
			cmd := exec.Command("bash", "-c", script)
			cmd.Env = append(os.Environ(),
				"CONFIG="+filepath.Join(root, ".autonomous", "config.sh"),
				"OUTPUT="+outputPath,
				"PATH="+binDir+":"+os.Getenv("PATH"),
			)
			if output, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("run_agent failed: %v\n%s", err, output)
			}

			output, err := os.ReadFile(outputPath)
			if err != nil {
				t.Fatalf("read captured arguments: %v", err)
			}
			args := string(output)
			if !strings.Contains(args, "--sandbox\n"+tt.wantSandbox+"\n") {
				t.Errorf("captured arguments = %q, want sandbox %q", args, tt.wantSandbox)
			}
			hasSearch := strings.Contains(args, "--search\n")
			if hasSearch != tt.wantSearch {
				t.Errorf("captured arguments search = %t, want %t; args: %q", hasSearch, tt.wantSearch, args)
			}
		})
	}
}
