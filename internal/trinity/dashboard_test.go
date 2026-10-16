package trinity

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCollectHealthMatchesCountsAndKnowledge(t *testing.T) {
	dir := createTestEcosystem(t)

	// Add a stale doc next to the fresh one from the fixture.
	stale := time.Now().AddDate(-1, 0, 0).Format(time.DateOnly)
	doc := "**Version**: 1.0\n**Created**: " + stale + "\n**Last Updated**: " + stale + "\n**Authors:** Test\n\n---\n\n" +
		strings.Repeat("Content line for testing purposes.\n", 50)
	writeFile(t, filepath.Join(dir, ".claude", "knowledge", "go-rules", "02-stale.md"), doc)

	report := CollectHealth(HealthOpts{ProjectDir: dir, MaxAgeMonths: 6})

	if report.Runtime != "claude" {
		t.Errorf("Runtime = %q, want claude", report.Runtime)
	}
	pass, warn, fail := 0, 0, 0
	for _, c := range report.Checks {
		switch c.Level {
		case "pass":
			pass++
		case "warn":
			warn++
		case "fail":
			fail++
		}
	}
	if pass != report.Pass || warn != report.Warn || fail != report.Fail {
		t.Errorf("check levels %d/%d/%d disagree with counts %d/%d/%d", pass, warn, fail, report.Pass, report.Warn, report.Fail)
	}
	if report.Fail != 0 {
		t.Errorf("Fail = %d, want 0: %+v", report.Fail, report.Checks)
	}

	if len(report.Knowledge) != 1 {
		t.Fatalf("Knowledge stacks = %d, want 1", len(report.Knowledge))
	}
	stack := report.Knowledge[0]
	if stack.Name != "go-rules" || !stack.HasIndex {
		t.Errorf("stack = %+v, want go-rules with index", stack)
	}
	statuses := map[string]string{}
	for _, d := range stack.Docs {
		statuses[d.File] = d.Status
	}
	if statuses["01-test-doc.md"] != "fresh" || statuses["02-stale.md"] != "stale" {
		t.Errorf("doc statuses = %v, want 01 fresh and 02 stale", statuses)
	}

	eco := report.Ecosystem
	if !eco.AgentsSupported || eco.AgentsExpected != len(ExpectedAgents) || eco.AgentsPresent != 0 {
		t.Errorf("Ecosystem = %+v, want agents supported, %d expected, 0 present", eco, len(ExpectedAgents))
	}
	if len(eco.Agents) != len(ExpectedAgents) || eco.Agents[0].Name != "manager" || eco.Agents[0].Present {
		t.Errorf("Ecosystem.Agents = %+v, want every expected agent listed as missing", eco.Agents)
	}
	if len(report.Hints) == 0 {
		t.Error("expected remediation hints when warnings exist")
	}
}

func TestCollectHealthMissingContextDir(t *testing.T) {
	report := CollectHealth(HealthOpts{ProjectDir: t.TempDir(), MaxAgeMonths: 6})
	if report.Fail != 1 || len(report.Checks) != 1 || report.Checks[0].Section != SectionRuntime {
		t.Errorf("report = %+v, want a single runtime failure", report)
	}
	if report.Knowledge == nil || report.Hints == nil {
		t.Error("empty slices must serialize as [], not null")
	}
}

