package matrixapp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// Long jobs (research, build loops) run for minutes or hours on the user's
// Claude Code or Codex account. Unlike short tasks they have no time limit,
// write their output straight to ~/.the-matrix/jobs/<id>/log.txt, keep their
// state in job.json next to it, and stop as a whole process group.

const (
	maxJobs       = 3
	keepJobs      = 20
	jobReadCap    = 256 << 10
	stopGrace     = 10 * time.Second
	jobLogName    = "log.txt"
	jobStateName  = "job.json"
	jobConfigName = "stack.json"
)

// step is one part of a job: a command, or a small Go action between commands.
type step struct {
	label string   // plain words shown while the step runs
	bin   string   // command to run; empty when do is set
	args  []string // arguments, passed without a shell
	dir   string
	do    func(log io.Writer) error
}

// Job is one long-running piece of work for a project.
type Job struct {
	mu       sync.Mutex
	ID       string    `json:"id"`
	Kind     string    `json:"kind"` // research | build
	Title    string    `json:"title"`
	Project  string    `json:"project"` // absolute project folder
	Subject  string    `json:"subject,omitempty"`
	Status   string    `json:"status"` // running | ok | fail | stopped | interrupted
	Step     string    `json:"step"`
	Steps    []string  `json:"steps"`
	Commands []string  `json:"commands"`
	StopFile string    `json:"stop_file,omitempty"` // loop STOP file, used after a crash
	Started  time.Time `json:"started"`
	Ended    time.Time `json:"ended,omitempty"`
	Exit     int       `json:"exit"`

	dir     string
	stopped bool
	stop    func()
	done    chan struct{}
}

// JobView is the JSON shape of a job; Output starts at the requested offset.
type JobView struct {
	ID         string   `json:"id"`
	Kind       string   `json:"kind"`
	Title      string   `json:"title"`
	Path       string   `json:"path"`
	Subject    string   `json:"subject,omitempty"`
	Status     string   `json:"status"`
	Step       string   `json:"step"`
	StepIndex  int      `json:"step_index"`
	StepCount  int      `json:"step_count"`
	Commands   []string `json:"commands"`
	Exit       int      `json:"exit"`
	StartedAt  int64    `json:"started_at"`
	DurationMS int64    `json:"duration_ms"`
	Output     string   `json:"output"`
	Next       int64    `json:"next"`
}

// View returns the job with log output from byte offset from (at most 256 KB;
// a negative offset returns no output).
func (j *Job) View(from int64, shown string) JobView {
	j.mu.Lock()
	end := j.Ended
	if end.IsZero() {
		end = time.Now()
	}
	idx := 0
	for i, s := range j.Steps {
		if s == j.Step {
			idx = i + 1
		}
	}
	v := JobView{
		ID: j.ID, Kind: j.Kind, Title: j.Title, Path: shown, Subject: j.Subject, Status: j.Status,
		Step: j.Step, StepIndex: idx, StepCount: len(j.Steps), Commands: append([]string{}, j.Commands...), Exit: j.Exit,
		StartedAt: j.Started.UnixMilli(), DurationMS: end.Sub(j.Started).Milliseconds(),
	}
	dir := j.dir
	j.mu.Unlock()

	f, err := os.Open(filepath.Join(dir, jobLogName))
	if err != nil {
		return v
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return v
	}
	size := info.Size()
	if from < 0 || from > size {
		from = size
	}
	if size-from > jobReadCap {
		from = size - jobReadCap // the newest output matters most
	}
	buf := make([]byte, size-from)
	n, _ := f.ReadAt(buf, from)
	v.Output = ansi.ReplaceAllString(string(buf[:n]), "")
	v.Next = from + int64(n)
	return v
}

func (j *Job) running() bool {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.Status == "running"
}

// save writes job.json. Caller holds j.mu.
func (j *Job) save() {
	raw, err := json.MarshalIndent(j, "", "  ")
	if err != nil {
		return
	}
	tmp := filepath.Join(j.dir, jobStateName+".tmp")
	if os.WriteFile(tmp, append(raw, '\n'), 0o600) == nil {
		_ = os.Rename(tmp, filepath.Join(j.dir, jobStateName))
	}
}

func (j *Job) set(fn func()) {
	j.mu.Lock()
	fn()
	j.save()
	j.mu.Unlock()
}

// Jobs keeps the long jobs of this app, on disk and in memory.
type Jobs struct {
	mu   sync.Mutex
	root string // ~/.the-matrix/jobs
	env  []string
	jobs []*Job
}

var (
	errJobsBusy   = errors.New("three long jobs are already running; wait for one to finish or stop one")
	errJobRunning = errors.New("this project already has a job running; stop it or wait for it to finish")
)

// newJobs loads earlier jobs from disk. Jobs that were still running when the
// app last stopped are marked interrupted, and their loop is asked to halt.
func newJobs(home string, env []string) *Jobs {
	js := &Jobs{root: filepath.Join(home, ".the-matrix", "jobs"), env: env}
	entries, _ := os.ReadDir(js.root)
	for _, e := range entries {
		dir := filepath.Join(js.root, e.Name())
		raw, err := os.ReadFile(filepath.Join(dir, jobStateName))
		if err != nil {
			continue
		}
		j := &Job{dir: dir}
		if json.Unmarshal(raw, j) != nil || j.ID != e.Name() {
			continue
		}
		if j.Status == "running" {
			j.Status, j.Ended = "interrupted", time.Now()
			if j.StopFile != "" && isDir(filepath.Dir(j.StopFile)) {
				_ = os.WriteFile(j.StopFile, []byte("stopped by The Matrix after a restart\n"), 0o644)
			}
			j.save()
		}
		js.jobs = append(js.jobs, j)
	}
	sort.Slice(js.jobs, func(a, b int) bool { return js.jobs[a].Started.Before(js.jobs[b].Started) })
	return js
}

