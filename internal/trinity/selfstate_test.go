package trinity

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// stubFetchers returns a fetchers struct whose subprocess-backed functions
// return canned values. Tests pass these to collectSelfState directly so no
// real git/gh subprocesses run during unit tests.
func stubFetchers() *fetchers {
	return &fetchers{
		gitBranch:       func(string) (string, error) { return "main", nil },
		gitHeadSHA:      func(string) (string, error) { return "01715a6f0123456789abcdef0123456789abcdef", nil },
		gitLatestTag:    func(string) (string, error) { return "v1.7.1", nil },
		gitCommitsSince: func(string, string) (int, error) { return 2, nil },
		gitDirty:        func(string) (bool, error) { return false, nil },
		gitAheadBehind:  func(string) (int, int, error) { return 0, 0, nil },
		prCount:         func(string) (int, error) { return 3, nil },
		issueCount:      func(string) (int, error) { return 5, nil },
	}
}

// writeTool writes a stub cmd/{tool}/main.go declaring `var version = ver`.
func writeTool(t *testing.T, root, tool, ver string) {
	t.Helper()
	dir := filepath.Join(root, "cmd", tool)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("mkdir tool dir: %v", err)
	}
	body := "package main\n\nvar version = \"" + ver + "\"\n"
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(body), 0644); err != nil {
		t.Fatalf("write tool main: %v", err)
	}
}

func TestSelfState_VersionExtraction(t *testing.T) {
	root := t.TempDir()
	writeTool(t, root, "neo", "1.7.2-dev")
	writeTool(t, root, "morpheus", "1.7.2-dev")
	writeTool(t, root, "oracle", "1.7.2-dev")
	writeTool(t, root, "trinity", "1.7.2-dev")

	state, _ := collectSelfState(root, stubFetchers())

	for _, name := range []string{"neo", "morpheus", "oracle", "trinity"} {
		tb, ok := state.Tools[name]
		if !ok || tb == nil {
			t.Fatalf("tool %s missing or nil", name)
		}
		if tb.Version != "1.7.2-dev" {
			t.Errorf("tool %s version = %q, want %q", name, tb.Version, "1.7.2-dev")
		}
		if len(tb.Commands) == 0 {
			t.Errorf("tool %s has empty commands", name)
		}
	}

	// trinity must list self-state in its commands
	found := false
	for _, c := range state.Tools["trinity"].Commands {
		if c == "self-state" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("trinity commands missing self-state: %v", state.Tools["trinity"].Commands)
	}
}

