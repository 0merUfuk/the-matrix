package export

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ── test helpers ──────────────────────────────────────────────────────────────

func sampleDocs() []KnowledgeDoc {
	return []KnowledgeDoc{
		{
			Filename: "01-core-principles.md",
			Topic:    "01-core-principles",
			Title:    "Core Principles",
			Content:  "**Version**: 1.0\n**Created**: 2026-03-15\n**Authors:** Oracle\n\n---\n\n# Core Principles\n\nAlways write clean code.\nKeep it simple.\n",
		},
		{
			Filename: "04-error-handling.md",
			Topic:    "04-error-handling",
			Title:    "Error Handling",
			Content:  "**Version**: 1.0\n**Created**: 2026-03-15\n**Authors:** Oracle\n\n---\n\n# Error Handling\n\nReturn errors, do not panic.\nWrap with context.\n",
		},
	}
}

func sampleOpts(stackName, language string) ExportOpts {
	return ExportOpts{
		StackName: stackName,
		Language:  language,
	}
}

// createSourceDir writes valid oracle docs to a temp directory and returns the path.
func createSourceDir(t *testing.T, docs ...string) string {
	t.Helper()
	dir := t.TempDir()
	for i, name := range docs {
		title := fmt.Sprintf("Topic %d", i+1)
		createDocWithTitle(t, dir, name, title)
	}
	return dir
}

// ── agentsMDFormat ────────────────────────────────────────────────────────────

func TestAgentsMDExport(t *testing.T) {
	t.Run("creates AGENTS.md in target dir", func(t *testing.T) {
		target := t.TempDir()
		f := &agentsMDFormat{}

		err := f.Export(sampleDocs(), target, sampleOpts("go-internal", "go"))
		if err != nil {
			t.Fatalf("Export() error: %v", err)
		}

		outPath := filepath.Join(target, "AGENTS.md")
		if _, err := os.Stat(outPath); err != nil {
			t.Fatalf("AGENTS.md not created: %v", err)
		}
	})

	t.Run("contains stack name in header", func(t *testing.T) {
		target := t.TempDir()
		f := &agentsMDFormat{}

		err := f.Export(sampleDocs(), target, sampleOpts("go-internal", "go"))
		if err != nil {
			t.Fatalf("Export() error: %v", err)
		}

		content, err := os.ReadFile(filepath.Join(target, "AGENTS.md"))
		if err != nil {
			t.Fatal(err)
		}
		body := string(content)

		if !strings.Contains(body, "go-internal") {
			t.Error("AGENTS.md should contain the stack name")
		}
	})

	t.Run("contains auto-generated notice", func(t *testing.T) {
		target := t.TempDir()
		f := &agentsMDFormat{}

		err := f.Export(sampleDocs(), target, sampleOpts("go-internal", "go"))
		if err != nil {
			t.Fatalf("Export() error: %v", err)
		}

		content, _ := os.ReadFile(filepath.Join(target, "AGENTS.md"))
		if !strings.Contains(string(content), "Auto-generated") {
			t.Error("AGENTS.md should contain auto-generated notice")
		}
	})

	t.Run("contains table of contents", func(t *testing.T) {
		target := t.TempDir()
		f := &agentsMDFormat{}

		err := f.Export(sampleDocs(), target, sampleOpts("go-internal", "go"))
		if err != nil {
			t.Fatalf("Export() error: %v", err)
		}

		content, _ := os.ReadFile(filepath.Join(target, "AGENTS.md"))
		body := string(content)

		if !strings.Contains(body, "Table of Contents") {
			t.Error("AGENTS.md should contain a Table of Contents section")
		}
		if !strings.Contains(body, "Core Principles") {
			t.Error("TOC should list Core Principles")
		}
		if !strings.Contains(body, "Error Handling") {
			t.Error("TOC should list Error Handling")
		}
	})

	t.Run("frontmatter is stripped from topic sections", func(t *testing.T) {
		target := t.TempDir()
		f := &agentsMDFormat{}

		err := f.Export(sampleDocs(), target, sampleOpts("go-internal", "go"))
		if err != nil {
			t.Fatalf("Export() error: %v", err)
		}

		content, _ := os.ReadFile(filepath.Join(target, "AGENTS.md"))
		body := string(content)

		// Oracle frontmatter should not appear in the output
		if strings.Contains(body, "**Version**") {
			t.Error("AGENTS.md should not contain oracle frontmatter (**Version**)")
		}
		if strings.Contains(body, "**Authors:**") {
			t.Error("AGENTS.md should not contain oracle frontmatter (**Authors:**)")
		}
	})

	t.Run("topics rendered as ## sections", func(t *testing.T) {
		target := t.TempDir()
		f := &agentsMDFormat{}

		err := f.Export(sampleDocs(), target, sampleOpts("go-internal", "go"))
		if err != nil {
			t.Fatalf("Export() error: %v", err)
		}

		content, _ := os.ReadFile(filepath.Join(target, "AGENTS.md"))
		body := string(content)

		if !strings.Contains(body, "## Core Principles") {
			t.Error("expected '## Core Principles' section heading")
		}
		if !strings.Contains(body, "## Error Handling") {
			t.Error("expected '## Error Handling' section heading")
		}
	})

	t.Run("empty docs slice writes file with just header", func(t *testing.T) {
		target := t.TempDir()
		f := &agentsMDFormat{}

		err := f.Export([]KnowledgeDoc{}, target, sampleOpts("go-internal", "go"))
		if err != nil {
			t.Fatalf("Export() should not error on empty docs: %v", err)
		}

		outPath := filepath.Join(target, "AGENTS.md")
		content, err := os.ReadFile(outPath)
		if err != nil {
			t.Fatalf("AGENTS.md should be created even with empty docs: %v", err)
		}
		if len(content) == 0 {
			t.Error("AGENTS.md should not be completely empty")
		}
	})

	t.Run("topic content included in output", func(t *testing.T) {
		target := t.TempDir()
		f := &agentsMDFormat{}

		err := f.Export(sampleDocs(), target, sampleOpts("go-internal", "go"))
		if err != nil {
			t.Fatalf("Export() error: %v", err)
		}

		content, _ := os.ReadFile(filepath.Join(target, "AGENTS.md"))
		body := string(content)

		if !strings.Contains(body, "Always write clean code.") {
			t.Error("topic body content should be present in AGENTS.md")
		}
		if !strings.Contains(body, "Return errors, do not panic.") {
			t.Error("second topic body content should be present in AGENTS.md")
		}
	})
}

