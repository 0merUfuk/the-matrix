package trinity

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"time"
)

// SelfStateOpts configures the trinity self-state command.
type SelfStateOpts struct {
	ProjectDir string
	JSONOnly   bool   // --json: write to stdout, no persistence
	OutPath    string // --out: custom output path (overrides default)
	Prune      bool   // --prune: enforce retention after writing
}

// CollectionError represents a non-fatal collection failure.
type CollectionError struct {
	Section string `json:"section"`
	Field   string `json:"field"`
	Error   string `json:"error"`
}

// MatrixBlock describes the repo-level git/version state.
type MatrixBlock struct {
	Branch           string `json:"branch"`
	HeadSHA          string `json:"head_sha"`
	LatestTag        string `json:"latest_tag"`
	CommitsSinceTag  int    `json:"commits_since_tag"`
	Dirty            bool   `json:"dirty"`
	SyncedWithRemote bool   `json:"synced_with_remote"`
	BehindRemote     int    `json:"behind_remote"`
	AheadRemote      int    `json:"ahead_remote"`
}

// ToolBlock describes one tool's version + commands.
type ToolBlock struct {
	Version  string   `json:"version"`
	Commands []string `json:"commands"`
}

// MemoryBlock describes one agent's memory directory state.
type MemoryBlock struct {
	Files int `json:"files"`
	Lines int `json:"lines"`
}

// BacklogBlock counts pending work across surfaces.
type BacklogBlock struct {
	NextStepsPending   int  `json:"next_steps_pending"`
	KnownIssuesOpen    int  `json:"known_issues_open"`
	OpenPRs            *int `json:"open_prs"`
	OpenIssues         *int `json:"open_issues"`
	StaleKnowledgeDocs *int `json:"stale_knowledge_docs"`
}

// VersionChange records a tool version delta between two snapshots.
type VersionChange struct {
	Tool string `json:"tool"`
	From string `json:"from"`
	To   string `json:"to"`
}

// DeltasBlock describes change since the previous snapshot.
type DeltasBlock struct {
	NextStepsAdded          int             `json:"next_steps_added"`
	NextStepsClosed         int             `json:"next_steps_closed"`
	IssuesOpened            int             `json:"issues_opened"`
	IssuesClosed            int             `json:"issues_closed"`
	MemoryGrowthLines       int             `json:"memory_growth_lines"`
	MemoryFilesAdded        int             `json:"memory_files_added"`
	VersionChanges          []VersionChange `json:"version_changes"`
	DispatchSignalsResolved []string        `json:"dispatch_signals_resolved"`
}

// SelfStateV1 is the typed mirror of the state.v1 schema. Prose blocks
// (ContextDrift, EcosystemHealth, Security, DispatchSignals) are pointers so
// they can be explicit JSON null in --quick mode.
type SelfStateV1 struct {
	Schema           string                 `json:"schema"`
	Mode             string                 `json:"mode"`
	ProducedAt       string                 `json:"produced_at"`
	PreviousSnapshot *string                `json:"previous_snapshot"`
	Matrix           MatrixBlock            `json:"matrix"`
	Tools            map[string]*ToolBlock  `json:"tools"`
	ContextDrift     *json.RawMessage       `json:"context_drift"`
	EcosystemHealth  *json.RawMessage       `json:"ecosystem_health"`
	Security         *json.RawMessage       `json:"security"`
	Backlog          BacklogBlock           `json:"backlog"`
	AgentMemory      map[string]MemoryBlock `json:"agent_memory"`
	DeltasSincePrev  *DeltasBlock           `json:"deltas_since_previous"`
	DispatchSignals  *json.RawMessage       `json:"dispatch_signals"`
	CollectionErrors []CollectionError      `json:"collection_errors"`
}

