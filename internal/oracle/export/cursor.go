package export

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/0merUfuk/the-matrix/internal/config"
)

func init() {
	Register(&cursorFormat{})
}

const cursorMaxChars = 6000

type cursorFormat struct{}

func (f *cursorFormat) Name() string        { return "cursor" }
func (f *cursorFormat) Description() string { return ".cursor/rules/ — one .mdc file per topic" }

func (f *cursorFormat) Export(docs []KnowledgeDoc, targetDir string, opts ExportOpts) error {
	rulesDir := filepath.Join(targetDir, ".cursor", "rules")
	if err := config.EnsureDir(rulesDir); err != nil {
		return fmt.Errorf("creating .cursor/rules/: %w", err)
	}

	glob := LanguageGlob(opts.Language)

	for _, doc := range docs {
		body := StripFrontmatter(doc.Content)

		// Build MDC frontmatter
		var description string
		if opts.Language != "" {
			description = fmt.Sprintf("%s patterns for %s services", doc.Title, opts.Language)
		} else {
			description = doc.Title + " patterns"
		}

		if len(body) <= cursorMaxChars {
			// Single file
			content := buildMDC(description, glob, body)
			outPath := filepath.Join(rulesDir, doc.Topic+".mdc")
			if err := os.WriteFile(outPath, []byte(content), 0644); err != nil {
				return fmt.Errorf("writing %s: %w", outPath, err)
			}
		} else {
			// Split into multiple files
			chunks := splitContent(body, cursorMaxChars)
			for i, chunk := range chunks {
				content := buildMDC(description, glob, chunk)
				suffix := fmt.Sprintf("-%d", i+1)
				outPath := filepath.Join(rulesDir, doc.Topic+suffix+".mdc")
				if err := os.WriteFile(outPath, []byte(content), 0644); err != nil {
					return fmt.Errorf("writing %s: %w", outPath, err)
				}
			}
		}
	}

	return nil
}

// buildMDC constructs a complete .mdc file with frontmatter and body.
func buildMDC(description, glob, body string) string {
	var b strings.Builder
	b.WriteString("---\n")
	fmt.Fprintf(&b, "description: %s\n", description)
	fmt.Fprintf(&b, "globs: %s\n", glob)
	b.WriteString("---\n")
	b.WriteString(body)
	if !strings.HasSuffix(body, "\n") {
		b.WriteString("\n")
	}
	return b.String()
}

// splitContent splits content into chunks of at most maxChars, breaking at line boundaries.
func splitContent(content string, maxChars int) []string {
	lines := strings.Split(content, "\n")
	var chunks []string
	var current strings.Builder

	for _, line := range lines {
		// If adding this line would exceed the limit and we have content, start a new chunk
		if current.Len() > 0 && current.Len()+len(line)+1 > maxChars {
			chunks = append(chunks, current.String())
			current.Reset()
		}
		if current.Len() > 0 {
			current.WriteString("\n")
		}
		current.WriteString(line)
	}

	if current.Len() > 0 {
		chunks = append(chunks, current.String())
	}

	return chunks
}
