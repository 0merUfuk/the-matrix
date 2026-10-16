**Version**: 2.6
**Created**: 2026-03-12
**Last Updated**: 2026-08-10
**Authors:** Ömer Ufuk

---

# the-matrix — Autonomous Agent Ecosystem

The-matrix is a CLI-driven, self-refreshing autonomous agent ecosystem for any software project. Its standalone tools provision, scaffold, and maintain runtime-specific ecosystems for Claude Code and OpenAI Codex, from knowledge synthesis to service scaffolding and live ecosystem maintenance.

> **Public clone note**: this repository no longer ships its own root `.claude/` ecosystem. That directory is local/private maintainer state, ignored by Git, and absent from fresh public clones.
>
> **Maintainer continuation**: if you are in a private maintainer checkout where local `.claude/` context exists, use it as supplemental continuation context. Otherwise start from `README.md`, `ARCHITECTURE.md`, `CONTRIBUTING.md`, and the live source files.

---

## Tool Inventory

| Tool | Entry Point | Latest Public Release | Role |
|------|-------------|-----------------------|------|
| `neo` | `cmd/neo` | v1.9.0 | Meta-CLI — provisions ecosystems, prints oracle/trinity guidance, invokes morp for multi-repo scaffolds |
| `morp` | `cmd/morp` | v1.9.0 | Service scaffolding + autonomous development loops |
| `oracle` | `cmd/oracle` | v1.9.0 | Researches any tech stack → best-practice knowledge docs |
| `trinity` | `cmd/trinity` | v1.9.0 | Maintenance runtime — keeps ecosystems alive and fresh |
| `matrix` | `cmd/matrix` | *(unreleased)* | The Matrix app — local browser UI that runs the four tools for people who do not use the CLI |
| `sentinel` | *(future)* | — | Ecosystem health monitoring |

> **Note**: All 4 tools are Go binaries in a single repo (`github.com/0merUfuk/the-matrix`). Node.js predecessors have been removed.
>
> **Release/version note**: v1.9.0 is the latest public release. The `v1.8.0` tag marks the clean root commit; `main` continues from that public baseline. GoReleaser injects the tag version into release binaries, while the current command source constants and Makefile defaults remain `1.9.0-dev`; plain source builds use that default unless version ldflags override it.

---

## System Topology

```
the-matrix (ecosystem name)
│
├── neo          ← meta-CLI: init / analyze / doctor / status / update / registry
│                  provisions ecosystems and prints oracle/trinity guidance
│                  invokes morp only for multi-repo scaffolds
│                  gains intelligence in v2+ (memory, patterns, self-improvement)
│
├── oracle       ← independently invoked knowledge synthesis
├── morp         ← standalone scaffolding + autonomous loops; neo invokes it
│                  only for multi-repo service scaffolds
├── trinity      ← independently invoked maintenance: sync, health, refresh
│
├── runtime      ← ARAL registry + Claude Code/Codex adapters
│                  selected by flag, project config, or context-file detection
│
└── sentinel     ← future: ecosystem health monitoring
```

### Data Flow

The `.claude/` paths here describe generated target-project knowledge or Claude integration. They are not files expected to exist at this repository root in a public clone.

```
--runtime / .matrix.yaml / context-file detection → Claude Code or Codex adapter
oracle (independent) → synthesizes stack knowledge → .claude/knowledge/
trinity (independent) → syncs output and performs runtime-aware health checks
morp (standalone) → scaffolds services, runtime context, and autonomous loops
neo → provisions runtime context, prints oracle/trinity commands, and invokes morp
      only for multi-repo service scaffolds
```

### Runtime Selection

`neo init`, `morp init`, `morp loop`, and `oracle research` accept `--runtime claude|codex`. An explicit choice is persisted to project-local `.matrix.yaml`/`.matrix.yml`; with no flag, selection uses project config, then `CLAUDE.md`, then `AGENTS.md`, then the Claude compatibility default. Global config is not used for runtime detection.

The runtime contract lives in `internal/runtime/`. Built-in adapters register through `init()` and are imported by `internal/runtime/builtin`. Runtime-independent Go calls use `subprocess.Invoke`; `subprocess.CallClaude` remains a Claude-only compatibility wrapper. Runtime-specific shell config defines `check_auth()`, `run_agent()`, and `run_finalizer()` for shared loop scripts.

---

## Agent Ecosystems

Neo provisions 6 core role definitions into generated target projects:

| Agent | Role |
|-------|------|
| `manager` | Orchestrator — coordinates the agent workflow |
| `developer` | Senior engineer — implements code in a scoped worktree |
| `tester` | QA engineer — writes and runs tests |
| `reviewer` | Adversarial reviewer — read-only quality pass |
| `strategist` | Product/technical strategist — researches and designs work |
| `security-reviewer` | OWASP/security reviewer — read-only security pass |

Claude ecosystems write these roles to `.claude/agents/` and also provision skills, rules, and settings. Codex ecosystems consolidate role guidance in `AGENTS.md` and intentionally omit separate generated agent and skill files. Both use `.claude/knowledge/` as the shared knowledge location. Morpheus embeds shared prompt bodies plus runtime-specific wrappers and shell configuration.

This repo's maintainer checkout may have extra private workflow agents and continuation context under its ignored root `.claude/` directory. Those files are not part of the public repository; do not assume they exist unless you are explicitly working in that private local checkout.

---

## Before Working Here

1. In a fresh public clone, read `README.md`, `ARCHITECTURE.md`, `CONTRIBUTING.md`, and the relevant source/tests.
2. In a maintainer-private checkout only, local `.claude/` context may provide supplemental vision, ADR, service-state, and next-step notes.
3. Treat live files as ground truth when docs and implementation disagree.

---

## Install

### From source

```bash
git clone https://github.com/0merUfuk/the-matrix.git
cd the-matrix
make build
# Binaries in ./bin/
```

### Via Homebrew

```bash
brew tap 0merUfuk/thematrix
brew install neo morp oracle trinity
brew install 0merUfuk/thematrix/matrix   # optional browser app; depends on the four tools
```

---

## Key Commands

```bash
# Build all tools
make build

# Or build individually
make neo morp oracle trinity

# Release (tags + pushes, triggers CI → GoReleaser → Homebrew tap)
make release VERSION=X.Y.Z SUMMARY="one-line release summary"

# neo (meta-CLI orchestrator)
neo init --runtime codex # Wizard → provisions AGENTS.md + runtime config
neo init --runtime claude # Wizard → provisions CLAUDE.md + .claude/ integration
neo analyze              # Auto-detect existing codebase → generate Claude integration (no runtime flag yet)
neo doctor               # Detect runtime and validate supported ecosystem files
neo status               # Ecosystem dashboard — knowledge freshness, tool versions
neo update               # Staleness check → prints suggested oracle/trinity commands
neo registry list        # List available packs from the knowledge registry
neo registry search      # Filter packs by language/framework
neo registry info        # Show detailed pack metadata and topic list
neo registry pull        # Download and validate pack docs to target directory

# morp (service scaffolding + autonomous loops)
morp init --runtime codex  # Scaffold a service with Codex context
morp loop --runtime codex  # Add a Codex-driven loop to an existing project
morp doctor              # Validate scaffolded service before running the loop
morp status              # Loop progress at a glance
morp finalize            # Post-loop project context updater
morp report              # Autonomous loop analysis

# oracle (knowledge synthesis)
oracle research --runtime codex  # Wizard-guided Codex knowledge collection
oracle status            # Research loop progress at a glance
oracle inject --from <dir> --to <dir>  # Validate + inject output into .claude/knowledge/
oracle update --workspace <dir> --knowledge <dir>   # Re-research stale topics only
oracle export --from <dir> --format <fmt> --to <dir>  # Export knowledge to AI tool formats

# trinity (maintenance runtime)
trinity health           # Read-only ecosystem integrity check
trinity health --path .  # Check a specific project
trinity sync --from <dir> --to <dir>  # Push oracle output to .claude/knowledge/
trinity sync             # Read from .trinity.json config
trinity sync --dry-run   # Preview without writing
trinity refresh          # Detect stale docs, suggest refresh workflow
trinity refresh --dry-run  # Preview stale topics only
trinity self-state       # Write state.v1 JSON snapshot to .claude/state/
trinity self-state --json  # Emit JSON to stdout, no persistence (~1-2s)
trinity self-state --out <path>       # Write snapshot to a custom path
trinity self-state --prune            # Enforce retention (keep last 12 snapshots)

# matrix (The Matrix app, for non-CLI users)
matrix                   # Open the app in the browser (127.0.0.1:7474)
matrix --no-open         # Run without opening a browser (Homebrew service)
matrix --path <dir>      # Add a project folder to the app's list
```

---

## Work Protocol