// Fetcher dependencies — injected for tests.
type fetchers struct {
	gitBranch       func(root string) (string, error)
	gitHeadSHA      func(root string) (string, error)
	gitLatestTag    func(root string) (string, error)
	gitCommitsSince func(root, tag string) (int, error)
	gitDirty        func(root string) (bool, error)
	gitAheadBehind  func(root string) (ahead, behind int, err error)
	prCount         func(root string) (int, error)
	issueCount      func(root string) (int, error)
}

func defaultFetchers() *fetchers {
	return &fetchers{
		gitBranch:       gitBranch,
		gitHeadSHA:      gitHeadSHA,
		gitLatestTag:    gitLatestTag,
		gitCommitsSince: gitCommitsSinceTag,
		gitDirty:        gitDirty,
		gitAheadBehind:  gitAheadBehind,
		prCount:         ghPRCount,
		issueCount:      ghIssueCount,
	}
}

// canonicalToolCommands is the ground-truth command list per tool — kept
// aligned with CLAUDE.md and cmd/{tool}/main.go cobra registrations.
var canonicalToolCommands = map[string][]string{
	"neo":      {"init", "analyze", "doctor", "status", "update", "registry"},
	"morpheus": {"init", "loop", "doctor", "status", "finalize", "report"},
	"oracle":   {"research", "status", "inject", "update", "export"},
	"trinity":  {"health", "sync", "refresh", "self-state"},
}

// RunSelfState executes the self-state collector.
func RunSelfState(opts SelfStateOpts) error {
	root, err := filepath.Abs(opts.ProjectDir)
	if err != nil || opts.ProjectDir == "" {
		root, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("resolve project dir: %w", err)
		}
	}

	state, errs := collectSelfState(root, defaultFetchers())
	if errs == nil {
		errs = []CollectionError{}
	}
	state.CollectionErrors = errs

	data, err := marshalIndent(state)
	if err != nil {
		return fmt.Errorf("marshal state: %w", err)
	}

	if opts.JSONOnly {
		_, err := os.Stdout.Write(append(data, '\n'))
		return err
	}

	stateDir := filepath.Join(root, ".claude", "state")
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		return fmt.Errorf("mkdir state dir: %w", err)
	}

	filename := snapshotFilename(time.Now().UTC())
	outPath := opts.OutPath
	if outPath == "" {
		outPath = filepath.Join(stateDir, filename)
	}

	if err := os.WriteFile(outPath, data, 0644); err != nil {
		return fmt.Errorf("write snapshot: %w", err)
	}

	// Update state-latest.json pointer to point at the snapshot we just wrote.
	if opts.OutPath == "" {
		if err := updateLatestPointer(stateDir, filename); err != nil {
			return fmt.Errorf("update latest pointer: %w", err)
		}
	}

	if opts.Prune {
		if err := pruneSnapshots(stateDir, 12); err != nil {
			return fmt.Errorf("prune snapshots: %w", err)
		}
	}

	fmt.Printf("self-state snapshot written: %s\n", outPath)
	return nil
}

// collectSelfState assembles the full quick-mode snapshot. It is white-box
// testable with custom fetchers.
func collectSelfState(root string, f *fetchers) (SelfStateV1, []CollectionError) {
	var errs []CollectionError

	state := SelfStateV1{
		Schema:     "self-state.v1",
		Mode:       "quick",
		ProducedAt: time.Now().UTC().Format(time.RFC3339),
	}

	// matrix block
	state.Matrix = collectMatrix(root, f, &errs)

	// tools block
	state.Tools = collectTools(root, &errs)

	// agent_memory block
	state.AgentMemory = collectAgentMemory(root, &errs)

	// backlog block
	state.Backlog = collectBacklog(root, f, &errs)

	// previous snapshot (resolved BEFORE we update the pointer)
	prevName, prevState, prevErrs := readPreviousSnapshot(root)
	errs = append(errs, prevErrs...)
	if prevName != "" {
		s := prevName
		state.PreviousSnapshot = &s
	}

	// deltas
	if prevState != nil {
		d := computeDeltas(prevState, &state)
		state.DeltasSincePrev = &d
	}

	// quick mode: prose blocks are explicit JSON null (already nil pointers)
	return state, errs
}