func TestSelfState_AgentMemoryCounts(t *testing.T) {
	root := t.TempDir()
	memDir := filepath.Join(root, ".claude", "agent-memory")

	// Agent "a": 2 files
	aDir := filepath.Join(memDir, "a")
	if err := os.MkdirAll(aDir, 0755); err != nil {
		t.Fatalf("mkdir a: %v", err)
	}
	if err := os.WriteFile(filepath.Join(aDir, "MEMORY.md"), []byte("line1\nline2\nline3\n"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := os.WriteFile(filepath.Join(aDir, "feedback.md"), []byte("one\ntwo\n"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	// Agent "b": 1 file
	bDir := filepath.Join(memDir, "b")
	if err := os.MkdirAll(bDir, 0755); err != nil {
		t.Fatalf("mkdir b: %v", err)
	}
	if err := os.WriteFile(filepath.Join(bDir, "MEMORY.md"), []byte("alpha\n"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	var errs []CollectionError
	got := collectAgentMemory(root, &errs)

	if len(errs) != 0 {
		t.Errorf("unexpected errors: %v", errs)
	}
	if got["a"].Files != 2 {
		t.Errorf("agent a files = %d, want 2", got["a"].Files)
	}
	if got["a"].Lines != 5 {
		t.Errorf("agent a lines = %d, want 5", got["a"].Lines)
	}
	if got["b"].Files != 1 {
		t.Errorf("agent b files = %d, want 1", got["b"].Files)
	}
	if got["b"].Lines != 1 {
		t.Errorf("agent b lines = %d, want 1", got["b"].Lines)
	}
}

func TestSelfState_BacklogCounting(t *testing.T) {
	root := t.TempDir()
	claudeDir := filepath.Join(root, ".claude")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	nextSteps := "# Next Steps\n\n- [ ] First item\n- [x] Done item\n- [ ] Second pending\n  - [ ] Indented (should not match)\n- [ ] Third pending\n"
	if err := os.WriteFile(filepath.Join(claudeDir, "NEXT_STEPS.md"), []byte(nextSteps), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	knownIssues := "# Known Issues\n\n## Issue 1\n**Status**: Open\n\n## Issue 2\n**Status**: ✅ Fixed\n\n## Issue 3\n**Status**: Investigating\n"
	if err := os.WriteFile(filepath.Join(claudeDir, "KNOWN_ISSUES.md"), []byte(knownIssues), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	f := stubFetchers()
	var errs []CollectionError
	got := collectBacklog(root, f, &errs)

	if got.NextStepsPending != 3 {
		t.Errorf("next_steps_pending = %d, want 3", got.NextStepsPending)
	}
	if got.KnownIssuesOpen != 2 {
		t.Errorf("known_issues_open = %d, want 2 (Issue 2 has ✅, must be excluded)", got.KnownIssuesOpen)
	}
	if got.OpenPRs == nil || *got.OpenPRs != 3 {
		t.Errorf("open_prs = %v, want 3", got.OpenPRs)
	}
	if got.OpenIssues == nil || *got.OpenIssues != 5 {
		t.Errorf("open_issues = %v, want 5", got.OpenIssues)
	}
}

func TestSelfState_StaleKnowledgeDocs(t *testing.T) {
	root := t.TempDir()
	knowledgeDir := filepath.Join(root, ".claude", "knowledge", "go-rules")
	if err := os.MkdirAll(knowledgeDir, 0755); err != nil {
		t.Fatalf("mkdir knowledge dir: %v", err)
	}

	// Stale doc: Created more than 6 months ago (2025-01-01 is well over 6 months from any 2026 date).
	staleContent := "**Version**: 1.0\n**Created**: 2025-01-01\n**Last Updated**: 2025-01-01\n\n# Stale Doc\n"
	if err := os.WriteFile(filepath.Join(knowledgeDir, "stale.md"), []byte(staleContent), 0644); err != nil {
		t.Fatalf("write stale doc: %v", err)
	}

	// Fresh doc: Created within 6 months — use a fixed recent date.
	// MonthsAgo computes relative to time.Now(), so pick a date that is guaranteed
	// to be < 6 months ago at test time. We use 1 month ago from May 2026.
	freshContent := "**Version**: 1.0\n**Created**: 2026-04-15\n**Last Updated**: 2026-04-15\n\n# Fresh Doc\n"
	if err := os.WriteFile(filepath.Join(knowledgeDir, "fresh.md"), []byte(freshContent), 0644); err != nil {
		t.Fatalf("write fresh doc: %v", err)
	}

	f := stubFetchers()
	var errs []CollectionError
	got := collectBacklog(root, f, &errs)

	if got.StaleKnowledgeDocs == nil {
		t.Fatal("stale_knowledge_docs should be non-nil when .claude/knowledge/ exists")
	}
	if *got.StaleKnowledgeDocs != 1 {
		t.Errorf("stale_knowledge_docs = %d, want 1 (1 stale + 1 fresh)", *got.StaleKnowledgeDocs)
	}

	// No collection errors expected for the knowledge walk
	for _, e := range errs {
		if e.Section == "backlog" && e.Field == "stale_knowledge_docs" {
			t.Errorf("unexpected collection error for stale_knowledge_docs: %s", e.Error)
		}
	}
}

func TestSelfState_DeltaComputation(t *testing.T) {
	prev := &SelfStateV1{
		Backlog: BacklogBlock{NextStepsPending: 10, KnownIssuesOpen: 4},
		AgentMemory: map[string]MemoryBlock{
			"a": {Files: 3, Lines: 100},
		},
		Tools: map[string]*ToolBlock{
			"neo": {Version: "1.7.0"},
		},
	}
	cur := &SelfStateV1{
		Backlog: BacklogBlock{NextStepsPending: 7, KnownIssuesOpen: 6},
		AgentMemory: map[string]MemoryBlock{
			"a": {Files: 5, Lines: 250},
		},
		Tools: map[string]*ToolBlock{
			"neo": {Version: "1.7.2-dev"},
		},
	}

	d := computeDeltas(prev, cur)

	if d.NextStepsClosed != 3 {
		t.Errorf("next_steps_closed = %d, want 3", d.NextStepsClosed)
	}
	if d.NextStepsAdded != 0 {
		t.Errorf("next_steps_added = %d, want 0", d.NextStepsAdded)
	}
	if d.IssuesOpened != 2 {
		t.Errorf("issues_opened = %d, want 2", d.IssuesOpened)
	}
	if d.MemoryFilesAdded != 2 {
		t.Errorf("memory_files_added = %d, want 2", d.MemoryFilesAdded)
	}
	if d.MemoryGrowthLines != 150 {
		t.Errorf("memory_growth_lines = %d, want 150", d.MemoryGrowthLines)
	}
	if len(d.VersionChanges) != 1 {
		t.Fatalf("expected 1 version change, got %d", len(d.VersionChanges))
	}
	vc := d.VersionChanges[0]
	if vc.Tool != "neo" {
		t.Errorf("expected Tool=neo, got %q", vc.Tool)
	}
	if vc.From != "1.7.0" {
		t.Errorf("expected From=1.7.0, got %q", vc.From)
	}
	if vc.To != "1.7.2-dev" {
		t.Errorf("expected To=1.7.2-dev, got %q", vc.To)
	}
}

func TestSelfState_JSONSchemaShape(t *testing.T) {
	root := t.TempDir()
	writeTool(t, root, "neo", "1.7.2-dev")
	writeTool(t, root, "morpheus", "1.7.2-dev")
	writeTool(t, root, "oracle", "1.7.2-dev")
	writeTool(t, root, "trinity", "1.7.2-dev")

	state, errs := collectSelfState(root, stubFetchers())
	state.CollectionErrors = errs

	data, err := marshalIndent(state)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got SelfStateV1
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got.Schema != "self-state.v1" {
		t.Errorf("schema = %q, want self-state.v1", got.Schema)
	}
	if got.Mode != "quick" {
		t.Errorf("mode = %q, want quick", got.Mode)
	}
	if got.ProducedAt == "" {
		t.Error("produced_at is empty")
	}
	if got.Matrix.Branch != "main" {
		t.Errorf("matrix.branch = %q, want main", got.Matrix.Branch)
	}
	if got.Matrix.HeadSHA == "" || len(got.Matrix.HeadSHA) != 40 {
		t.Errorf("matrix.head_sha = %q, want 40-char SHA", got.Matrix.HeadSHA)
	}
	if got.Matrix.LatestTag == "" {
		t.Error("matrix.latest_tag empty")
	}
	if len(got.Tools) != 4 {
		t.Errorf("tools count = %d, want 4", len(got.Tools))
	}
	// Quick mode: prose blocks are explicit JSON null
	if got.ContextDrift != nil {
		t.Errorf("context_drift should be nil in quick mode, got %s", string(*got.ContextDrift))
	}
	if got.EcosystemHealth != nil {
		t.Errorf("ecosystem_health should be nil in quick mode")
	}
	if got.Security != nil {
		t.Errorf("security should be nil in quick mode")
	}
	if got.DispatchSignals != nil {
		t.Errorf("dispatch_signals should be nil in quick mode")
	}

	// Verify nullability fields in raw JSON
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("raw unmarshal: %v", err)
	}
	for _, key := range []string{"context_drift", "ecosystem_health", "security", "dispatch_signals", "previous_snapshot", "deltas_since_previous"} {
		if _, present := raw[key]; !present {
			t.Errorf("key %s missing from JSON", key)
		}
		if raw[key] != nil {
			t.Errorf("key %s should be JSON null, got %v", key, raw[key])
		}
	}
}

func TestSelfState_RetentionPruning(t *testing.T) {
	root := t.TempDir()
	stateDir := filepath.Join(root, ".claude", "state")
	archivesDir := filepath.Join(stateDir, "archives")
	if err := os.MkdirAll(archivesDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	// Create 15 snapshots with progressively newer mtimes
	base := time.Now().Add(-15 * time.Hour)
	for i := 0; i < 15; i++ {
		ts := base.Add(time.Duration(i) * time.Hour)
		name := snapshotFilename(ts)
		path := filepath.Join(stateDir, name)
		if err := os.WriteFile(path, []byte(`{}`), 0644); err != nil {
			t.Fatalf("write: %v", err)
		}
		if err := os.Chtimes(path, ts, ts); err != nil {
			t.Fatalf("chtimes: %v", err)
		}
	}

	// 2 archive files (must NOT be touched)
	a1 := filepath.Join(archivesDir, "state-2025-01-01T00-00-00Z.json")
	a2 := filepath.Join(archivesDir, "state-2025-02-01T00-00-00Z.json")
	if err := os.WriteFile(a1, []byte("a1"), 0644); err != nil {
		t.Fatalf("archive write: %v", err)
	}
	if err := os.WriteFile(a2, []byte("a2"), 0644); err != nil {
		t.Fatalf("archive write: %v", err)
	}

	// state-latest.json (must NOT be touched)
	latestPath := filepath.Join(stateDir, "state-latest.json")
	if err := os.WriteFile(latestPath, []byte("latest"), 0644); err != nil {
		t.Fatalf("latest write: %v", err)
	}

	if err := pruneSnapshots(stateDir, 12); err != nil {
		t.Fatalf("prune: %v", err)
	}

	// Count surviving snapshots in main dir
	entries, _ := os.ReadDir(stateDir)
	count := 0
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasPrefix(name, "state-") || !strings.HasSuffix(name, ".json") {
			continue
		}
		if name == "state-latest.json" {
			continue
		}
		count++
	}
	if count != 12 {
		t.Errorf("expected 12 surviving snapshots, got %d", count)
	}

	// state-latest.json must still exist
	if _, err := os.Stat(latestPath); err != nil {
		t.Errorf("state-latest.json removed by prune: %v", err)
	}

	// Archives must be untouched
	if _, err := os.Stat(a1); err != nil {
		t.Errorf("archive a1 removed: %v", err)
	}
	if _, err := os.Stat(a2); err != nil {
		t.Errorf("archive a2 removed: %v", err)
	}
}

func TestSelfState_SymlinkUpdate(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink semantics differ on Windows")
	}
	root := t.TempDir()
	stateDir := filepath.Join(root, ".claude", "state")
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	target := "state-2026-05-08T03-23-45Z.json"
	if err := os.WriteFile(filepath.Join(stateDir, target), []byte(`{"x":1}`), 0644); err != nil {
		t.Fatalf("write target: %v", err)
	}

	if err := updateLatestPointer(stateDir, target); err != nil {
		t.Fatalf("updateLatestPointer: %v", err)
	}

	pointerPath := filepath.Join(stateDir, "state-latest.json")
	info, err := os.Lstat(pointerPath)
	if err != nil {
		t.Fatalf("lstat pointer: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatal("expected symlink, got regular file")
	}
	resolved, err := os.Readlink(pointerPath)
	if err != nil {
		t.Fatalf("readlink: %v", err)
	}
	if resolved != target {
		t.Errorf("symlink target = %q, want %q", resolved, target)
	}

	// Update again with a new target — atomic replace must succeed.
	target2 := "state-2026-05-09T00-00-00Z.json"
	if err := os.WriteFile(filepath.Join(stateDir, target2), []byte(`{"x":2}`), 0644); err != nil {
		t.Fatalf("write target2: %v", err)
	}
	if err := updateLatestPointer(stateDir, target2); err != nil {
		t.Fatalf("second updateLatestPointer: %v", err)
	}
	resolved2, _ := os.Readlink(pointerPath)
	if resolved2 != target2 {
		t.Errorf("after replace, symlink target = %q, want %q", resolved2, target2)
	}
}

func TestSelfState_SymlinkFallbackToCopy(t *testing.T) {
	root := t.TempDir()
	stateDir := filepath.Join(root, ".claude", "state")
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	target := "state-2026-05-08T03-23-45Z.json"
	srcContent := []byte(`{"hello":"world"}`)
	if err := os.WriteFile(filepath.Join(stateDir, target), srcContent, 0644); err != nil {
		t.Fatalf("write target: %v", err)
	}

	// Direct call to fallback path — bypasses symlink-first logic.
	if err := copySnapshotAsLatest(stateDir, target); err != nil {
		t.Fatalf("copySnapshotAsLatest: %v", err)
	}

	pointerPath := filepath.Join(stateDir, "state-latest.json")
	got, err := os.ReadFile(pointerPath)
	if err != nil {
		t.Fatalf("read pointer: %v", err)
	}
	if string(got) != string(srcContent) {
		t.Errorf("pointer content = %q, want %q", string(got), string(srcContent))
	}

	// Pointer must be a regular file, not a symlink.
	info, err := os.Lstat(pointerPath)
	if err != nil {
		t.Fatalf("lstat: %v", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		t.Error("expected regular file, got symlink")
	}
}

func TestSelfState_QuickMode(t *testing.T) {
	root := t.TempDir()
	writeTool(t, root, "neo", "1.7.2-dev")
	writeTool(t, root, "morpheus", "1.7.2-dev")
	writeTool(t, root, "oracle", "1.7.2-dev")
	writeTool(t, root, "trinity", "1.7.2-dev")

	state, errs := collectSelfState(root, stubFetchers())
	state.CollectionErrors = errs

	if state.Mode != "quick" {
		t.Errorf("mode = %q, want quick", state.Mode)
	}
	if state.ContextDrift != nil {
		t.Errorf("context_drift should be JSON null in quick mode")
	}
	if state.EcosystemHealth != nil {
		t.Errorf("ecosystem_health should be JSON null")
	}
	if state.Security != nil {
		t.Errorf("security should be JSON null")
	}
	if state.DispatchSignals != nil {
		t.Errorf("dispatch_signals should be JSON null")
	}
	// Deterministic blocks must be populated
	if state.Matrix.Branch == "" {
		t.Error("matrix.branch should be populated in quick mode")
	}
	if len(state.Tools) != 4 {
		t.Errorf("tools count = %d, want 4 in quick mode", len(state.Tools))
	}
}

func TestSelfState_DetachedHEAD(t *testing.T) {
	root := t.TempDir()
	writeTool(t, root, "neo", "1.7.2-dev")
	writeTool(t, root, "morpheus", "1.7.2-dev")
	writeTool(t, root, "oracle", "1.7.2-dev")
	writeTool(t, root, "trinity", "1.7.2-dev")

	f := stubFetchers()
	f.gitBranch = func(string) (string, error) { return "(detached)", nil }

	state, _ := collectSelfState(root, f)

	if state.Matrix.Branch != "(detached)" {
		t.Errorf("branch = %q, want (detached)", state.Matrix.Branch)
	}
	// Should not crash — matrix block still populated otherwise.
	if state.Matrix.HeadSHA == "" {
		t.Error("head_sha empty even though branch is detached")
	}
}

func TestSelfState_SyncedWithRemote(t *testing.T) {
	root := t.TempDir()
	writeTool(t, root, "neo", "1.7.2-dev")
	writeTool(t, root, "morpheus", "1.7.2-dev")
	writeTool(t, root, "oracle", "1.7.2-dev")
	writeTool(t, root, "trinity", "1.7.2-dev")

	f := stubFetchers()
	f.gitAheadBehind = func(string) (int, int, error) { return 0, 0, nil }

	state, errs := collectSelfState(root, f)

	if !state.Matrix.SyncedWithRemote {
		t.Errorf("synced_with_remote = false, want true when ahead=0 behind=0")
	}
	if state.Matrix.BehindRemote != 0 {
		t.Errorf("behind_remote = %d, want 0", state.Matrix.BehindRemote)
	}
	if state.Matrix.AheadRemote != 0 {
		t.Errorf("ahead_remote = %d, want 0", state.Matrix.AheadRemote)
	}
	for _, e := range errs {
		if e.Section == "matrix" && e.Field == "synced_with_remote" {
			t.Errorf("unexpected sync collection error: %s", e.Error)
		}
	}
}

func TestSelfState_BehindRemote(t *testing.T) {
	root := t.TempDir()
	writeTool(t, root, "neo", "1.7.2-dev")
	writeTool(t, root, "morpheus", "1.7.2-dev")
	writeTool(t, root, "oracle", "1.7.2-dev")
	writeTool(t, root, "trinity", "1.7.2-dev")

	f := stubFetchers()
	f.gitAheadBehind = func(string) (int, int, error) { return 0, 3, nil }

	state, _ := collectSelfState(root, f)

	if state.Matrix.SyncedWithRemote {
		t.Errorf("synced_with_remote = true, want false when behind=3")
	}
	if state.Matrix.BehindRemote != 3 {
		t.Errorf("behind_remote = %d, want 3", state.Matrix.BehindRemote)
	}
	if state.Matrix.AheadRemote != 0 {
		t.Errorf("ahead_remote = %d, want 0", state.Matrix.AheadRemote)
	}
}

func TestSelfState_NoUpstream(t *testing.T) {
	root := t.TempDir()
	writeTool(t, root, "neo", "1.7.2-dev")
	writeTool(t, root, "morpheus", "1.7.2-dev")
	writeTool(t, root, "oracle", "1.7.2-dev")
	writeTool(t, root, "trinity", "1.7.2-dev")

	f := stubFetchers()
	f.gitAheadBehind = func(string) (int, int, error) {
		return 0, 0, errors.New("no upstream tracking branch")
	}

	state, errs := collectSelfState(root, f)

	if state.Matrix.SyncedWithRemote {
		t.Errorf("synced_with_remote = true, want false when upstream lookup fails")
	}
	if state.Matrix.BehindRemote != 0 {
		t.Errorf("behind_remote = %d, want 0 on upstream error", state.Matrix.BehindRemote)
	}
	if state.Matrix.AheadRemote != 0 {
		t.Errorf("ahead_remote = %d, want 0 on upstream error", state.Matrix.AheadRemote)
	}
	found := false
	for _, e := range errs {
		if e.Section == "matrix" && e.Field == "synced_with_remote" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected collection_errors entry for matrix.synced_with_remote on upstream error")
	}
}

func TestSelfState_NoAgentMemoryDir(t *testing.T) {
	root := t.TempDir()
	writeTool(t, root, "neo", "1.7.2-dev")
	writeTool(t, root, "morpheus", "1.7.2-dev")
	writeTool(t, root, "oracle", "1.7.2-dev")
	writeTool(t, root, "trinity", "1.7.2-dev")

	state, errs := collectSelfState(root, stubFetchers())

	if state.AgentMemory == nil {
		t.Fatal("agent_memory should be a (possibly empty) map, not nil")
	}
	if len(state.AgentMemory) != 0 {
		t.Errorf("expected empty agent_memory, got %d entries", len(state.AgentMemory))
	}
	for _, e := range errs {
		if e.Section == "agent_memory" {
			t.Errorf("missing dir should not produce error, got %v", e)
		}
	}

	// Marshal and verify the JSON is `{}` not `null`.
	data, err := marshalIndent(state)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	mem, ok := raw["agent_memory"]
	if !ok {
		t.Fatal("agent_memory key missing")
	}
	memMap, ok := mem.(map[string]any)
	if !ok {
		t.Fatalf("agent_memory should be object, got %T", mem)
	}
	if len(memMap) != 0 {
		t.Errorf("agent_memory should be empty object, got %d keys", len(memMap))
	}
}

func TestSelfState_MissingToolBinary(t *testing.T) {
	root := t.TempDir()
	// Only create 3 of 4 tools — trinity missing.
	writeTool(t, root, "neo", "1.7.2-dev")
	writeTool(t, root, "morpheus", "1.7.2-dev")
	writeTool(t, root, "oracle", "1.7.2-dev")

	state, errs := collectSelfState(root, stubFetchers())

	tb, ok := state.Tools["trinity"]
	if !ok {
		t.Fatal("trinity key missing from tools map")
	}
	if tb != nil {
		t.Errorf("trinity should be nil when cmd/trinity/main.go absent, got %+v", tb)
	}

	// collection_errors must include the trinity tool failure
	found := false
	for _, e := range errs {
		if e.Section == "tools" && e.Field == "trinity" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected collection_errors entry for missing trinity tool")
	}
}

func TestSelfState_MalformedPreviousSnapshot(t *testing.T) {
	root := t.TempDir()
	stateDir := filepath.Join(root, ".claude", "state")
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	// Write a malformed JSON file as the pointer (no symlink) — copy fallback path.
	pointerPath := filepath.Join(stateDir, "state-latest.json")
	if err := os.WriteFile(pointerPath, []byte("{not valid json"), 0644); err != nil {
		t.Fatalf("write pointer: %v", err)
	}

	writeTool(t, root, "neo", "1.7.2-dev")
	writeTool(t, root, "morpheus", "1.7.2-dev")
	writeTool(t, root, "oracle", "1.7.2-dev")
	writeTool(t, root, "trinity", "1.7.2-dev")

	state, errs := collectSelfState(root, stubFetchers())

	if state.PreviousSnapshot != nil {
		t.Errorf("previous_snapshot should be nil on parse error, got %v", *state.PreviousSnapshot)
	}
	if state.DeltasSincePrev != nil {
		t.Error("deltas_since_previous should be nil when previous fails to parse")
	}
	// collection_errors are returned as the second value — check errs directly.
	found := false
	for _, e := range errs {
		if e.Section == "previous_snapshot" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected collection_errors entry for previous_snapshot parse failure")
	}
}

func TestSelfState_GhNotAvailable(t *testing.T) {
	root := t.TempDir()
	writeTool(t, root, "neo", "1.7.2-dev")
	writeTool(t, root, "morpheus", "1.7.2-dev")
	writeTool(t, root, "oracle", "1.7.2-dev")
	writeTool(t, root, "trinity", "1.7.2-dev")

	f := stubFetchers()
	f.prCount = func(string) (int, error) { return 0, errors.New("gh: command not found") }
	f.issueCount = func(string) (int, error) { return 0, errors.New("gh: command not found") }

	state, errs := collectSelfState(root, f)

	if state.Backlog.OpenPRs != nil {
		t.Errorf("open_prs should be nil on gh failure, got %v", *state.Backlog.OpenPRs)
	}
	if state.Backlog.OpenIssues != nil {
		t.Errorf("open_issues should be nil on gh failure, got %v", *state.Backlog.OpenIssues)
	}
	prFound := false
	issueFound := false
	for _, e := range errs {
		if e.Section == "backlog" && e.Field == "open_prs" {
			prFound = true
		}
		if e.Section == "backlog" && e.Field == "open_issues" {
			issueFound = true
		}
	}
	if !prFound {
		t.Error("missing collection_errors entry for open_prs")
	}
	if !issueFound {
		t.Error("missing collection_errors entry for open_issues")
	}
}

func TestSelfState_DispatchSignalIDDeterministic(t *testing.T) {
	// Same inputs must produce the same ID every time.
	id1 := dispatchSignalID("stale_doc", ".claude/SERVICE_CONTEXT.md")
	id2 := dispatchSignalID("stale_doc", ".claude/SERVICE_CONTEXT.md")
	if id1 != id2 {
		t.Errorf("dispatchSignalID not deterministic: %q != %q", id1, id2)
	}
	// Different inputs must produce different IDs.
	id3 := dispatchSignalID("stale_doc", ".claude/NEXT_STEPS.md")
	if id1 == id3 {
		t.Errorf("expected different IDs for different targets: both %q", id1)
	}
	// ID length must be exactly 12 hex chars.
	if len(id1) != 12 {
		t.Errorf("expected 12-char ID, got %d chars: %q", len(id1), id1)
	}
	// Separator change (\x00 not |): kind "a|b" + target "c" must differ from kind "a" + target "b|c".
	idA := dispatchSignalID("a|b", "c")
	idB := dispatchSignalID("a", "b|c")
	if idA == idB {
		t.Errorf("pipe-in-kind and pipe-in-target produce same ID — separator not effective")
	}
}

func TestSelfState_PruneNoArchives(t *testing.T) {
	root := t.TempDir()
	stateDir := filepath.Join(root, ".claude", "state")
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	// 5 snapshots, fewer than retention — no-op expected
	base := time.Now().Add(-5 * time.Hour)
	var names []string
	for i := 0; i < 5; i++ {
		ts := base.Add(time.Duration(i) * time.Hour)
		name := snapshotFilename(ts)
		path := filepath.Join(stateDir, name)
		if err := os.WriteFile(path, []byte(`{}`), 0644); err != nil {
			t.Fatalf("write: %v", err)
		}
		if err := os.Chtimes(path, ts, ts); err != nil {
			t.Fatalf("chtimes: %v", err)
		}
		names = append(names, name)
	}

	if err := pruneSnapshots(stateDir, 12); err != nil {
		t.Fatalf("prune: %v", err)
	}

	for _, name := range names {
		if _, err := os.Stat(filepath.Join(stateDir, name)); err != nil {
			t.Errorf("snapshot %s removed when count < retention: %v", name, err)
		}
	}
}