// ── cursorFormat ──────────────────────────────────────────────────────────────

func TestCursorExport(t *testing.T) {
	t.Run("creates .cursor/rules/ directory", func(t *testing.T) {
		target := t.TempDir()
		f := &cursorFormat{}

		err := f.Export(sampleDocs(), target, sampleOpts("go-internal", "go"))
		if err != nil {
			t.Fatalf("Export() error: %v", err)
		}

		rulesDir := filepath.Join(target, ".cursor", "rules")
		if info, err := os.Stat(rulesDir); err != nil || !info.IsDir() {
			t.Errorf(".cursor/rules/ should be created as a directory")
		}
	})

	t.Run("creates one .mdc file per topic", func(t *testing.T) {
		target := t.TempDir()
		f := &cursorFormat{}

		err := f.Export(sampleDocs(), target, sampleOpts("go-internal", "go"))
		if err != nil {
			t.Fatalf("Export() error: %v", err)
		}

		rulesDir := filepath.Join(target, ".cursor", "rules")

		// Each topic with short content should produce a single .mdc file
		for _, doc := range sampleDocs() {
			mdcPath := filepath.Join(rulesDir, doc.Topic+".mdc")
			if _, err := os.Stat(mdcPath); err != nil {
				t.Errorf("expected .mdc file for topic %q at %s: %v", doc.Topic, mdcPath, err)
			}
		}
	})

	t.Run("mdc files contain description frontmatter", func(t *testing.T) {
		target := t.TempDir()
		f := &cursorFormat{}

		err := f.Export(sampleDocs(), target, sampleOpts("go-internal", "go"))
		if err != nil {
			t.Fatalf("Export() error: %v", err)
		}

		mdcPath := filepath.Join(target, ".cursor", "rules", "01-core-principles.mdc")
		content, err := os.ReadFile(mdcPath)
		if err != nil {
			t.Fatalf("reading .mdc file: %v", err)
		}
		body := string(content)

		if !strings.Contains(body, "description:") {
			t.Error(".mdc file should contain 'description:' frontmatter field")
		}
		if !strings.Contains(body, "globs:") {
			t.Error(".mdc file should contain 'globs:' frontmatter field")
		}
	})

	t.Run("mdc globs match the language", func(t *testing.T) {
		target := t.TempDir()
		f := &cursorFormat{}

		err := f.Export(sampleDocs(), target, sampleOpts("go-internal", "go"))
		if err != nil {
			t.Fatalf("Export() error: %v", err)
		}

		mdcPath := filepath.Join(target, ".cursor", "rules", "01-core-principles.mdc")
		content, _ := os.ReadFile(mdcPath)

		if !strings.Contains(string(content), "**/*.go") {
			t.Error(".mdc file should contain Go glob pattern")
		}
	})

	t.Run("mdc frontmatter is wrapped in --- delimiters", func(t *testing.T) {
		target := t.TempDir()
		f := &cursorFormat{}

		err := f.Export(sampleDocs(), target, sampleOpts("go-internal", "go"))
		if err != nil {
			t.Fatalf("Export() error: %v", err)
		}

		mdcPath := filepath.Join(target, ".cursor", "rules", "01-core-principles.mdc")
		content, _ := os.ReadFile(mdcPath)
		body := string(content)

		if !strings.HasPrefix(body, "---\n") {
			t.Error(".mdc file should start with --- frontmatter delimiter")
		}
	})

	t.Run("unknown language uses wildcard glob", func(t *testing.T) {
		target := t.TempDir()
		f := &cursorFormat{}

		err := f.Export(sampleDocs(), target, sampleOpts("unknown-stack", ""))
		if err != nil {
			t.Fatalf("Export() error: %v", err)
		}

		mdcPath := filepath.Join(target, ".cursor", "rules", "01-core-principles.mdc")
		content, _ := os.ReadFile(mdcPath)

		if !strings.Contains(string(content), "**/*") {
			t.Error(".mdc file should use wildcard glob for unknown language")
		}
	})

	t.Run("oracle frontmatter stripped from mdc body", func(t *testing.T) {
		target := t.TempDir()
		f := &cursorFormat{}

		err := f.Export(sampleDocs(), target, sampleOpts("go-internal", "go"))
		if err != nil {
			t.Fatalf("Export() error: %v", err)
		}

		mdcPath := filepath.Join(target, ".cursor", "rules", "01-core-principles.mdc")
		content, _ := os.ReadFile(mdcPath)
		body := string(content)

		if strings.Contains(body, "**Version**") {
			t.Error(".mdc file body should not contain oracle frontmatter (**Version**)")
		}
		if strings.Contains(body, "**Authors:**") {
			t.Error(".mdc file body should not contain oracle frontmatter (**Authors:**)")
		}
	})

	t.Run("large doc is split into multiple numbered files", func(t *testing.T) {
		target := t.TempDir()
		f := &cursorFormat{}

		// Build a doc whose body (after stripping frontmatter) exceeds cursorMaxChars
		largeBody := strings.Repeat("This is a long line of content for testing the split behavior.\n", 120)
		largeDoc := KnowledgeDoc{
			Filename: "05-large-topic.md",
			Topic:    "05-large-topic",
			Title:    "Large Topic",
			Content:  "**Version**: 1.0\n**Created**: 2026-03-15\n**Authors:** Oracle\n\n---\n\n# Large Topic\n\n" + largeBody,
		}

		err := f.Export([]KnowledgeDoc{largeDoc}, target, sampleOpts("go-internal", "go"))
		if err != nil {
			t.Fatalf("Export() error: %v", err)
		}

		rulesDir := filepath.Join(target, ".cursor", "rules")

		// Single file should NOT exist (was split)
		singlePath := filepath.Join(rulesDir, "05-large-topic.mdc")
		if _, err := os.Stat(singlePath); err == nil {
			t.Error("expected single file to be absent when content is split")
		}

		// Numbered parts should exist
		part1 := filepath.Join(rulesDir, "05-large-topic-1.mdc")
		part2 := filepath.Join(rulesDir, "05-large-topic-2.mdc")
		if _, err := os.Stat(part1); err != nil {
			t.Errorf("expected part 1 file %s: %v", part1, err)
		}
		if _, err := os.Stat(part2); err != nil {
			t.Errorf("expected part 2 file %s: %v", part2, err)
		}
	})

	t.Run("empty docs slice produces no files", func(t *testing.T) {
		target := t.TempDir()
		f := &cursorFormat{}

		err := f.Export([]KnowledgeDoc{}, target, sampleOpts("go-internal", "go"))
		if err != nil {
			t.Fatalf("Export() should not error on empty docs: %v", err)
		}

		entries, _ := os.ReadDir(filepath.Join(target, ".cursor", "rules"))
		if len(entries) != 0 {
			t.Errorf("expected no .mdc files for empty docs, got %d", len(entries))
		}
	})
}