// collectMatrix populates branch/sha/tag info via fetchers.
func collectMatrix(root string, f *fetchers, errs *[]CollectionError) MatrixBlock {
	m := MatrixBlock{}

	if v, err := f.gitBranch(root); err != nil {
		*errs = append(*errs, CollectionError{Section: "matrix", Field: "branch", Error: err.Error()})
	} else {
		m.Branch = v
	}

	if v, err := f.gitHeadSHA(root); err != nil {
		*errs = append(*errs, CollectionError{Section: "matrix", Field: "head_sha", Error: err.Error()})
	} else {
		m.HeadSHA = v
	}

	if v, err := f.gitLatestTag(root); err != nil {
		*errs = append(*errs, CollectionError{Section: "matrix", Field: "latest_tag", Error: err.Error()})
	} else {
		m.LatestTag = v
	}

	if m.LatestTag != "" {
		if n, err := f.gitCommitsSince(root, m.LatestTag); err != nil {
			*errs = append(*errs, CollectionError{Section: "matrix", Field: "commits_since_tag", Error: err.Error()})
		} else {
			m.CommitsSinceTag = n
		}
	}

	if d, err := f.gitDirty(root); err != nil {
		*errs = append(*errs, CollectionError{Section: "matrix", Field: "dirty", Error: err.Error()})
	} else {
		m.Dirty = d
	}

	// local-vs-origin sync state. If there is no upstream tracking branch the
	// fetcher returns an error — handle gracefully: synced=false, ahead/behind=0,
	// plus a collection error so the loop can refuse to dispatch from a snapshot
	// that cannot prove it is in sync with origin.
	if ahead, behind, err := f.gitAheadBehind(root); err != nil {
		*errs = append(*errs, CollectionError{Section: "matrix", Field: "synced_with_remote", Error: err.Error()})
		m.SyncedWithRemote = false
		m.AheadRemote = 0
		m.BehindRemote = 0
	} else {
		m.AheadRemote = ahead
		m.BehindRemote = behind
		m.SyncedWithRemote = ahead == 0 && behind == 0
	}

	return m
}

// collectTools walks cmd/{tool}/main.go for each tool, extracts the version
// literal and pairs it with the canonical command list.
func collectTools(root string, errs *[]CollectionError) map[string]*ToolBlock {
	out := make(map[string]*ToolBlock)
	for _, name := range []string{"neo", "morpheus", "oracle", "trinity"} {
		mainPath := filepath.Join(root, "cmd", name, "main.go")
		v, err := extractVersion(mainPath)
		if err != nil {
			out[name] = nil
			*errs = append(*errs, CollectionError{
				Section: "tools",
				Field:   name,
				Error:   err.Error(),
			})
			continue
		}
		cmds := canonicalToolCommands[name]
		out[name] = &ToolBlock{Version: v, Commands: append([]string(nil), cmds...)}
	}
	return out
}

var versionRe = regexp.MustCompile(`var\s+version\s*=\s*"([^"]+)"`)

// extractVersion reads `var version = "..."` from a Go source file.
func extractVersion(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	m := versionRe.FindSubmatch(b)
	if len(m) < 2 {
		return "", fmt.Errorf("no version literal in %s", path)
	}
	return string(m[1]), nil
}

// collectAgentMemory walks .claude/agent-memory/{name}/ and counts files+lines.
func collectAgentMemory(root string, errs *[]CollectionError) map[string]MemoryBlock {
	out := make(map[string]MemoryBlock)
	memDir := filepath.Join(root, ".claude", "agent-memory")
	entries, err := os.ReadDir(memDir)
	if err != nil {
		// Missing directory is acceptable (returns empty map). Only report
		// errors that aren't "not exist" so testing absent dirs is clean.
		if !os.IsNotExist(err) {
			*errs = append(*errs, CollectionError{
				Section: "agent_memory",
				Field:   "_root",
				Error:   err.Error(),
			})
		}
		return out
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		files, lines, err := countMemoryDir(filepath.Join(memDir, name))
		if err != nil {
			*errs = append(*errs, CollectionError{
				Section: "agent_memory",
				Field:   name,
				Error:   err.Error(),
			})
			continue
		}
		out[name] = MemoryBlock{Files: files, Lines: lines}
	}
	return out
}

