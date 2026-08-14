// Package runtime defines the agent runtime abstraction layer.
package runtime

// Runtime represents an AI agent execution runtime (Claude Code, Codex, etc.)
type Runtime interface {
	// Identity
	Name() string        // "claude", "codex", "gemini", "hermes"
	DisplayName() string // "Claude Code", "OpenAI Codex", etc.

	// Capabilities — what this runtime supports
	Capabilities() Capabilities

	// Invocation — build the command to run an agent headlessly
	BuildCommand(opts InvokeOpts) (*Command, error)

	// Auth — check if the runtime is installed and authenticated
	CheckAuth() AuthStatus

	// Context files — what files this runtime reads for project context
	ContextFiles() ContextFileSpec

	// Agent definitions — how agent roles are expressed in this runtime
	AgentFormat() AgentFormat

	// Skill definitions — how skills are expressed (if supported)
	SkillFormat() *SkillFormat

	// Settings format — permission/tool-restriction format
	SettingsFormat() SettingsFormat

	// Environment — env vars to set/unset for subprocess invocation
	EnvModifications() []EnvModification

	// Tool translation — translate universal tool names to runtime-specific format
	TranslateTools(universalTools []string) string

	// Model resolution — translate universal model tiers to runtime-specific model names
	ResolveModel(tier ModelTier) string
}

// Capabilities describes what a runtime can do.
type Capabilities struct {
	HeadlessMode       bool // can run non-interactively with a prompt
	ToolRestrictions   bool // can restrict which tools an agent uses
	MaxTurnsLimit      bool // supports turn/iteration limits
	ModelSelection     bool // supports choosing between models
	JSONOutput         bool // can output structured JSON
	SessionPersistence bool // supports session state (to disable for headless)
	AgentDefinitions   bool // supports named agent profiles
	SkillDefinitions   bool // supports named skill/command workflows
	WorktreeIsolation  bool // supports git worktree isolation
	Memory             bool // supports cross-session memory
	MCPIntegration     bool // supports MCP tool servers
	AuthCheck          bool // has a CLI auth status command
}

// InvokeOpts contains the universal parameters for invoking an agent.
type InvokeOpts struct {
	Prompt     string
	Role       string   // developer, tester, reviewer, etc.
	Tools      []string // universal tool names
	MaxTurns   int
	Model      ModelTier // Fast, Balanced, Powerful
	OutputJSON bool
	WorkingDir string
}

// Command is the resolved command to execute.
type Command struct {
	Binary string
	Args   []string
	Env    map[string]string // env vars to set
	Unset  []string          // env vars to unset
}

// ModelTier is a universal model classification.
type ModelTier int

const (
	ModelFast     ModelTier = iota // cheap, fast (haiku-class)
	ModelBalanced                  // mid-tier (sonnet-class)
	ModelPowerful                  // most capable (opus-class)
)

// AuthStatus reports runtime installation and authentication.
type AuthStatus struct {
	Installed     bool
	Authenticated bool
	BinaryName    string
	Error         string
}

// ContextFileSpec describes what context files a runtime reads.
type ContextFileSpec struct {
	PrimaryContext string // "CLAUDE.md" or "AGENTS.md"
	ContextDir     string // ".claude/" or "" (none)
	AgentDir       string // ".claude/agents/" or ""
	SkillDir       string // ".claude/skills/" or ""
	RulesDir       string // ".claude/rules/" or ""
	SettingsFile   string // ".claude/settings.json" or ""
	KnowledgeDir   string // ".claude/knowledge/" (neutral, always present)
}

// AgentFormat describes how agent definitions are structured.
type AgentFormat struct {
	Type        AgentFormatType    // frontmatter, prompt-file, agents-md-section
	Directory   string             // where agent files go
	FilePattern string             // "developer.md" or "developer.txt"
	Frontmatter []FrontmatterField // required fields and their names
}

// AgentFormatType identifies an agent definition format.
type AgentFormatType int

const (
	AgentFormatFrontmatter     AgentFormatType = iota // .claude/agents/*.md with YAML frontmatter
	AgentFormatPromptFile                             // standalone prompt files
	AgentFormatAgentsMDSection                        // sections within AGENTS.md
)

// SkillFormat describes how skill definitions are structured.
type SkillFormat struct {
	Type        SkillFormatType
	Directory   string
	FilePattern string
	Frontmatter []FrontmatterField
}

// SkillFormatType identifies a skill definition format.
type SkillFormatType int

const (
	SkillFormatFrontmatter SkillFormatType = iota
	SkillFormatPromptFile
	SkillFormatNone
)

// SettingsFormat describes the permission and tool-restriction format.
type SettingsFormat struct {
	Type       SettingsFormatType // json, yaml, sandbox-flags, none, toml
	Path       string             // file path if file-based
	ToolSyntax ToolSyntax         // how tools are expressed
}

// SettingsFormatType identifies a runtime settings format.
type SettingsFormatType int

const (
	SettingsJSON SettingsFormatType = iota
	SettingsYAML
	SettingsSandboxFlags
	SettingsNone
	SettingsTOML
)

// ToolSyntax identifies how tool restrictions are expressed.
type ToolSyntax int

const (
	ToolSyntaxBashPattern ToolSyntax = iota
	ToolSyntaxCSV
	ToolSyntaxNone
	ToolSyntaxSandboxMode
)

// EnvModification describes an environment variable to set or unset.
type EnvModification struct {
	Action string // "set" or "unset"
	Key    string
	Value  string // (for "set")
}

// FrontmatterField describes a field in an agent or skill definition.
type FrontmatterField struct {
	Name     string
	Required bool
	Type     string // "string", "list", "bool"
}