func TestReadSnapshotHistory(t *testing.T) {
	stateDir := filepath.Join(t.TempDir(), "state")
	writeSnapshot(t, stateDir, "state-2026-01-01T00-00-00Z.json", 5, map[string]MemoryBlock{"a": {Files: 1, Lines: 10}})
	writeSnapshot(t, stateDir, "state-2026-02-01T00-00-00Z.json", 3, map[string]MemoryBlock{"a": {Files: 1, Lines: 10}, "b": {Files: 2, Lines: 5}})
	writeSnapshot(t, stateDir, "state-2026-03-01T00-00-00Z.json", 1, nil)
	writeFile(t, filepath.Join(stateDir, "state-latest.json"), "{}")
	writeFile(t, filepath.Join(stateDir, "README.md"), "# state")
	writeFile(t, filepath.Join(stateDir, "state-broken.json"), "not json")

	points, err := readSnapshotHistory(stateDir, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(points) != 2 {
		t.Fatalf("points = %d, want 2 (limit applied, newest kept)", len(points))
	}
	// state-broken sorts last and is skipped, leaving Feb and Mar oldest-first.
	if points[0].File != "state-2026-02-01T00-00-00Z.json" || points[1].File != "state-2026-03-01T00-00-00Z.json" {
		t.Errorf("points = %+v, want Feb then Mar", points)
	}

	all, _ := readSnapshotHistory(stateDir, 10)
	if len(all) != 3 {
		t.Fatalf("all points = %d, want 3", len(all))
	}
	if all[0].NextStepsPending != 5 || all[1].MemoryLines != 15 || all[2].NextStepsPending != 1 {
		t.Errorf("history = %+v", all)
	}

	missing, err := readSnapshotHistory(filepath.Join(t.TempDir(), "nope"), 10)
	if err != nil || missing != nil {
		t.Errorf("missing dir = %v, %v; want nil, nil", missing, err)
	}
}

func TestReadSyncLog(t *testing.T) {
	path := filepath.Join(t.TempDir(), "trinity-log.md")
	writeFile(t, path, "# Trinity Sync Log\n\n> header\n\n"+
		"2026-03-01 10:00 | sync | go → .claude/knowledge/go | 17 docs | 17 new\n"+
		"not a log line\n"+
		"2026-03-02 11:30 | sync | ts → .claude/knowledge/ts | 4 docs | 1 updated")

	entries, err := readSyncLog(path, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("entries = %d, want 2", len(entries))
	}
	if entries[0].When != "2026-03-02 11:30" || entries[0].Docs != "4 docs" || entries[0].Details != "1 updated" {
		t.Errorf("newest entry = %+v", entries[0])
	}

	limited, _ := readSyncLog(path, 1)
	if len(limited) != 1 || limited[0].When != "2026-03-02 11:30" {
		t.Errorf("limited = %+v", limited)
	}

	missing, err := readSyncLog(filepath.Join(t.TempDir(), "none.md"), 10)
	if err != nil || missing != nil {
		t.Errorf("missing log = %v, %v; want nil, nil", missing, err)
	}
}

func TestCollectDashboard(t *testing.T) {
	dir := createTestEcosystem(t)
	writeSnapshot(t, filepath.Join(dir, ".claude", "state"), "state-2026-01-01T00-00-00Z.json", 2, nil)
	writeFile(t, filepath.Join(dir, ".claude", "trinity-log.md"), "# log\n\n2026-03-01 10:00 | sync | go → x | 1 docs | 1 new")

	state := CollectDashboard(DashboardOpts{ProjectDir: dir, MaxAgeMonths: 6, Version: "test"})

	if state.Schema != DashboardSchema || state.Version != "test" || state.Project != filepath.Base(dir) {
		t.Errorf("header = %s %s %s", state.Schema, state.Version, state.Project)
	}
	if len(state.History) != 1 || len(state.SyncLog) != 1 {
		t.Errorf("history=%d synclog=%d, want 1 and 1", len(state.History), len(state.SyncLog))
	}
	// No state-latest.json pointer: self_state stays nil but history is listed.
	if state.SelfState != nil {
		t.Errorf("SelfState = %+v, want nil without a latest pointer", state.SelfState)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeSnapshot(t *testing.T, dir, name string, pending int, memory map[string]MemoryBlock) {
	t.Helper()
	s := SelfStateV1{Schema: "self-state.v1", ProducedAt: "2026-01-01T00:00:00Z", AgentMemory: memory}
	s.Backlog.NextStepsPending = pending
	data, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, name), string(data))
}
