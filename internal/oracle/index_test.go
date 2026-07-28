package oracle

import (
	"strings"
	"testing"
)

func TestBuildIndex_Format(t *testing.T) {
	docs := []DocInfo{
		{Filename: "01-test.md", Content: "**Version**: 1.0\n**Created**: 2026-03-15\n" + strings.Repeat("line\n", 55)},
		{Filename: "02-test.md", Content: "**Version**: 1.0\n**Created**: 2026-03-10\n" + strings.Repeat("line\n", 30)},
	}

	result := BuildIndex(docs, "/tmp/source", "/tmp/target")

	// Check frontmatter
	if !strings.Contains(result, "**Version**: 1.0") {
		t.Error("expected **Version** in index")
	}
	if !strings.Contains(result, "**Authors:** Oracle CLI (injected)") {
		t.Error("expected authors in index")
	}

	// Check metadata
	if !strings.Contains(result, "**Docs**: 2") {
		t.Error("expected doc count 2")
	}
	if !strings.Contains(result, "# Knowledge Index") {
		t.Error("expected title")
	}

	// Check table
	if !strings.Contains(result, "| `01-test.md`") {
		t.Error("expected 01-test.md in table")
	}
	if !strings.Contains(result, "| `02-test.md`") {
		t.Error("expected 02-test.md in table")
	}

	// Check dates parsed from content
	if !strings.Contains(result, "2026-03-15") {
		t.Error("expected created date 2026-03-15 from doc content")
	}
	if !strings.Contains(result, "2026-03-10") {
		t.Error("expected created date 2026-03-10 from doc content")
	}
}

func TestBuildIndex_UnknownDate(t *testing.T) {
	docs := []DocInfo{
		{Filename: "01-nodate.md", Content: "# No date frontmatter\n" + strings.Repeat("line\n", 55)},
	}

	result := BuildIndex(docs, "/tmp/source", "/tmp/target")

	if !strings.Contains(result, "unknown") {
		t.Error("expected 'unknown' for doc without Created date")
	}
}
