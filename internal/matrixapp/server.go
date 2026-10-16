package matrixapp

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/0merUfuk/the-matrix/internal/cli"
	agentruntime "github.com/0merUfuk/the-matrix/internal/runtime"
	"github.com/0merUfuk/the-matrix/internal/trinity"
)

//go:embed web
var webAssets embed.FS

// DefaultAddr is the loopback address the app listens on.
const DefaultAddr = "127.0.0.1:7474"

var toolIDs = []string{"neo", "oracle", "morp", "trinity"}

// Options configures the app.
type Options struct {
	ProjectDir   string // optional project to add to the list at startup
	Addr         string // host:port; host must be loopback
	MaxAgeMonths float64
	Version      string
	Open         bool
	BinDir       string // where the tool binaries live; default: next to this executable, then PATH
}

type server struct {
	opts     Options
	home     string
	store    *projectStore
	runner   Runner
	jobs     *Jobs
	binOnce  sync.Once
	binaries map[string]toolBin
	path     string // PATH for commands: the tools' folders, then the usual places
}

type toolBin struct {
	Path    string `json:"-"`
	Version string `json:"version"`
}

// NewHandler builds the HTTP handler: embedded UI plus the JSON API.
func NewHandler(opts Options) (http.Handler, error) {
	s, err := newServer(opts)
	if err != nil {
		return nil, err
	}
	return s.handler()
}

func newServer(opts Options) (*server, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	s := &server{opts: opts, home: home, store: newProjectStore(home)}
	s.path = commandPath(home, s.tools())
	env := append(os.Environ(), "PATH="+s.path, "NO_COLOR=1", "CLICOLOR=0", "TERM=dumb")
	s.runner.env = env
	s.jobs = newJobs(home, env)
	if opts.ProjectDir != "" {
		abs, _ := filepath.Abs(opts.ProjectDir)
		if dir, err := resolve(home, abs); err == nil {
			_ = s.store.add(dir)
		}
	}
	return s, nil
}

func (s *server) handler() (http.Handler, error) {
	static, err := fs.Sub(webAssets, "web")
	if err != nil {
		return nil, err
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/projects", s.handleProjects)
	mux.HandleFunc("POST /api/projects", s.handleAddProject)
	mux.HandleFunc("POST /api/projects/remove", s.handleRemoveProject)
	mux.HandleFunc("GET /api/browse", s.handleBrowse)
	mux.HandleFunc("GET /api/project", s.handleProject)
	mux.HandleFunc("POST /api/tasks", s.handleTask)
	mux.HandleFunc("GET /api/tasks/{id}", s.handleTaskGet)
	mux.HandleFunc("POST /api/tasks/{id}/cancel", s.handleCancel)
	mux.HandleFunc("POST /api/jobs", s.handleJob)
	mux.HandleFunc("GET /api/jobs/{id}", s.handleJobGet)
	mux.HandleFunc("POST /api/jobs/{id}/stop", s.handleJobStop)
	mux.HandleFunc("POST /api/research/approve", s.handleApprove)
	mux.HandleFunc("GET /api/research/file", s.handleResearchFile)
	files := http.FileServerFS(static)
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		// Embedded files carry no modification time; revalidate so an upgrade shows the new UI.
		w.Header().Set("Cache-Control", "no-cache")
		files.ServeHTTP(w, r)
	})

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("Content-Security-Policy", "default-src 'self'; img-src 'self' data:; frame-ancestors 'none'; base-uri 'none'; form-action 'none'")
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "no-referrer")
		if !isLoopbackHost(r.Host) {
			http.Error(w, "forbidden: The Matrix only answers on this computer", http.StatusForbidden)
			return
		}
		if r.Method == http.MethodPost && !sameOriginJSON(r) {
			http.Error(w, "forbidden: cross-origin or non-JSON request", http.StatusForbidden)
			return
		}
		if r.Method != http.MethodGet && r.Method != http.MethodHead && r.Method != http.MethodPost {
			w.Header().Set("Allow", "GET, HEAD, POST")
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		mux.ServeHTTP(w, r)
	}), nil
}

// sameOriginJSON blocks cross-site form posts: browsers always send Origin on
// POST, and a JSON body cannot be sent cross-origin without a CORS preflight.
func sameOriginJSON(r *http.Request) bool {
	return r.Header.Get("Origin") == "http://"+r.Host &&
		strings.HasPrefix(r.Header.Get("Content-Type"), "application/json")
}