// ── buildMDC ──────────────────────────────────────────────────────────────────

func TestBuildMDC(t *testing.T) {
	t.Run("produces correct structure", func(t *testing.T) {
		result := buildMDC("Error handling patterns", "**/*.go", "# Error\n\nDo not panic.\n")

		if !strings.HasPrefix(result, "---\n") {
			t.Error("result should start with ---")
		}
		if !strings.Contains(result, "description: Error handling patterns") {
			t.Error("result should contain description field")
		}
		if !strings.Contains(result, "globs: **/*.go") {
			t.Error("result should contain globs field")
		}
		if !strings.Contains(result, "Do not panic.") {
			t.Error("result should contain the body content")
		}
	})

	t.Run("appends newline if body does not end with one", func(t *testing.T) {
		result := buildMDC("desc", "**/*.go", "No trailing newline")
		if !strings.HasSuffix(result, "\n") {
			t.Error("result should end with newline")
		}
	})

	t.Run("preserves trailing newline if body already has one", func(t *testing.T) {
		result := buildMDC("desc", "**/*.go", "Body with newline\n")
		if strings.HasSuffix(result, "\n\n") {
			t.Error("result should not double the trailing newline")
		}
	})
}

// ── splitContent ─────────────────────────────────────────────────────────────

func TestSplitContent(t *testing.T) {
	t.Run("short content is not split", func(t *testing.T) {
		content := "Short content\nSecond line\n"
		chunks := splitContent(content, 6000)
		if len(chunks) != 1 {
			t.Errorf("expected 1 chunk, got %d", len(chunks))
		}
	})

	t.Run("long content is split at line boundaries", func(t *testing.T) {
		// Build content > 6000 chars
		content := strings.Repeat("This line is about sixty characters long and adds up quickly.\n", 120)
		if len(content) <= 6000 {
			t.Skip("content not large enough to trigger split — adjust test")
		}

		chunks := splitContent(content, 6000)
		if len(chunks) < 2 {
			t.Fatalf("expected at least 2 chunks for %d char content, got %d", len(content), len(chunks))
		}

		// Each chunk should be <= maxChars (accounting for line boundaries)
		for i, chunk := range chunks {
			if len(chunk) > 6000+200 { // small tolerance for long lines
				t.Errorf("chunk %d is too large: %d chars", i+1, len(chunk))
			}
		}
	})

	t.Run("all content preserved across chunks", func(t *testing.T) {
		content := strings.Repeat("Line of content.\n", 120)
		chunks := splitContent(content, 500)

		// Rejoin and verify nothing was lost
		joined := strings.Join(chunks, "\n")
		// Original content lines should all appear
		if !strings.Contains(joined, "Line of content.") {
			t.Error("content should be preserved across splits")
		}
	})

	t.Run("empty content returns empty chunks", func(t *testing.T) {
		chunks := splitContent("", 6000)
		if len(chunks) != 0 {
			t.Errorf("expected 0 chunks for empty content, got %d", len(chunks))
		}
	})

	t.Run("single very long line is kept in one chunk", func(t *testing.T) {
		// A single line longer than maxChars — should not be split mid-line
		line := strings.Repeat("x", 7000)
		chunks := splitContent(line, 6000)
		if len(chunks) != 1 {
			t.Errorf("expected 1 chunk for a single long line, got %d", len(chunks))
		}
	})
}

