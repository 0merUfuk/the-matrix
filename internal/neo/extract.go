package neo

import (
	"path/filepath"
	"regexp"
	"strings"

	"github.com/0merUfuk/the-matrix/internal/config"
)

// ProjectExtract holds build, CI/CD, and existing AI context information
// extracted from a project.
type ProjectExtract struct {
	BuildCommands map[string]string // {"build": "make build", "test": "make test", "dev": "make dev"}
	HasGitHub     bool
	HasGitLab     bool
	HasCircleCI   bool
	ExistingAI    string // content of existing CLAUDE.md or .cursorrules if present
}

// RunExtraction executes all extraction functions against a project directory
// and returns the combined result.
func RunExtraction(projectPath string) ProjectExtract {
	hasGitHub, hasGitLab, hasCircleCI := extractCICD(projectPath)
	return ProjectExtract{
		BuildCommands: extractBuildCommands(projectPath),
		HasGitHub:     hasGitHub,
		HasGitLab:     hasGitLab,
		HasCircleCI:   hasCircleCI,
		ExistingAI:    extractExistingAI(projectPath),
	}
}

// extractBuildCommands parses Makefile targets and package.json scripts to
// produce a map of command-name to full invocation string.
func extractBuildCommands(projectPath string) map[string]string {
	commands := make(map[string]string)

	// Parse Makefile
	makefileContent := readFileQuiet(filepath.Join(projectPath, "Makefile"))
	if makefileContent != "" {
		targets := parseMakefileTargets(makefileContent)
		for _, t := range targets {
			commands[t] = "make " + t
		}
	}

	// Parse package.json scripts
	pkg := readPackageJSON(filepath.Join(projectPath, "package.json"))
	if pkg != nil && len(pkg.Scripts) > 0 {
		for name := range pkg.Scripts {
			// Map common npm script names to npm run commands.
			// Don't overwrite Makefile entries — Makefile takes precedence.
			switch name {
			case "start", "test", "build", "dev", "lint":
				if _, exists := commands[name]; !exists {
					commands[name] = "npm run " + name
				}
			default:
				if _, exists := commands[name]; !exists {
					commands[name] = "npm run " + name
				}
			}
		}
	}

	return commands
}

// makefileTargetRe matches potential Makefile target lines like "target:" or
// "target: dep1 dep2". Post-match filtering excludes variable assignments
// (VAR := value).
var makefileTargetRe = regexp.MustCompile(`^([a-zA-Z][a-zA-Z0-9_-]*)\s*:(.*)$`)

// parseMakefileTargets extracts target names from Makefile content.
func parseMakefileTargets(content string) []string {
	var targets []string
	seen := make(map[string]bool)

	for _, line := range strings.Split(content, "\n") {
		// Skip empty, comment, and recipe lines
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "\t") || strings.HasPrefix(line, " ") {
			continue
		}

		// Skip special targets
		if strings.HasPrefix(line, ".") {
			continue
		}

		matches := makefileTargetRe.FindStringSubmatch(line)
		if len(matches) < 3 {
			continue
		}

		// Skip variable assignments: NAME := value, NAME ::= value, NAME = value
		rest := matches[2]
		if strings.HasPrefix(rest, "=") || strings.HasPrefix(rest, ":=") {
			continue
		}

		target := matches[1]
		if !seen[target] {
			seen[target] = true
			targets = append(targets, target)
		}
	}

	return targets
}

// extractCICD checks for the presence of CI/CD configuration files/directories.
func extractCICD(projectPath string) (hasGitHub, hasGitLab, hasCircleCI bool) {
	hasGitHub = config.IsDir(filepath.Join(projectPath, ".github", "workflows"))
	hasGitLab = config.FileExists(filepath.Join(projectPath, ".gitlab-ci.yml"))
	hasCircleCI = config.IsDir(filepath.Join(projectPath, ".circleci"))
	return
}

// extractExistingAI reads the first existing AI context file found.
// Priority: CLAUDE.md > .cursorrules > AGENTS.md
func extractExistingAI(projectPath string) string {
	candidates := []string{
		"CLAUDE.md",
		".cursorrules",
		"AGENTS.md",
		".github/copilot-instructions.md",
	}

	for _, name := range candidates {
		content := readFileQuiet(filepath.Join(projectPath, name))
		if content != "" {
			return content
		}
	}
	return ""
}
