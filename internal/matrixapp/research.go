package matrixapp

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/0merUfuk/the-matrix/internal/oracle"
)

// Research runs oracle for one stack of a project. oracle research prepares a
// workspace and its loop does the paid work: four research rounds, a review
// gate (tasks/SYNTHESIZE_APPROVED), then writing the knowledge docs to
// output/<stack>/. The workspace lives under ~/.the-matrix/research so the
// project folder only receives the finished docs, through trinity sync.

// ResearchParams is the knowledge form.
type ResearchParams struct {
	Stack     string `json:"stack"`
	Framework string `json:"framework"`
	AppType   string `json:"app_type"`
	DataLayer string `json:"data_layer"`
	Testing   string `json:"testing"`
	Queue     bool   `json:"queue"`
	Review    bool   `json:"review"`
}

// ResearchState is what the knowledge step shows for one stack.
type ResearchState struct {
	Stack    string         `json:"stack"`
	Defaults ResearchParams `json:"defaults"`
	Phase    string         `json:"phase"` // none | running | review | review-timeout | writing | ready | added | failed | stopped
	Cycle    int            `json:"cycle"`
	Cycles   int            `json:"cycles"`
	Found    []Finding      `json:"found"`
	Docs     int            `json:"docs"`
	Job      *JobView       `json:"job"`
}

// Finding is one research file waiting for review.
type Finding struct {
	Name  string `json:"name"`
	Title string `json:"title"`
}

var (
	slug     = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)
	appTypes = map[string]string{"full-stack": "hybrid", "api-only": "n/a", "frontend": "client-state", "cli": "n/a", "library": "n/a"}
)

type stackInfo struct {
	Name             string `json:"name"`
	Language         string `json:"language"`
	Framework        string `json:"framework"`
	ArchStyle        string `json:"archStyle"`
	TestingFramework string `json:"testingFramework"`
	HasQueue         bool   `json:"hasQueue"`
	DataLayer        string `json:"dataLayer"`
}

// projectStacks reads the stacks neo recorded in .neo.json.
func projectStacks(dir string) []stackInfo {
	raw, err := os.ReadFile(filepath.Join(dir, ".neo.json")) // #nosec G304 G703 -- reads the project's own .neo.json; dir is confined to $HOME by resolve()
	if err != nil {
		return nil
	}
	var p struct {
		Stacks []stackInfo `json:"stacks"`
	}
	if json.Unmarshal(raw, &p) != nil {
		return nil
	}
	var out []stackInfo
	for _, s := range p.Stacks {
		if slug.MatchString(s.Name) && len(s.Name) <= 40 {
			out = append(out, s)
		}
	}
	return out
}

func findStack(dir, name string) (stackInfo, bool) {
	for _, s := range projectStacks(dir) {
		if s.Name == name {
			return s, true
		}
	}
	return stackInfo{}, false
}

// researchWorkspace is where oracle works for a project's stack.
func researchWorkspace(home, project, stack string) string {
	sum := sha256.Sum256([]byte(project))
	key := filepath.Base(project) + "-" + hex.EncodeToString(sum[:4])
	return filepath.Join(home, ".the-matrix", "research", key, stack)
}

// cleanText trims a form value and refuses control characters and long text.
func cleanText(label, v string, max int, required bool) (string, error) {
	v = strings.TrimSpace(v)
	if v == "" && required {
		return "", fmt.Errorf("fill in %s", label)
	}
	if len(v) > max {
		return "", fmt.Errorf("%s is too long (keep it under %d characters)", label, max)
	}
	for _, r := range v {
		if r < 0x20 || r == 0x7f {
			return "", fmt.Errorf("%s has characters that cannot be used", label)
		}
	}
	return v, nil
}

// stackContext turns the form into the JSON oracle research --config reads.
func stackContext(s stackInfo, p ResearchParams) (oracle.StackContext, error) {
	var err error
	ctx := oracle.StackContext{StackName: s.Name, ArchStyle: s.ArchStyle, SelectedTopics: oracle.AllTopicValues()}
	if ctx.FrameworkVersion, err = cleanText("what it is built with", p.Framework, 120, true); err != nil {
		return ctx, err
	}
	state, ok := appTypes[p.AppType]
	if !ok {
		return ctx, errors.New("choose what kind of software this is")
	}
	ctx.AppType, ctx.StateApproach = p.AppType, state
	if ctx.DataLayer, err = cleanText("where it keeps data", p.DataLayer, 120, false); err != nil {
		return ctx, err
	}
	if ctx.DataLayer == "" {
		ctx.DataLayer = "none"
	}
	if ctx.TestingFramework, err = cleanText("how it is tested", p.Testing, 120, false); err != nil {
		return ctx, err
	}
	switch ctx.ArchStyle {
	case "layered", "feature-based", "flat", "hybrid":
	default:
		ctx.ArchStyle = "hybrid"
	}
	ctx.HasQueue, ctx.HasBackgroundJobs = p.Queue, p.Queue
	ctx.TopicCount = len(ctx.SelectedTopics)
	ctx.CreatedDate = time.Now().Format("2006-01-02")
	return ctx, nil
}

