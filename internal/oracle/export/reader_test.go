package export

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ── helpers ──────────────────────────────────────────────────────────────────

func createValidDoc(t *testing.T, dir, filename string) {
	t.Helper()
	content := "**Version**: 1.0\n**Created**: 2026-03-15\n**Last Updated**: 2026-03-15\n**Authors:** Oracle CLI (test)\n\n---\n\n# Test Topic\n\n"
	content += strings.Repeat("Content line for testing.\n", 50)
	if err := os.WriteFile(filepath.Join(dir, filename), []byte(content), 0644); err != nil {
		t.Fatalf("createValidDoc: %v", err)
	}
}

func createDocWithTitle(t *testing.T, dir, filename, title string) {
	t.Helper()
	content := fmt.Sprintf("**Version**: 1.0\n**Created**: 2026-03-15\n**Last Updated**: 2026-03-15\n**Authors:** Oracle CLI (test)\n\n---\n\n# %s\n\n", title)
	content += strings.Repeat("Content line for testing.\n", 50)
	if err := os.WriteFile(filepath.Join(dir, filename), []byte(content), 0644); err != nil {
		t.Fatalf("createDocWithTitle: %v", err)
	}
}

// ── DetectLanguage ────────────────────────────────────────────────────────────

func TestDetectLanguage(t *testing.T) {
	tests := []struct {
		name     string
		dirName  string
		expected string
	}{
		// prefix-based patterns
		{"go-internal prefix", "go-internal", "go"},
		{"go-chi prefix", "go-chi", "go"},
		{"golang-service prefix", "golang-service", "go"},
		{"nodejs-express prefix", "nodejs-express", "javascript"},
		{"node-api prefix", "node-api", "javascript"},
		{"javascript-app prefix", "javascript-app", "javascript"},
		{"typescript-nestjs prefix", "typescript-nestjs", "typescript"},
		{"ts-api prefix", "ts-api", "typescript"},
		{"python-fastapi prefix", "python-fastapi", "python"},
		{"py-service prefix", "py-service", "python"},
		{"flutter-dart prefix", "flutter-dart", "dart"},
		{"dart-app prefix", "dart-app", "dart"},
		{"rust-axum prefix", "rust-axum", "rust"},
		{"ruby-rails prefix", "ruby-rails", "ruby"},
		{"java-spring prefix", "java-spring", "java"},
		{"kotlin-ktor prefix", "kotlin-ktor", "kotlin"},
		{"swift-vapor prefix", "swift-vapor", "swift"},
		{"csharp-asp prefix", "csharp-asp", "csharp"},
		{"dotnet-api prefix", "dotnet-api", "csharp"},
		// exact matches
		{"go exact", "go", "go"},
		{"golang exact", "golang", "go"},
		{"nodejs exact", "nodejs", "javascript"},
		{"node exact", "node", "javascript"},
		{"javascript exact", "javascript", "javascript"},
		{"typescript exact", "typescript", "typescript"},
		{"python exact", "python", "python"},
		{"flutter exact", "flutter", "dart"},
		{"dart exact", "dart", "dart"},
		{"rust exact", "rust", "rust"},
		{"ruby exact", "ruby", "ruby"},
		{"java exact", "java", "java"},
		{"kotlin exact", "kotlin", "kotlin"},
		{"swift exact", "swift", "swift"},
		{"csharp exact", "csharp", "csharp"},
		// case insensitive
		{"Go uppercase", "Go", "go"},
		{"GO-originating project uppercase prefix", "GO-originating project", "go"},
		// unknown
		{"unknown-stack returns empty", "unknown-stack", ""},
		{"empty string returns empty", "", ""},
		{"random word returns empty", "haskell", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectLanguage(tt.dirName)
			if got != tt.expected {
				t.Errorf("DetectLanguage(%q) = %q, want %q", tt.dirName, got, tt.expected)
			}
		})
	}
}

// ── LanguageGlob ──────────────────────────────────────────────────────────────

func TestLanguageGlob(t *testing.T) {
	tests := []struct {
		name     string
		lang     string
		expected string
	}{
		{"go", "go", "**/*.go"},
		{"javascript", "javascript", "**/*.{js,ts,jsx,tsx}"},
		{"typescript", "typescript", "**/*.{ts,tsx}"},
		{"python", "python", "**/*.py"},
		{"dart", "dart", "**/*.dart"},
		{"rust", "rust", "**/*.rs"},
		{"ruby", "ruby", "**/*.rb"},
		{"java", "java", "**/*.java"},
		{"kotlin", "kotlin", "**/*.{kt,kts}"},
		{"swift", "swift", "**/*.swift"},
		{"csharp", "csharp", "**/*.cs"},
		{"unknown falls back to wildcard", "unknown", "**/*"},
		{"empty falls back to wildcard", "", "**/*"},
		{"haskell not registered", "haskell", "**/*"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := LanguageGlob(tt.lang)
			if got != tt.expected {
				t.Errorf("LanguageGlob(%q) = %q, want %q", tt.lang, got, tt.expected)
			}
		})
	}
}

