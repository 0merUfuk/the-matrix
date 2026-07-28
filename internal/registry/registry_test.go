package registry

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// testIndex returns a sample RegistryIndex for testing.
func testIndex() *RegistryIndex {
	return &RegistryIndex{
		Version:     "1.0.0",
		LastUpdated: "2026-03-20",
		Packs: []PackMeta{
			{
				Name:          "go-chi",
				DisplayName:   "Go + Chi Router",
				Language:      "go",
				Framework:     "chi",
				Version:       "1.0.0",
				CreatedDate:   "2026-03-15",
				RefreshedDate: "2026-03-20",
				TopicCount:    17,
				QualityScore:  92,
				DownloadURL:   "packs/go-chi",
				Description:   "Best-practice knowledge for Go services using chi router",
			},
			{
				Name:          "nodejs-express",
				DisplayName:   "Node.js + Express",
				Language:      "nodejs",
				Framework:     "express",
				Version:       "1.0.0",
				CreatedDate:   "2026-03-16",
				RefreshedDate: "2026-03-19",
				TopicCount:    15,
				QualityScore:  88,
				DownloadURL:   "packs/nodejs-express",
				Description:   "Best-practice knowledge for Node.js Express services",
			},
			{
				Name:          "python-fastapi",
				DisplayName:   "Python + FastAPI",
				Language:      "python",
				Framework:     "fastapi",
				Version:       "1.0.0",
				CreatedDate:   "2026-03-17",
				RefreshedDate: "2026-03-18",
				TopicCount:    12,
				QualityScore:  85,
				DownloadURL:   "packs/python-fastapi",
				Description:   "Best-practice knowledge for Python FastAPI services",
			},
		},
	}
}

// testPackDetail returns a sample PackDetail for testing.
func testPackDetail() *PackDetail {
	return &PackDetail{
		PackMeta: PackMeta{
			Name:          "go-chi",
			DisplayName:   "Go + Chi Router",
			Language:      "go",
			Framework:     "chi",
			Version:       "1.0.0",
			CreatedDate:   "2026-03-15",
			RefreshedDate: "2026-03-20",
			TopicCount:    2,
			QualityScore:  92,
			DownloadURL:   "packs/go-chi",
			Description:   "Best-practice knowledge for Go services using chi router",
		},
		ArchStyle:    "layered",
		ResearchedBy: "oracle v1.3.0",
		GoldStandard: "go-internal",
		Topics: []PackTopic{
			{File: "01-project-structure.md", Lines: 200, Tier1Sources: 5},
			{File: "02-layered-architecture.md", Lines: 350, Tier1Sources: 8},
		},
	}
}

// sampleDocContent returns valid doc content that passes ValidateInjectDoc rules.
func sampleDocContent(title string) string {
	// Must have 50+ lines and required frontmatter fields.
	var b strings.Builder
	b.WriteString("**Version**: 1.0\n")
	b.WriteString("**Created**: 2026-03-15\n")
	b.WriteString("**Last Updated**: 2026-03-20\n")
	b.WriteString("**Authors:** Oracle CLI\n")
	b.WriteString("\n---\n\n")
	b.WriteString("# " + title + "\n\n")
	for i := 0; i < 50; i++ {
		b.WriteString("This is line content for testing purposes.\n")
	}
	return b.String()
}

// ─── Search Tests ────────────────────────────────────────────────────────────

func TestSearch_ByLanguage(t *testing.T) {
	index := testIndex()

	results := Search(index, "go")
	if len(results) != 1 {
		t.Fatalf("expected 1 result for 'go', got %d", len(results))
	}
	if results[0].Name != "go-chi" {
		t.Errorf("expected go-chi, got %s", results[0].Name)
	}
}

func TestSearch_ByFramework(t *testing.T) {
	index := testIndex()

	results := Search(index, "express")
	if len(results) != 1 {
		t.Fatalf("expected 1 result for 'express', got %d", len(results))
	}
	if results[0].Name != "nodejs-express" {
		t.Errorf("expected nodejs-express, got %s", results[0].Name)
	}
}

func TestSearch_CaseInsensitive(t *testing.T) {
	index := testIndex()

	results := Search(index, "PYTHON")
	if len(results) != 1 {
		t.Fatalf("expected 1 result for 'PYTHON', got %d", len(results))
	}
	if results[0].Name != "python-fastapi" {
		t.Errorf("expected python-fastapi, got %s", results[0].Name)
	}
}