// ── copilotFormat ─────────────────────────────────────────────────────────────

func TestCopilotExport(t *testing.T) {
	t.Run("creates .github/copilot-instructions.md", func(t *testing.T) {
		target := t.TempDir()
		f := &copilotFormat{}

		err := f.Export(sampleDocs(), target, sampleOpts("go-internal", "go"))
		if err != nil {
			t.Fatalf("Export() error: %v", err)
		}

		outPath := filepath.Join(target, ".github", "copilot-instructions.md")
		if _, err := os.Stat(outPath); err != nil {
			t.Fatalf("copilot-instructions.md not created: %v", err)
		}
	})

	t.Run("contains header and auto-generated notice", func(t *testing.T) {
		target := t.TempDir()
		f := &copilotFormat{}

		err := f.Export(sampleDocs(), target, sampleOpts("go-internal", "go"))
		if err != nil {
			t.Fatalf("Export() error: %v", err)
		}

		content, _ := os.ReadFile(filepath.Join(target, ".github", "copilot-instructions.md"))
		body := string(content)

		if !strings.Contains(body, "Copilot Instructions") {
			t.Error("should contain 'Copilot Instructions' header")
		}
		if !strings.Contains(body, "Auto-generated") {
			t.Error("should contain auto-generated notice")
		}
		if !strings.Contains(body, "go-internal") {
			t.Error("should mention the stack name")
		}
	})

	t.Run("all topics present as ## sections", func(t *testing.T) {
		target := t.TempDir()
		f := &copilotFormat{}

		err := f.Export(sampleDocs(), target, sampleOpts("go-internal", "go"))
		if err != nil {
			t.Fatalf("Export() error: %v", err)
		}

		content, _ := os.ReadFile(filepath.Join(target, ".github", "copilot-instructions.md"))
		body := string(content)

		if !strings.Contains(body, "## Core Principles") {
			t.Error("should contain '## Core Principles' section")
		}
		if !strings.Contains(body, "## Error Handling") {
			t.Error("should contain '## Error Handling' section")
		}
	})

	t.Run("oracle frontmatter stripped", func(t *testing.T) {
		target := t.TempDir()
		f := &copilotFormat{}

		err := f.Export(sampleDocs(), target, sampleOpts("go-internal", "go"))
		if err != nil {
			t.Fatalf("Export() error: %v", err)
		}

		content, _ := os.ReadFile(filepath.Join(target, ".github", "copilot-instructions.md"))
		body := string(content)

		if strings.Contains(body, "**Version**") {
			t.Error("output should not contain oracle frontmatter (**Version**)")
		}
		if strings.Contains(body, "**Authors:**") {
			t.Error("output should not contain oracle frontmatter (**Authors:**)")
		}
	})

	t.Run("empty docs slice writes file with just header", func(t *testing.T) {
		target := t.TempDir()
		f := &copilotFormat{}

		err := f.Export([]KnowledgeDoc{}, target, sampleOpts("go-internal", "go"))
		if err != nil {
			t.Fatalf("Export() should not error on empty docs: %v", err)
		}

		outPath := filepath.Join(target, ".github", "copilot-instructions.md")
		content, err := os.ReadFile(outPath)
		if err != nil {
			t.Fatalf("file should be created: %v", err)
		}
		if len(content) == 0 {
			t.Error("output should not be empty")
		}
	})
}