// isLoopbackHost reports whether a host or host:port names the local machine.
func isLoopbackHost(hostport string) bool {
	host := hostport
	if h, _, err := net.SplitHostPort(hostport); err == nil {
		host = h
	}
	host = strings.Trim(host, "[]")
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func (s *server) tools() map[string]toolBin {
	s.binOnce.Do(func() {
		dir := s.opts.BinDir
		if dir == "" {
			if exe, err := os.Executable(); err == nil {
				dir = filepath.Dir(exe)
			}
		}
		s.binaries = map[string]toolBin{}
		for _, id := range toolIDs {
			path := filepath.Join(dir, id)
			if info, err := os.Stat(path); err != nil || info.IsDir() {
				p, err := exec.LookPath(id)
				if err != nil {
					continue
				}
				path = p
			}
			s.binaries[id] = toolBin{Path: path, Version: version(path)}
		}
	})
	return s.binaries
}

// commandPath is the PATH given to every command: the folders the tools were
// found in, this process's PATH, then places Claude Code, Codex and Homebrew
// usually live (a Homebrew service starts with a very short PATH).
func commandPath(home string, bins map[string]toolBin) string {
	var dirs []string
	for _, id := range toolIDs {
		if b, ok := bins[id]; ok {
			dirs = append(dirs, filepath.Dir(b.Path))
		}
	}
	dirs = append(dirs, filepath.SplitList(os.Getenv("PATH"))...)
	dirs = append(dirs, filepath.Join(home, ".local", "bin"), filepath.Join(home, ".claude", "local"), "/opt/homebrew/bin", "/usr/local/bin", "/usr/bin", "/bin")
	seen := map[string]bool{}
	var out []string
	for _, d := range dirs {
		if d != "" && !seen[d] && isDir(d) {
			seen[d] = true
			out = append(out, d)
		}
	}
	return strings.Join(out, string(os.PathListSeparator))
}

// lookPath finds a command on the app's command PATH.
func (s *server) lookPath(name string) bool {
	for _, d := range filepath.SplitList(s.path) {
		if info, err := os.Stat(filepath.Join(d, name)); err == nil && !info.IsDir() && info.Mode()&0o111 != 0 { // #nosec G703 -- probes PATH entries for a fixed tool name, like exec.LookPath
			return true
		}
	}
	return false
}

// agentRuntime is the project's runtime and whether its CLI is installed.
func (s *server) agentRuntime(dir string) (name, display string, ok bool) {
	name = agentruntime.DetectRuntime(dir)
	display = map[string]string{"claude": "Claude Code", "codex": "Codex"}[name]
	if display == "" {
		return name, name, false
	}
	return name, display, s.lookPath(name)
}

func version(bin string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, bin, "--version").Output()
	if f := strings.Fields(string(out)); err == nil && len(f) > 0 {
		return f[len(f)-1]
	}
	return ""
}

// ── projects ────────────────────────────────────────────

type projectSummary struct {
	Name   string `json:"name"`
	Path   string `json:"path"`
	Exists bool   `json:"exists"`
	SetUp  bool   `json:"set_up"`
	Docs   int    `json:"docs"`
	Stale  int    `json:"stale"`
	Issues int    `json:"issues"`
}

func (s *server) summary(dir string) projectSummary {
	sum := projectSummary{Name: filepath.Base(dir), Path: cli.PathToHome(dir)}
	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		return sum
	}
	sum.Exists, sum.SetUp = true, isSetUp(dir)
	if !sum.SetUp {
		return sum
	}
	h := trinity.CollectHealth(trinity.HealthOpts{ProjectDir: dir, MaxAgeMonths: s.maxAge()})
	sum.Issues = h.Warn + h.Fail
	for _, st := range h.Knowledge {
		for _, d := range st.Docs {
			sum.Docs++
			if d.Status == "stale" {
				sum.Stale++
			}
		}
	}
	return sum
}

func (s *server) maxAge() float64 {
	if s.opts.MaxAgeMonths > 0 {
		return s.opts.MaxAgeMonths
	}
	return 6
}