// Start begins a job for project. Steps run in order and the job stops at the
// first failing step.
func (js *Jobs) Start(kind, title, project, subject, stopFile string, prepare func(jobDir string) ([]step, error)) (*Job, error) {
	js.mu.Lock()
	defer js.mu.Unlock()
	active := 0
	for _, j := range js.jobs {
		if j.running() {
			active++
			if j.Project == project {
				return nil, errJobRunning
			}
		}
	}
	if active >= maxJobs {
		return nil, errJobsBusy
	}

	id := newID()
	dir := filepath.Join(js.root, id)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	steps, err := prepare(dir)
	if err != nil {
		_ = os.RemoveAll(dir)
		return nil, err
	}
	logFile, err := os.OpenFile(filepath.Join(dir, jobLogName), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		_ = os.RemoveAll(dir)
		return nil, err
	}
	j := &Job{
		ID: id, Kind: kind, Title: title, Project: project, Subject: subject, Status: "running",
		StopFile: stopFile, Started: time.Now(), dir: dir, done: make(chan struct{}),
	}
	for _, s := range steps {
		j.Steps = append(j.Steps, s.label)
		if s.bin != "" {
			j.Commands = append(j.Commands, filepath.Base(s.bin)+" "+strings.Join(s.args, " "))
		}
	}
	j.Step = j.Steps[0]
	ctx, cancel := context.WithCancel(context.Background())
	j.stop = cancel
	j.mu.Lock()
	j.save()
	j.mu.Unlock()
	js.jobs = append(js.jobs, j)
	js.prune()

	go js.run(ctx, j, steps, logFile)
	return j, nil
}

func (js *Jobs) run(ctx context.Context, j *Job, steps []step, logFile *os.File) {
	defer close(j.done)
	defer logFile.Close()
	status, code := "ok", 0
	for _, s := range steps {
		if ctx.Err() != nil {
			break
		}
		j.set(func() { j.Step = s.label })
		fmt.Fprintf(logFile, "\n── %s ──\n", s.label)
		if s.do != nil {
			if err := s.do(logFile); err != nil {
				fmt.Fprintln(logFile, err)
				status, code = "fail", 1
				break
			}
			continue
		}
		cmd := exec.Command(s.bin, s.args...)
		cmd.Dir = s.dir
		cmd.Env = js.env
		cmd.Stdout, cmd.Stderr = logFile, logFile
		ownGroup(cmd)
		if err := cmd.Start(); err != nil {
			fmt.Fprintln(logFile, err)
			status, code = "fail", -1
			break
		}
		waited := make(chan error, 1)
		go func() { waited <- cmd.Wait() }()
		var err error
		select {
		case err = <-waited:
		case <-ctx.Done():
			terminateGroup(cmd, false)
			select {
			case err = <-waited:
			case <-time.After(stopGrace):
				terminateGroup(cmd, true)
				err = <-waited
			}
		}
		if err != nil {
			status, code = "fail", cmd.ProcessState.ExitCode()
			break
		}
	}
	j.set(func() {
		if j.stopped {
			status = "stopped"
		}
		j.Status, j.Exit, j.Ended = status, code, time.Now()
	})
	fmt.Fprintf(logFile, "\n── finished: %s ──\n", status)
}

// Get returns a job by id.
func (js *Jobs) Get(id string) *Job {
	js.mu.Lock()
	defer js.mu.Unlock()
	for _, j := range js.jobs {
		if j.ID == id {
			return j
		}
	}
	return nil
}

// Stop ends a running job: its whole process group gets SIGTERM, then
// SIGKILL if it is still there after a few seconds.
func (js *Jobs) Stop(id string) bool {
	j := js.Get(id)
	if j == nil || !j.running() || j.stop == nil {
		return false
	}
	j.mu.Lock()
	j.stopped = true
	j.mu.Unlock()
	j.stop()
	return true
}

// StopAll stops every running job and waits for them, used when the app quits.
func (js *Jobs) StopAll() {
	js.mu.Lock()
	var running []*Job
	for _, j := range js.jobs {
		if j.running() && j.stop != nil {
			running = append(running, j)
		}
	}
	js.mu.Unlock()
	for _, j := range running {
		js.Stop(j.ID)
	}
	for _, j := range running {
		<-j.done
	}
}

// ForProject returns the project's jobs, newest first.
func (js *Jobs) ForProject(project string, limit int) []*Job {
	js.mu.Lock()
	defer js.mu.Unlock()
	var out []*Job
	for i := len(js.jobs) - 1; i >= 0 && len(out) < limit; i-- {
		if js.jobs[i].Project == project {
			out = append(out, js.jobs[i])
		}
	}
	return out
}

// Latest returns the newest job of a kind for a project, or nil.
func (js *Jobs) Latest(project, kind, subject string) *Job {
	for _, j := range js.ForProject(project, keepJobs) {
		if j.Kind == kind && (subject == "" || j.Subject == subject) {
			return j
		}
	}
	return nil
}

// prune removes the oldest finished jobs beyond keepJobs. Caller holds js.mu.
func (js *Jobs) prune() {
	for len(js.jobs) > keepJobs {
		idx := -1
		for i, j := range js.jobs {
			if !j.running() {
				idx = i
				break
			}
		}
		if idx < 0 {
			return
		}
		_ = os.RemoveAll(js.jobs[idx].dir)
		js.jobs = append(js.jobs[:idx], js.jobs[idx+1:]...)
	}
}

func isDir(p string) bool {
	info, err := os.Stat(p)
	return err == nil && info.IsDir()
}