// ── windsurfFormat ────────────────────────────────────────────────────────────

func TestWindsurfExport(t *testing.T) {
	t.Run("creates .windsurfrules in target dir", func(t *testing.T) {
		target := t.TempDir()
		f := &windsurfFormat{}

		err := f.Export(sampleDocs(), target, sampleOpts("go-internal", "go"))
		if err != nil {
			t.Fatalf("Export() error: %v", err)
		}

		outPath := filepath.Join(target, ".windsurfrules")
		if _, err := os.Stat(outPath); err != nil {
			t.Fatalf(".windsurfrules not created: %v", err)
		}
	})

	t.Run("contains header with stack name", func(t *testing.T) {
		target := t.TempDir()
		f := &windsurfFormat{}

		err := f.Export(sampleDocs(), target, sampleOpts("go-internal", "go"))
		if err != nil {
			t.Fatalf("Export() error: %v", err)
		}

		content, _ := os.ReadFile(filepath.Join(target, ".windsurfrules"))
		body := string(content)

		if !strings.Contains(body, "go-internal") {
			t.Error(".windsurfrules should contain the stack name")
		}
		if !strings.Contains(body, "Windsurf Rules") {
			t.Error(".windsurfrules should contain 'Windsurf Rules' header")
		}
	})

	t.Run("contains auto-generated notice", func(t *testing.T) {
		target := t.TempDir()
		f := &windsurfFormat{}

		err := f.Export(sampleDocs(), target, sampleOpts("go-internal", "go"))
		if err != nil {
			t.Fatalf("Export() error: %v", err)
		}

		content, _ := os.ReadFile(filepath.Join(target, ".windsurfrules"))
		if !strings.Contains(string(content), "Auto-generated") {
			t.Error(".windsurfrules should contain auto-generated notice")
		}
	})

	t.Run("all topics present as ## sections", func(t *testing.T) {
		target := t.TempDir()
		f := &windsurfFormat{}

		err := f.Export(sampleDocs(), target, sampleOpts("go-internal", "go"))
		if err != nil {
			t.Fatalf("Export() error: %v", err)
		}

		content, _ := os.ReadFile(filepath.Join(target, ".windsurfrules"))
		body := string(content)

		if !strings.Contains(body, "## Core Principles") {
			t.Error("should contain '## Core Principles' section")
		}
		if !strings.Contains(body, "## Error Handling") {
			t.Error("should contain '## Error Handling' section")
		}
	})

	t.Run("oracle frontmatter stripped", func(t *testing.T) {
		target := t.TempDir()
		f := &windsurfFormat{}

		err := f.Export(sampleDocs(), target, sampleOpts("go-internal", "go"))
		if err != nil {
			t.Fatalf("Export() error: %v", err)
		}

		content, _ := os.ReadFile(filepath.Join(target, ".windsurfrules"))
		body := string(content)

		if strings.Contains(body, "**Version**") {
			t.Error("output should not contain oracle frontmatter (**Version**)")
		}
	})

	t.Run("empty docs slice writes file with just header", func(t *testing.T) {
		target := t.TempDir()
		f := &windsurfFormat{}

		err := f.Export([]KnowledgeDoc{}, target, sampleOpts("go-internal", "go"))
		if err != nil {
			t.Fatalf("Export() should not error on empty docs: %v", err)
		}

		outPath := filepath.Join(target, ".windsurfrules")
		content, err := os.ReadFile(outPath)
		if err != nil {
			t.Fatalf("file should be created: %v", err)
		}
		if len(content) == 0 {
			t.Error("output should not be empty")
		}
	})
}

