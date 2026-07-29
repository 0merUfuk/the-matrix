package neo

import (
	"sort"
	"time"
)

// PresetDefinition holds the fixed values for a named preset.
// ProjectName, OutputPath, and CreatedDate are always set at runtime.
type PresetDefinition struct {
	DisplayName string
	ProjectType string
	TeamSize    string
	IsInternal  bool
	Stacks      []StackProfile
}

// builtinPresets contains all built-in named presets.
var builtinPresets = map[string]PresetDefinition{
	"internal-go-service": {
		DisplayName: "originating project Go Microservice",
		ProjectType: "single-app",
		TeamSize:    "small",
		IsInternal:  true,
		Stacks: []StackProfile{{
			Name:             "go-internal",
			Language:         "go",
			Framework:        "chi v5",
			ArchStyle:        "layered",
			TestingFramework: "ginkgo + gomega",
			HasQueue:         true,
			DataLayer:        "bun",
		}},
	},
	"go-service": {
		DisplayName: "Go Microservice",
		ProjectType: "single-app",
		TeamSize:    "solo",
		IsInternal:  false,
		Stacks: []StackProfile{{
			Name:             "go-service",
			Language:         "go",
			Framework:        "chi v5",
			ArchStyle:        "layered",
			TestingFramework: "testing",
			HasQueue:         false,
			DataLayer:        "none",
		}},
	},
	"nextjs-solo": {
		DisplayName: "Next.js Solo Project",
		ProjectType: "single-app",
		TeamSize:    "solo",
		IsInternal:  false,
		Stacks: []StackProfile{{
			Name:             "nextjs",
			Language:         "nodejs",
			Framework:        "Next.js",
			ArchStyle:        "feature-based",
			TestingFramework: "jest",
			HasQueue:         false,
			DataLayer:        "prisma",
		}},
	},
}

// GetPreset returns a partially populated ProjectProfile for the named preset.
// ProjectName, OutputPath, and CreatedDate are left empty — caller must fill them.
// Returns (profile, true) if found, (zero, false) if unknown.
// The returned profile owns its own slice memory — safe to mutate without
// affecting future calls.
func GetPreset(name string) (ProjectProfile, bool) {
	def, ok := builtinPresets[name]
	if !ok {
		return ProjectProfile{}, false
	}

	// Deep-copy the stacks slice so callers can't mutate the builtin definition.
	stacks := make([]StackProfile, len(def.Stacks))
	copy(stacks, def.Stacks)

	return ProjectProfile{
		ProjectType: def.ProjectType,
		TeamSize:    def.TeamSize,
		IsInternal:  def.IsInternal,
		CreatedDate: time.Now().Format("2006-01-02"),
		Stacks:      stacks,
	}, true
}

// ListPresets returns all built-in preset names in sorted order.
func ListPresets() []string {
	names := make([]string, 0, len(builtinPresets))
	for name := range builtinPresets {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