// ── StripFrontmatter ──────────────────────────────────────────────────────────

func TestStripFrontmatter(t *testing.T) {
	strPtr := func(s string) *string { return &s }

	tests := []struct {
		name     string
		input    string
		contains string  // non-empty: result must contain this
		excludes string  // non-empty: result must NOT contain this
		exact    *string // non-nil: exact match (including empty string)
	}{
		{
			name:     "oracle frontmatter stripped up to first separator",
			input:    "**Version**: 1.0\n**Created**: 2026-03-15\n**Last Updated**: 2026-03-15\n**Authors:** Oracle\n\n---\n\n# Core Principles\n\nSome content here.\n",
			excludes: "**Version**",
			contains: "Core Principles",
		},
		{
			name:     "content starts after the separator",
			input:    "Frontmatter line\n---\nBody content here\n",
			contains: "Body content here",
			excludes: "Frontmatter line",
		},
		{
			name:  "no separator returns content as-is",
			input: "No separator in this content\nJust plain text\n",
			exact: strPtr("No separator in this content\nJust plain text\n"),
		},
		{
			name:  "only separator line",
			input: "---\n",
			exact: strPtr(""),
		},
		{
			name:     "multiple separators — strips only through first",
			input:    "Frontmatter\n---\nBody\n---\nMore body\n",
			contains: "More body",
			excludes: "Frontmatter",
		},
		{
			name:  "empty content returns empty",
			input: "",
			exact: strPtr(""),
		},
		{
			name:     "separator with leading whitespace is recognized",
			input:    "Meta\n  ---  \nBody\n",
			contains: "Body",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StripFrontmatter(tt.input)

			if tt.exact != nil {
				if got != *tt.exact {
					t.Errorf("StripFrontmatter() exact mismatch\ngot:  %q\nwant: %q", got, *tt.exact)
				}
				return
			}
			if tt.contains != "" && !strings.Contains(got, tt.contains) {
				t.Errorf("StripFrontmatter() result missing %q\ngot: %q", tt.contains, got)
			}
			if tt.excludes != "" && strings.Contains(got, tt.excludes) {
				t.Errorf("StripFrontmatter() result should not contain %q\ngot: %q", tt.excludes, got)
			}
		})
	}
}

// ── extractTitle ──────────────────────────────────────────────────────────────

func TestExtractTitle(t *testing.T) {
	tests := []struct {
		name     string
		topic    string
		content  string
		expected string
	}{
		{
			name:     "extracts h1 heading",
			topic:    "04-error-handling",
			content:  "**Version**: 1.0\n\n---\n\n# Error Handling\n\nSome content.\n",
			expected: "Error Handling",
		},
		{
			name:     "ignores h2 heading and falls back to topic name",
			topic:    "04-error-handling",
			content:  "**Version**: 1.0\n\n---\n\n## Error Handling\n\nSome content.\n",
			expected: "Error Handling",
		},
		{
			name:     "no heading derives from topic name",
			topic:    "04-error-handling",
			content:  "**Version**: 1.0\n\nSome content without heading.\n",
			expected: "Error Handling",
		},
		{
			name:     "topic 00-core-principles derives correctly",
			topic:    "00-core-principles",
			content:  "No heading here.\n",
			expected: "Core Principles",
		},
		{
			name:     "topic without numeric prefix",
			topic:    "testing",
			content:  "No heading here.\n",
			expected: "Testing",
		},
		{
			name:     "topic with multi-word derives correctly",
			topic:    "99-session-completion-protocol",
			content:  "No heading.\n",
			expected: "Session Completion Protocol",
		},
		{
			name:     "bare # heading falls back to topic name",
			topic:    "04-error-handling",
			content:  "**Version**: 1.0\n\n---\n\n# \n\nSome content.\n",
			expected: "Error Handling",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractTitle(tt.topic, tt.content)
			if got != tt.expected {
				t.Errorf("extractTitle(%q, ...) = %q, want %q", tt.topic, got, tt.expected)
			}
		})
	}
}

// ── topicToTitle ──────────────────────────────────────────────────────────────

func TestTopicToTitle(t *testing.T) {
	tests := []struct {
		topic    string
		expected string
	}{
		{"04-error-handling", "Error Handling"},
		{"00-core-principles", "Core Principles"},
		{"99-session-completion-protocol", "Session Completion Protocol"},
		{"01-project-structure", "Project Structure"},
		{"testing", "Testing"},
		{"error-handling", "Error Handling"},
		{"a", "A"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.topic, func(t *testing.T) {
			got := topicToTitle(tt.topic)
			if got != tt.expected {
				t.Errorf("topicToTitle(%q) = %q, want %q", tt.topic, got, tt.expected)
			}
		})
	}
}

