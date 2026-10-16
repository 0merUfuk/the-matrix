package matrixapp

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// The build loop runs morp for a project. morp loop --non-interactive writes
// .autonomous/ and turns the goal into tasks/todo.md (one agent call), then
// .autonomous/loop.sh runs rounds of developer, tester, reviewer and security
// agents until both reviews approve, the rounds run out, or the agents need a
// decision from the user.

// BuildParams is the goal form.
type BuildParams struct {
	Goal     string `json:"goal"`
	Size     string `json:"size"`     // small | medium | large
	Continue bool   `json:"continue"` // run the existing loop again, no new plan
}

// BuildState is what the build step shows.
type BuildState struct {
	Exists    bool     `json:"exists"`
	Git       bool     `json:"git"`
	Goal      string   `json:"goal"`
	Done      int      `json:"done"`
	Total     int      `json:"total"`
	Round     int      `json:"round"`
	Rounds    int      `json:"rounds"`
	Activity  string   `json:"activity"` // what the agents are doing right now, in plain words
	Result    string   `json:"result"`   // approved | rounds-used | decisions | stopped | failed | ""
	Decisions []string `json:"decisions"`
	Job       *JobView `json:"job"`
}

var (
	buildSizes = map[string]string{"small": "3", "medium": "6", "large": "10"}
	todoItem   = regexp.MustCompile(`(?m)^\s*[-*] \[( |x|X)\]`)
	roundLine  = regexp.MustCompile(`CYCLE (\d+) / (\d+)`)
	roleLine   = regexp.MustCompile(`Cycle \d+ \| ([A-Z]+) \| max-turns`)
	activities = map[string]string{"MANAGER": "planning the round", "STRATEGIST": "making a decision", "DEVELOPER": "writing code", "TESTER": "testing", "REVIEWER": "reviewing", "SECURITY": "checking security"}
	exitLine   = regexp.MustCompile(`Exit reason:\s+([A-Z_]+)`)
	results    = map[string]string{"ALL_GATES_APPROVED": "approved", "MAX_CYCLES_REACHED": "rounds-used", "DECISIONS_PENDING": "decisions", "STOPPED_BY_USER": "stopped", "RUNTIME_ERROR": "failed"}
)

// buildSteps validates the goal form and returns the job's steps.
func (s *server) buildSteps(dir string, p BuildParams, runtime string) (string, func(string) ([]step, error), error) {
	if !isSetUp(dir) {
		return "", nil, errors.New("set up the agents first")
	}
	if !exists(dir, ".git") {
		return "", nil, errors.New("this folder is not tracked with Git yet; the agents save every step with Git, so the project needs it first")
	}
	var plan []step
	if p.Continue {
		if !exists(dir, ".autonomous/loop.sh") {
			return "", nil, errors.New("there is no plan to continue yet; describe what to build first")
		}
	} else {
		goal, err := cleanGoal(p.Goal)
		if err != nil {
			return "", nil, err
		}
		size, ok := buildSizes[p.Size]
		if !ok {
			return "", nil, errors.New("choose how long the agents may work")
		}
		plan = append(plan, step{
			label: "Planning the work", bin: s.tools()["morp"].Path, dir: dir,
			args: []string{"loop", "--path", dir, "--non-interactive", "--goal", goal, "--max-cycles", size, "--runtime", runtime},
		})
	}
	stop := filepath.Join(dir, "tasks", "STOP")
	prepare := func(string) ([]step, error) {
		steps := append(append([]step{}, plan...), step{label: "Getting ready", do: func(w io.Writer) error {
			if err := os.Remove(stop); err == nil {
				fmt.Fprintln(w, "Cleared an old stop request.")
			} else if !errors.Is(err, os.ErrNotExist) {
				return err
			}
			return nil
		}})
		steps = append(steps, step{label: "Agents are building", bin: "bash", dir: dir, args: []string{filepath.Join(dir, ".autonomous", "loop.sh")}})
		return steps, nil
	}
	return stop, prepare, nil
}

// cleanGoal allows line breaks and tabs in a goal but no other control characters.
func cleanGoal(v string) (string, error) {
	v = strings.TrimSpace(v)
	if len(v) < 10 {
		return "", errors.New("describe what to build in a sentence or two")
	}
	if len(v) > 2000 {
		return "", errors.New("the goal is too long (keep it under 2000 characters)")
	}
	for _, r := range v {
		if (r < 0x20 && r != '\n' && r != '\t') || r == 0x7f {
			return "", errors.New("the goal has characters that cannot be used")
		}
	}
	return v, nil
}

// buildState reads the plan, the latest job's log and pending decisions.
func (s *server) buildState(dir string) BuildState {
	b := BuildState{Exists: exists(dir, ".autonomous/loop.sh"), Git: exists(dir, ".git"), Decisions: []string{}}
	if raw, err := os.ReadFile(filepath.Join(dir, "tasks", "todo.md")); err == nil {
		for _, m := range todoItem.FindAllStringSubmatch(string(raw), -1) {
			b.Total++
			if m[1] != " " {
				b.Done++
			}
		}
	}
	if raw, err := os.ReadFile(filepath.Join(dir, "tasks", "decisions-pending.md")); err == nil {
		for _, line := range strings.Split(string(raw), "\n") {
			if t, ok := strings.CutPrefix(line, "## "); ok && len(b.Decisions) < 10 {
				b.Decisions = append(b.Decisions, strings.TrimSpace(t))
			}
		}
	}
	job := s.jobs.Latest(dir, "build", "")
	if job == nil {
		return b
	}
	v := s.jobView(job, 0)
	log := v.Output
	v.Output = ""
	b.Job, b.Goal = &v, job.Subject
	if m := roundLine.FindAllStringSubmatch(log, -1); len(m) > 0 {
		b.Round, _ = strconv.Atoi(m[len(m)-1][1])
		b.Rounds, _ = strconv.Atoi(m[len(m)-1][2])
	}
	if m := roleLine.FindAllStringSubmatch(log, -1); len(m) > 0 && v.Status == "running" {
		b.Activity = activities[m[len(m)-1][1]]
	}
	if m := exitLine.FindAllStringSubmatch(log, -1); len(m) > 0 && v.Status != "running" {
		b.Result = results[m[len(m)-1][1]]
	}
	switch {
	case v.Status == "stopped":
		b.Result = "stopped"
	case v.Status == "fail", v.Status == "interrupted":
		if b.Result == "" || b.Result == "approved" {
			b.Result = "failed"
		}
	}
	return b
}
