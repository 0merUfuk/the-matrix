# Architecture

This document covers the-matrix's system design, tool boundaries, data flow, and key architectural decisions. The maintainer's full founding context and ADR log now live in local/private `.claude/` state that is not shipped in public clones; this file is the public architecture summary.

---

## Overview

the-matrix is four Go CLIs that provision and maintain autonomous [Claude Code](https://docs.anthropic.com/en/docs/claude-code) agent ecosystems. Each tool has a single responsibility and clean boundaries. Neo provisions the ecosystem and guides the flow; oracle and trinity remain independently invoked commands, while neo invokes morpheus only for multi-repo service scaffolds.

```
the-matrix (ecosystem name)
│
├── neo          ← meta-CLI: init / analyze / doctor / status / update / registry
│                  provisions ecosystems, prints oracle/trinity guidance,
│                  and invokes morpheus only for multi-repo scaffolds
│
├── oracle       ← independently invoked knowledge synthesis
├── morpheus     ← service scaffolding + autonomous dev loops; standalone,
│                  or invoked by neo for multi-repo scaffolds
├── trinity      ← independently invoked maintenance: sync, health, refresh, self-state
│
└── sentinel     ← future: ecosystem health monitoring (not yet built)
```

---

## The Core Insight

> **You fix AI agent reliability at setup time, not at runtime.**

Most Claude Code setups are hand-crafted, generic, and stale within weeks. the-matrix attacks the layer *around* the agent — permissions, tool allowlists, agent roles, ADRs, knowledge quality — at provisioning time. These are setup-time decisions that determine runtime reliability.

---

## Tool Boundaries

### Why 4 binaries instead of 1?

Each tool has a distinct operational shape:

| Tool | Runtime | Duration | Replaced by? |
|------|---------|----------|--------------|
| **neo** | CLI wizard + ecosystem generation | 30–60s | — |
| **oracle** | 4-cycle research loop with Claude | 3–8 min | Any tool that produces `.claude/knowledge/*.md` |
| **morpheus** | Template rendering + `claude -p` subprocess | 10–30s | Any scaffolder that produces `.claude/` + `.autonomous/` |
| **trinity** | File I/O + staleness calculation | <2s | Any tool that syncs knowledge docs |

Oracle's research loop runs for minutes and is embarrassingly parallel. Trinity's sync is sub-second. Mixing them in one binary would create a tool with unclear scope and incompatible operational shapes. The contract between them is the **output directory layout** — not a protocol, not an API.

The original ADR rationale behind this split is maintainer-local; the public boundary is the command and directory contract summarized here.

---

## Data Flow

The `.claude/` paths below are generated target-project ecosystem paths, not root files expected to exist in this public repository.

```
Step 1: oracle researches target stack
  → tasks/research/*.md (Tier-1 sourced, quality-gated)
  → human reviews research
  → output/{stack}/ (17 standard knowledge docs)

Step 2: trinity syncs into agent ecosystem
  trinity sync --from output/go/ --to .claude/knowledge/go-rules/
  → .claude/knowledge/go-rules/*.md (fresh, versioned)

Step 3: morpheus scaffolds a service
  morpheus init
  → reads .claude/knowledge/ (gold standards from oracle output)
  → generates service/ with .autonomous/loop.sh + .claude/ + tasks/

Step 4: autonomous loop builds the service
  bash .autonomous/loop.sh
  → Developer → Tester → Reviewer → Security-Reviewer cycles
  → produces production-ready microservice

Step 5: trinity maintains freshness
  trinity refresh (run independently)
  → reads .claude/knowledge/*.md frontmatter (Created date)
  → identifies docs older than threshold (default: 6 months)
  → prints oracle re-research and trinity sync guidance

Step 6: neo detects and guides
  neo init → provisions the ecosystem and prints oracle/trinity commands
           → invokes morpheus only for multi-repo service scaffolds
  neo update → detects staleness and prints suggested refresh commands
  neo doctor → validates ecosystem integrity at any point
```

---

## Repository Structure

```
the-matrix/
├── cmd/               # CLI entrypoints (one per tool)
│   ├── neo/
│   ├── morpheus/
│   ├── oracle/
│   └── trinity/
├── internal/          # Shared packages + per-tool implementations
│   ├── cli/           # Terminal formatting (lipgloss)
│   ├── config/        # JSON/file I/O
│   ├── tmpl/          # Template engine (text/template + 17 FuncMap helpers)
│   ├── wizard/        # Interactive prompts (huh v2)
│   ├── subprocess/    # Claude subprocess wrapper
│   ├── staleness/     # Document age calculation
│   ├── matrixcfg/     # .matrix.yaml unified Config + per-tool options
│   ├── neo/           # neo implementation + 23 .tmpl files (go:embed)
│   ├── morpheus/      # morpheus implementation + 93 artifacts (92 .tmpl + loop.sh)
│   ├── oracle/        # oracle implementation + export/ + 7 artifacts (5 .tmpl + loop.sh + update.sh)
│   │   └── export/    # Format adapter pattern: Cursor, Copilot, Windsurf, AGENTS.md
│   ├── registry/      # Knowledge registry (pack catalog + validation)
│   └── trinity/       # trinity implementation (no templates)
├── .github/workflows/ # CI: build, test, vet, security gates, release
├── go.mod             # Single module, minimum Go 1.25.8
├── Makefile           # build, test, vet, release
└── .goreleaser.yml    # Cross-platform binary builds + Homebrew tap
```

Tracked `internal/**/templates/.claude/` paths are embedded generation assets. A root `.claude/` directory in a maintainer checkout is ignored local/private agent state and is absent from a fresh public clone.

---

## Key Design Decisions

### go:embed for Template Distribution

All 123 template artifacts (120 `.tmpl` files and 3 shell scripts) are compiled into the binaries via `go:embed`. No external template files are required at runtime; Claude-powered workflows still require the Claude Code CLI and Bash. Template changes require recompilation, but embedding eliminates the "template not found" failure mode common in filesystem-based template loaders.

### Shared Internal Packages

All 4 tools share `cli/`, `config/`, `tmpl/`, `wizard/`, `subprocess/`, `staleness/`, `matrixcfg/`. This reduces duplication and ensures consistent behavior across tools (e.g., the same wizard library, the same staleness calculation).

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
| `cursor` | `.cursor/rules/*.mdc` (MDC frontmatter, auto-splits >6K) |
| `copilot` | `.github/copilot-instructions.md` |
| `windsurf` | `.windsurfrules` |

Architecture: format adapter pattern with `init()`-based auto-registration. New formats are added by implementing the `Format` interface and calling `Register()` in `init()`.

### Unified Configuration

`.matrix.yaml` — optional YAML config file at the project root (or `~/.config/the-matrix/config.yaml` globally). Priority chain: CLI flag > config file > hardcoded default. Only oracle and trinity consume config in v1; neo and morpheus will be wired when they gain configurable flags.

The configuration contract is public in code and command behavior; deeper ADR notes are maintainer-local.

---

## Agent Ecosystem

The-matrix has two related `.claude/` concepts:

1. **Generated target-project ecosystems** — public product output created by `neo` or `morpheus` from embedded templates.
2. **This repo's maintainer-local ecosystem** — ignored private state in the root `.claude/` directory when present locally. It is not tracked and is absent from fresh public clones.

### Generated Core Agents

Neo embeds 6 core Claude Code agent templates for generated ecosystems:

| Agent | Role |
|-------|------|
| `manager` | Orchestrator — coordinates the agent workflow |
| `developer` | Senior engineer — implements code in a scoped worktree |
| `tester` | QA engineer — writes and runs tests |
| `reviewer` | Adversarial reviewer — read-only quality pass |
| `strategist` | Product/technical strategist — researches and designs work |
| `security-reviewer` | OWASP/security reviewer — read-only security pass |

### Generated Skills And Rules

Neo embeds 10 core skills for generated ecosystems: `/audit`, `/commit`, `/continue`, `/dep-audit`, `/doublecheck`, `/fix`, `/issue`, `/owasp-review`, `/secret-scan`, and `/security-scan`.

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
make release VERSION=1.8.0 SUMMARY="public launch"
```

---

## Technology Stack

| Component | Choice | Why |
|-----------|--------|-----|
| Language | Go 1.25.8+ | Single-binary distribution and compile-time type safety |
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
