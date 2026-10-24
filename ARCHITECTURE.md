# Architecture

This document covers the-matrix's system design, tool boundaries, data flow, and key architectural decisions. The maintainer's full founding context and ADR log now live in local/private `.claude/` state that is not shipped in public clones; this file is the public architecture summary.

---

## Overview

the-matrix is four Go CLIs that provision and maintain autonomous agent ecosystems. The Agent Runtime Abstraction Layer (ARAL) makes Claude Code and OpenAI Codex first-class runtime choices while preserving each tool's boundary. Neo provisions the ecosystem and guides the flow; oracle and trinity remain independently invoked commands, while neo invokes morp only for multi-repo service scaffolds.

```
the-matrix (ecosystem name)
│
├── neo          ← meta-CLI: init / analyze / doctor / status / update / registry
│                  provisions ecosystems, prints oracle/trinity guidance,
│                  and invokes morp only for multi-repo scaffolds
│
├── morp         ← service scaffolding + autonomous dev loops; standalone,
│                  or invoked by neo for multi-repo scaffolds
├── oracle       ← independently invoked knowledge synthesis
├── trinity      ← independently invoked maintenance: sync, health, refresh, self-state
│
└── sentinel     ← future: ecosystem health monitoring (not yet built)
```

---

## The Core Insight

> **You fix AI agent reliability at setup time, not at runtime.**

Most agent-runtime setups are hand-crafted, generic, and stale within weeks. the-matrix attacks the layer *around* the agent — permissions, tool restrictions, role guidance, ADRs, and knowledge quality — at provisioning time. These are setup-time decisions that determine runtime reliability.

---

## Tool Boundaries

### Why 4 binaries instead of 1?

Each tool has a distinct operational shape:

| Tool | Runtime | Duration | Replaced by? |
|------|---------|----------|--------------|
| **neo** | CLI wizard + ecosystem generation | 30–60s | — |
| **oracle** | 4-cycle research loop with the selected runtime | 3–8 min | Any tool that produces compatible knowledge docs |
| **morp** | Template rendering + runtime-neutral agent invocation | 10–30s | Any scaffolder that produces runtime context + `.autonomous/` |
| **trinity** | File I/O + staleness calculation | <2s | Any tool that syncs knowledge docs |

Oracle's research loop runs for minutes and is embarrassingly parallel. Trinity's sync is sub-second. Mixing them in one binary would create a tool with unclear scope and incompatible operational shapes. The contract between them is the **output directory layout** — not a protocol, not an API.

The original ADR rationale behind this split is maintainer-local; the public boundary is the command and directory contract summarized here.

---

## Data Flow

The `.claude/` paths below are generated target-project knowledge or Claude-integration paths, not root files expected to exist in this public repository.

```
Step 1: oracle researches target stack
  → tasks/research/*.md (Tier-1 sourced, quality-gated)
  → human reviews research
  → output/{stack}/ (17 standard knowledge docs)

Step 2: trinity syncs into agent ecosystem
  trinity sync --from output/go/ --to .claude/knowledge/go-rules/
  → .claude/knowledge/go-rules/*.md (fresh, versioned)

Step 3: morp scaffolds a service
  morp init --runtime claude|codex
  → reads .claude/knowledge/ (gold standards from oracle output)
  → generates service/ with .autonomous/loop.sh + tasks/
  → writes CLAUDE.md + .claude/ integration, or AGENTS.md for Codex

Step 4: autonomous loop builds the service
  bash .autonomous/loop.sh
  → (cycle > 1) Manager reviews and reprioritizes
  → (if decisions pending) Strategist resolves them
  → Developer → Tester → Reviewer → Security pipeline
  → both Reviewer AND Security must approve before finalizing
  → produces production-ready microservice

Step 5: trinity maintains freshness
  trinity refresh (run independently)
  → reads .claude/knowledge/*.md frontmatter (Created date)
  → identifies docs older than threshold (default: 6 months)
  → prints oracle re-research and trinity sync guidance

Step 6: neo detects and guides
  neo init --runtime claude|codex
           → provisions runtime context and prints oracle/trinity commands
           → invokes morp only for multi-repo service scaffolds
  neo update → detects staleness and prints suggested refresh commands
  neo doctor → detects the runtime and validates supported integration files
```