func (s *server) handleProjects(w http.ResponseWriter, _ *http.Request) {
	out := []projectSummary{}
	for _, dir := range s.store.list() {
		out = append(out, s.summary(dir))
	}
	missing := []string{}
	for _, id := range toolIDs {
		if _, ok := s.tools()[id]; !ok {
			missing = append(missing, id)
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"projects": out, "missing_tools": missing, "version": s.opts.Version})
}

type pathRequest struct {
	Path string `json:"path"`
}

func (s *server) handleAddProject(w http.ResponseWriter, r *http.Request) {
	var req pathRequest
	if !decode(w, r, &req) {
		return
	}
	dir, err := resolve(s.home, req.Path)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errBody(err))
		return
	}
	if err := s.store.add(dir); err != nil {
		writeJSON(w, http.StatusInternalServerError, errBody(err))
		return
	}
	writeJSON(w, http.StatusOK, s.summary(dir))
}

func (s *server) handleRemoveProject(w http.ResponseWriter, r *http.Request) {
	var req pathRequest
	if !decode(w, r, &req) {
		return
	}
	dir := req.Path
	if strings.HasPrefix(dir, "~/") {
		dir = filepath.Join(s.home, dir[2:])
	}
	if err := s.store.remove(filepath.Clean(dir)); err != nil {
		writeJSON(w, http.StatusInternalServerError, errBody(err))
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"removed": true})
}

func (s *server) handleBrowse(w http.ResponseWriter, r *http.Request) {
	raw := r.URL.Query().Get("path")
	if raw == "" {
		raw = "~"
	}
	dir, err := resolve(s.home, raw)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errBody(err))
		return
	}
	list, err := browse(s.home, dir)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errBody(err))
		return
	}
	writeJSON(w, http.StatusOK, list)
}

type projectView struct {
	Name          string                 `json:"name"`
	Path          string                 `json:"path"`
	SetUp         bool                   `json:"set_up"`
	State         trinity.DashboardState `json:"state"`
	ResearchReady []string               `json:"research_ready"`
	Loop          bool                   `json:"loop"`
	Runtime       string                 `json:"runtime"`
	RuntimeName   string                 `json:"runtime_name"`
	RuntimeReady  bool                   `json:"runtime_ready"`
	Research      []ResearchState        `json:"research"`
	Build         BuildState             `json:"build"`
	Presets       []Preset               `json:"presets"`
	Tools         map[string]toolBin     `json:"tools"`
	Runs          []RunView              `json:"runs"`
}

func (s *server) handleProject(w http.ResponseWriter, r *http.Request) {
	dir, err := resolve(s.home, r.URL.Query().Get("path"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errBody(err))
		return
	}
	shown := cli.PathToHome(dir)
	runs := []RunView{}
	for _, v := range s.runner.Recent(30) {
		if v.Path == shown && len(runs) < 6 {
			runs = append(runs, v)
		}
	}
	rt, rtName, rtReady := s.agentRuntime(dir)
	writeJSON(w, http.StatusOK, projectView{
		Name: filepath.Base(dir), Path: shown, SetUp: isSetUp(dir),
		Runtime: rt, RuntimeName: rtName, RuntimeReady: rtReady,
		Research:      s.researchStates(dir),
		Build:         s.buildState(dir),
		State:         trinity.CollectDashboard(trinity.DashboardOpts{ProjectDir: dir, MaxAgeMonths: s.maxAge(), Version: s.opts.Version}),
		ResearchReady: emptyIfNil(researchReady(dir)),
		Loop:          exists(dir, ".autonomous/loop.sh"),
		Presets:       presets(),
		Tools:         s.tools(),
		Runs:          runs,
	})
}

// ── tasks ───────────────────────────────────────────────

type taskRequest struct {
	Task    string     `json:"task"`
	Path    string     `json:"path"`
	Params  TaskParams `json:"params"`
	Confirm bool       `json:"confirm"`
}

