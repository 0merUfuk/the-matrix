package trinity

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/0merUfuk/the-matrix/internal/cli"
	agentruntime "github.com/0merUfuk/the-matrix/internal/runtime"
)

// DashboardSchema identifies the ecosystem-state contract served by neo dashboard.
const DashboardSchema = "trinity-dashboard.v1"

const (
	maxHistorySnapshots = 24
	maxLogEntries       = 20
)

// DashboardOpts configures CollectDashboard.
type DashboardOpts struct {
	ProjectDir   string
	MaxAgeMonths float64
	Version      string
}

// SnapshotPoint is the trend-relevant subset of one self-state snapshot.
type SnapshotPoint struct {
	File             string `json:"file"`
	ProducedAt       string `json:"produced_at"`
	CommitsSinceTag  int    `json:"commits_since_tag"`
	NextStepsPending int    `json:"next_steps_pending"`
	KnownIssuesOpen  int    `json:"known_issues_open"`
	OpenPRs          *int   `json:"open_prs"`
	OpenIssues       *int   `json:"open_issues"`
	MemoryLines      int    `json:"memory_lines"`
}

// SyncLogEntry is one parsed line of trinity-log.md.
type SyncLogEntry struct {
	When    string `json:"when"`
	Action  string `json:"action"`
	Subject string `json:"subject"`
	Docs    string `json:"docs"`
	Details string `json:"details"`
}

// DashboardState is the read-only ecosystem state shown by neo dashboard.
type DashboardState struct {
	Schema       string          `json:"schema"`
	ProducedAt   string          `json:"produced_at"`
	Version      string          `json:"version"`
	Project      string          `json:"project"`
	Root         string          `json:"root"`
	Health       HealthReport    `json:"health"`
	SelfState    *SelfStateV1    `json:"self_state"`
	SnapshotFile string          `json:"snapshot_file"`
	History      []SnapshotPoint `json:"history"`
	SyncLog      []SyncLogEntry  `json:"sync_log"`
	Errors       []string        `json:"errors"`
}

// CollectDashboard gathers health, knowledge, self-state snapshots and the
// sync log for the project. It only reads; nothing is written.
func CollectDashboard(opts DashboardOpts) DashboardState {
	root := resolveRoot(opts.ProjectDir)
	health := CollectHealth(HealthOpts{ProjectDir: root, MaxAgeMonths: opts.MaxAgeMonths})
	health.Root = cli.PathToHome(health.Root)

	state := DashboardState{
		Schema:     DashboardSchema,
		ProducedAt: time.Now().UTC().Format(time.RFC3339),
		Version:    opts.Version,
		Project:    filepath.Base(root),
		Root:       cli.PathToHome(root),
		Health:     health,
		History:    []SnapshotPoint{},
		SyncLog:    []SyncLogEntry{},
		Errors:     []string{},
	}

	file, latest, errs := readPreviousSnapshot(root)
	for _, e := range errs {
		state.Errors = append(state.Errors, fmt.Sprintf("self-state %s: %s", e.Field, e.Error))
	}
	state.SelfState, state.SnapshotFile = latest, file

	history, err := readSnapshotHistory(filepath.Join(root, ".claude", "state"), maxHistorySnapshots)
	if err != nil {
		state.Errors = append(state.Errors, "snapshot history: "+err.Error())
	}
	if history != nil {
		state.History = history
	}

	rt := agentruntime.Registry.Get(health.Runtime)
	if rt == nil {
		rt = agentruntime.Registry.Default()
	}
	if logPath := trinityLogPath(root, rt.ContextFiles()); logPath != "" {
		entries, err := readSyncLog(logPath, maxLogEntries)
		if err != nil {
			state.Errors = append(state.Errors, "sync log: "+err.Error())
		}
		if entries != nil {
			state.SyncLog = entries
		}
	}

	return state
}

// isSnapshotFile reports whether name is a timestamped self-state snapshot
// (not a latest pointer or other state-dir file).
func isSnapshotFile(name string) bool {
	if name == "state-latest.json" || name == "state-quick-latest.json" {
		return false
	}
	return strings.HasPrefix(name, "state-") && strings.HasSuffix(name, ".json")
}

// readSnapshotHistory returns up to limit snapshots, oldest first. A missing
// state directory is not an error.
func readSnapshotHistory(stateDir string, limit int) ([]SnapshotPoint, error) {
	entries, err := os.ReadDir(stateDir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var names []string
	for _, e := range entries {
		if !e.IsDir() && isSnapshotFile(e.Name()) {
			names = append(names, e.Name())
		}
	}
	// Snapshot filenames sort lexicographically by time; walk newest first so
	// unreadable files do not eat into the limit.
	sort.Strings(names)

	var points []SnapshotPoint
	for i := len(names) - 1; i >= 0 && len(points) < limit; i-- {
		name := names[i]
		data, err := os.ReadFile(filepath.Join(stateDir, name)) // #nosec G304 -- snapshot files listed from the project's own .claude/state directory
		if err != nil {
			continue
		}
		var s SelfStateV1
		if err := json.Unmarshal(data, &s); err != nil {
			continue
		}
		memory := 0
		for _, m := range s.AgentMemory {
			memory += m.Lines
		}
		points = append(points, SnapshotPoint{
			File:             name,
			ProducedAt:       s.ProducedAt,
			CommitsSinceTag:  s.Matrix.CommitsSinceTag,
			NextStepsPending: s.Backlog.NextStepsPending,
			KnownIssuesOpen:  s.Backlog.KnownIssuesOpen,
			OpenPRs:          s.Backlog.OpenPRs,
			OpenIssues:       s.Backlog.OpenIssues,
			MemoryLines:      memory,
		})
	}
	for i, j := 0, len(points)-1; i < j; i, j = i+1, j-1 {
		points[i], points[j] = points[j], points[i]
	}
	return points, nil
}

// syncLogLine matches entries written by BuildLogEntry:
// "YYYY-MM-DD HH:MM | action | subject | N docs | details".
var syncLogLine = regexp.MustCompile(`^\d{4}-\d{2}-\d{2} \d{2}:\d{2} \|`)

// readSyncLog returns up to limit log entries, newest first. A missing log is
// not an error.
func readSyncLog(path string, limit int) ([]SyncLogEntry, error) {
	f, err := os.Open(path) // #nosec G304 -- the project's own trinity sync log
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var entries []SyncLogEntry
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024) // a long details line must not hide the whole history
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !syncLogLine.MatchString(line) {
			continue
		}
		parts := strings.Split(line, " | ")
		for len(parts) < 5 {
			parts = append(parts, "")
		}
		entries = append(entries, SyncLogEntry{
			When:    parts[0],
			Action:  parts[1],
			Subject: parts[2],
			Docs:    parts[3],
			Details: strings.Join(parts[4:], " | "),
		})
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// Newest first, bounded.
	for i, j := 0, len(entries)-1; i < j; i, j = i+1, j-1 {
		entries[i], entries[j] = entries[j], entries[i]
	}
	if len(entries) > limit {
		entries = entries[:limit]
	}
	return entries, nil
}
