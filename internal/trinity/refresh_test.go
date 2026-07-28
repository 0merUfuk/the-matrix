package trinity

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFindStaleDocs_Fresh(t *testing.T) {
	dir := t.TempDir()

	// Doc created recently (2026-03-01) — should be fresh at 6 months
	content := "**Version**: 1.0\n**Created**: 2026-03-01\n\n" + strings.Repeat("line\n", 55)
	os.WriteFile(filepath.Join(dir, "01-test.md"), []byte(content), 0644)

	stale := findStaleDocs(dir, 6, nil)
	if len(stale) != 0 {
		t.Errorf("expected no stale docs, got %d", len(stale))
	}
}

func TestFindStaleDocs_Stale(t *testing.T) {
	dir := t.TempDir()

	// Doc created long ago — should be stale at 6 months
	content := "**Version**: 1.0\n**Created**: 2024-01-01\n\n" + strings.Repeat("line\n", 55)
	os.WriteFile(filepath.Join(dir, "01-old.md"), []byte(content), 0644)

	stale := findStaleDocs(dir, 6, nil)
	if len(stale) != 1 {
		t.Fatalf("expected 1 stale doc, got %d", len(stale))
	}
	if stale[0].Filename != "01-old.md" {
		t.Errorf("expected '01-old.md', got '%s'", stale[0].Filename)
	}
	if stale[0].DateStr != "2024-01-01" {
		t.Errorf("expected date '2024-01-01', got '%s'", stale[0].DateStr)
	}
}

func TestFindStaleDocs_NoDate(t *testing.T) {
	dir := t.TempDir()

	// Doc with no date frontmatter
	content := strings.Repeat("no date frontmatter\n", 55)
	os.WriteFile(filepath.Join(dir, "01-nodate.md"), []byte(content), 0644)

	stale := findStaleDocs(dir, 6, nil)
	if len(stale) != 1 {
		t.Fatalf("expected 1 stale doc (no date), got %d", len(stale))
	}
	if stale[0].DateStr != "?" {
		t.Errorf("expected date '?', got '%s'", stale[0].DateStr)
	}
	if stale[0].AgeStr != "no date found" {
		t.Errorf("expected age 'no date found', got '%s'", stale[0].AgeStr)
	}
}

func TestFindStaleDocs_TopicFilter(t *testing.T) {
	dir := t.TempDir()

	// Two docs, both stale
	oldContent := "**Created**: 2024-01-01\n\n" + strings.Repeat("line\n", 55)
	os.WriteFile(filepath.Join(dir, "08-api-design.md"), []byte(oldContent), 0644)
	os.WriteFile(filepath.Join(dir, "14-api-gateway.md"), []byte(oldContent), 0644)
	os.WriteFile(filepath.Join(dir, "05-testing.md"), []byte(oldContent), 0644)

	// Filter by "api" — should match 08-api-design and 14-api-gateway (substring match)
	stale := findStaleDocs(dir, 6, []string{"api"})
	if len(stale) != 2 {
		t.Fatalf("expected 2 stale docs matching 'api', got %d", len(stale))
	}

	names := []string{stale[0].Filename, stale[1].Filename}
	for _, name := range names {
		if !strings.Contains(name, "api") {
			t.Errorf("expected 'api' in filename, got '%s'", name)
		}
	}
}

func TestFindStaleDocs_TopicFilter_NoMatch(t *testing.T) {
	dir := t.TempDir()

	oldContent := "**Created**: 2024-01-01\n\n" + strings.Repeat("line\n", 55)
	os.WriteFile(filepath.Join(dir, "05-testing.md"), []byte(oldContent), 0644)

	stale := findStaleDocs(dir, 6, []string{"nonexistent-topic"})
	if len(stale) != 0 {
		t.Errorf("expected 0 stale docs with non-matching filter, got %d", len(stale))
	}
}

func TestFindStaleDocs_MaxAgeZero(t *testing.T) {
	dir := t.TempDir()

	// Even a very recent doc should be stale at maxAge=0
	content := "**Created**: 2026-03-15\n\n" + strings.Repeat("line\n", 55)
	os.WriteFile(filepath.Join(dir, "01-recent.md"), []byte(content), 0644)

	stale := findStaleDocs(dir, 0, nil)
	if len(stale) != 1 {
		t.Errorf("expected 1 stale doc at maxAge=0, got %d", len(stale))
	}
}

func TestFindStaleDocs_ExcludesIndexMD(t *testing.T) {
	dir := t.TempDir()

	oldContent := "**Created**: 2024-01-01\n\n" + strings.Repeat("line\n", 55)
	os.WriteFile(filepath.Join(dir, "_index.md"), []byte(oldContent), 0644)
	os.WriteFile(filepath.Join(dir, "01-real.md"), []byte(oldContent), 0644)

	stale := findStaleDocs(dir, 6, nil)
	if len(stale) != 1 {
		t.Fatalf("expected 1 stale doc (excluding _index.md), got %d", len(stale))
	}
	if stale[0].Filename == "_index.md" {
		t.Error("_index.md should be excluded")
	}
}