---

## Repository Structure

```
the-matrix/
├── cmd/               # CLI entrypoints (one per tool)
│   ├── neo/
│   ├── morp/          # morp CLI entrypoint (calls internal/morpheus)
│   ├── oracle/
│   └── trinity/
├── internal/          # Shared packages + per-tool implementations
│   ├── cli/           # Terminal formatting (lipgloss)
│   ├── config/        # JSON/file I/O
│   ├── tmpl/          # Template engine (text/template + 17 FuncMap helpers)
│   ├── wizard/        # Interactive prompts (huh v2)
│   ├── runtime/       # ARAL interface, registry, detection, and adapters
│   │   ├── builtin/   # Registers built-in adapters
│   │   ├── claude/    # Claude Code adapter
│   │   └── codex/     # OpenAI Codex adapter
│   ├── subprocess/    # Runtime-neutral invocation + legacy CallClaude wrapper
│   ├── staleness/     # Document age calculation
│   ├── matrixcfg/     # .matrix.yaml schema and runtime persistence
│   ├── neo/           # neo implementation + embedded runtime templates
│   ├── morpheus/      # morp implementation + shared/runtime template layers
│   ├── oracle/        # oracle implementation + export/runtime templates
│   │   └── export/    # Format adapter pattern: Antigravity, Cursor, Copilot, Windsurf, AGENTS.md
│   ├── registry/      # Knowledge registry (pack catalog + validation)
│   └── trinity/       # trinity implementation (no templates)
├── .github/workflows/ # CI: build, test, vet, security gates, release
├── go.mod             # Single module, minimum Go 1.26.10
├── Makefile           # build, test, vet, release
└── .goreleaser.yml    # Cross-platform binary builds + Homebrew tap
```

Template trees use `shared/` for runtime-independent content and `runtimes/<name>/` for adapter-specific wrappers and shell configuration. Tracked `internal/**/templates/.claude/` paths that remain are embedded Claude target-project assets. A root `.claude/` directory in a maintainer checkout is ignored local/private agent state and is absent from a fresh public clone.

---

## Key Design Decisions

### go:embed for Template Distribution

All template artifacts are compiled into the binaries via `go:embed`. No external template files are required at runtime; agent-powered workflows require the selected runtime CLI and the script loops require Bash. Template changes require recompilation, but embedding eliminates the "template not found" failure mode common in filesystem-based template loaders.

### Shared Internal Packages

All 4 tools share `cli/`, `config/`, `tmpl/`, `wizard/`, `runtime/`, `subprocess/`, `staleness/`, and `matrixcfg/`. This reduces duplication and ensures consistent behavior across tools (for example, the same runtime selection and staleness calculation).

### Agent Runtime Abstraction Layer (ARAL)

`internal/runtime.Runtime` defines identity, capabilities, command construction, authentication, context-file layout, agent/skill/settings formats, tool translation, environment changes, and model-tier resolution. Adapters register themselves in `init()`; `internal/runtime/builtin` imports the built-in Claude Code and Codex adapters so binaries can select either without central switch statements.

Runtime selection follows this order:

1. An explicit `--runtime claude|codex` flag.
2. A project-local `.matrix.yaml` or `.matrix.yml` `runtime:` value.
3. Context-file detection: `CLAUDE.md` before `AGENTS.md`.
4. Claude Code as the compatibility default.

Only an explicit flag is persisted. `neo init`, `morp init`, `morp loop`, and `oracle research` write that selection to the target project's Matrix config; inferred/default selection leaves config untouched. Global Matrix config does not participate in runtime detection.