// countMemoryDir returns count of .md files and total line count under dir.
func countMemoryDir(dir string) (int, int, error) {
	files := 0
	lines := 0
	err := filepath.WalkDir(dir, func(p string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(p, ".md") {
			return nil
		}
		files++
		n, err := lineCount(p)
		if err != nil {
			return err
		}
		lines += n
		return nil
	})
	return files, lines, err
}

// lineCount counts newline-delimited lines in a file.
func lineCount(path string) (int, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	s.Buffer(make([]byte, 64*1024), 1024*1024)
	n := 0
	for s.Scan() {
		n++
	}
	return n, s.Err()
}

// collectBacklog reads NEXT_STEPS.md, KNOWN_ISSUES.md, and queries gh.
func collectBacklog(root string, f *fetchers, errs *[]CollectionError) BacklogBlock {
	b := BacklogBlock{}

	// next_steps_pending: count `^- [ ]` lines
	nextStepsPath := filepath.Join(root, ".claude", "NEXT_STEPS.md")
	if n, err := countMatchingLines(nextStepsPath, regexp.MustCompile(`^- \[ \]`), nil); err != nil {
		if !os.IsNotExist(err) {
			*errs = append(*errs, CollectionError{Section: "backlog", Field: "next_steps_pending", Error: err.Error()})
		}
	} else {
		b.NextStepsPending = n
	}

	// known_issues_open: lines matching `^**Status**:` not containing "✅"
	issuesPath := filepath.Join(root, ".claude", "KNOWN_ISSUES.md")
	if n, err := countMatchingLines(
		issuesPath,
		regexp.MustCompile(`^\*\*Status\*\*:`),
		func(line string) bool { return !strings.Contains(line, "✅") },
	); err != nil {
		if !os.IsNotExist(err) {
			*errs = append(*errs, CollectionError{Section: "backlog", Field: "known_issues_open", Error: err.Error()})
		}
	} else {
		b.KnownIssuesOpen = n
	}

	// open_prs / open_issues: gh API; nil on failure
	if n, err := f.prCount(root); err != nil {
		*errs = append(*errs, CollectionError{Section: "backlog", Field: "open_prs", Error: err.Error()})
	} else {
		b.OpenPRs = &n
	}
	if n, err := f.issueCount(root); err != nil {
		*errs = append(*errs, CollectionError{Section: "backlog", Field: "open_issues", Error: err.Error()})
	} else {
		b.OpenIssues = &n
	}

	// stale_knowledge_docs: nil if .claude/knowledge/ absent; computed by walking
	// each subdirectory for docs older than 6 months.
	knowledgeDir := filepath.Join(root, ".claude", "knowledge")
	if _, err := os.Stat(knowledgeDir); err == nil {
		topicEntries, walkErr := os.ReadDir(knowledgeDir)
		if walkErr != nil {
			*errs = append(*errs, CollectionError{Section: "backlog", Field: "stale_knowledge_docs", Error: walkErr.Error()})
		} else {
			totalStale := 0
			for _, te := range topicEntries {
				if !te.IsDir() {
					continue
				}
				topicDir := filepath.Join(knowledgeDir, te.Name())
				docs := findStaleDocs(topicDir, 6, nil)
				totalStale += len(docs)
			}
			b.StaleKnowledgeDocs = &totalStale
		}
	}

	return b
}

// countMatchingLines reads path line-by-line and counts lines matching re.
// If filterFn is non-nil, only lines for which filterFn returns true are counted.
func countMatchingLines(path string, re *regexp.Regexp, filterFn func(string) bool) (int, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	s.Buffer(make([]byte, 64*1024), 1024*1024)
	n := 0
	for s.Scan() {
		line := s.Text()
		if !re.MatchString(line) {
			continue
		}
		if filterFn != nil && !filterFn(line) {
			continue
		}
		n++
	}
	return n, s.Err()
}