func (s *server) handleTask(w http.ResponseWriter, r *http.Request) {
	var req taskRequest
	if !decode(w, r, &req) {
		return
	}
	t, ok := tasks[req.Task]
	if !ok {
		writeJSON(w, http.StatusNotFound, errBody(errors.New("unknown task")))
		return
	}
	if t.write && !req.Confirm {
		writeJSON(w, http.StatusPreconditionRequired, errBody(errors.New("this changes files in your project; confirm it first")))
		return
	}
	dir, err := resolve(s.home, req.Path)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errBody(err))
		return
	}
	req.Params.home = s.home
	if err := t.ready(dir, req.Params); err != nil {
		writeJSON(w, http.StatusConflict, errBody(err))
		return
	}
	args, err := t.args(dir, req.Params)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errBody(err))
		return
	}
	bin, ok := s.tools()[t.tool]
	if !ok {
		writeJSON(w, http.StatusServiceUnavailable, errBody(fmt.Errorf("%s is not installed; run: brew install 0merUfuk/thematrix/%s", t.tool, t.tool)))
		return
	}
	shown := cli.PathToHome(dir)
	display := t.tool + " " + strings.ReplaceAll(strings.Join(args, " "), dir, shown)
	run, err := s.runner.Start(t.tool, req.Task, t.title, display, bin.Path, args, dir, shown)
	if err != nil {
		writeJSON(w, http.StatusTooManyRequests, errBody(err))
		return
	}
	writeJSON(w, http.StatusAccepted, run.View(0))
}

func (s *server) handleTaskGet(w http.ResponseWriter, r *http.Request) {
	run := s.runner.Get(r.PathValue("id"))
	if run == nil {
		writeJSON(w, http.StatusNotFound, errBody(errors.New("unknown task run")))
		return
	}
	from := 0
	_, _ = fmt.Sscanf(r.URL.Query().Get("from"), "%d", &from)
	writeJSON(w, http.StatusOK, run.View(from))
}

func (s *server) handleCancel(w http.ResponseWriter, r *http.Request) {
	if !s.runner.Cancel(r.PathValue("id")) {
		writeJSON(w, http.StatusConflict, errBody(errors.New("not running")))
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"canceled": true})
}

// ── helpers ─────────────────────────────────────────────

func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8<<10)).Decode(v); err != nil {
		writeJSON(w, http.StatusBadRequest, errBody(errors.New("invalid request")))
		return false
	}
	return true
}

func errBody(err error) map[string]string { return map[string]string{"error": err.Error()} }

func emptyIfNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

// Serve runs the app until ctx is cancelled.
func Serve(ctx context.Context, opts Options) error {
	if opts.Addr == "" {
		opts.Addr = DefaultAddr
	}
	if !isLoopbackHost(opts.Addr) {
		return fmt.Errorf("--addr %q: The Matrix only listens on this computer (127.0.0.1, ::1, localhost)", opts.Addr)
	}
	s, err := newServer(opts)
	if err != nil {
		return err
	}
	handler, err := s.handler()
	if err != nil {
		return err
	}
	// Long jobs spend the user's agent account; never leave them running
	// unseen after the app is closed.
	defer s.jobs.StopAll()
	ln, err := net.Listen("tcp", opts.Addr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w (is The Matrix already open? try --addr 127.0.0.1:0)", opts.Addr, err)
	}
	url := "http://" + ln.Addr().String() + "/"

	theme := cli.GetTheme()
	fmt.Print(theme.Banner("The Matrix", "local app"))
	fmt.Println(theme.KeyValue([]cli.KV{{Key: "Open", Value: url}, {Key: "Stop", Value: "Ctrl+C"}}))
	fmt.Println()
	if opts.Open {
		if err := openBrowser(url); err != nil {
			fmt.Fprintf(os.Stderr, "  warning: could not open the browser: %v\n", err)
		}
	}

	srv := &http.Server{Handler: handler, ReadHeaderTimeout: 5 * time.Second}
	errc := make(chan error, 1)
	go func() { errc <- srv.Serve(ln) }()
	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	case err := <-errc:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}

func openBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url) // #nosec G204 -- fixed opener with the app's own loopback URL as a single argument, no shell
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url) // #nosec G204 -- fixed opener with the app's own loopback URL as a single argument, no shell
	default:
		cmd = exec.Command("xdg-open", url) // #nosec G204 -- fixed opener with the app's own loopback URL as a single argument, no shell
	}
	return cmd.Start()
}

// ── long jobs ───────────────────────────────────────────

type jobRequest struct {
	Job      string         `json:"job"` // research | build
	Path     string         `json:"path"`
	Research ResearchParams `json:"research"`
	Build    BuildParams    `json:"build"`
	Confirm  bool           `json:"confirm"`
}