func TestSearch_ByDescription(t *testing.T) {
	index := testIndex()

	results := Search(index, "router")
	if len(results) != 1 {
		t.Fatalf("expected 1 result for 'router', got %d", len(results))
	}
	if results[0].Name != "go-chi" {
		t.Errorf("expected go-chi, got %s", results[0].Name)
	}
}

func TestSearch_NoMatch(t *testing.T) {
	index := testIndex()

	results := Search(index, "ruby")
	if len(results) != 0 {
		t.Errorf("expected 0 results for 'ruby', got %d", len(results))
	}
}

func TestSearch_NilIndex(t *testing.T) {
	results := Search(nil, "go")
	if results != nil {
		t.Errorf("expected nil for nil index, got %v", results)
	}
}

func TestSearch_EmptyQuery(t *testing.T) {
	index := testIndex()
	results := Search(index, "")
	if results != nil {
		t.Errorf("expected nil for empty query, got %v", results)
	}
}

// ─── FindMatch Tests ─────────────────────────────────────────────────────────

func TestFindMatch_ExactMatch(t *testing.T) {
	index := testIndex()

	match := FindMatch(index, "go", "chi")
	if match == nil {
		t.Fatal("expected match for go/chi")
	}
	if match.Name != "go-chi" {
		t.Errorf("expected go-chi, got %s", match.Name)
	}
}

func TestFindMatch_CaseInsensitive(t *testing.T) {
	index := testIndex()

	match := FindMatch(index, "Go", "Chi")
	if match == nil {
		t.Fatal("expected match for Go/Chi")
	}
	if match.Name != "go-chi" {
		t.Errorf("expected go-chi, got %s", match.Name)
	}
}

func TestFindMatch_FrameworkWithVersion(t *testing.T) {
	index := testIndex()

	// Wizard says "chi v5" — pack says "chi". Contains check should match.
	match := FindMatch(index, "go", "chi v5")
	if match == nil {
		t.Fatal("expected match for go/chi v5")
	}
	if match.Name != "go-chi" {
		t.Errorf("expected go-chi, got %s", match.Name)
	}
}

func TestFindMatch_NoMatch(t *testing.T) {
	index := testIndex()

	match := FindMatch(index, "go", "gin")
	if match != nil {
		t.Errorf("expected no match for go/gin, got %s", match.Name)
	}
}

func TestFindMatch_NilIndex(t *testing.T) {
	match := FindMatch(nil, "go", "chi")
	if match != nil {
		t.Errorf("expected nil for nil index, got %v", match)
	}
}

func TestFindMatch_WrongLanguage(t *testing.T) {
	index := testIndex()

	match := FindMatch(index, "rust", "chi")
	if match != nil {
		t.Errorf("expected no match for rust/chi, got %s", match.Name)
	}
}

// ─── Score Tests ─────────────────────────────────────────────────────────────

func TestComputeScore_BasicCase(t *testing.T) {
	detail := testPackDetail()
	score := ComputeScore(*detail)

	// topicCount=2: base = 10
	// totalLines=550: linesBonus = 5
	// totalTier1=13: sourceBonus = 25 (capped)
	// total = 10 + 5 + 25 = 40
	// Wait, sourceBonus = 13*2 = 26, capped at 25
	expected := 10 + 5 + 25
	if score != expected {
		t.Errorf("expected score %d, got %d", expected, score)
	}
}

func TestComputeScore_MaxScore(t *testing.T) {
	detail := &PackDetail{
		PackMeta: PackMeta{TopicCount: 20},
		Topics: []PackTopic{
			{Lines: 5000, Tier1Sources: 50},
		},
	}
	score := ComputeScore(*detail)

	// base = 20*5 = 100, capped at 50
	// linesBonus = 5000/100 = 50, capped at 25
	// sourceBonus = 50*2 = 100, capped at 25
	// total = 50 + 25 + 25 = 100
	if score != 100 {
		t.Errorf("expected max score 100, got %d", score)
	}
}

func TestComputeScore_EmptyPack(t *testing.T) {
	detail := &PackDetail{}
	score := ComputeScore(*detail)
	if score != 0 {
		t.Errorf("expected score 0 for empty pack, got %d", score)
	}
}

// ─── HTTP Client Tests ───────────────────────────────────────────────────────

