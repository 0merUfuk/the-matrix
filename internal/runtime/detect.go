package runtime

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/0merUfuk/the-matrix/internal/matrixcfg"
)

const defaultRuntime = "claude"

// DetectRuntime determines which agent runtime a project uses.
// Priority: project-local config, context-file presence, then Claude.
func DetectRuntime(projectDir string) string {
	if cfg, err := matrixcfg.LoadProject(projectDir); err == nil && cfg.Runtime != nil {
		if name, ok := knownRuntime(*cfg.Runtime); ok {
			return name
		}
		// Unknown or unavailable configured runtimes fall back to context-file
		// detection so an invalid local setting does not block the project.
	}

	if fileExists(filepath.Join(projectDir, "CLAUDE.md")) {
		return "claude"
	}
	if fileExists(filepath.Join(projectDir, "AGENTS.md")) {
		return "codex"
	}

	return defaultRuntime
}

func knownRuntime(name string) (string, bool) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", false
	}
	if Registry.Get(name) != nil {
		return name, true
	}
	return "", false
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