// ── ReadDocs ──────────────────────────────────────────────────────────────────

func TestReadDocs(t *testing.T) {
	t.Run("valid oracle docs returns KnowledgeDoc slices", func(t *testing.T) {
		dir := t.TempDir()
		createDocWithTitle(t, dir, "01-core-principles.md", "Core Principles")
		createDocWithTitle(t, dir, "02-error-handling.md", "Error Handling")

		result, err := ReadDocs(dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result.Docs) != 2 {
			t.Fatalf("expected 2 docs, got %d", len(result.Docs))
		}
		if result.Total != 2 {
			t.Errorf("expected Total=2, got %d", result.Total)
		}
		if len(result.Skipped) != 0 {
			t.Errorf("expected 0 skipped, got %d", len(result.Skipped))
		}

		// Sorted alphabetically
		if result.Docs[0].Filename != "01-core-principles.md" {
			t.Errorf("first doc should be 01-core-principles.md, got %s", result.Docs[0].Filename)
		}
		if result.Docs[1].Topic != "02-error-handling" {
			t.Errorf("second doc topic should be 02-error-handling, got %s", result.Docs[1].Topic)
		}
		if result.Docs[0].Title != "Core Principles" {
			t.Errorf("first doc title should be Core Principles, got %s", result.Docs[0].Title)
		}
		if result.Docs[0].Content == "" {
			t.Error("first doc content should not be empty")
		}
	})

	t.Run("directory with no md files returns empty result", func(t *testing.T) {
		dir := t.TempDir()

		result, err := ReadDocs(dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Docs != nil {
			t.Errorf("expected nil docs for empty dir, got %v", result.Docs)
		}
		if result.Total != 0 {
			t.Errorf("expected Total=0, got %d", result.Total)
		}
	})

	t.Run("skips _index.md", func(t *testing.T) {
		dir := t.TempDir()
		// _index.md should be skipped by ListMDFiles
		os.WriteFile(filepath.Join(dir, "_index.md"), []byte("index content"), 0644)
		createValidDoc(t, dir, "01-topic.md")

		result, err := ReadDocs(dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result.Docs) != 1 {
			t.Fatalf("expected 1 doc (skipping _index.md), got %d", len(result.Docs))
		}
		if result.Docs[0].Filename == "_index.md" {
			t.Error("_index.md should be skipped")
		}
	})

	t.Run("skips invalid docs and reports them", func(t *testing.T) {
		dir := t.TempDir()
		// Too short — will fail validation
		shortContent := "**Version**: 1.0\n**Created**: 2026-03-15\n**Authors:** Test\n\n# Short\n\nToo short.\n"
		os.WriteFile(filepath.Join(dir, "short.md"), []byte(shortContent), 0644)
		// Valid doc also in same dir
		createValidDoc(t, dir, "01-valid.md")

		result, err := ReadDocs(dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result.Docs) != 1 {
			t.Fatalf("expected 1 doc (short.md should be skipped), got %d", len(result.Docs))
		}
		if result.Docs[0].Filename != "01-valid.md" {
			t.Errorf("expected 01-valid.md, got %s", result.Docs[0].Filename)
		}
		if result.Total != 2 {
			t.Errorf("expected Total=2, got %d", result.Total)
		}
		if len(result.Skipped) != 1 {
			t.Fatalf("expected 1 skipped, got %d", len(result.Skipped))
		}
		if result.Skipped[0] != "short.md" {
			t.Errorf("expected skipped[0]='short.md', got %q", result.Skipped[0])
		}
	})

	t.Run("skips docs with EJS tags", func(t *testing.T) {
		dir := t.TempDir()
		ejsContent := "**Version**: 1.0\n**Created**: 2026-03-15\n**Authors:** Test\n\n---\n\n# EJS Doc\n\n"
		ejsContent += strings.Repeat("Line.\n", 48)
		ejsContent += "<% someEJSTag %>\n"
		os.WriteFile(filepath.Join(dir, "ejs.md"), []byte(ejsContent), 0644)
		createValidDoc(t, dir, "01-valid.md")

		result, err := ReadDocs(dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result.Docs) != 1 {
			t.Fatalf("expected 1 doc (ejs.md should be skipped), got %d", len(result.Docs))
		}
		if len(result.Skipped) != 1 {
			t.Errorf("expected 1 skipped (ejs.md), got %d", len(result.Skipped))
		}
	})

	t.Run("non-existent directory returns error", func(t *testing.T) {
		_, err := ReadDocs("/non/existent/path/abc123")
		if err == nil {
			t.Fatal("expected error for non-existent directory, got nil")
		}
	})
}