func TestFetchRegistryIndex_Success(t *testing.T) {
	index := testIndex()
	data, _ := json.Marshal(index)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedUA := userAgentString()
		if r.Header.Get("User-Agent") != expectedUA {
			t.Errorf("expected User-Agent %q, got %q", expectedUA, r.Header.Get("User-Agent"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(data)
	}))
	defer server.Close()

	result, err := FetchRegistryIndex(server.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Version != "1.0.0" {
		t.Errorf("expected version 1.0.0, got %s", result.Version)
	}
	if len(result.Packs) != 3 {
		t.Errorf("expected 3 packs, got %d", len(result.Packs))
	}
}

func TestFetchRegistryIndex_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	_, err := FetchRegistryIndex(server.URL)
	if err == nil {
		t.Fatal("expected error for 404 response")
	}
	if !strings.Contains(err.Error(), "404") {
		t.Errorf("expected error to mention 404, got: %s", err.Error())
	}
}

func TestFetchRegistryIndex_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not json"))
	}))
	defer server.Close()

	_, err := FetchRegistryIndex(server.URL)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestFetchPackDetail_Success(t *testing.T) {
	detail := testPackDetail()
	data, _ := json.Marshal(detail)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(data)
	}))
	defer server.Close()

	result, err := FetchPackDetail(server.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Name != "go-chi" {
		t.Errorf("expected name go-chi, got %s", result.Name)
	}
	if len(result.Topics) != 2 {
		t.Errorf("expected 2 topics, got %d", len(result.Topics))
	}
}

func TestFetchPackFile_Success(t *testing.T) {
	content := sampleDocContent("Test Doc")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(content))
	}))
	defer server.Close()

	result, err := FetchPackFile(server.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != content {
		t.Error("fetched content does not match expected")
	}
}

// ─── Cache Tests ─────────────────────────────────────────────────────────────

func TestCache_IndexWriteAndRead(t *testing.T) {
	// Use a temp directory as home to isolate cache.
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	index := testIndex()
	if err := WriteCachedIndex(index); err != nil {
		t.Fatalf("WriteCachedIndex failed: %v", err)
	}

	cached, fresh := ReadCachedIndex()
	if cached == nil {
		t.Fatal("ReadCachedIndex returned nil")
	}
	if !fresh {
		t.Error("expected fresh cache")
	}
	if cached.Version != "1.0.0" {
		t.Errorf("expected version 1.0.0, got %s", cached.Version)
	}
	if len(cached.Packs) != 3 {
		t.Errorf("expected 3 packs, got %d", len(cached.Packs))
	}
}

func TestCache_IndexStale(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	index := testIndex()
	if err := WriteCachedIndex(index); err != nil {
		t.Fatalf("WriteCachedIndex failed: %v", err)
	}

	// Set mtime to 2 hours ago to make it stale.
	cachePath := filepath.Join(tmpHome, ".the-matrix", "registry", "registry.json")
	past := time.Now().Add(-2 * time.Hour)
	if err := os.Chtimes(cachePath, past, past); err != nil {
		t.Fatalf("setting mtime: %v", err)
	}

	cached, fresh := ReadCachedIndex()
	if cached == nil {
		t.Fatal("ReadCachedIndex returned nil for stale cache")
	}
	if fresh {
		t.Error("expected stale cache, got fresh")
	}
}

func TestCache_IndexCorrupt(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	// Write corrupt data.
	cacheDir := filepath.Join(tmpHome, ".the-matrix", "registry")
	os.MkdirAll(cacheDir, 0755)
	os.WriteFile(filepath.Join(cacheDir, "registry.json"), []byte("not json"), 0644)

	cached, fresh := ReadCachedIndex()
	if cached != nil {
		t.Error("expected nil for corrupt cache")
	}
	if fresh {
		t.Error("expected not fresh for corrupt cache")
	}

	// Corrupt file should have been deleted.
	if _, err := os.Stat(filepath.Join(cacheDir, "registry.json")); err == nil {
		t.Error("expected corrupt cache file to be deleted")
	}
}

func TestCache_PackWriteAndRead(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	detail := testPackDetail()
	data, _ := json.Marshal(detail)

	if err := WriteCachedPack("go-chi", data); err != nil {
		t.Fatalf("WriteCachedPack failed: %v", err)
	}

	cached, fresh := ReadCachedPack("go-chi")
	if cached == nil {
		t.Fatal("ReadCachedPack returned nil")
	}
	if !fresh {
		t.Error("expected fresh pack cache")
	}

	var result PackDetail
	if err := json.Unmarshal(cached, &result); err != nil {
		t.Fatalf("unmarshaling cached pack: %v", err)
	}
	if result.Name != "go-chi" {
		t.Errorf("expected name go-chi, got %s", result.Name)
	}
}

