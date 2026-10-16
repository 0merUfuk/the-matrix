package morpheus

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/0merUfuk/the-matrix/internal/config"
	"github.com/0merUfuk/the-matrix/internal/tmpl"
)

// generateManifestFiles renders and writes a manifest shared by init and loop.
func generateManifestFiles(manifest []ManifestEntry, ctx any) ([]string, error) {
	var written []string

	for _, entry := range manifest {
		if err := config.EnsureDir(filepath.Dir(entry.OutputPath)); err != nil {
			return nil, fmt.Errorf("creating directory: %w", err)
		}

		if entry.IsStatic {
			data, err := morpheusFS.ReadFile("templates/" + entry.TemplatePath)
			if err != nil {
				return nil, fmt.Errorf("reading embedded file %s: %w", entry.TemplatePath, err)
			}
			if err := os.WriteFile(entry.OutputPath, data, 0755); err != nil { // #nosec G306 -- static manifest entries are the loop's shell scripts, which must be 0755
				return nil, fmt.Errorf("writing %s: %w", entry.OutputPath, err)
			}
		} else {
			data, err := morpheusFS.ReadFile("templates/" + entry.TemplatePath)
			if err != nil {
				return nil, fmt.Errorf("reading template %s: %w", entry.TemplatePath, err)
			}

			var rendered string
			partialPaths := append([]string(nil), entry.PartialPaths...)
			if entry.PartialPath != "" {
				partialPaths = append(partialPaths, entry.PartialPath)
			}
			if len(partialPaths) == 0 {
				rendered, err = tmpl.Render(entry.TemplatePath, string(data), ctx)
			} else {
				partials := make(map[string]string, len(partialPaths))
				for _, partialPath := range partialPaths {
					partial, readErr := morpheusFS.ReadFile("templates/" + partialPath)
					if readErr != nil {
						return nil, fmt.Errorf("reading template partial %s: %w", partialPath, readErr)
					}
					partialName := strings.TrimSuffix(partialPath, ".tmpl")
					partials[partialName] = string(partial)
				}
				rendered, err = tmpl.RenderWithPartials(
					entry.TemplatePath,
					string(data),
					partials,
					ctx,
				)
			}
			if err != nil {
				return nil, fmt.Errorf("template render failed for %s: %w", entry.TemplatePath, err)
			}

			if err := os.WriteFile(entry.OutputPath, []byte(rendered), 0644); err != nil { // #nosec G306 -- rendered template, regular 0644 output file
				return nil, fmt.Errorf("writing %s: %w", entry.OutputPath, err)
			}

			if strings.HasSuffix(entry.OutputPath, ".sh") {
				_ = os.Chmod(entry.OutputPath, 0755) // #nosec G302 -- rendered .sh scripts must be executable (0755)
			}
		}

		written = append(written, entry.OutputPath)
	}

	return written, nil
}