// researchSteps validates the form and returns the job's steps.
func (s *server) researchSteps(dir string, p ResearchParams, runtime string) (string, func(string) ([]step, error), error) {
	st, ok := findStack(dir, p.Stack)
	if !ok {
		return "", nil, errors.New("choose which part of the project to research")
	}
	ctx, err := stackContext(st, p)
	if err != nil {
		return "", nil, err
	}
	ws := researchWorkspace(s.home, dir, st.Name)
	prepare := func(jobDir string) ([]step, error) {
		cfg := filepath.Join(jobDir, jobConfigName)
		raw, err := json.MarshalIndent(ctx, "", "  ")
		if err != nil {
			return nil, err
		}
		if err := os.WriteFile(cfg, raw, 0o600); err != nil {
			return nil, err
		}
		if err := os.MkdirAll(filepath.Dir(ws), 0o700); err != nil {
			return nil, err
		}
		steps := []step{{
			label: "Preparing the research",
			bin:   s.tools()["oracle"].Path, dir: jobDir,
			args: []string{"research", "--config", cfg, "--output-dir", ws, "--force", "--runtime", runtime},
		}}
		if !p.Review {
			steps = append(steps, step{label: "Skipping the review", do: func(w io.Writer) error {
				fmt.Fprintln(w, "The docs will be written without a review pause.")
				return touch(filepath.Join(ws, "tasks", "SYNTHESIZE_APPROVED"))
			}})
		}
		steps = append(steps, step{label: "Researching best practices", bin: "bash", dir: ws, args: []string{filepath.Join(ws, ".autonomous", "loop.sh")}})
		return steps, nil
	}
	return filepath.Join(ws, "tasks", "STOP"), prepare, nil
}

// researchStates describes the knowledge step for every stack of a project.
func (s *server) researchStates(dir string) []ResearchState {
	out := []ResearchState{}
	for _, st := range projectStacks(dir) {
		rs := ResearchState{
			Stack: st.Name, Cycles: len(oracle.CycleGroups), Found: []Finding{},
			Defaults: ResearchParams{Stack: st.Name, Framework: strings.TrimSpace(st.Framework), DataLayer: st.DataLayer, Testing: st.TestingFramework, Queue: st.HasQueue, Review: true, AppType: guessAppType(st)},
		}
		if st.DataLayer == "none" {
			rs.Defaults.DataLayer = ""
		}
		ws := researchWorkspace(s.home, dir, st.Name)
		job := s.jobs.Latest(dir, "research", st.Name)
		running := false
		if job != nil {
			v := s.jobView(job, -1)
			rs.Job, running = &v, v.Status == "running"
		}
		if last, ok := lastAnswers(job); ok {
			rs.Defaults = last
		}
		statusFile := filepath.Join(ws, "tasks", "status.txt")
		status := readTrim(statusFile)
		if info, err := os.Stat(statusFile); running && err == nil && info.ModTime().Before(job.Started) {
			status = "" // left over from an earlier run, not yet replaced by this one
		}
		if n, err := fmt.Sscanf(status, "RESEARCH_CYCLE_%d", &rs.Cycle); n != 1 || err != nil {
			rs.Cycle = 0
		}
		rs.Found = findings(filepath.Join(ws, "tasks", "research"))
		rs.Docs = len(mdFiles(filepath.Join(ws, "output", st.Name)))
		switch {
		case !isDir(ws) && job == nil:
			rs.Phase = "none"
		case status == "SYNTHESIS_COMPLETE" && rs.Docs > 0:
			rs.Phase = "ready"
			if added(filepath.Join(ws, "output", st.Name), filepath.Join(dir, ".claude", "knowledge", st.Name)) {
				rs.Phase = "added"
			}
		case running && status == "RESEARCH_COMPLETE":
			rs.Phase, rs.Cycle = "review", rs.Cycles
		case running && status == "SYNTHESIS_RUNNING":
			rs.Phase = "writing"
		case running:
			rs.Phase = "running"
		case rs.Job != nil && (rs.Job.Status == "stopped" || rs.Job.Status == "interrupted"), status == "STOPPED_BY_USER":
			// Checked before the review-timeout and failed fallbacks, because a
			// stopped loop leaves its last status behind. Finished docs still win.
			rs.Phase = "stopped"
		case status == "RESEARCH_COMPLETE":
			rs.Phase = "review-timeout"
		default:
			rs.Phase = "failed"
		}
		out = append(out, rs)
	}
	return out
}