| Adapter | Context contract | Product capabilities used |
|---------|------------------|---------------------------|
| Claude Code | `CLAUDE.md`, `.claude/agents/`, `.claude/skills/`, rules, JSON settings | Named agents, skills, per-call tool allowlists, turn limits, worktree isolation |
| OpenAI Codex | `AGENTS.md`, shared `.claude/knowledge/`, no separate generated agent/skill files | Consolidated role guidance, sandbox modes, model selection, JSONL event output |

The runtime-specific `config.sh` defines `check_auth()`, `run_agent()`, and where needed `run_finalizer()`. Shared `loop.sh`, `finalizer.sh`, and oracle scripts call those functions without knowing the CLI syntax. Go call sites use `subprocess.Invoke`; `subprocess.CallClaude` remains as a backward-compatible wrapper that always selects Claude.

To add a runtime, implement `Runtime`, register the adapter from `init()`, include it in `runtime/builtin`, add runtime-specific context/config templates, and cover selection, generation, health, and subprocess behavior with tests. Shared prompts should refer to runtime-derived context paths rather than hardcoded CLI-specific files.

### Oracle Source Quality Tiers

Oracle classifies research sources into 4 tiers:

| Tier | Sources | Trust |
|------|---------|-------|
| **T1** | Official docs, RFCs, specs, changelogs | Always prefer |
| **T2** | GitHub ≥500★, maintained <12mo | High trust |
| **T3** | Conference talks, established tech blogs | Use carefully |
| **T4** | Tutorials, "for beginners" | Ignore |

The reviewer agent enforces: ≥2 Tier-1 sources, ≥5 code examples, ≥3 anti-patterns per topic.

### Oracle Export — Universal Context Generator

Oracle's knowledge docs can be exported to formats consumed by all major AI coding tools:

| Format | Output |
|--------|--------|
| `agents-md` | `AGENTS.md` (consolidated, with TOC) |
| `antigravity` | `.agents/rules/*.md` (YAML frontmatter with `trigger: model_decision`, auto-splits <24KB) |
| `cursor` | `.cursor/rules/*.mdc` (MDC frontmatter, auto-splits >6K) |
| `copilot` | `.github/copilot-instructions.md` |
| `windsurf` | `.windsurfrules` |

Architecture: format adapter pattern with `init()`-based auto-registration. New formats are added by implementing the `Format` interface and calling `Register()` in `init()`.

### The Matrix App

`matrix` (`cmd/matrix`, `internal/matrixapp`) is a fifth binary for people who do not use the command line. It is not part of any single tool: it is the browser face of the whole ecosystem and runs the four installed binaries as separate processes. Its Homebrew formula depends on the four tool formulas and can run as a `brew services` service.

The app offers a fixed table of tasks (`tasks.go`) rather than arbitrary commands. Wizards run through their existing non-interactive paths: setup is `neo init --preset <preset> --runtime <runtime>`, with the preset and runtime chosen in a form and validated against the public presets. Because `neo init` overwrites `CLAUDE.md` and `.claude/` files without asking, the app only offers setup for projects that are not set up yet; setting up again requires an explicit replace confirmation. Sync is offered only when a stack's oracle output (from `.trinity.json`) exists. Data shown in the app comes from trinity's structured collectors (`CollectHealth`, `CollectDashboard`).

Short tasks run in the project folder without a shell, with a 60 s timeout, a 256 KB output cap and at most two at a time.

Research and building are long jobs (`jobs.go`) with their own model: no time limit, a sequence of steps (a tool command, then the generated `.autonomous/loop.sh`), state in `~/.the-matrix/jobs/<id>/job.json` and output streamed straight to `log.txt` beside it (0700 folders, 0600 files, last 20 jobs kept), at most one per project and three in total. Each command runs in its own process group, so stopping a job, or quitting the app, ends the loop and the agent it started. A job still marked running after a crash is marked interrupted on the next start and its loop gets a `tasks/STOP` file, which the loops already honour; the next build job clears it.