// readPreviousSnapshot resolves and decodes state-latest.json's target.
func readPreviousSnapshot(root string) (string, *SelfStateV1, []CollectionError) {
	stateDir := filepath.Join(root, ".claude", "state")
	pointerPath := filepath.Join(stateDir, "state-latest.json")

	info, err := os.Lstat(pointerPath)
	if err != nil {
		return "", nil, nil
	}

	var targetName string
	if info.Mode()&os.ModeSymlink != 0 {
		t, err := os.Readlink(pointerPath)
		if err != nil {
			return "", nil, []CollectionError{{Section: "previous_snapshot", Field: "readlink", Error: err.Error()}}
		}
		targetName = filepath.Base(t)
	} else {
		// Copy fallback: the pointer file *is* the snapshot. Read it.
		data, err := os.ReadFile(pointerPath)
		if err != nil {
			return "", nil, []CollectionError{{Section: "previous_snapshot", Field: "read", Error: err.Error()}}
		}
		// We don't have a name to report when the pointer is a copy — derive
		// from the snapshot's produced_at if parseable, else label as the pointer.
		var prev SelfStateV1
		if err := json.Unmarshal(data, &prev); err != nil {
			return "", nil, []CollectionError{{Section: "previous_snapshot", Field: "parse", Error: err.Error()}}
		}
		// Synthesize the canonical filename from the snapshot's produced_at.
		if t, perr := time.Parse(time.RFC3339, prev.ProducedAt); perr == nil {
			targetName = snapshotFilename(t.UTC())
		} else {
			targetName = "state-latest.json"
		}
		return targetName, &prev, nil
	}

	// Symlink case: read the target.
	targetPath := filepath.Join(stateDir, targetName)
	data, err := os.ReadFile(targetPath)
	if err != nil {
		return targetName, nil, []CollectionError{{Section: "previous_snapshot", Field: "read", Error: err.Error()}}
	}
	var prev SelfStateV1
	if err := json.Unmarshal(data, &prev); err != nil {
		return targetName, nil, []CollectionError{{Section: "previous_snapshot", Field: "parse", Error: err.Error()}}
	}
	return targetName, &prev, nil
}

// computeDeltas returns the delta object between previous and current snapshots.
// Only deterministic-block deltas are computed; prose-block deltas are the
// skill's responsibility.
func computeDeltas(prev, cur *SelfStateV1) DeltasBlock {
	d := DeltasBlock{
		VersionChanges:          []VersionChange{},
		DispatchSignalsResolved: []string{},
	}

	// Backlog deltas
	prevPending := prev.Backlog.NextStepsPending
	curPending := cur.Backlog.NextStepsPending
	if curPending > prevPending {
		d.NextStepsAdded = curPending - prevPending
	} else if curPending < prevPending {
		d.NextStepsClosed = prevPending - curPending
	}

	prevOpen := prev.Backlog.KnownIssuesOpen
	curOpen := cur.Backlog.KnownIssuesOpen
	if curOpen > prevOpen {
		d.IssuesOpened = curOpen - prevOpen
	} else if curOpen < prevOpen {
		d.IssuesClosed = prevOpen - curOpen
	}

	// Memory deltas
	prevFiles := 0
	prevLines := 0
	for _, m := range prev.AgentMemory {
		prevFiles += m.Files
		prevLines += m.Lines
	}
	curFiles := 0
	curLines := 0
	for _, m := range cur.AgentMemory {
		curFiles += m.Files
		curLines += m.Lines
	}
	if curFiles > prevFiles {
		d.MemoryFilesAdded = curFiles - prevFiles
	}
	if curLines > prevLines {
		d.MemoryGrowthLines = curLines - prevLines
	}

	// Version changes
	for tool, prevTool := range prev.Tools {
		curTool := cur.Tools[tool]
		if prevTool == nil || curTool == nil {
			continue
		}
		if prevTool.Version != curTool.Version {
			d.VersionChanges = append(d.VersionChanges, VersionChange{
				Tool: tool,
				From: prevTool.Version,
				To:   curTool.Version,
			})
		}
	}
	sort.Slice(d.VersionChanges, func(i, j int) bool {
		return d.VersionChanges[i].Tool < d.VersionChanges[j].Tool
	})

	return d
}