func (s *server) handleJob(w http.ResponseWriter, r *http.Request) {
	var req jobRequest
	if !decode(w, r, &req) {
		return
	}
	tool := map[string]string{"research": "oracle", "build": "morp"}[req.Job]
	if tool == "" {
		writeJSON(w, http.StatusNotFound, errBody(errors.New("unknown job")))
		return
	}
	if !req.Confirm {
		writeJSON(w, http.StatusPreconditionRequired, errBody(errors.New("this uses your agent account and changes files; confirm it first")))
		return
	}
	dir, err := resolve(s.home, req.Path)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errBody(err))
		return
	}
	if err := needSetUp(dir, TaskParams{}); err != nil {
		writeJSON(w, http.StatusConflict, errBody(err))
		return
	}
	if _, ok := s.tools()[tool]; !ok {
		writeJSON(w, http.StatusServiceUnavailable, errBody(fmt.Errorf("%s is not installed; run: brew install 0merUfuk/thematrix/%s", tool, tool)))
		return
	}
	rt, rtName, rtReady := s.agentRuntime(dir)
	if !rtReady {
		writeJSON(w, http.StatusServiceUnavailable, errBody(fmt.Errorf("%s is not installed on this computer; the agents need it to work", rtName)))
		return
	}

	var (
		title, subject, stopFile string
		prepare                  func(string) ([]step, error)
	)
	switch req.Job {
	case "research":
		title, subject = "Research best practices", req.Research.Stack
		stopFile, prepare, err = s.researchSteps(dir, req.Research, rt)
	case "build":
		title = "Build with the agents"
		subject = strings.TrimSpace(req.Build.Goal)
		if req.Build.Continue {
			if prev := s.jobs.Latest(dir, "build", ""); prev != nil {
				subject = prev.Subject
			}
		}
		stopFile, prepare, err = s.buildSteps(dir, req.Build, rt)
	}
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errBody(err))
		return
	}
	job, err := s.jobs.Start(req.Job, title, dir, subject, stopFile, prepare)
	switch {
	case errors.Is(err, errJobRunning), errors.Is(err, errJobsBusy):
		writeJSON(w, http.StatusTooManyRequests, errBody(err))
		return
	case err != nil:
		writeJSON(w, http.StatusInternalServerError, errBody(err))
		return
	}
	writeJSON(w, http.StatusAccepted, s.jobView(job, 0))
}

func (s *server) jobView(j *Job, from int64) JobView {
	v := j.View(from, cli.PathToHome(j.Project))
	for i, c := range v.Commands {
		v.Commands[i] = strings.ReplaceAll(c, s.home, "~")
	}
	return v
}

func (s *server) handleJobGet(w http.ResponseWriter, r *http.Request) {
	job := s.jobs.Get(r.PathValue("id"))
	if job == nil {
		writeJSON(w, http.StatusNotFound, errBody(errors.New("unknown job")))
		return
	}
	var from int64
	_, _ = fmt.Sscanf(r.URL.Query().Get("from"), "%d", &from)
	writeJSON(w, http.StatusOK, s.jobView(job, from))
}

func (s *server) handleJobStop(w http.ResponseWriter, r *http.Request) {
	if !s.jobs.Stop(r.PathValue("id")) {
		writeJSON(w, http.StatusConflict, errBody(errors.New("not running")))
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"stopping": true})
}

type approveRequest struct {
	Path    string `json:"path"`
	Stack   string `json:"stack"`
	Confirm bool   `json:"confirm"`
}

func (s *server) handleApprove(w http.ResponseWriter, r *http.Request) {
	var req approveRequest
	if !decode(w, r, &req) {
		return
	}
	if !req.Confirm {
		writeJSON(w, http.StatusPreconditionRequired, errBody(errors.New("confirm it first")))
		return
	}
	dir, err := resolve(s.home, req.Path)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errBody(err))
		return
	}
	if err := s.approveResearch(dir, req.Stack); err != nil {
		writeJSON(w, http.StatusConflict, errBody(err))
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"approved": true})
}

func (s *server) handleResearchFile(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	dir, err := resolve(s.home, q.Get("path"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errBody(err))
		return
	}
	text, err := s.researchFile(dir, q.Get("stack"), q.Get("name"))
	if err != nil {
		writeJSON(w, http.StatusNotFound, errBody(err))
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"name": q.Get("name"), "text": text})
}