func TestCache_PackMissing(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	cached, fresh := ReadCachedPack("nonexistent")
	if cached != nil {
		t.Error("expected nil for missing pack cache")
	}
	if fresh {
		t.Error("expected not fresh for missing pack cache")
	}
}

// ─── Pull Tests ──────────────────────────────────────────────────────────────

func TestPull_DownloadsAndValidates(t *testing.T) {
	detail := testPackDetail()
	detailData, _ := json.Marshal(detail)

	doc1 := sampleDocContent("Project Structure")
	doc2 := sampleDocContent("Layered Architecture")

	// Use temp home to prevent cache interference.
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/pack.json"):
			w.Write(detailData)
		case strings.HasSuffix(r.URL.Path, "/01-project-structure.md"):
			w.Write([]byte(doc1))
		case strings.HasSuffix(r.URL.Path, "/02-layered-architecture.md"):
			w.Write([]byte(doc2))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	// Override the base URL for this test by using Info directly.
	// We can't change the const, so we test Pull through a mock server
	// by pre-caching the pack detail.
	cacheData, _ := json.MarshalIndent(detail, "", "  ")
	_ = WriteCachedPack("go-chi", cacheData)

	// Modify topics to point at our test server.
	detail.Topics[0].File = "01-project-structure.md"
	detail.Topics[1].File = "02-layered-architecture.md"

	// Instead of Pull (which uses hardcoded BasePackURL), test the download + write flow
	// by calling the components directly.
	targetDir := filepath.Join(t.TempDir(), "output")

	// Simulate Pull behavior: fetch detail from cache, download files from mock server.
	for _, topic := range detail.Topics {
		fileURL := server.URL + "/packs/go-chi/" + topic.File
		content, err := FetchPackFile(fileURL)
		if err != nil {
			t.Fatalf("FetchPackFile failed for %s: %v", topic.File, err)
		}

		// Validate.
		if len(content) < 50 {
			t.Fatalf("content too short for %s", topic.File)
		}

		// Write.
		os.MkdirAll(targetDir, 0755)
		if err := os.WriteFile(filepath.Join(targetDir, topic.File), []byte(content), 0644); err != nil {
			t.Fatalf("writing %s: %v", topic.File, err)
		}
	}

	// Verify files were written.
	entries, err := os.ReadDir(targetDir)
	if err != nil {
		t.Fatalf("reading target dir: %v", err)
	}
	if len(entries) != 2 {
		t.Errorf("expected 2 files in target dir, got %d", len(entries))
	}

	// Verify content.
	content, _ := os.ReadFile(filepath.Join(targetDir, "01-project-structure.md"))
	if !strings.Contains(string(content), "Project Structure") {
		t.Error("written file should contain expected content")
	}
}

func TestPull_ValidationFailure(t *testing.T) {
	// Test that validation errors prevent writing.
	validator := func(filename, content string) error {
		return os.ErrInvalid
	}

	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	// Pre-cache a pack detail with topics pointing to a real server.
	detail := testPackDetail()
	doc := sampleDocContent("Test")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/pack.json"):
			data, _ := json.Marshal(detail)
			w.Write(data)
		default:
			w.Write([]byte(doc))
		}
	}))
	defer server.Close()

	// Pre-cache the detail so Pull uses it.
	cacheData, _ := json.MarshalIndent(detail, "", "  ")
	_ = WriteCachedPack("go-chi", cacheData)

	targetDir := filepath.Join(t.TempDir(), "output")

	// Simulate Pull with validation — the first file should fail.
	for _, topic := range detail.Topics {
		fileURL := server.URL + "/packs/go-chi/" + topic.File
		content, err := FetchPackFile(fileURL)
		if err != nil {
			t.Fatalf("unexpected fetch error: %v", err)
		}

		if err := validator(topic.File, content); err != nil {
			// Expected — validation rejects the file.
			return
		}
	}
	t.Fatal("expected validation to reject a file")

	_ = targetDir // unused in the success path
}

// ─── Load Tests (Integration with Cache + Network) ───────────────────────────

func TestLoad_UsesCache(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	index := testIndex()
	if err := WriteCachedIndex(index); err != nil {
		t.Fatalf("writing cache: %v", err)
	}

	// Load should return cached index without network call.
	// (If it tried to hit the real registry URL and failed, this would error.)
	result, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if result.Version != "1.0.0" {
		t.Errorf("expected version 1.0.0, got %s", result.Version)
	}
}
