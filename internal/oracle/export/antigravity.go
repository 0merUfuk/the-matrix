package export

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/0merUfuk/the-matrix/internal/config"
)

func init() {
	Register(&antigravityFormat{})
}

// antigravityMaxChars keeps each generated rule file below Antigravity's
// 24,000-byte per-file rule cap after YAML frontmatter is prepended.
const antigravityMaxChars = 23000

type antigravityFormat struct{}

func (f *antigravityFormat) Name() string { return "antigravity" }
func (f *antigravityFormat) Description() string {
	return ".agents/rules/ — one .md rule per topic"
}

func (f *antigravityFormat) Export(docs []KnowledgeDoc, targetDir string, opts ExportOpts) error {
	rulesDir := filepath.Join(targetDir, ".agents", "rules")
	if err := config.EnsureDir(rulesDir); err != nil {
		return fmt.Errorf("creating .agents/rules/: %w", err)
	}

	for _, doc := range docs {
		body := StripFrontmatter(doc.Content)

		var description string
		if opts.Language != "" {
			description = fmt.Sprintf("%s patterns for %s services", doc.Title, opts.Language)
		} else {
			description = doc.Title + " patterns"
		}

		if len(body) <= antigravityMaxChars {
			content := buildAntigravityRule(description, body)
			outPath := filepath.Join(rulesDir, doc.Topic+".md")
			if err := os.WriteFile(outPath, []byte(content), 0644); err != nil { // #nosec G306 -- exported Antigravity rule, regular 0644 output file
				return fmt.Errorf("writing %s: %w", outPath, err)
			}
		} else {
			chunks := splitContent(body, antigravityMaxChars)
			for i, chunk := range chunks {
				content := buildAntigravityRule(description, chunk)
				suffix := fmt.Sprintf("-%d", i+1)
				outPath := filepath.Join(rulesDir, doc.Topic+suffix+".md")
				if err := os.WriteFile(outPath, []byte(content), 0644); err != nil { // #nosec G306 -- exported Antigravity rule, regular 0644 output file
					return fmt.Errorf("writing %s: %w", outPath, err)
				}
			}
		}
	}

	return nil
}

// buildAntigravityRule constructs a complete .md rule file with YAML frontmatter and body.
func buildAntigravityRule(description, body string) string {
	var b strings.Builder
	b.WriteString("---\n")
	b.WriteString("trigger: model_decision\n")
	fmt.Fprintf(&b, "description: %s\n", description)
	b.WriteString("---\n")
	b.WriteString(body)
	if !strings.HasSuffix(body, "\n") {
		b.WriteString("\n")
	}
	return b.String()
}