// snapshotFilename produces a lexicographically-sortable RFC3339-ish filename.
// Colons are replaced with hyphens for filesystem safety; UTC `Z` is explicit.
func snapshotFilename(t time.Time) string {
	stamp := t.UTC().Format("2006-01-02T15-04-05Z")
	return "state-" + stamp + ".json"
}

// updateLatestPointer points state-latest.json at targetFilename. Tries
// symlink first; falls back to copy on error. Atomic via os.Rename of a temp
// file or temp link.
func updateLatestPointer(stateDir, targetFilename string) error {
	finalPath := filepath.Join(stateDir, "state-latest.json")
	tmpPath := finalPath + ".tmp"

	// Clean any leftover temp from a prior failure.
	_ = os.Remove(tmpPath)

	// Try symlink first.
	if err := os.Symlink(targetFilename, tmpPath); err == nil {
		if err := os.Rename(tmpPath, finalPath); err == nil {
			return nil
		}
		_ = os.Remove(tmpPath)
	}

	// Symlink failed — fall back to copy.
	return copySnapshotAsLatest(stateDir, targetFilename)
}

// copySnapshotAsLatest copies the named snapshot to state-latest.json
// atomically via a temp file + rename.
func copySnapshotAsLatest(stateDir, targetFilename string) error {
	srcPath := filepath.Join(stateDir, targetFilename)
	finalPath := filepath.Join(stateDir, "state-latest.json")
	tmpPath := finalPath + ".tmp"

	src, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("open source snapshot: %w", err)
	}
	defer src.Close()

	dst, err := os.OpenFile(tmpPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("open temp pointer: %w", err)
	}
	if _, err := io.Copy(dst, src); err != nil {
		_ = dst.Close()
		_ = os.Remove(tmpPath)
		return fmt.Errorf("copy snapshot: %w", err)
	}
	if err := dst.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("close temp pointer: %w", err)
	}
	if err := os.Rename(tmpPath, finalPath); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("rename pointer: %w", err)
	}
	return nil
}

// pruneSnapshots keeps the N most recent state-*.json files in stateDir
// (ordered by filename descending), removing the rest. state-latest.json,
// state-quick-latest.json, and archives/ are always preserved.
func pruneSnapshots(stateDir string, keep int) error {
	entries, err := os.ReadDir(stateDir)
	if err != nil {
		return err
	}

	type snap struct {
		name string
	}
	var snaps []snap
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if name == "state-latest.json" || name == "state-quick-latest.json" || name == "README.md" || name == ".gitkeep" {
			continue
		}
		if !strings.HasPrefix(name, "state-") || !strings.HasSuffix(name, ".json") {
			continue
		}
		snaps = append(snaps, snap{name: name})
	}

	// Sort newest-first by filename (lexicographic descending). Snapshot filenames
	// use the format state-YYYY-MM-DDTHH-MM-SSZ.json which is lexicographically
	// sortable — avoids divergent mtimes from copied/restored files.
	sort.Slice(snaps, func(i, j int) bool { return snaps[i].name > snaps[j].name })

	for i, s := range snaps {
		if i < keep {
			continue
		}
		_ = os.Remove(filepath.Join(stateDir, s.name))
	}
	return nil
}

// marshalIndent returns the canonical JSON representation of state.
func marshalIndent(state SelfStateV1) ([]byte, error) {
	return json.MarshalIndent(state, "", "  ")
}