- **Session start**: Fresh public clone: read `README.md`, `ARCHITECTURE.md`, `CONTRIBUTING.md`, and relevant source/tests. Maintainer-private checkout: also consult local `.claude/` context if it exists.
- **Active work**: Track progress in `tasks/` (gitignored, session-local)
- **Architectural decisions**: Public contributors should capture rationale in issues, PRs, or tracked docs. Maintainers may also update private local ADR context when present.
- **Slice index access**: Slices populated from analysis results (`ProjectContext.Stacks`, `.Services`, etc.) must be guarded with `len() > 0` before `[0]` index access — analysis may return empty slices on unrecognized stacks (see issue #90)
- **Template default changes**: When changing a template default value (e.g., `SECURITY_MAX_TURNS`, `MAX_TURNS`), grep `*_test.go` for the old value before finalizing — stale assertions from prior sessions aren't caught until the next CI run
- **Issues found**: Use GitHub issues, PR notes, tracked docs, or local maintainer context depending on scope
- **Session end**: Update tracked docs when public behavior changed; update private maintainer context only if it exists locally
- **Never commit**: service source code to this repo
- **Tool work**: all 4 tools live in this repo under `cmd/` and `internal/`
- **Runtime changes**: preserve the no-flag Claude path, test explicit Claude and Codex generation, and keep shared templates free of hardcoded runtime context paths

---

## Workflow Orchestration

### 1. Plan Mode Default

- Enter plan mode for ANY cross-tool change — effects cascade across morp, oracle, trinity, neo
- If a tool boundary decision becomes unclear, STOP and re-plan — don't assign work to the wrong tool
- Use plan mode for `/audit` remediation sequences and release planning
- In maintainer-private checkouts, local tool action files may exist under `.claude/`; in public clones, track action items in `tasks/` or the issue/PR

### 2. Subagent Strategy

- Use subagents liberally when the available environment supports them — especially for audit-style reviews
- Offload per-tool verification to Explore subagents; keep orchestration context clean
- For ecosystem-wide analysis, launch one subagent per tool simultaneously
- One bounded scope per subagent — version check, code check, docs check, structure check are each their own agent

### 3. Self-Improvement Loop

- After ANY correction: update `tasks/lessons.md` with the pattern
- Ecosystem-level lessons (wrong tool boundary, wrong context file) go in `tasks/lessons.md`
- Tool-level lessons go in that tool's own session `tasks/lessons.md`
- Review lessons at session start before touching any tool

### 4. Verification Before Done

- Never mark a context file updated without checking ground truth (git, code, filesystem)
- Run an audit/review pass before ending any session that touched context files
- Ask yourself: "If the next session starts cold from these context files, will they be accurate?"
- Audit output beats your own assumptions about what's current

### 5. Demand Elegance (Balanced)

- For tool boundary decisions: ask "does this belong here or should neo orchestrate it?"
- If a cross-tool flow feels convoluted, simplify — these tools are designed to chain cleanly
- Skip this for simple context file fixes — don't over-engineer the obvious
- Resist feature creep across tool boundaries; the right tool for the right job

### 6. Autonomous Work

- When given ecosystem drift (stale context, wrong versions, open issues): audit, patch, and verify before reporting done
- Use the release policy and git log to compute versions; do not tag or push unless the task explicitly asks for a release
- Zero hand-holding needed — read the action items, understand the scope, execute

---

## Task Management

1. **Plan First**: Write plan to `tasks/todo.md` with checkable items
2. **Verify Plan**: Check in before starting implementation
3. **Track Progress**: Mark items complete as you go
4. **Explain Changes**: High-level summary at each step
5. **Document Results**: Add review section to `tasks/todo.md`
6. **Capture Lessons**: Update `tasks/lessons.md` after corrections

---

## Core Principles

- **Simplicity First**: Minimal changes. Don't add capabilities to a tool without a tracked action item.
- **No Laziness**: Verify claims against ground truth. Never trust context files without checking.
- **Minimal Impact**: A change to one tool should not break another. Respect tool boundaries.

---

## Reference Index

| Document | Purpose |
|----------|---------|
| `README.md` | Install, command overview, release note, and repository hygiene |
| `ARCHITECTURE.md` | Public system design and tool boundaries |
| `CONTRIBUTING.md` | Fresh-clone contributor workflow |
| `go.mod` | Minimum Go version and module dependencies |
| `Makefile` | Build, test, vet, and release commands |
| `internal/runtime/` | ARAL interface, registry, detection, selection, and built-in adapters |
| `.github/workflows/` | Public CI, security, and release automation |
| Maintainer-local `.claude/` context | Private continuation notes, ADRs, agent state, and action lists; absent from public clones |
