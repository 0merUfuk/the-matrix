package export

import (
	"path/filepath"
	"strings"

	"github.com/0merUfuk/the-matrix/internal/config"
	"github.com/0merUfuk/the-matrix/internal/oracle"
)

// ReadResult contains the results of reading docs from a directory.
type ReadResult struct {
	Docs    []KnowledgeDoc
	Skipped []string // filenames that failed validation
	Total   int      // total .md files found (before validation)
}

// ReadDocs reads all .md files from a directory and returns a ReadResult.
// Skips _index.md. Uses config.ListMDFiles for consistency with oracle inject.
// Validates each doc using oracle.ValidateInjectDoc; invalid docs are recorded in Skipped.
func ReadDocs(dir string) (ReadResult, error) {
	mdFiles, err := config.ListMDFiles(dir)
	if err != nil {
		return ReadResult{}, err
	}

	result := ReadResult{
		Total: len(mdFiles),
	}

	for _, filename := range mdFiles {
		filePath := filepath.Join(dir, filename)
		content, err := config.ReadFileString(filePath)
		if err != nil {
			return ReadResult{}, err
		}

		// Validate using oracle's inject validation
		errs := oracle.ValidateInjectDoc(filename, content)
		if len(errs) > 0 {
			result.Skipped = append(result.Skipped, filename)
			continue
		}

		topic := strings.TrimSuffix(filename, ".md")
		title := extractTitle(topic, content)

		result.Docs = append(result.Docs, KnowledgeDoc{
			Filename: filename,
			Topic:    topic,
			Title:    title,
			Content:  content,
		})
	}

	return result, nil
}

// DetectLanguage infers the programming language from a directory name.
// "go-internal" -> "go", "nodejs-express" -> "javascript", "flutter-dart" -> "dart",
// "python-fastapi" -> "python", "typescript-nestjs" -> "typescript"
// Returns empty string if detection fails.
func DetectLanguage(dirName string) string {
	lower := strings.ToLower(dirName)

	// Check prefix-based patterns
	prefixes := map[string]string{
		"go-":         "go",
		"golang-":     "go",
		"nodejs-":     "javascript",
		"node-":       "javascript",
		"javascript-": "javascript",
		"typescript-": "typescript",
		"ts-":         "typescript",
		"python-":     "python",
		"py-":         "python",
		"flutter-":    "dart",
		"dart-":       "dart",
		"rust-":       "rust",
		"ruby-":       "ruby",
		"java-":       "java",
		"kotlin-":     "kotlin",
		"swift-":      "swift",
		"csharp-":     "csharp",
		"dotnet-":     "csharp",
	}

	for prefix, lang := range prefixes {
		if strings.HasPrefix(lower, prefix) {
			return lang
		}
	}

	// Check if the entire name is a language
	exact := map[string]string{
		"go":         "go",
		"golang":     "go",
		"nodejs":     "javascript",
		"node":       "javascript",
		"javascript": "javascript",
		"typescript": "typescript",
		"python":     "python",
		"flutter":    "dart",
		"dart":       "dart",
		"rust":       "rust",
		"ruby":       "ruby",
		"java":       "java",
		"kotlin":     "kotlin",
		"swift":      "swift",
		"csharp":     "csharp",
	}

	if lang, ok := exact[lower]; ok {
		return lang
	}

	return ""
}

// LanguageGlob returns the glob pattern for a language.
// "go" -> "**/*.go", "javascript" -> "**/*.{js,ts,jsx,tsx}", "python" -> "**/*.py"
// Empty/unknown -> "**/*"
func LanguageGlob(language string) string {
	globs := map[string]string{
		"go":         "**/*.go",
		"javascript": "**/*.{js,ts,jsx,tsx}",
		"typescript": "**/*.{ts,tsx}",
		"python":     "**/*.py",
		"dart":       "**/*.dart",
		"rust":       "**/*.rs",
		"ruby":       "**/*.rb",
		"java":       "**/*.java",
		"kotlin":     "**/*.{kt,kts}",
		"swift":      "**/*.swift",
		"csharp":     "**/*.cs",
	}

	if g, ok := globs[language]; ok {
		return g
	}
	return "**/*"
}

// StripFrontmatter removes oracle's markdown frontmatter from content.
// Strips everything from the start through the first "---" line (inclusive).
// If no "---" separator is found, returns the original content unchanged.
func StripFrontmatter(content string) string {
	lines := strings.Split(content, "\n")

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "---" {
			// Return everything after the --- line
			remaining := strings.Join(lines[i+1:], "\n")
			return strings.TrimLeft(remaining, "\n")
		}
	}

	// No frontmatter separator found — return as-is
	return content
}

// extractTitle pulls the first # heading from content, or derives from topic name.
// Falls back to topicToTitle if the heading text is empty (e.g. bare "# " line).
func extractTitle(topic, content string) string {
	// Look for first # heading (not ##, ###, etc.)
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "# ") && !strings.HasPrefix(trimmed, "## ") {
			title := strings.TrimPrefix(trimmed, "# ")
			if title != "" {
				return title
			}
		}
	}

	// Derive from topic name: "04-error-handling" -> "Error Handling"
	return topicToTitle(topic)
}

// topicToTitle converts a topic slug to a human-readable title.
// "04-error-handling" -> "Error Handling", "00-core-principles" -> "Core Principles"
func topicToTitle(topic string) string {
	// Strip leading number prefix (e.g., "04-")
	parts := strings.SplitN(topic, "-", 2)
	name := topic
	if len(parts) == 2 && isNumericPrefix(parts[0]) {
		name = parts[1]
	}

	// Convert kebab-case to Title Case
	words := strings.Split(name, "-")
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(words, " ")
}

// isNumericPrefix checks if a string is a pure number prefix like "00", "04", "99".
func isNumericPrefix(s string) bool {
	if len(s) == 0 {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}