// dispatchSignalID returns the first 12 hex chars of sha256(kind + "\x00" + target).
// The null-byte separator prevents collisions when future kind values contain "|".
// Not used in quick-mode collection but documented as the canonical algorithm for
// Subagent D — see SKILL.md dispatch_signals.id section.
func dispatchSignalID(kind, target string) string {
	h := sha256.Sum256([]byte(kind + "\x00" + target))
	return hex.EncodeToString(h[:])[:12]
}

// --- Real fetcher implementations (excluded from unit tests via injection) --

func gitBranch(root string) (string, error) {
	out, err := runGit(root, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", err
	}
	branch := strings.TrimSpace(out)
	if branch == "HEAD" {
		return "(detached)", nil
	}
	return branch, nil
}

func gitHeadSHA(root string) (string, error) {
	out, err := runGit(root, "rev-parse", "HEAD")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

func gitLatestTag(root string) (string, error) {
	out, err := runGit(root, "describe", "--tags", "--abbrev=0")
	if err != nil {
		// No tags is non-fatal — return empty
		return "", nil
	}
	return strings.TrimSpace(out), nil
}

func gitCommitsSinceTag(root, tag string) (int, error) {
	if tag == "" {
		return 0, nil
	}
	out, err := runGit(root, "rev-list", "--count", tag+"..HEAD")
	if err != nil {
		return 0, err
	}
	var n int
	_, _ = fmt.Sscanf(strings.TrimSpace(out), "%d", &n)
	return n, nil
}

func gitDirty(root string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", "diff", "--quiet", "HEAD")
	cmd.Dir = root
	if err := cmd.Run(); err != nil {
		// Exit code 1 means there ARE diffs; treat that as dirty=true.
		if ee, ok := err.(*exec.ExitError); ok && ee.ExitCode() == 1 {
			return true, nil
		}
		return false, err
	}
	return false, nil
}

// gitAheadBehind reports how many commits local HEAD is ahead of and behind its
// upstream tracking branch (@{u}). It first resolves the upstream SHA via
// `git rev-parse @{u}` — if that fails (no upstream tracking branch), it returns
// an error so the caller can record "no upstream tracking branch" and mark the
// snapshot unsynced. Counts come from `git rev-list --count --left-right
// HEAD...@{u}`, where the left count is ahead and the right count is behind.
func gitAheadBehind(root string) (ahead, behind int, err error) {
	if _, err := runGit(root, "rev-parse", "@{u}"); err != nil {
		return 0, 0, fmt.Errorf("no upstream tracking branch")
	}
	out, err := runGit(root, "rev-list", "--count", "--left-right", "HEAD...@{u}")
	if err != nil {
		return 0, 0, err
	}
	fields := strings.Fields(strings.TrimSpace(out))
	if len(fields) != 2 {
		return 0, 0, fmt.Errorf("unexpected rev-list output: %q", strings.TrimSpace(out))
	}
	if _, err := fmt.Sscanf(fields[0], "%d", &ahead); err != nil {
		return 0, 0, fmt.Errorf("parse ahead count %q: %w", fields[0], err)
	}
	if _, err := fmt.Sscanf(fields[1], "%d", &behind); err != nil {
		return 0, 0, fmt.Errorf("parse behind count %q: %w", fields[1], err)
	}
	return ahead, behind, nil
}

func runGit(root string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func ghPRCount(root string) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "gh", "pr", "list", "--state", "open", "--json", "number", "--limit", "200")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return 0, err
	}
	return countJSONArray(out)
}

func ghIssueCount(root string) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "gh", "issue", "list", "--state", "open", "--json", "number", "--limit", "200")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return 0, err
	}
	return countJSONArray(out)
}

func countJSONArray(b []byte) (int, error) {
	var arr []map[string]any
	if err := json.Unmarshal(b, &arr); err != nil {
		return 0, err
	}
	return len(arr), nil
}

// runtimeIsWindows is a hook for tests; the real implementation is the
// straightforward runtime check.
var runtimeIsWindows = func() bool { return runtime.GOOS == "windows" }
