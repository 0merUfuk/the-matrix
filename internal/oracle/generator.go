package oracle

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/0merUfuk/the-matrix/internal/config"
	"github.com/0merUfuk/the-matrix/internal/tmpl"
)

//go:embed templates/*
//go:embed templates/prompts/*
var templateFS embed.FS

// ManifestEntry describes a file to generate.
type ManifestEntry struct {
	TemplatePath string // relative to templates/oracle/
	OutputPath   string // absolute output path
	IsStatic     bool   // true = copy as-is, false = render template
}

// BuildFileManifest returns the 6-file manifest for oracle research workspace.
func BuildFileManifest(ctx *StackContext, outputDir string) []ManifestEntry {
	o := func(rel string) string { return filepath.Join(outputDir, rel) }

	return []ManifestEntry{
		{TemplatePath: "loop.sh", OutputPath: o(".autonomous/loop.sh"), IsStatic: true},
		{TemplatePath: "config.sh.tmpl", OutputPath: o(".autonomous/config.sh"), IsStatic: false},
		{TemplatePath: "CLAUDE.md.tmpl", OutputPath: o("CLAUDE.md"), IsStatic: false},
		{TemplatePath: "prompts/researcher.md.tmpl", OutputPath: o(".autonomous/prompts/researcher.md"), IsStatic: false},
		{TemplatePath: "prompts/reviewer.md.tmpl", OutputPath: o(".autonomous/prompts/reviewer.md"), IsStatic: false},
		{TemplatePath: "prompts/synthesizer.md.tmpl", OutputPath: o(".autonomous/prompts/synthesizer.md"), IsStatic: false},
	}
}

// GenerateFiles renders all templates and writes files to disk.
// Returns the list of written file paths.
// Gold standards must be resolved before calling this function (ctx.GoldStandards).
func GenerateFiles(ctx *StackContext, outputDir string) ([]string, error) {
	// Ensure GoldStandards is never nil (templates need a non-nil map)
	if ctx.GoldStandards == nil {
		ctx.GoldStandards = map[string]string{}
	}

	manifest := BuildFileManifest(ctx, outputDir)
	var written []string

	for _, entry := range manifest {
		if err := config.EnsureDir(filepath.Dir(entry.OutputPath)); err != nil {
			return nil, fmt.Errorf("creating directory: %w", err)
		}

		if entry.IsStatic {
			// Read from embedded FS and copy
			data, err := templateFS.ReadFile("templates/" + entry.TemplatePath)
			if err != nil {
				return nil, fmt.Errorf("reading embedded file %s: %w", entry.TemplatePath, err)
			}
			if err := os.WriteFile(entry.OutputPath, data, 0755); err != nil {
				return nil, fmt.Errorf("writing %s: %w", entry.OutputPath, err)
			}
		} else {
			// Read template from embedded FS and render
			data, err := templateFS.ReadFile("templates/" + entry.TemplatePath)
			if err != nil {
				return nil, fmt.Errorf("reading template %s: %w", entry.TemplatePath, err)
			}

			rendered, err := tmpl.Render(entry.TemplatePath, string(data), ctx)
			if err != nil {
				return nil, fmt.Errorf("EJS render failed for %s: %w", entry.TemplatePath, err)
			}

			if err := os.WriteFile(entry.OutputPath, []byte(rendered), 0644); err != nil {
				return nil, fmt.Errorf("writing %s: %w", entry.OutputPath, err)
			}

			// Make .sh files executable
			if strings.HasSuffix(entry.OutputPath, ".sh") {
				if err := os.Chmod(entry.OutputPath, 0755); err != nil {
					return nil, fmt.Errorf("chmod %s: %w", entry.OutputPath, err)
				}
			}
		}

		written = append(written, entry.OutputPath)
	}

	// Create required task directories and seed files
	seedDirs := []string{
		filepath.Join(outputDir, "tasks", "research"),
		filepath.Join(outputDir, "tasks", "review"),
		filepath.Join(outputDir, "tasks", "cycles"),
		filepath.Join(outputDir, "output", ctx.StackName),
	}
	for _, d := range seedDirs {
		if err := config.EnsureDir(d); err != nil {
			return nil, fmt.Errorf("creating %s: %w", d, err)
		}
	}

	// Write status.txt
	statusPath := filepath.Join(outputDir, "tasks", "status.txt")
	if err := os.WriteFile(statusPath, []byte("INITIALIZED\n"), 0644); err != nil {
		return nil, fmt.Errorf("writing status.txt: %w", err)
	}

	// Write .gitignore
	gitignore := "tasks/\noutput/\n.autonomous/cycles/\nnode_modules/\n*.log\n"
	gitignorePath := filepath.Join(outputDir, ".gitignore")
	if err := os.WriteFile(gitignorePath, []byte(gitignore), 0644); err != nil {
		return nil, fmt.Errorf("writing .gitignore: %w", err)
	}

	return written, nil
}

// DryRunEntry represents a manifest entry for --dry-run display.
type DryRunEntry struct {
	OutputPath   string
	IsStatic     bool
	TemplateName string
}

// BuildDryRunManifest returns manifest entries without writing — for --dry-run mode.
func BuildDryRunManifest(ctx *StackContext, outputDir string) []DryRunEntry {
	manifest := BuildFileManifest(ctx, outputDir)
	entries := make([]DryRunEntry, len(manifest))
	for i, m := range manifest {
		entries[i] = DryRunEntry{
			OutputPath:   m.OutputPath,
			IsStatic:     m.IsStatic,
			TemplateName: m.TemplatePath,
		}
	}
	return entries
}
