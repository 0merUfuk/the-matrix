package matrixapp

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"os"
	"os/exec"
	"regexp"
	"sync"
	"time"
)

const (
	runTimeout   = 60 * time.Second
	outputCap    = 256 << 10
	maxActive    = 2
	keepFinished = 30
)

var ansi = regexp.MustCompile(`\x1b\[[0-9;?]*[A-Za-z]`)

// Run is one command execution with its captured output.
type Run struct {
	mu        sync.Mutex
	ID        string
	Tool      string
	Action    string
	Title     string
	Command   string
	Path      string
	Started   time.Time
	Ended     time.Time
	Status    string // running | ok | fail | canceled | timeout
	Exit      int
	out       []byte
	truncated bool
	cancel    context.CancelFunc
}

// RunView is the JSON shape of a run; Output starts at the requested offset.
type RunView struct {
	ID         string `json:"id"`
	Tool       string `json:"tool"`
	Action     string `json:"action"`
	Title      string `json:"title"`
	Command    string `json:"command"`
	Path       string `json:"path"`
	Status     string `json:"status"`
	Exit       int    `json:"exit"`
	StartedAt  int64  `json:"started_at"`
	DurationMS int64  `json:"duration_ms"`
	Output     string `json:"output"`
	Next       int    `json:"next"`
	Truncated  bool   `json:"truncated"`
}

func (r *Run) Write(p []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if room := outputCap - len(r.out); room > 0 {
		if len(p) > room {
			r.out = append(r.out, p[:room]...)
			r.truncated = true
		} else {
			r.out = append(r.out, p...)
		}
	} else {
		r.truncated = true
	}
	return len(p), nil
}

// View returns the run with output from byte offset from.
func (r *Run) View(from int) RunView {
	r.mu.Lock()
	defer r.mu.Unlock()
	if from < 0 || from > len(r.out) {
		from = len(r.out)
	}
	end := r.Ended
	if end.IsZero() {
		end = time.Now()
	}
	return RunView{
		ID: r.ID, Tool: r.Tool, Action: r.Action, Title: r.Title, Command: r.Command, Path: r.Path,
		Status: r.Status, Exit: r.Exit, StartedAt: r.Started.UnixMilli(), DurationMS: end.Sub(r.Started).Milliseconds(),
		Output: ansi.ReplaceAllString(string(r.out[from:]), ""), Next: len(r.out), Truncated: r.truncated,
	}
}

// Runner starts commands and keeps a short in-memory history.
type Runner struct {
	mu   sync.Mutex
	env  []string
	runs []*Run
}

var errBusy = errors.New("two commands are already running; wait for one to finish")

// Start runs bin with args in dir. Arguments go straight to the process (no
// shell), stdin is empty and colors are disabled.
func (rn *Runner) Start(tool, action, title, display, bin string, args []string, dir, shown string) (*Run, error) {
	rn.mu.Lock()
	active := 0
	for _, r := range rn.runs {
		if r.status() == "running" {
			active++
		}
	}
	if active >= maxActive {
		rn.mu.Unlock()
		return nil, errBusy
	}
	ctx, cancel := context.WithTimeout(context.Background(), runTimeout)
	run := &Run{ID: newID(), Tool: tool, Action: action, Title: title, Command: display, Path: shown, Started: time.Now(), Status: "running", cancel: cancel}
	rn.runs = append(rn.runs, run)
	rn.trim()
	rn.mu.Unlock()

	cmd := exec.CommandContext(ctx, bin, args...) // #nosec G204 -- argv-only exec of a resolved matrix tool binary with validated task arguments, no shell
	cmd.Dir = dir
	cmd.Env = rn.env
	if cmd.Env == nil {
		cmd.Env = append(os.Environ(), "NO_COLOR=1", "CLICOLOR=0", "TERM=dumb")
	}
	cmd.Stdout, cmd.Stderr = run, run
	if err := cmd.Start(); err != nil {
		cancel()
		run.finish("fail", -1)
		_, _ = run.Write([]byte(err.Error() + "\n"))
		return run, nil
	}
	go func() {
		err := cmd.Wait()
		status, code := "ok", 0
		if err != nil {
			status, code = "fail", cmd.ProcessState.ExitCode()
		}
		switch ctx.Err() {
		case context.DeadlineExceeded:
			status = "timeout"
		case context.Canceled:
			// CommandContext kills with SIGKILL, so exit 0 here means the tool
			// finished before the cancel landed: report what actually happened.
			if status != "ok" {
				status = "canceled"
			}
		}
		cancel()
		run.finish(status, code)
	}()
	return run, nil
}

func (r *Run) status() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.Status
}

func (r *Run) finish(status string, code int) {
	r.mu.Lock()
	r.Status, r.Exit, r.Ended = status, code, time.Now()
	r.mu.Unlock()
}

// trim drops the oldest finished runs beyond keepFinished. Caller holds rn.mu.
func (rn *Runner) trim() {
	for len(rn.runs) > keepFinished {
		idx := -1
		for i, r := range rn.runs {
			if r.status() != "running" {
				idx = i
				break
			}
		}
		if idx < 0 {
			return
		}
		rn.runs = append(rn.runs[:idx], rn.runs[idx+1:]...)
	}
}

// Get returns a run by id.
func (rn *Runner) Get(id string) *Run {
	rn.mu.Lock()
	defer rn.mu.Unlock()
	for _, r := range rn.runs {
		if r.ID == id {
			return r
		}
	}
	return nil
}

// Cancel stops a running command.
func (rn *Runner) Cancel(id string) bool {
	r := rn.Get(id)
	if r == nil || r.status() != "running" {
		return false
	}
	r.cancel()
	return true
}

// Recent returns finished and running runs, newest first, without output.
func (rn *Runner) Recent(limit int) []RunView {
	rn.mu.Lock()
	defer rn.mu.Unlock()
	out := []RunView{}
	for i := len(rn.runs) - 1; i >= 0 && len(out) < limit; i-- {
		v := rn.runs[i].View(-1)
		out = append(out, v)
	}
	return out
}

func newID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
