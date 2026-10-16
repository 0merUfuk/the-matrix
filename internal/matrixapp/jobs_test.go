package matrixapp

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

// Fake oracle and morp write a loop.sh that behaves like the real loops
// (status file, research files, review gate, STOP file, round lines) without
// calling any agent. The fake claude only has to exist.
const fakeOracle = `#!/bin/bash
if [ "$1" = --version ]; then echo "oracle version 9.9.9"; exit 0; fi
echo "ran oracle $*"
[ "$1" = research ] || exit 0
while [ $# -gt 0 ]; do case "$1" in --config) cfg=$2; shift;; --output-dir) out=$2; shift;; esac; shift; done
stack=$(sed -n 's/.*"stackName": *"\([^"]*\)".*/\1/p' "$cfg")
mkdir -p "$out/.autonomous" "$out/tasks"
cp "$cfg" "$out/stack.json"
cat > "$out/.autonomous/loop.sh" <<EOS
#!/bin/bash
cd "\$(dirname "\$0")/.."
mkdir -p tasks/research output/$stack
echo RESEARCH_CYCLE_1 > tasks/status.txt
echo "# Core principles" > tasks/research/00-core-principles.md
echo "# Testing" > tasks/research/05-testing.md
sleep \${FAKE_RESEARCH_SECONDS:-0}
echo RESEARCH_COMPLETE > tasks/status.txt
while [ ! -f tasks/SYNTHESIZE_APPROVED ]; do
  if [ -f tasks/STOP ]; then echo STOPPED_BY_USER > tasks/status.txt; exit 0; fi
  sleep 0.05
done
echo SYNTHESIS_RUNNING > tasks/status.txt
echo "# doc" > output/$stack/00-core-principles.md
echo SYNTHESIS_COMPLETE > tasks/status.txt
EOS
`

const fakeMorp = `#!/bin/bash
if [ "$1" = --version ]; then echo "morp version 9.9.9"; exit 0; fi
echo "ran morp $*"
[ "$1" = loop ] || exit 0
while [ $# -gt 0 ]; do case "$1" in --path) dir=$2; shift;; --max-cycles) max=$2; shift;; esac; shift; done
mkdir -p "$dir/.autonomous" "$dir/tasks"
printf -- '- [x] plan\n- [ ] build\n' > "$dir/tasks/todo.md"
cat > "$dir/.autonomous/loop.sh" <<EOS
#!/bin/bash
cd "\$(dirname "\$0")/.."
if [ -n "\$FAKE_BUILD_SLEEP" ]; then sleep 30 & echo \$! > tasks/child.pid; wait; fi
echo "CYCLE 1 / $max"
echo "[t] ─── Cycle 1 | DEVELOPER | max-turns=30 ───"
if [ -n "\$FAKE_BUILD_PAUSE" ]; then sleep 1; fi
echo "Exit reason:      ALL_GATES_APPROVED"
EOS
`

