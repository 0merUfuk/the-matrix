package matrixapp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// fakeBins writes shell scripts named like the four tools that print their
// arguments, so tests see exactly what would have run.
func fakeBins(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for _, id := range toolIDs {
		script := "#!/bin/sh\nif [ \"$1\" = --version ]; then echo \"" + id + " version 9.9.9\"; exit 0; fi\n" +
			"echo \"ran " + id + " $*\"\nif [ \"$1\" = slow ]; then sleep 5; fi\n"
		if err := os.WriteFile(filepath.Join(dir, id), []byte(script), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// env sets up a fake home with one fresh folder and one set-up project.
func env(t *testing.T) (h http.Handler, home, fresh, setup string) {
	t.Helper()
	home = t.TempDir()
	t.Setenv("HOME", home)
	fresh = filepath.Join(home, "code", "fresh")
	setup = filepath.Join(home, "code", "ready")
	for _, d := range []string{fresh, filepath.Join(setup, ".claude", "knowledge", "go")} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(setup, "CLAUDE.md"), []byte("# ready\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	h, err := NewHandler(Options{MaxAgeMonths: 6, Version: "test", BinDir: fakeBins(t)})
	if err != nil {
		t.Fatal(err)
	}
	return h, home, fresh, setup
}

func call(h http.Handler, method, target string, body any, mutate func(*http.Request)) *httptest.ResponseRecorder {
	var raw string
	if body != nil {
		b, _ := json.Marshal(body)
		raw = string(b)
	}
	req := httptest.NewRequest(method, target, strings.NewReader(raw))
	req.Host = "127.0.0.1:7474"
	if method == http.MethodPost {
		req.Header.Set("Origin", "http://127.0.0.1:7474")
		req.Header.Set("Content-Type", "application/json")
	}
	if mutate != nil {
		mutate(req)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func waitRun(t *testing.T, h http.Handler, id string) RunView {
	t.Helper()
	var v RunView
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		_ = json.Unmarshal(call(h, http.MethodGet, "/api/tasks/"+id+"?from=0", nil, nil).Body.Bytes(), &v)
		if v.Status != "running" {
			return v
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("run %s did not finish", id)
	return v
}

func TestPresetsArePublicOnly(t *testing.T) {
	ps := presets()
	if len(ps) == 0 {
		t.Fatal("no presets")
	}
	for _, p := range ps {
		if p.ID == "internal-go-service" || p.Name == "" {
			t.Errorf("unexpected preset %+v", p)
		}
	}
}

func TestProjectListAddBrowseRemove(t *testing.T) {
	h, home, fresh, setup := env(t)
	for _, dir := range []string{fresh, "~/code/ready"} {
		if rec := call(h, http.MethodPost, "/api/projects", pathRequest{Path: dir}, nil); rec.Code != http.StatusOK {
			t.Fatalf("add %s: %d %s", dir, rec.Code, rec.Body)
		}
	}
	if rec := call(h, http.MethodPost, "/api/projects", pathRequest{Path: "/etc"}, nil); rec.Code != http.StatusBadRequest {
		t.Errorf("add /etc status = %d, want 400", rec.Code)
	}

	var list struct {
		Projects []projectSummary `json:"projects"`
	}
	_ = json.Unmarshal(call(h, http.MethodGet, "/api/projects", nil, nil).Body.Bytes(), &list)
	if len(list.Projects) != 2 || list.Projects[0].SetUp || !list.Projects[1].SetUp {
		t.Fatalf("projects = %+v", list.Projects)
	}
	if info, err := os.Stat(filepath.Join(home, ".the-matrix", "projects.json")); err != nil || info.Mode().Perm() != 0o600 {
		t.Errorf("projects.json mode = %v, %v; want 0600", info, err)
	}

	var fl folderList
	_ = json.Unmarshal(call(h, http.MethodGet, "/api/browse?path=~/code", nil, nil).Body.Bytes(), &fl)
	if len(fl.Folders) != 2 || fl.Folders[1].Name != "ready" || !fl.Folders[1].Project || fl.Parent != "~" {
		t.Errorf("browse = %+v", fl)
	}
	if rec := call(h, http.MethodGet, "/api/browse?path=/", nil, nil); rec.Code != http.StatusBadRequest {
		t.Errorf("browse / status = %d, want 400", rec.Code)
	}

	call(h, http.MethodPost, "/api/projects/remove", pathRequest{Path: "~/code/ready"}, nil)
	_ = json.Unmarshal(call(h, http.MethodGet, "/api/projects", nil, nil).Body.Bytes(), &list)
	if len(list.Projects) != 1 || !strings.HasSuffix(list.Projects[0].Path, "fresh") {
		t.Errorf("after remove = %+v", list.Projects)
	}
	_ = setup
}

func TestSetupRunsNeoInitOnlyOnFreshProjects(t *testing.T) {
	h, _, fresh, setup := env(t)
	good := TaskParams{Preset: "go-service", Runtime: "claude"}

	tests := []struct {
		name string
		req  taskRequest
		want int
	}{
		{"no confirm", taskRequest{Task: "setup", Path: fresh, Params: good}, http.StatusPreconditionRequired},
		{"bad preset", taskRequest{Task: "setup", Path: fresh, Params: TaskParams{Preset: "internal-go-service", Runtime: "claude"}, Confirm: true}, http.StatusBadRequest},
		{"bad runtime", taskRequest{Task: "setup", Path: fresh, Params: TaskParams{Preset: "go-service", Runtime: "gpt"}, Confirm: true}, http.StatusBadRequest},
		{"already set up", taskRequest{Task: "setup", Path: setup, Params: good, Confirm: true}, http.StatusConflict},
		{"check before setup", taskRequest{Task: "check-setup", Path: fresh}, http.StatusConflict},
		{"sync without research", taskRequest{Task: "sync", Path: setup, Confirm: true}, http.StatusConflict},
		{"unknown task", taskRequest{Task: "init", Path: fresh}, http.StatusNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if rec := call(h, http.MethodPost, "/api/tasks", tt.req, nil); rec.Code != tt.want {
				t.Errorf("status = %d, want %d: %s", rec.Code, tt.want, rec.Body)
			}
		})
	}

	rec := call(h, http.MethodPost, "/api/tasks", taskRequest{Task: "setup", Path: fresh, Params: good, Confirm: true}, nil)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("setup status = %d: %s", rec.Code, rec.Body)
	}
	var run RunView
	_ = json.Unmarshal(rec.Body.Bytes(), &run)
	done := waitRun(t, h, run.ID)
	want := "ran neo init --preset go-service --path " + fresh + " --runtime claude"
	if done.Status != "ok" || !strings.Contains(done.Output, want) {
		t.Errorf("run = %+v, want output %q", done, want)
	}
	if done.Command != "neo init --preset go-service --path ~/code/fresh --runtime claude" || done.Path != "~/code/fresh" {
		t.Errorf("display = %q in %q", done.Command, done.Path)
	}

	replace := call(h, http.MethodPost, "/api/tasks", taskRequest{Task: "setup", Path: setup, Params: TaskParams{Preset: "go-service", Runtime: "codex", Replace: true}, Confirm: true}, nil)
	if replace.Code != http.StatusAccepted {
		t.Errorf("replace status = %d: %s", replace.Code, replace.Body)
	}
}

func TestSyncWhenResearchIsReady(t *testing.T) {
	h, _, _, setup := env(t)
	cfg := `{"stacks":[{"name":"go","from":"oracle-output/go/","to":".claude/knowledge/go/"}]}`
	if err := os.WriteFile(filepath.Join(setup, ".trinity.json"), []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := researchReady(setup); len(got) != 0 {
		t.Fatalf("researchReady before output = %v", got)
	}
	out := filepath.Join(setup, "oracle-output", "go")
	_ = os.MkdirAll(out, 0o755)
	_ = os.WriteFile(filepath.Join(out, "01-doc.md"), []byte("x"), 0o644)
	if got := researchReady(setup); len(got) != 1 || got[0] != "go" {
		t.Fatalf("researchReady = %v, want [go]", got)
	}
	var pv projectView
	_ = json.Unmarshal(call(h, http.MethodGet, "/api/project?path="+setup, nil, nil).Body.Bytes(), &pv)
	if !pv.SetUp || len(pv.ResearchReady) != 1 || pv.Tools["neo"].Version != "9.9.9" || len(pv.Presets) == 0 {
		t.Errorf("project view = %+v", pv)
	}
	if rec := call(h, http.MethodPost, "/api/tasks", taskRequest{Task: "sync", Path: setup, Confirm: true}, nil); rec.Code != http.StatusAccepted {
		t.Errorf("sync status = %d: %s", rec.Code, rec.Body)
	}
}

func TestRequestGuards(t *testing.T) {
	h, _, fresh, _ := env(t)
	body := taskRequest{Task: "check-setup", Path: fresh}
	for name, mutate := range map[string]func(*http.Request){
		"cross-origin":   func(r *http.Request) { r.Header.Set("Origin", "https://evil.example") },
		"form post":      func(r *http.Request) { r.Header.Set("Content-Type", "application/x-www-form-urlencoded") },
		"rebinding host": func(r *http.Request) { r.Host = "evil.example"; r.Header.Set("Origin", "http://evil.example") },
	} {
		if rec := call(h, http.MethodPost, "/api/tasks", body, mutate); rec.Code != http.StatusForbidden {
			t.Errorf("%s: status = %d, want 403", name, rec.Code)
		}
	}
	if rec := call(h, http.MethodDelete, "/api/projects", nil, nil); rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("DELETE status = %d, want 405", rec.Code)
	}
	rec := call(h, http.MethodGet, "/", nil, nil)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Header().Get("Content-Security-Policy"), "default-src 'self'") {
		t.Errorf("index = %d, CSP %q", rec.Code, rec.Header().Get("Content-Security-Policy"))
	}
}

func TestRunnerCancelAndBusy(t *testing.T) {
	bin := filepath.Join(fakeBins(t), "neo")
	var rn Runner
	dir := t.TempDir()
	a, _ := rn.Start("neo", "x", "slow", "neo slow", bin, []string{"slow"}, dir, dir)
	b, _ := rn.Start("neo", "x", "slow", "neo slow", bin, []string{"slow"}, dir, dir)
	if _, err := rn.Start("neo", "x", "slow", "neo slow", bin, []string{"slow"}, dir, dir); err != errBusy {
		t.Errorf("third run err = %v, want errBusy", err)
	}
	for _, r := range []*Run{a, b} {
		if !rn.Cancel(r.ID) {
			t.Fatalf("cancel %s failed", r.ID)
		}
	}
	deadline := time.Now().Add(5 * time.Second)
	for a.status() == "running" && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	if a.status() != "canceled" {
		t.Errorf("status = %s, want canceled", a.status())
	}
}

func TestServeRejectsNonLoopbackAndStops(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	if err := Serve(context.Background(), Options{Addr: "0.0.0.0:0"}); err == nil || !strings.Contains(err.Error(), "only listens on this computer") {
		t.Errorf("err = %v, want loopback-only rejection", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- Serve(ctx, Options{Addr: "127.0.0.1:0", BinDir: fakeBins(t)}) }()
	time.Sleep(100 * time.Millisecond)
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Serve returned %v after cancel", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Serve did not stop")
	}
}

// A symlink inside home must not lead outside it; one that stays inside works.
func TestResolveFollowsSymlinksForHomeConfinement(t *testing.T) {
	home, outside := t.TempDir(), t.TempDir()
	inside := filepath.Join(home, "code", "app")
	if err := os.MkdirAll(inside, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(home, "escape")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(inside, filepath.Join(home, "shortcut")); err != nil {
		t.Fatal(err)
	}

	if _, err := resolve(home, "~/escape"); err == nil || !strings.Contains(err.Error(), "inside your home folder") {
		t.Fatalf("link to a folder outside home: err = %v, want refusal", err)
	}
	got, err := resolve(home, "~/shortcut")
	if err != nil {
		t.Fatalf("link to a folder inside home: %v", err)
	}
	if got != filepath.Join(home, "shortcut") {
		t.Fatalf("resolve kept %q, want the path the user chose", got)
	}
	if _, err := resolve(home, "~/code/app"); err != nil {
		t.Fatalf("plain folder: %v", err)
	}
}