- **Research** (`research.go`) turns a short form, prefilled from the stack in `.neo.json`, into an oracle `StackContext`, runs `oracle research --config … --output-dir … --force --runtime …`, then the research loop. The workspace lives in `~/.the-matrix/research/<project>-<hash>/<stack>/` so the project only receives finished docs. The app shows the loop's `tasks/status.txt` as plain progress, lists the findings for reading at the review gate and opens the gate (`tasks/SYNTHESIZE_APPROVED`) when the user approves; the review can also be skipped up front. Finished docs from `output/<stack>/` are added with `trinity sync --from … --to .claude/knowledge/<stack> --path …`.
- **Build** (`build.go`) runs `morp loop --path … --non-interactive --goal … --max-cycles 3|6|10 --runtime …` and then the build loop, or only the loop to continue. Progress comes from the job log (round, current role, exit reason), `tasks/todo.md` checkboxes and `tasks/decisions-pending.md`. The project must be a Git repository because the loop commits each step.

The runtime is the project's detected one; long jobs start only when its CLI is installed. Commands get a PATH of the tools' folders, the app's own PATH and the usual install places, because a Homebrew service starts with a very short PATH. The server listens on loopback only, answers only loopback hosts, requires a same-origin JSON request for every change, refuses write tasks without confirmation and only accepts project folders inside the home directory. The project list lives in `~/.the-matrix/projects.json` (0600). The UI is plain HTML/CSS/JS embedded with `go:embed`.

### Unified Configuration

`.matrix.yaml` is the optional unified project config; `.matrix.yml` is also recognized. General tool configuration still follows each command's documented precedence. ARAL adds a project-local `runtime:` field consumed by runtime selection, `neo doctor`, and `trinity health`, and written when a supported command receives an explicit `--runtime` flag.

The configuration contract is public in code and command behavior; deeper ADR notes are maintainer-local.

---

## Agent Ecosystem

The-matrix has two related `.claude/` concepts even when Codex is selected:

1. **Generated target-project integration and knowledge** — Claude uses `.claude/` for agents, skills, rules, settings, and knowledge; Codex uses it only as the shared knowledge location.
2. **This repo's maintainer-local ecosystem** — ignored private state in the root `.claude/` directory when present locally. It is not tracked and is absent from fresh public clones.

### Generated Core Agents

Neo embeds 6 core roles for generated ecosystems. Claude writes them as separate agent definitions; Codex consolidates the role guidance into `AGENTS.md`:

| Agent | Role |
|-------|------|
| `manager` | Orchestrator — coordinates the agent workflow |
| `developer` | Senior engineer — implements code in a scoped worktree |
| `tester` | QA engineer — writes and runs tests |
| `reviewer` | Adversarial reviewer — read-only quality pass |
| `strategist` | Product/technical strategist — researches and designs work |
| `security-reviewer` | OWASP/security reviewer — read-only security pass |

### Generated Skills And Rules

Claude ecosystems receive 10 core skills: `/audit`, `/commit`, `/continue`, `/dep-audit`, `/doublecheck`, `/fix`, `/issue`, `/owasp-review`, `/secret-scan`, and `/security-scan`. The current Codex product contract does not generate separate skill files.

Neo also generates a session protocol rule plus per-stack service rules. Morpheus adds service-loop-specific agents, prompts, skills, and rules when scaffolding a service or installing the autonomous loop.

### Maintainer-Local Context

The maintainer checkout may contain additional private agents, skills, rules, ADRs, and self-improvement context under the ignored root `.claude/` directory. Public contributors do not need those files to build, test, or understand the public architecture.

---

## Self-Improvement Loop

The maintainer-local dogfood workflow is designed around a 6-layer self-improvement architecture:

