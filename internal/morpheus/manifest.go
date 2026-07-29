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
			if err := os.WriteFile(entry.OutputPath, data, 0755); err != nil {
				return nil, fmt.Errorf("writing %s: %w", entry.OutputPath, err)
			}
		} else {
			data, err := morpheusFS.ReadFile("templates/" + entry.TemplatePath)
			if err != nil {
				return nil, fmt.Errorf("reading template %s: %w", entry.TemplatePath, err)
			}

			rendered, err := tmpl.Render(entry.TemplatePath, string(data), ctx)
			if err != nil {
				return nil, fmt.Errorf("template render failed for %s: %w", entry.TemplatePath, err)
			}

			if err := os.WriteFile(entry.OutputPath, []byte(rendered), 0644); err != nil {
				return nil, fmt.Errorf("writing %s: %w", entry.OutputPath, err)
			}

			if strings.HasSuffix(entry.OutputPath, ".sh") {
				_ = os.Chmod(entry.OutputPath, 0755)
			}
		}

		written = append(written, entry.OutputPath)
	}

	return written, nil
}
