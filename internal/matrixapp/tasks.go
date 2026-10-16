// Package matrixapp is the local web app behind the `matrix` command: one
// screen for people who do not use the CLI. It runs the neo, oracle, morp and
// trinity binaries for them, as fixed tasks with validated inputs.
package matrixapp

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"

	"github.com/0merUfuk/the-matrix/internal/neo"
)

// TaskParams are the user's choices for a task.
type TaskParams struct {
	Preset  string `json:"preset"`
	Runtime string `json:"runtime"`
	Replace bool   `json:"replace"`
	Stack   string `json:"stack"` // add-knowledge: which researched stack

	home string
}

type task struct {
	tool  string
	title string
	write bool
	args  func(dir string, p TaskParams) ([]string, error)
	ready func(dir string, p TaskParams) error
}

var errAlreadySetUp = errors.New("this project is already set up; setting it up again replaces CLAUDE.md and the .claude files")

// tasks is the complete list of things the app can run. Wizards run through
// their non-interactive flags; nothing else is reachable from the browser.
var tasks = map[string]task{
	"setup": {
		tool: "neo", title: "Set up agents", write: true,
		args: func(dir string, p TaskParams) ([]string, error) {
			if !publicPreset(p.Preset) {
				return nil, errors.New("choose a project type")
			}
			if p.Runtime != "claude" && p.Runtime != "codex" {
				return nil, errors.New("choose Claude Code or Codex")
			}
			return []string{"init", "--preset", p.Preset, "--path", dir, "--runtime", p.Runtime}, nil
		},
		ready: func(dir string, p TaskParams) error {
			if isSetUp(dir) && !p.Replace {
				return errAlreadySetUp
			}
			return nil
		},
	},
	"check-setup": {
		tool: "neo", title: "Check setup",
		args:  fixed("doctor", "--path", "{dir}"),
		ready: needSetUp,
	},
	"health": {
		tool: "trinity", title: "Health check",
		args:  fixed("health", "--path", "{dir}"),
		ready: needSetUp,
	},
	"sync": {
		tool: "trinity", title: "Add researched knowledge", write: true,
		args: fixed("sync", "--path", "{dir}"),
		ready: func(dir string, _ TaskParams) error {
			if len(researchReady(dir)) == 0 {
				return errors.New("there is no finished research to add yet")
			}
			return nil
		},
	},
	"add-knowledge": {
		tool: "trinity", title: "Add knowledge for the agents", write: true,
		args: func(dir string, p TaskParams) ([]string, error) {
			ws := researchWorkspace(p.home, dir, p.Stack)
			return []string{"sync", "--from", filepath.Join(ws, "output", p.Stack), "--to", filepath.Join(dir, ".claude", "knowledge", p.Stack), "--path", dir}, nil
		},
		ready: func(dir string, p TaskParams) error {
			if _, ok := findStack(dir, p.Stack); !ok {
				return errors.New("choose which knowledge to add")
			}
			if len(mdFiles(filepath.Join(researchWorkspace(p.home, dir, p.Stack), "output", p.Stack))) == 0 {
				return errors.New("there is no finished research to add yet")
			}
			return nil
		},
	},
	"snapshot": {
		tool: "trinity", title: "Save a status snapshot", write: true,
		args:  fixed("self-state", "--path", "{dir}"),
		ready: needSetUp,
	},
	"loop-status": {
		tool: "morp", title: "Build loop progress",
		args: fixed("status"),
		ready: func(dir string, _ TaskParams) error {
			if !exists(dir, ".autonomous/loop.sh") {
				return errors.New("this project has no build loop yet")
			}
			return nil
		},
	},
}

func fixed(args ...string) func(string, TaskParams) ([]string, error) {
	return func(dir string, _ TaskParams) ([]string, error) {
		out := make([]string, len(args))
		for i, a := range args {
			if a == "{dir}" {
				a = dir
			}
			out[i] = a
		}
		return out, nil
	}
}

func needSetUp(dir string, _ TaskParams) error {
	if !isSetUp(dir) {
		return errors.New("set up the agents first")
	}
	return nil
}

func exists(dir, rel string) bool {
	_, err := os.Stat(filepath.Join(dir, filepath.FromSlash(rel)))
	return err == nil
}

// isSetUp reports whether a project already has an agent ecosystem.
func isSetUp(dir string) bool {
	return exists(dir, "CLAUDE.md") || exists(dir, "AGENTS.md")
}

// researchReady lists stacks whose oracle output is waiting to be synced,
// read from the project's .trinity.json.
func researchReady(dir string) []string {
	data, err := os.ReadFile(filepath.Join(dir, ".trinity.json"))
	if err != nil {
		return nil
	}
	var cfg struct {
		Stacks []struct {
			Name string `json:"name"`
			From string `json:"from"`
		} `json:"stacks"`
	}
	if json.Unmarshal(data, &cfg) != nil {
		return nil
	}
	var ready []string
	for _, s := range cfg.Stacks {
		from := s.From
		if !filepath.IsAbs(from) {
			from = filepath.Join(dir, from)
		}
		if entries, err := os.ReadDir(from); err == nil && len(entries) > 0 {
			ready = append(ready, s.Name)
		}
	}
	return ready
}

// Preset is a project type offered in the setup form.
type Preset struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Stack string `json:"stack"`
}

// presets returns the public neo presets (maintainer-only ones are hidden).
func presets() []Preset {
	var out []Preset
	for _, id := range neo.ListPresets() {
		p, ok := neo.GetPreset(id)
		if !ok || p.IsInternal {
			continue
		}
		stack := ""
		if len(p.Stacks) > 0 {
			stack = p.Stacks[0].Framework
		}
		out = append(out, Preset{ID: id, Name: neo.PresetDisplayName(id), Stack: stack})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func publicPreset(id string) bool {
	for _, p := range presets() {
		if p.ID == id {
			return true
		}
	}
	return false
}