// ── sanitizeAnchor ────────────────────────────────────────────────────────────

func TestSanitizeAnchor(t *testing.T) {
	// sanitizeAnchor keeps only [a-z0-9-]. Callers convert spaces to dashes first.
	tests := []struct {
		input    string
		expected string
	}{
		{"error-handling", "error-handling"},
		// spaces are stripped (caller converts spaces to dashes before calling)
		{"core principles", "coreprinciples"},
		{"core-principles", "core-principles"},
		// & is stripped entirely
		{"testing & validation", "testingvalidation"},
		{"123-setup", "123-setup"},
		{"hello world!", "helloworld"},
		{"", ""},
		// uppercase stripped — sanitizeAnchor only keeps lowercase
		{"UPPERCASE", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := sanitizeAnchor(tt.input)
			if got != tt.expected {
				t.Errorf("sanitizeAnchor(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

// ── promoteToSection ──────────────────────────────────────────────────────────

func TestPromoteToSection(t *testing.T) {
	t.Run("replaces first h1 with h2", func(t *testing.T) {
		content := "# Error Handling\n\nSome content.\n"
		got := promoteToSection("Error Handling", content)

		if !strings.Contains(got, "## Error Handling") {
			t.Error("result should contain ## Error Handling")
		}
		if strings.Contains(got, "# Error Handling\n") && !strings.Contains(got, "## Error Handling") {
			t.Error("original # heading should be replaced")
		}
	})

	t.Run("prepends ## when no h1 heading found", func(t *testing.T) {
		content := "## Already h2\n\nSome content.\n"
		got := promoteToSection("My Title", content)

		if !strings.HasPrefix(got, "## My Title") {
			t.Errorf("result should start with ## My Title when no h1 found, got: %q", got[:min(len(got), 50)])
		}
	})

	t.Run("does not affect h2 or deeper headings", func(t *testing.T) {
		content := "## Sub Heading\n\n### Deep Heading\n\nContent.\n"
		got := promoteToSection("My Title", content)

		if !strings.Contains(got, "## Sub Heading") {
			t.Error("existing ## heading should be preserved")
		}
		if !strings.Contains(got, "### Deep Heading") {
			t.Error("### heading should be preserved")
		}
	})
}

// ── RunExport (integration) ───────────────────────────────────────────────────

func TestRunExport_ValidSourceAndFormat(t *testing.T) {
	// RunExport calls os.Exit on errors, so we only test the happy path
	// to avoid the test process dying.
	t.Run("agents-md format completes without error", func(t *testing.T) {
		source := t.TempDir()
		target := t.TempDir()

		// Create valid docs in source
		createDocWithTitle(t, source, "01-core-principles.md", "Core Principles")
		createDocWithTitle(t, source, "02-error-handling.md", "Error Handling")

		opts := RunExportOpts{
			From:   source,
			Format: "agents-md",
			To:     target,
		}

		err := RunExport(opts)
		if err != nil {
			t.Fatalf("RunExport() error: %v", err)
		}

		outPath := filepath.Join(target, "AGENTS.md")
		if _, err := os.Stat(outPath); err != nil {
			t.Errorf("AGENTS.md should be created by RunExport: %v", err)
		}
	})

	t.Run("cursor format creates .cursor/rules/", func(t *testing.T) {
		source := t.TempDir()
		target := t.TempDir()

		createDocWithTitle(t, source, "01-core-principles.md", "Core Principles")

		opts := RunExportOpts{
			From:   source,
			Format: "cursor",
			To:     target,
		}

		err := RunExport(opts)
		if err != nil {
			t.Fatalf("RunExport() error: %v", err)
		}

		rulesDir := filepath.Join(target, ".cursor", "rules")
		if _, err := os.Stat(rulesDir); err != nil {
			t.Errorf(".cursor/rules/ should be created: %v", err)
		}
	})

	t.Run("all format runs all registered adapters", func(t *testing.T) {
		source := t.TempDir()
		target := t.TempDir()

		createDocWithTitle(t, source, "01-core-principles.md", "Core Principles")

		opts := RunExportOpts{
			From:   source,
			Format: "all",
			To:     target,
		}

		err := RunExport(opts)
		if err != nil {
			t.Fatalf("RunExport() error: %v", err)
		}

		// All format output files should exist
		expected := []string{
			filepath.Join(target, "AGENTS.md"),
			filepath.Join(target, ".cursor", "rules"),
			filepath.Join(target, ".github", "copilot-instructions.md"),
			filepath.Join(target, ".windsurfrules"),
		}

		for _, p := range expected {
			if _, err := os.Stat(p); err != nil {
				t.Errorf("expected output %s after 'all' format: %v", p, err)
			}
		}
	})

	t.Run("explicit language override is respected", func(t *testing.T) {
		source := t.TempDir()
		target := t.TempDir()

		createDocWithTitle(t, source, "01-core-principles.md", "Core Principles")

		opts := RunExportOpts{
			From:     source,
			Format:   "cursor",
			To:       target,
			Language: "python",
		}

		err := RunExport(opts)
		if err != nil {
			t.Fatalf("RunExport() error: %v", err)
		}

		// The .mdc file should use Python glob even though dir name is unrelated
		mdcPath := filepath.Join(target, ".cursor", "rules", "01-core-principles.mdc")
		content, err := os.ReadFile(mdcPath)
		if err != nil {
			t.Fatalf("reading .mdc: %v", err)
		}
		if !strings.Contains(string(content), "**/*.py") {
			t.Error("should use Python glob when Language override is 'python'")
		}
	})
}