1. **Self-Perception** — `trinity self-state` produces a versioned JSON snapshot
2. **Goal-Perception** — synthesizes improvement candidates from dogfood signals, external signals, and strategy docs
3. **Action Loop** — `self-improver` agent dispatches the manager→developer→tester→reviewer pipeline per candidate
4. **Quality Verification** — CI and the PR security workflow provide build/test/vet plus report-only security checks
5. **External Signal Ingestion** — watchers poll Anthropic releases, Go releases, CVE feeds
6. **Trigger & Scheduling** — cron + event triggers with budget caps and kill switch

### Autonomy Levels

| Level | Name | Who acts | Who gates | Current status |
|-------|------|----------|-----------|----------------|
| L0 | Manual | Human | Human | Baseline |
| **L1** | Suggestive | Agent proposes | Human executes | **Current default** |
| L2 | Autonomous + merge gate | Agent opens PRs | Human merges | Unlocked by founder |
| L3 | Categorically autonomous | Agent opens + merges | CI + path allowlist | Earned per category |
| L4 | Mostly autonomous | Agent acts across all | Async human review | Defined only — not activatable |

The loop currently operates at **L1** (propose-to-file). The `self-improver` agent writes proposal files; the founder reviews and executes manually.

### Safety Gates

- **Kill switch**: `.no-autonomy` file at repo root halts loop activity instantly
- **Scope limits**: governing context, CI workflows, rules, and release pipeline stay out of autonomous edit scope
- **Budget caps**: token, spend, and PR-count caps bound each cycle
- **Founder approval gate**: human review controls merge decisions

---

## CI/CD

### CI Pipeline (`.github/workflows/ci.yml`)

On every push/PR to `main`:
1. `go vet ./...`
2. `go test ./... -race -count=1`
3. `go build` all 4 binaries

### Security Gates (`.github/workflows/security.yml`)

Non-blocking (report-only) on pull requests to `main`:
- `govulncheck` — Go vulnerability scanner
- `gosec` — Go security analyzer
- `gitleaks` — secret scanner
- `golangci-lint` — Go linter

### Release Pipeline (`.github/workflows/release.yml`)

Triggered by `v*` tag push:
1. GoReleaser builds 16 cross-platform binaries (darwin/linux × amd64/arm64 × 4 tools)
2. GitHub Release created with checksums
3. Homebrew formulae pushed to `0merUfuk/homebrew-thematrix`

```bash
make release VERSION=X.Y.Z SUMMARY="one-line release summary"
```

---

## Technology Stack

| Component | Choice | Why |
|-----------|--------|-----|
| Language | Go 1.26.10+ | Single-binary distribution and compile-time type safety |
| CLI framework | cobra | Standard Go CLI — commands, flags, help text |
| TUI prompts | charm.land/huh v2 | Form-based interactive prompts |
| Terminal styling | charm.land/lipgloss v2 | Consistent styled output |
| Templates | text/template + go:embed | Compiled into binary, 17 custom helpers |
| Config | gopkg.in/yaml.v3 | Optional unified YAML configuration |
| Release | GoReleaser + GitHub Actions | Cross-platform binaries, Homebrew tap automation |

---

## Validation History

| Project | Type | Cycles | Result |
|---------|------|--------|--------|
| webhook-service | originating project Go | 12 | 8,644 lines, 268 tests — first validation |
| cron-service | originating project Go | 6 | 137/137 tasks, all CLEAN reviews |
| mythix-api | Non-originating project Go | 5 | 65+ tests, ALL_GATES_APPROVED — 6-agent pipeline validated |
| notification-service | originating project Go | 1 | Manager agent caught real production defect (Redis key collision) |

---

## Further Reading

| Document | Purpose |
|----------|---------|
| [README.md](README.md) | Install, command overview, and repository hygiene notes |
| [CONTRIBUTING.md](CONTRIBUTING.md) | How to contribute from a fresh public clone |
| Maintainer-local `.claude/` context | Private continuation notes and ADRs, absent from public clones |
