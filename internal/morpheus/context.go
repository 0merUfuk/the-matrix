package morpheus

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/0merUfuk/the-matrix/internal/cli"
	"github.com/0merUfuk/the-matrix/internal/wizard"
)

// ContextProfile represents the .neo.json file produced by neo init.
// Defined here (not imported from neo) to avoid cross-package dependency.
type ContextProfile struct {
	ProjectName string `json:"projectName"`
	// ProjectType is parsed from .neo.json but not mapped to ProjectContext.ProjectType —
	// morp derives its own ProjectType ("internal" vs "general") from IsInternal instead.
	// Future: could map to wizard defaults (e.g., "microservice" → default port suggestions).
	ProjectType string         `json:"projectType"`
	IsInternal  bool           `json:"isInternal"`
	Stacks      []ContextStack `json:"stacks"`
}

// ContextStack represents a single technology stack entry in the context profile.
type ContextStack struct {
	// Name is the stack name as set by neo init (e.g. "go-chi").
	// Reserved for future use — MapContextToProject does not currently read this field.
	Name string `json:"name"`

	Language string `json:"language"` // "go" | "nodejs" | "python" | "dart" | "other"

	// Framework is the framework variant within the stack (e.g. "chi").
	// Reserved for future use — MapContextToProject does not currently read this field.
	Framework string `json:"framework"`
}

// LoadContextProfile reads and validates a .neo.json context file.
// Supported language values in stacks[].language: "go", "nodejs".
// Other values (e.g. "python", "dart") are accepted but will not pre-fill
// the tech stack — the wizard collects it instead.
func LoadContextProfile(path string) (*ContextProfile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("context file not found: %s", path)
		}
		return nil, fmt.Errorf("failed to read context file: %w", err)
	}

	var profile ContextProfile
	if err := json.Unmarshal(data, &profile); err != nil {
		return nil, fmt.Errorf("failed to parse context file: %w", err)
	}

	// Validate required fields
	if profile.ProjectName == "" {
		return nil, fmt.Errorf("context file missing required field: projectName")
	}
	// Validate projectName format — non-TTY path uses this value to derive the
	// service output directory (which is later passed to os.RemoveAll). Without
	// this check, a crafted .neo.json with projectName like "../../../etc"
	// could escape the intended scaffold root. The TTY path validates via
	// wizard.ValidateKebabCase already; mirror that here.
	if err := wizard.ValidateKebabCase(profile.ProjectName); err != nil {
		return nil, fmt.Errorf("context file %q has invalid projectName %q: %w", path, profile.ProjectName, err)
	}
	if len(profile.Stacks) == 0 {
		return nil, fmt.Errorf("context file must include at least one stack")
	}

	return &profile, nil
}

// ApplyContextDefaults fills in the ProjectContext fields normally collected by
// the delta wizard with sensible non-interactive defaults. Used when
// `morp init --context` is invoked without a TTY (e.g. as a subprocess
// from `neo init`, in CI, or via scripted automation), so the chain still
// produces a complete scaffold instead of silently aborting.
//
// Identity, tech, and IsInternal fields are assumed to already be populated by
// MapContextToProject. Only fields the wizard would otherwise prompt for are
// touched, and only when they are still at their zero value — caller-supplied
// overrides are preserved.
func ApplyContextDefaults(ctx *ProjectContext) {
	// Description placeholder — used by templates; loop will refine.
	if ctx.Description == "" {
		ctx.Description = fmt.Sprintf("%s service (scaffolded non-interactively from neo context)", ctx.ServiceName)
	}

	// Tech-specific defaults.
	if ctx.IsGo && ctx.GoVersion == "" {
		ctx.GoVersion = "1.24"
	}

	// Ports — pick the next free originating project port, or 8080 for general projects.
	if ctx.ApiPort == 0 {
		if ctx.IsInternal {
			ctx.ApiPort = SuggestNextApiPort()
		} else {
			ctx.ApiPort = 8080
		}
	}

	// Module path — match the wizard defaults.
	if ctx.ModulePath == "" {
		org := "yourorg"
		if ctx.IsInternal {
			org = "internal"
		}
		if ctx.IsNode {
			ctx.ModulePath = fmt.Sprintf("@%s/%s", org, ctx.ServiceName)
		} else {
			ctx.ModulePath = fmt.Sprintf("github.com/%s/%s", org, ctx.ServiceName)
		}
	}

	// Cycle config — auto-calculate from (empty) entities and patterns. Same
	// formula the wizards use when the user accepts the suggestion.
	if ctx.MaxCycles <= 0 || ctx.DeveloperMaxTurns <= 0 || ctx.TesterMaxTurns <= 0 || ctx.ReviewerMaxTurns <= 0 {
		cfg := CalculateCycles(ctx.Entities, ctx.SpecialPatterns)
		if ctx.MaxCycles <= 0 {
			ctx.MaxCycles = cfg.MaxCycles
		}
		if ctx.DeveloperMaxTurns <= 0 {
			ctx.DeveloperMaxTurns = cfg.DeveloperMaxTurns
		}
		if ctx.TesterMaxTurns <= 0 {
			ctx.TesterMaxTurns = cfg.TesterMaxTurns
		}
		if ctx.ReviewerMaxTurns <= 0 {
			ctx.ReviewerMaxTurns = cfg.ReviewerMaxTurns
		}
	}
}

// MapContextToProject creates a partially populated ProjectContext from a ContextProfile.
// The returned context has identity and tech stack fields filled in; remaining fields
// (ports, entities, auth, etc.) are left at zero values for the wizard to collect.
func MapContextToProject(ctx *ContextProfile) *ProjectContext {
	pc := &ProjectContext{
		ServiceName:       ctx.ProjectName,
		ServiceNamePascal: wizard.ToPascalCase(ctx.ProjectName),
		IsInternal:        ctx.IsInternal,
		CreatedDate:       time.Now().Format("2006-01-02"),
	}

	if ctx.IsInternal {
		pc.ProjectType = "internal"
	} else {
		pc.ProjectType = "general"
	}

	// Map primary stack (first entry)
	if len(ctx.Stacks) > 0 {
		stack := ctx.Stacks[0]
		switch stack.Language {
		case "go":
			pc.Tech = "go"
			pc.IsGo = true
		case "nodejs":
			pc.Tech = "node"
			pc.IsNode = true
			pc.NodeArchitecture = "4-layer"
		default:
			// Unknown language — tech fields remain empty, wizard will collect.
			if stack.Language != "" {
				cli.PrintWarning(fmt.Sprintf("Unknown language %q in context file, tech stack will not be pre-filled", stack.Language))
			}
		}
	}

	return pc
}