// lastAnswers reads the form answers a research job was started with.
func lastAnswers(j *Job) (ResearchParams, bool) {
	if j == nil {
		return ResearchParams{}, false
	}
	raw, err := os.ReadFile(filepath.Join(j.dir, jobConfigName)) // #nosec G703 -- job directory the app created under ~/.the-matrix/jobs
	var c oracle.StackContext
	if err != nil || json.Unmarshal(raw, &c) != nil {
		return ResearchParams{}, false
	}
	p := ResearchParams{Stack: c.StackName, Framework: c.FrameworkVersion, AppType: c.AppType, DataLayer: c.DataLayer, Testing: c.TestingFramework, Queue: c.HasQueue, Review: true}
	if p.DataLayer == "none" {
		p.DataLayer = ""
	}
	return p, true
}

// approveResearch opens the review gate of a running research job.
func (s *server) approveResearch(dir, stack string) error {
	for _, rs := range s.researchStates(dir) {
		if rs.Stack == stack && rs.Phase == "review" {
			return touch(filepath.Join(researchWorkspace(s.home, dir, stack), "tasks", "SYNTHESIZE_APPROVED"))
		}
	}
	return errors.New("this research is not waiting for a review")
}

// researchFile returns one research file for reading in the review step.
func (s *server) researchFile(dir, stack, name string) (string, error) {
	if !slug.MatchString(stack) || filepath.Base(name) != name || !strings.HasSuffix(name, ".md") {
		return "", errors.New("unknown file")
	}
	p := filepath.Join(researchWorkspace(s.home, dir, stack), "tasks", "research", name)
	info, err := os.Lstat(p)
	if err != nil || !info.Mode().IsRegular() {
		return "", errors.New("unknown file")
	}
	if info.Size() > 512<<10 {
		return "", errors.New("this file is too large to show here")
	}
	raw, err := os.ReadFile(p) // #nosec G304 -- name is a bare .md file name, stack a slug, symlinks and large files rejected above
	return string(raw), err
}

// guessAppType picks the form's starting answer from the stack neo recorded.
func guessAppType(s stackInfo) string {
	fw := strings.ToLower(s.Framework)
	switch {
	case strings.Contains(fw, "next"), strings.Contains(fw, "nuxt"), strings.Contains(fw, "rails"), strings.Contains(fw, "django"):
		return "full-stack"
	case strings.Contains(fw, "react"), strings.Contains(fw, "vue"), strings.Contains(fw, "svelte"), s.Language == "flutter":
		return "frontend"
	}
	return "api-only"
}

func findings(dir string) []Finding {
	out := []Finding{}
	for _, name := range mdFiles(dir) {
		title := strings.TrimSuffix(name, ".md")
		for _, t := range oracle.AllTopics {
			if strings.HasPrefix(name, t.Value) {
				title = strings.TrimSpace(strings.TrimLeft(t.Name, "0123456789"))
			}
		}
		out = append(out, Finding{Name: name, Title: title})
	}
	return out
}

func mdFiles(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		if e.Type().IsRegular() && strings.HasSuffix(e.Name(), ".md") && !strings.HasPrefix(e.Name(), ".") {
			out = append(out, e.Name())
		}
	}
	return out
}

// added reports whether every doc in from is already in to and not older.
func added(from, to string) bool {
	files := mdFiles(from)
	if len(files) == 0 {
		return false
	}
	for _, name := range files {
		src, err1 := os.Stat(filepath.Join(from, name))
		dst, err2 := os.Stat(filepath.Join(to, name)) // #nosec G703 -- compares files the app listed itself in the project's knowledge folder
		if err1 != nil || err2 != nil || dst.ModTime().Before(src.ModTime()) {
			return false
		}
	}
	return true
}

func readTrim(p string) string {
	raw, err := os.ReadFile(p) // #nosec G304 -- reads small marker files the app or the loop wrote in the project workspace
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(raw))
}

func touch(p string) error {
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil { // #nosec G301 -- 0755 project directory per security-baseline
		return err
	}
	return os.WriteFile(p, nil, 0o644) // #nosec G306 -- empty marker file the loop scripts read; regular 0644 output per security-baseline
}