func jobEnv(t *testing.T) (s *server, h http.Handler, home, project string) {
	t.Helper()
	home = t.TempDir()
	t.Setenv("HOME", home)
	bins := fakeBins(t)
	for name, body := range map[string]string{"oracle": fakeOracle, "morp": fakeMorp, "claude": "#!/bin/sh\nexit 0\n"} {
		if err := os.WriteFile(filepath.Join(bins, name), []byte(body), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	project = filepath.Join(home, "code", "shop")
	if err := os.MkdirAll(filepath.Join(project, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(project, "CLAUDE.md"), "# shop\n")
	write(t, filepath.Join(project, ".neo.json"), `{"stacks":[{"name":"go-service","language":"go","framework":"chi v5","archStyle":"layered","dataLayer":"none"}]}`)
	var err error
	if s, err = newServer(Options{MaxAgeMonths: 6, Version: "test", BinDir: bins}); err != nil {
		t.Fatal(err)
	}
	if h, err = s.handler(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.jobs.StopAll)
	return s, h, home, project
}

func write(t *testing.T, p, body string) {
	t.Helper()
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func project(t *testing.T, h http.Handler, dir string) projectView {
	t.Helper()
	var v projectView
	rec := call(h, http.MethodGet, "/api/project?path="+dir, nil, nil)
	if rec.Code != http.StatusOK || json.Unmarshal(rec.Body.Bytes(), &v) != nil {
		t.Fatalf("project: %d %s", rec.Code, rec.Body)
	}
	return v
}

func waitFor(t *testing.T, what string, ok func() bool) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if ok() {
			return
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", what)
}

func startJob(t *testing.T, h http.Handler, body map[string]any, want int) JobView {
	t.Helper()
	rec := call(h, http.MethodPost, "/api/jobs", body, nil)
	if rec.Code != want {
		t.Fatalf("start %v: got %d %s, want %d", body["job"], rec.Code, rec.Body, want)
	}
	var v JobView
	_ = json.Unmarshal(rec.Body.Bytes(), &v)
	return v
}

func TestResearchJobWithReviewThenAddKnowledge(t *testing.T) {
	_, h, home, dir := jobEnv(t)
	form := map[string]any{"stack": "go-service", "framework": "Go 1.24, chi v5", "app_type": "api-only", "review": true}

	startJob(t, h, map[string]any{"job": "research", "path": dir, "research": form}, http.StatusPreconditionRequired)
	bad := map[string]any{"stack": "../../etc", "framework": "Go", "app_type": "api-only"}
	startJob(t, h, map[string]any{"job": "research", "path": dir, "research": bad, "confirm": true}, http.StatusBadRequest)
	bad = map[string]any{"stack": "go-service", "framework": "Go", "app_type": "rocket"}
	startJob(t, h, map[string]any{"job": "research", "path": dir, "research": bad, "confirm": true}, http.StatusBadRequest)
	startJob(t, h, map[string]any{"job": "research", "path": "/etc", "research": form, "confirm": true}, http.StatusBadRequest)

	if rs := project(t, h, dir).Research; len(rs) != 1 || rs[0].Phase != "none" || rs[0].Defaults.Framework != "chi v5" {
		t.Fatalf("before research: %+v", rs)
	}
	job := startJob(t, h, map[string]any{"job": "research", "path": dir, "research": form, "confirm": true}, http.StatusAccepted)
	if job.Status != "running" || job.StepCount != 2 {
		t.Fatalf("job = %+v", job)
	}
	startJob(t, h, map[string]any{"job": "build", "path": dir, "build": map[string]any{"goal": "add a checkout page", "size": "small"}, "confirm": true}, http.StatusTooManyRequests)

	var rs ResearchState
	waitFor(t, "review gate", func() bool { rs = project(t, h, dir).Research[0]; return rs.Phase == "review" })
	if len(rs.Found) != 2 || rs.Found[0].Title != "Core Principles" {
		t.Fatalf("found = %+v", rs.Found)
	}
	ws := researchWorkspace(home, dir, "go-service")
	if !strings.HasPrefix(ws, filepath.Join(home, ".the-matrix", "research")+string(os.PathSeparator)) {
		t.Fatalf("workspace %s is outside ~/.the-matrix/research", ws)
	}
	var cfg map[string]any
	raw, _ := os.ReadFile(filepath.Join(ws, "stack.json"))
	if json.Unmarshal(raw, &cfg) != nil || cfg["stackName"] != "go-service" || cfg["frameworkVersion"] != "Go 1.24, chi v5" || cfg["archStyle"] != "layered" || cfg["stateApproach"] != "n/a" || cfg["dataLayer"] != "none" {
		t.Fatalf("stack context = %s", raw)
	}

	if rec := call(h, http.MethodGet, "/api/research/file?path="+dir+"&stack=go-service&name=05-testing.md", nil, nil); rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "# Testing") {
		t.Fatalf("read finding: %d %s", rec.Code, rec.Body)
	}
	for _, name := range []string{"../stack.json", "..%2Fstack.json", "status.txt", ""} {
		if rec := call(h, http.MethodGet, "/api/research/file?path="+dir+"&stack=go-service&name="+name, nil, nil); rec.Code != http.StatusNotFound {
			t.Fatalf("name %q: got %d", name, rec.Code)
		}
	}
	if rec := call(h, http.MethodPost, "/api/tasks", map[string]any{"task": "add-knowledge", "path": dir, "params": map[string]any{"stack": "go-service"}, "confirm": true}, nil); rec.Code != http.StatusConflict {
		t.Fatalf("add before docs: %d", rec.Code)
	}

	if rec := call(h, http.MethodPost, "/api/research/approve", map[string]any{"path": dir, "stack": "go-service"}, nil); rec.Code != http.StatusPreconditionRequired {
		t.Fatalf("approve without confirm: %d", rec.Code)
	}
	if rec := call(h, http.MethodPost, "/api/research/approve", map[string]any{"path": dir, "stack": "go-service", "confirm": true}, nil); rec.Code != http.StatusOK {
		t.Fatalf("approve: %d %s", rec.Code, rec.Body)
	}
	waitFor(t, "docs", func() bool { rs = project(t, h, dir).Research[0]; return rs.Phase == "ready" })
	if rs.Docs != 1 || rs.Job == nil || rs.Job.Status != "ok" || rs.Defaults.Framework != "Go 1.24, chi v5" {
		t.Fatalf("ready state = %+v", rs)
	}
	if rec := call(h, http.MethodPost, "/api/research/approve", map[string]any{"path": dir, "stack": "go-service", "confirm": true}, nil); rec.Code != http.StatusConflict {
		t.Fatalf("approve twice: %d", rec.Code)
	}

	rec := call(h, http.MethodPost, "/api/tasks", map[string]any{"task": "add-knowledge", "path": dir, "params": map[string]any{"stack": "go-service"}, "confirm": true}, nil)
	var run RunView
	if rec.Code != http.StatusAccepted || json.Unmarshal(rec.Body.Bytes(), &run) != nil {
		t.Fatalf("add knowledge: %d %s", rec.Code, rec.Body)
	}
	got := waitRun(t, h, run.ID).Output
	want := "ran trinity sync --from " + filepath.Join(ws, "output", "go-service") + " --to " + filepath.Join(dir, ".claude", "knowledge", "go-service") + " --path " + dir
	if !strings.Contains(got, want) {
		t.Fatalf("output %q, want %q", got, want)
	}
}

func TestResearchStoppedAtReviewSaysStopped(t *testing.T) {
	_, h, _, dir := jobEnv(t)
	form := map[string]any{"stack": "go-service", "framework": "Go", "app_type": "api-only", "review": true}
	job := startJob(t, h, map[string]any{"job": "research", "path": dir, "research": form, "confirm": true}, http.StatusAccepted)
	waitFor(t, "review gate", func() bool { return project(t, h, dir).Research[0].Phase == "review" })
	if rec := call(h, http.MethodPost, "/api/jobs/"+job.ID+"/stop", map[string]any{}, nil); rec.Code != http.StatusOK {
		t.Fatalf("stop: %d", rec.Code)
	}
	waitFor(t, "stopped", func() bool { return project(t, h, dir).Research[0].Phase == "stopped" })
}

func TestResearchWithoutReviewWritesDocsStraightAway(t *testing.T) {
	_, h, _, dir := jobEnv(t)
	form := map[string]any{"stack": "go-service", "framework": "Go", "app_type": "api-only", "review": false}
	job := startJob(t, h, map[string]any{"job": "research", "path": dir, "research": form, "confirm": true}, http.StatusAccepted)
	if job.StepCount != 3 {
		t.Fatalf("steps = %d, want 3 (prepare, skip review, research)", job.StepCount)
	}
	waitFor(t, "docs", func() bool { return project(t, h, dir).Research[0].Phase == "ready" })
}

func TestBuildJobPlansThenRuns(t *testing.T) {
	t.Setenv("FAKE_BUILD_PAUSE", "1")
	_, h, home, dir := jobEnv(t)
	goal := "Add a checkout page with card payments"

	startJob(t, h, map[string]any{"job": "build", "path": dir, "build": map[string]any{"goal": "short", "size": "small"}, "confirm": true}, http.StatusBadRequest)
	startJob(t, h, map[string]any{"job": "build", "path": dir, "build": map[string]any{"goal": goal, "size": "huge"}, "confirm": true}, http.StatusBadRequest)
	startJob(t, h, map[string]any{"job": "build", "path": dir, "build": map[string]any{"continue": true}, "confirm": true}, http.StatusBadRequest)
	startJob(t, h, map[string]any{"job": "nope", "path": dir, "confirm": true}, http.StatusNotFound)
	if err := os.Rename(filepath.Join(dir, ".git"), filepath.Join(dir, "git")); err != nil {
		t.Fatal(err)
	}
	startJob(t, h, map[string]any{"job": "build", "path": dir, "build": map[string]any{"goal": goal, "size": "small"}, "confirm": true}, http.StatusBadRequest)
	_ = os.Rename(filepath.Join(dir, "git"), filepath.Join(dir, ".git"))

	job := startJob(t, h, map[string]any{"job": "build", "path": dir, "build": map[string]any{"goal": goal, "size": "small"}, "confirm": true}, http.StatusAccepted)
	var b BuildState
	waitFor(t, "agents at work", func() bool { b = project(t, h, dir).Build; return b.Activity == "writing code" && b.Round == 1 })
	if cmds := strings.Join(b.Job.Commands, " "); strings.Contains(cmds, home) || !strings.Contains(cmds, "~/code/shop") {
		t.Fatalf("commands should show ~ paths: %s", cmds)
	}
	waitFor(t, "build", func() bool { b = project(t, h, dir).Build; return b.Job != nil && b.Job.Status != "running" })
	if b.Job.ID != job.ID || b.Job.Status != "ok" || b.Result != "approved" || b.Round != 1 || b.Rounds != 3 || b.Done != 1 || b.Total != 2 || b.Goal != goal {
		t.Fatalf("build state = %+v job = %+v", b, b.Job)
	}
	var full JobView
	_ = json.Unmarshal(call(h, http.MethodGet, "/api/jobs/"+job.ID+"?from=0", nil, nil).Body.Bytes(), &full)
	if !strings.Contains(full.Output, "ran morp loop --path "+dir+" --non-interactive --goal "+goal+" --max-cycles 3 --runtime claude") {
		t.Fatalf("output = %q", full.Output)
	}

	write(t, filepath.Join(dir, "tasks", "STOP"), "")
	again := startJob(t, h, map[string]any{"job": "build", "path": dir, "build": map[string]any{"continue": true}, "confirm": true}, http.StatusAccepted)
	waitFor(t, "continue", func() bool { b = project(t, h, dir).Build; return b.Job.ID == again.ID && b.Job.Status == "ok" })
	if exists(dir, "tasks/STOP") || b.Goal != goal || again.StepCount != 2 {
		t.Fatalf("continue: stop file left=%v goal=%q steps=%d", exists(dir, "tasks/STOP"), b.Goal, again.StepCount)
	}
}

func TestJobStopEndsTheWholeProcessGroup(t *testing.T) {
	t.Setenv("FAKE_BUILD_SLEEP", "1") // the fake loop waits on a child sleep
	_, h2, _, dir := jobEnv(t)
	job := startJob(t, h2, map[string]any{"job": "build", "path": dir, "build": map[string]any{"goal": "Add a checkout page", "size": "small"}, "confirm": true}, http.StatusAccepted)
	waitFor(t, "agents step", func() bool {
		var v JobView
		_ = json.Unmarshal(call(h2, http.MethodGet, "/api/jobs/"+job.ID, nil, nil).Body.Bytes(), &v)
		return v.Step == "Agents are building"
	})
	time.Sleep(200 * time.Millisecond)
	start := time.Now()
	if rec := call(h2, http.MethodPost, "/api/jobs/"+job.ID+"/stop", map[string]any{}, nil); rec.Code != http.StatusOK {
		t.Fatalf("stop: %d %s", rec.Code, rec.Body)
	}
	var v JobView
	waitFor(t, "stopped", func() bool {
		_ = json.Unmarshal(call(h2, http.MethodGet, "/api/jobs/"+job.ID, nil, nil).Body.Bytes(), &v)
		return v.Status != "running"
	})
	if v.Status != "stopped" || time.Since(start) > 5*time.Second {
		t.Fatalf("status %s after %s", v.Status, time.Since(start))
	}
	pid, err := strconv.Atoi(strings.TrimSpace(readTrim(filepath.Join(dir, "tasks", "child.pid"))))
	if err != nil {
		t.Fatalf("child pid: %v", err)
	}
	child, _ := os.FindProcess(pid)
	waitFor(t, "agent child to exit", func() bool { return child.Signal(syscall.Signal(0)) != nil })
	if rec := call(h2, http.MethodPost, "/api/jobs/"+job.ID+"/stop", map[string]any{}, nil); rec.Code != http.StatusConflict {
		t.Fatalf("stop twice: %d", rec.Code)
	}
}

func TestJobsSurviveRestartAndInterruptedLoopsAreHalted(t *testing.T) {
	s, h, home, dir := jobEnv(t)
	job := startJob(t, h, map[string]any{"job": "build", "path": dir, "build": map[string]any{"goal": "Add a checkout page", "size": "medium"}, "confirm": true}, http.StatusAccepted)
	waitFor(t, "build", func() bool { return !s.jobs.Get(job.ID).running() })

	// A job still marked running on disk, as after a crash.
	ghost := filepath.Join(home, ".the-matrix", "jobs", "0123456789abcdef")
	if err := os.MkdirAll(ghost, 0o700); err != nil {
		t.Fatal(err)
	}
	stop := filepath.Join(dir, "tasks", "STOP")
	raw, _ := json.Marshal(map[string]any{"id": "0123456789abcdef", "kind": "build", "project": dir, "status": "running", "stop_file": stop, "started": time.Now()})
	write(t, filepath.Join(ghost, jobStateName), string(raw))

	again, err := newServer(Options{BinDir: s.opts.BinDir})
	if err != nil {
		t.Fatal(err)
	}
	if j := again.jobs.Get(job.ID); j == nil || j.Status != "ok" || !strings.Contains(j.View(0, "").Output, "ran morp loop") {
		t.Fatalf("finished job not reloaded: %+v", j)
	}
	if g := again.jobs.Get("0123456789abcdef"); g == nil || g.Status != "interrupted" {
		t.Fatalf("ghost = %+v", g)
	}
	if !exists(dir, "tasks/STOP") {
		t.Fatal("interrupted loop was not asked to stop")
	}
	info, _ := os.Stat(filepath.Join(home, ".the-matrix", "jobs", job.ID, jobLogName))
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("log mode = %v", info.Mode().Perm())
	}
}
