**Version**: 2.1
**Created**: 2026-03-17
**Last Updated**: 2026-07-29
**Authors:** Ömer Ufuk

---

# Agent Ecosystem Handbook

> **Scope note (v2.1):** This handbook documents the **full maintainer ecosystem**. The
> public binaries (`neo`, `morpheus`) ship a **subset** of these agents and skills —
> the rest live in the maintainer's private root `.claude/` directory and are not
> present in fresh public clones or in the embedded templates.
>
> | | Public (embedded in binaries) | Maintainer-private (root `.claude/`) |
> |---|---|---|
> | **Agents** | manager, developer, tester, reviewer, strategist, security-reviewer (6) | + architect, product-lead, tech-lead, growth-lead (4) |
> | **Skills** | audit, commit, continue, dep-audit, doublecheck, fix, issue, owasp-review, secret-scan, security-scan (10) | + release, provision, strategy-weekly, strategy-monthly, session-learn (5) |
>
> Sections marked **[private]** below describe agents/skills that are not shipped in
> the public binaries. A fresh `neo init` or `morpheus init` will not produce them.

This handbook explains the-matrix's autonomous agent ecosystem: 10 specialized agents and 15 skills that together enable coordinated, multi-agent development workflows for the Go monorepo.

---

## Table of Contents

1. [Architecture Overview](#architecture-overview)
2. [Agents](#agents)
   - [Manager](#manager)
   - [Developer](#developer)
   - [Tester](#tester)
   - [Reviewer](#reviewer)
   - [Strategist](#strategist)
   - [Security-Reviewer](#security-reviewer)
   - [Architect](#architect)
   - [Product-Lead](#product-lead)
   - [Tech-Lead](#tech-lead)
   - [Growth-Lead](#growth-lead)
3. [Skills](#skills)
   - [Core Quality Gates](#core-quality-gates) — `/audit`, `/doublecheck`, `/fix`
   - [Release + Workflow](#release--workflow) — `/release`, `/commit`, `/issue`
   - [Security Suite](#security-suite) — `/owasp-review`, `/secret-scan`, `/dep-audit`, `/security-scan`
   - [Provisioning + Strategy](#provisioning--strategy) — `/provision`, `/strategy-weekly`, `/strategy-monthly`, `/session-learn`
4. [Workflows](#workflows)
   - [Full Autonomous Pipeline](#full-autonomous-pipeline)
   - [Direct Agent Usage](#direct-agent-usage)
   - [Strategy Loop](#strategy-loop)
   - [Skill Invocation](#skill-invocation)
5. [Agent Memory](#agent-memory)
6. [How Agents Communicate](#how-agents-communicate)
7. [Configuration Reference](#configuration-reference)
8. [Troubleshooting](#troubleshooting)

---

## Architecture Overview

The ecosystem has two categories:

**Agents** (`.claude/agents/`) — specialized personas with isolated context, persistent memory, and defined tool access. Each agent runs in its own context window and does not pollute the main conversation.

**Skills** (`.claude/skills/`) — task-oriented commands that execute within the current conversation (or a forked context). Each skill is a directory with a `SKILL.md` file, following the Agent Skills open standard.

### At a Glance

| Category | Count | Examples |
|----------|-------|----------|
| Agents | 10 (6 public + 4 private) | manager, developer, tester, reviewer, strategist, security-reviewer, architect [private], product-lead [private], tech-lead [private], growth-lead [private] |
| Skills | 15 (10 public + 5 private) | audit, commit, continue, dep-audit, doublecheck, fix, issue, owasp-review, secret-scan, security-scan, release [private], provision [private], strategy-weekly [private], strategy-monthly [private], session-learn [private] |

### Coordinator Topology

```
User
│
└── claude --agent manager           ← coordinated execution
         │
         ├── spawns strategist       ← researches web/GitHub, returns strategy
         │       └── returns feature spec + competitive analysis
         │
         ├── spawns developer        ← implements in isolated worktree (+ context7 docs)
         │       └── returns code + summary
         │
         ├── spawns tester           ← validates in isolated worktree (+ context7 docs)
         │       └── returns test results
         │
         ├── spawns reviewer         ← read-only quality gate (+ GitHub best practices)
         │       └── returns review verdict
         │
         ├── spawns security-reviewer ← OWASP/ASI audit (read-only) when scope warrants
         │       └── returns SECURITY_APPROVED | SECURITY_BLOCKED
         │
         └── creates PR via GitHub MCP
```

### Strategy Topology (monthly cadence)

```
User / manager
│
└── /strategy-monthly (skill)
         │
         ├── spawns product-lead     ← CEO perspective — priorities, kill criteria
         ├── spawns tech-lead        ← CTO perspective — architecture drift, dep health (read-only)
         └── spawns growth-lead      ← CMO perspective — channels, content, retention
                 │
                 └── synthesized assessment
                         │
                         └── (if strategy changed) architect → /provision → ecosystem refresh
```

### Design Principles

1. **One agent, one job** — developer implements, tester tests, reviewer reviews. No overlap.
2. **Isolation by default** — developer and tester work in git worktrees. Their changes can't conflict with each other or the main branch.
3. **Read-only gates** — reviewer, security-reviewer, and tech-lead are framework-restricted read-only (`disallowedTools: Write, Edit, NotebookEdit`). strategist is conversational by prompt instruction — its tools list omits Write and Edit entirely (the restriction is structural rather than an explicit `disallowedTools` block). Their value is in analysis and discussion, not code changes.
4. **Memory compounds** — all agents use `memory: project`, learning from past sessions. The developer remembers codebase patterns. The reviewer remembers recurring issues. The strategist remembers product decisions.
5. **Skills are auto-discoverable** — `/audit`, `/doublecheck`, `/strategy-weekly`, and `/strategy-monthly` can be triggered by Claude automatically when verification or periodic review is warranted. `/fix`, `/release`, `/commit`, `/issue`, `/owasp-review`, `/provision`, and `/session-learn` require explicit invocation.
6. **Separation of concerns**: execution agents (developer, tester) vs. review agents (reviewer, security-reviewer) vs. strategy agents (strategist, product-lead, tech-lead, growth-lead) vs. ecosystem agents (architect). Each category has distinct read/write boundaries.

---

## Agents

### Manager

**File**: `.claude/agents/manager.md`
**Model**: opus
**Purpose**: Orchestrator that breaks tasks into subtasks, spawns the right agent for each, and manages handoffs until quality gates pass.

**When to use**: Complex features requiring coordinated implementation, testing, and review. Any task that touches multiple files or tools.

**Invocation**:
```bash
claude --agent manager
```

**Key configuration** (authoritative values from `.claude/agents/manager.md`):
```yaml
tools: Agent(strategist, developer, tester, reviewer, security-reviewer, product-lead, tech-lead, growth-lead, architect), Read, Edit, Write, Grep, Glob, Bash, Skill, mcp__plugin_github_github__create_branch, mcp__plugin_github_github__create_pull_request, mcp__plugin_github_github__list_issues, mcp__plugin_github_github__add_issue_comment, mcp__plugin_github_github__search_issues
model: opus
memory: project
maxTurns: 100
permissionMode: bypassPermissions
skills:
  - audit
  - doublecheck
  - fix
  - commit
  - issue
  - release
  - owasp-review
  - strategy-weekly
  - strategy-monthly
  - session-learn
```

**What it does**:
1. Checks periodic strategy cadence — invokes `/strategy-monthly` if >30 days since last run (spawns product-lead + tech-lead + growth-lead in parallel)
2. Spawns strategist for feature-specific research when direction is needed (Phase 0)
3. Reads context files (SERVICE_CONTEXT.md, NEXT_STEPS.md, action items)
4. Decomposes the task into agent-sized subtasks with acceptance criteria
5. Writes the plan to `tasks/todo.md`
6. Spawns developer in a worktree for implementation
7. Spawns tester in a worktree for validation
8. Spawns reviewer for quality gate
9. Spawns security-reviewer when scope warrants (auth, file I/O, subprocess, deps, agent/CLI code)
10. Handles feedback loops (reviewer finds issue → developer fixes → tester re-validates → reviewer re-checks)
11. Runs `/doublecheck` before marking complete
12. Creates PR via GitHub MCP when work is ready to ship

**Tool restriction**: `Agent(strategist, developer, tester, reviewer, security-reviewer, product-lead, tech-lead, growth-lead, architect)` means the manager can spawn any of these nine agents (every agent in the ecosystem except itself). The manager cannot spawn another manager.

**Confidence threshold protocol**: Every reviewer verdict is scored 0–100%. The manager accepts verdicts ≥ 95%; < 95% triggers an automatic second pass; two consecutive < 95% passes escalate to the user.

**Has Write/Edit**: Yes — the manager writes `tasks/todo.md` and `tasks/session-summary.md`. It does NOT write Go source code (enforced by prompt, not tools).

---

### Developer

**File**: `.claude/agents/developer.md`
**Model**: opus
**Purpose**: Senior Go engineer that implements features, fixes bugs, refactors code, and creates templates.

**When to use**: Any code change — new commands, template modifications, shared package updates, bug fixes.

**Invocation**:
```bash
# Direct (you manage the workflow)
claude --agent developer

# Via manager (manager coordinates)
# Manager spawns developer automatically
```

**Key configuration**:
```yaml
tools: Read, Write, Edit, Grep, Glob, Bash, mcp__context7__resolve-library-id, mcp__context7__query-docs, mcp__MCP_DOCKER__sequentialthinking
model: opus
memory: project
isolation: worktree
maxTurns: 80
permissionMode: bypassPermissions
skills:
  - audit
  - commit
```

**What it does**:
1. Reads the relevant `cmd/{tool}/main.go` and `internal/{tool}/` packages
2. Understands existing patterns before making changes
3. Implements the requested feature following established codebase conventions
4. Verifies with `make build`, `make test`, `make vet`
5. Reports back: files changed, decisions made, issues found

**Isolation**: Runs in a git worktree — an isolated copy of the repository. Changes don't affect the main branch until explicitly merged. The worktree is auto-cleaned if no changes were made.

**Preloaded skills**: `/audit` and `/commit` are injected into the developer's context at startup. This gives the developer awareness of verification checks and commit conventions, even though the full audit subagent pipeline cannot run from within a spawned subagent.

**What it knows**: Go 1.25, cobra CLI, charmbracelet/huh v2, lipgloss v2, text/template, go:embed, and the 4-tool monorepo architecture. Can look up latest library docs via context7 and use structured reasoning for complex decisions via `mcp__MCP_DOCKER__sequentialthinking`.

---

### Tester

**File**: `.claude/agents/tester.md`
**Model**: sonnet
**Purpose**: QA engineer that writes tests, runs test suites, validates implementations, and checks edge cases.

**When to use**: After developer delivers code, or upfront for TDD.

**Invocation**:
```bash
# Direct
claude --agent tester

# Via manager
# Manager spawns tester automatically after developer
```

**Key configuration**:
```yaml
tools: Read, Write, Edit, Grep, Glob, Bash, mcp__context7__resolve-library-id, mcp__context7__query-docs
model: sonnet
memory: project
isolation: worktree
maxTurns: 50
permissionMode: bypassPermissions
skills:
  - issue
```

**What it does**:
1. Reads the implementation to understand what to test
2. Checks existing tests in the same package for patterns
3. Looks up Go testing patterns and assertion libraries via context7 when needed
4. Writes table-driven Go tests covering: happy path, error cases, edge cases
5. Runs `make test` and `go test -race`
6. Reports: tests written, pass/fail count, coverage, bugs found

**Preloaded skill**: `/issue` is available so the tester can file well-formed GitHub Issues when tests reveal a bug the developer did not introduce.

**Does NOT fix bugs** — if tests reveal implementation issues, the tester reports them back. The developer fixes.

---

### Reviewer

**File**: `.claude/agents/reviewer.md`
**Model**: sonnet
**Purpose**: Adversarial code reviewer that performs quality gate checks. Read-only — cannot modify files.

**When to use**: After implementation and testing, as the final quality gate before marking work complete.

**Invocation**:
```bash
# Direct
claude --agent reviewer

# Via manager
# Manager spawns reviewer after developer + tester
```

**Key configuration**:
```yaml
tools: Read, Grep, Glob, Bash, mcp__MCP_DOCKER__sequentialthinking, mcp__plugin_github_github__search_code
disallowedTools: Write, Edit, NotebookEdit
model: sonnet
permissionMode: bypassPermissions
memory: project
maxTurns: 50
```

**What it does** — three mandatory passes:

1. **Pass 1: Code Correctness & Safety** — error handling, nil safety, concurrency, naming, template safety, cobra CLI correctness, shared package impact
2. **Pass 2: Documentation Accuracy** — cross-references every factual claim in docs against actual source code
3. **Pass 3: Convention Compliance** — frontmatter, code blocks, tree diagrams, Go style

**Output format**:
```
REVIEW COMPLETE — {scope}

Pass 1 (Code): {PASS | N findings}
Pass 2 (Docs): {PASS | N findings}
Pass 3 (Conventions): {PASS | N findings}

CRITICAL (must fix): ...
WARNING (should fix): ...
MINOR (nice to have): ...

Verdict: {APPROVE | APPROVE WITH FIXES | NEEDS CHANGES}
Confidence: {0-100}%
```

**The mindset**: "Assume at least one flaw exists. Look until you find it." "Looks good" is forbidden.

**Has Bash but no Write/Edit**: Reviewer needs Bash to run `make build`, `make test`, `git diff`, etc. It cannot use the Write or Edit tools to modify files. Bash write operations (e.g., `echo > file`) are prevented by prompt instruction.

---

### Strategist

**File**: `.claude/agents/strategist.md`
**Model**: opus
**Purpose**: Product strategist for roadmap discussions, feature prioritization, tool boundary decisions, and ecosystem evolution.

**When to use**: Before starting a major feature, when questioning priorities, when discussing the-matrix as a product.

**Invocation**:
```bash
claude --agent strategist
```

**Key configuration**:
```yaml
tools: Read, Grep, Glob, WebSearch, WebFetch, mcp__MCP_DOCKER__fetch, mcp__MCP_DOCKER__sequentialthinking, mcp__context7__resolve-library-id, mcp__context7__query-docs, mcp__plugin_github_github__search_repositories, mcp__plugin_github_github__search_code, mcp__plugin_github_github__list_releases
model: opus
permissionMode: bypassPermissions
memory: project
maxTurns: 60
```

**What it does**:
- Researches competitors on GitHub (star counts, release frequency, features)
- Fetches web pages for market analysis, blog posts, documentation
- Looks up library capabilities via context7
- Uses structured reasoning for complex trade-off analysis
- Discusses product strategy, roadmap, and priorities
- Challenges feature proposals with data-backed arguments
- Builds on previous conversations via project memory

**What it knows**: Two audiences (originating project + outsiders), 5-tool topology, phase roadmap, all ADRs. Plus: can actively research the web and GitHub for up-to-date competitive intelligence.

**Research-capable, not operational**: Strategist can read, search, and analyze — but cannot write code, run commands, or modify files. Its output is researched strategy, not implementation.

**Graceful degradation**: If MCP tools are unavailable, strategist falls back through: web research → GitHub search → context7 docs → repo context only. Never fails completely — always provides value from whatever sources are available.

**Managed by manager**: The manager can spawn the strategist as the first step of a feature pipeline. Strategist researches, returns strategy, manager decomposes into implementation tasks. Strategist can also be used standalone for pure discussion.

---

### Security-Reviewer

**File**: `.claude/agents/security-reviewer.md`
**Model**: sonnet
**Purpose**: On-demand security vulnerability auditor. Checks code against OWASP Top 10:2025 and ASI01–ASI10 (Agentic Security Intelligence). Read-only — cannot modify any files.

**When to use**: After developer + tester + reviewer pass, whenever changes touch authentication, file I/O, subprocess execution, dependency updates, or agent/CLI code. Also invoked on demand via `/owasp-review` and active in all morpheus loop paths since Phase 3a (PR #47).

**Invocation**:
```bash
# Direct
claude --agent security-reviewer

# Via skill
/owasp-review

# Via manager
# Manager spawns after developer + tester + reviewer pass, when scope warrants
```

**Key configuration**:
```yaml
tools: Read, Grep, Glob, Bash(make:*), Bash(go:*), Bash(git:*), Bash(npm:*), Bash(ls:*), Bash(find:*), mcp__MCP_DOCKER__sequentialthinking, mcp__plugin_github_github__search_code
disallowedTools: Write, Edit, NotebookEdit
model: sonnet
permissionMode: bypassPermissions
memory: project
maxTurns: 50
```

**What it does**:
1. Reviews the diff scope against OWASP Top 10:2025 (A01 broken access control through A10 server-side request forgery)
2. Reviews the diff scope against ASI01–ASI10 (prompt injection, data leakage, tool misuse, excessive agency, etc.)
3. Runs `govulncheck`, parses known-vulnerability signals from `go.mod`, inspects shell/subprocess invocations
4. Produces a SECURITY_APPROVED or SECURITY_BLOCKED verdict
5. For `SECURITY_BLOCKED`: returns specific findings with file:line references and remediation guidance

**Bash tool scoping**: Bash is restricted to the specific command prefixes listed above — this is a security requirement encoded in the tool allowlist, not just a prompt instruction. The agent cannot run arbitrary shell commands.

---

### Architect [private]

**File**: `.claude/agents/architect.md`
**Model**: opus
**Purpose**: Ecosystem architect. Reads strategy documents, audits the current `.claude/` ecosystem, identifies gaps between strategy and ecosystem, and provisions agents, skills, and rules via the `/provision` skill.

**When to use**: After strategy changes (monthly assessment flagged a shift), after adopting a new project template, or when the `.claude/` ecosystem drifts from the intended design.

**Invocation**:
```bash
# Direct
claude --agent architect

# Via manager
# Manager spawns after /strategy-monthly signals ecosystem refresh needed
```

**Key configuration**:
```yaml
tools: Read, Write, Edit, Grep, Glob, Bash, Skill
model: opus
memory: project
maxTurns: 100
permissionMode: bypassPermissions
skills:
  - provision
```

**What it does**:
1. Reads `tasks/monthly/` strategy outputs and current `.claude/` ecosystem
2. Computes the delta: which agents/skills/rules exist vs. what the strategy implies should exist
3. Invokes `/provision` to create or modify agent files, skill directories, and rule files per the ecosystem-conventions policy
4. Applies required YAML frontmatter (`name`, `description`, `tools`, `model`, `maxTurns`, `memory`, `permissionMode`, plus `isolation` or `disallowedTools` as appropriate)
5. Creates memory directories for any agent declaring `memory: project`
6. Reports what was added, modified, or removed with rationale per change

**The only agent besides manager with full Write access to `.claude/`**: Architect is the sanctioned modifier of the ecosystem. Developer cannot edit `.claude/` definitions (enforced by prompt); architect is explicitly scoped to do so.

---

### Product-Lead [private]

**File**: `.claude/agents/product-lead.md`
**Model**: opus
**Purpose**: Product lead (CEO perspective). Periodically reviews project health against strategy, makes priority decisions, identifies pivot triggers, and refreshes the product strategy.

**When to use**: Via `/strategy-monthly` at the start of each calendar month, after a production incident, when competitive intelligence suggests a meaningful shift, or when the roadmap needs a priority reset.

**Invocation**:
```bash
# Direct
claude --agent product-lead

# Via skill
/strategy-monthly
```

**Key configuration**:
```yaml
tools: Read, Write, Edit, Grep, Glob, WebSearch, WebFetch, Bash, Agent, mcp__MCP_DOCKER__sequentialthinking, mcp__MCP_DOCKER__fetch
model: opus
memory: project
maxTurns: 80
permissionMode: bypassPermissions
```

**What it does**:
1. Reads strategy-output/ (or the project's equivalent roadmap docs) and current metrics
2. Spawns research subagents for market context, competitor releases, community signal
3. Produces a monthly assessment: what is working, what is not, which kill criteria are approaching, what to cut or accelerate
4. Writes the assessment to `tasks/monthly/YYYY-MM.md`
5. Flags when the ecosystem needs a refresh (triggers architect + `/provision`)

**Has Write access**: Yes — product-lead writes monthly assessment files. It does NOT modify code, tool source, or `.claude/` agent definitions.

**Spawns research subagents**: Product-lead has the `Agent` tool and can spawn focused research subagents for competitor analysis, market sizing, or user research deep-dives.

---

### Tech-Lead [private]

**File**: `.claude/agents/tech-lead.md`
**Model**: opus
**Purpose**: Technical lead (CTO perspective). Reviews codebase health, architecture drift, dependency vulnerabilities, test coverage, and performance. Read-only — cannot modify code or files.

**When to use**: Bi-weekly, after major merges, or as part of `/strategy-monthly` for the technical health pulse.

**Invocation**:
```bash
# Direct
claude --agent tech-lead

# Via skill
/strategy-monthly    # spawned alongside product-lead and growth-lead
```

**Key configuration**:
```yaml
tools: Read, Grep, Glob, Bash(make:*), Bash(go:*), Bash(git:*), Bash(govulncheck:*), Bash(ls:*), Bash(find:*), mcp__MCP_DOCKER__sequentialthinking, mcp__context7__resolve-library-id, mcp__context7__query-docs
disallowedTools: Write, Edit, NotebookEdit
model: opus
memory: project
maxTurns: 60
permissionMode: bypassPermissions
skills:
  - dep-audit
  - security-scan
  - secret-scan
```

**What it does**:
1. Audits dependency freshness via `/dep-audit` (`govulncheck`, trivy, npm audit)
2. Runs security scans via `/security-scan` (gosec, semgrep SAST)
3. Runs secret scans via `/secret-scan` (gitleaks)
4. Assesses architecture drift (are tool boundaries respected? are shared packages accumulating responsibility?)
5. Reports refactoring priorities, dep bumps, and technical debt with severity rankings

**Read-only like reviewer and security-reviewer**: Tech-lead finds issues and reports. It does not fix them. Fixes go through developer + reviewer.

**Bash tool scoping**: Restricted to specific command prefixes (make, go, git, govulncheck, ls, find) — cannot run arbitrary shell commands.

---

### Growth-Lead [private]

**File**: `.claude/agents/growth-lead.md`
**Model**: opus
**Purpose**: Growth lead (CMO perspective). Reviews acquisition channels, retention metrics, content performance, ASO, and community health. Produces growth strategy updates.

**When to use**: Weekly/monthly growth assessments, after a major feature ships (post-mortem for learnings), when launching a new acquisition channel.

**Invocation**:
```bash
# Direct
claude --agent growth-lead

# Via skill
/strategy-monthly
```

**Key configuration**:
```yaml
tools: Read, Write, Edit, Grep, Glob, WebSearch, WebFetch, Bash, Agent, mcp__MCP_DOCKER__sequentialthinking, mcp__MCP_DOCKER__fetch
model: opus
memory: project
maxTurns: 80
permissionMode: bypassPermissions
```

**What it does**:
1. Reviews acquisition channels (Show HN, social, SEO, tap listings, podcast rotations, etc.)
2. Analyzes retention and funnel metrics for the tool ecosystem
3. Audits content calendar and proposes adjustments
4. Spawns research subagents for competitor content benchmarking
5. Writes growth sections of `tasks/monthly/YYYY-MM.md`

**Spawns research subagents**: Has the `Agent` tool — can delegate focused channel research to subagents for e.g., competitor content cadence, keyword tracking, community ROI analysis.

---

## Skills

Skills are slash commands defined in `.claude/skills/`. Each skill is a directory containing `SKILL.md` (and optionally supporting files). They execute within the current conversation context.

The ecosystem has 15 skills (10 public + 5 private) organized into four groups.

### Core Quality Gates

#### /audit

**File**: `.claude/skills/audit/SKILL.md`
**Auto-triggerable**: Yes — Claude can invoke this automatically when verification is needed.

```bash
/audit               # audit all 4 tools
/audit morpheus      # audit morpheus only
```

**What it does**: Launches 4 parallel Explore subagents that verify every documented claim against ground truth (git, code, filesystem):

1. **Version Agent** — checks version strings across main.go, Makefile, git tags, SERVICE_CONTEXT.md, CHANGELOG.md, CLAUDE.md
2. **Code Verification Agent** — verifies cobra commands, embedded templates, template counts, dependencies
3. **Docs Accuracy Agent** — validates all .claude/ context files against code reality
4. **Structure Agent** — verifies monorepo structure, build system, distribution config

**Output**: Terminal report + `tasks/audit-report.md` with "Safe to Auto-Fix" section for `/fix`.

**Dynamic context injection**: The skill injects current git status, tool versions, and latest tags into the prompt before Claude sees it.

---

#### /doublecheck

**File**: `.claude/skills/doublecheck/SKILL.md`
**Auto-triggerable**: Yes — Claude can invoke this automatically after completing work.

```bash
/doublecheck              # verify recent implementation
/doublecheck morpheus     # verify specific tool
/doublecheck plan oracle  # attack a plan before implementation
```

**What it does**: Launches 4 adversarial subagents that attack the implementation from different angles:

1. **Gap Finder** — delta between spec and delivered code
2. **Assumption Attacker** — surfaces hidden assumptions, challenges each one
3. **Ground Truth Verifier** — runs make build/test/vet, verifies versions, tests binaries
4. **Devil's Advocate** — finds the most likely real-world failure mode

**Verdict scoring**: 0–100% confidence. 95%+ = SHIP. 80–94% = Fix warnings. 60–79% = FIX FIRST. <60% = REDESIGN.

---

#### /fix

**File**: `.claude/skills/fix/SKILL.md`
**Auto-triggerable**: No — user must invoke explicitly.

```bash
/fix             # apply all safe fixes from last audit
/fix --dry-run   # preview without applying
```

**What it does**: Reads `tasks/audit-report.md` from the last `/audit` run. Applies only safe mechanical fixes: context file corrections, version updates, stale path replacements. Never touches Go source, templates, Makefile, or git history.

---

### Release + Workflow

#### /release [private]

**File**: `.claude/skills/release/SKILL.md`
**Auto-triggerable**: No — user must invoke explicitly.

```bash
/release neo              # single tool, auto-compute version
/release neo v1.1.0       # explicit version
/release all v1.1.0       # unified release
```

**What it does**: Full release workflow — change analysis, version bump determination, CHANGELOG.md update, version bump in source, commit, annotated git tag, push, GitHub release, context file update, dev version bump.

---

#### /commit

**File**: `.claude/skills/commit/SKILL.md`
**Auto-triggerable**: No — user must invoke explicitly.

```bash
/commit            # group uncommitted changes and commit with conventional format
/commit session    # treat all changes as one coherent session commit
```

**What it does**: Analyzes `git status` + `git diff`, groups changes by logical intent, drafts a conventional commit message per group, and previews the full commit plan before executing. Never commits without approval. Enforces conventional commit format (`type(scope): description`) with scopes matching the monorepo (`neo`, `morpheus`, `oracle`, `trinity`, `cli`, `tmpl`, etc.).

**Key constraints**: Stages files by name (never `git add .`), never amends previous commits, never bypasses hooks, never pushes to remote unless explicitly asked.

---

#### /issue

**File**: `.claude/skills/issue/SKILL.md`
**Auto-triggerable**: No — user must invoke explicitly.

```bash
/issue              # create a well-formed GitHub Issue with enforced conventions
```

**What it does**: Creates GitHub Issues with the 14-label taxonomy (priority P0–P3, type, tool dimensions) introduced in v1.2.0 (PR #9). Ensures each issue has a scope, acceptance criteria, and appropriate labels. Used by the tester agent when tests reveal a bug that needs tracking but is out of scope for the current session.

---

### Security Suite

Four security skills. All shipped in v1.5.0 (Phase 3b/3c, PR #50). They deploy as templates to morpheus/neo output and run on the-matrix itself.

#### /owasp-review

**File**: `.claude/skills/owasp-review/SKILL.md`
**Auto-triggerable**: No — user or manager invokes explicitly.

```bash
/owasp-review               # full OWASP Top 10:2025 + ASI01–10 audit of current diff
```

**What it does**: Spawns the `security-reviewer` agent. Returns `SECURITY_APPROVED` or `SECURITY_BLOCKED` with specific file:line findings. The skill itself has `Agent` in `allowed-tools` so it can spawn the subagent (this was missing until v1.6.0 — see PR #74).

---

#### /secret-scan

**File**: `.claude/skills/secret-scan/SKILL.md`
**Auto-triggerable**: No — user invokes explicitly.

```bash
/secret-scan               # gitleaks scan for committed credentials
```

**What it does**: Runs `gitleaks` against the current tree. Reports any detected secrets with file:line. Intended for pre-commit verification when touching config files or scripts.

---

#### /dep-audit

**File**: `.claude/skills/dep-audit/SKILL.md`
**Auto-triggerable**: No — user invokes explicitly.

```bash
/dep-audit                 # govulncheck + trivy + npm audit dependency sweep
```

**What it does**: Runs Go's `govulncheck` for Go deps, `trivy` for container/system deps (if applicable), and `npm audit` for Node.js deps (if a `package.json` exists). Consolidates findings into a severity-ranked report. Used by tech-lead during bi-weekly or monthly health checks.

---

#### /security-scan

**File**: `.claude/skills/security-scan/SKILL.md`
**Auto-triggerable**: No — user invokes explicitly.

```bash
/security-scan             # gosec + semgrep SAST sweep
```

**What it does**: Runs `gosec` static analysis for Go-specific security patterns plus `semgrep` for cross-language SAST rules. Reports issues with file:line and rule ID. Distinct from `/owasp-review` (which uses an LLM reviewer agent) — this skill is pure tool-backed scanning.

---

### Provisioning + Strategy

#### /provision [private]

**File**: `.claude/skills/provision/SKILL.md`
**Auto-triggerable**: No — architect invokes explicitly.

```bash
/provision agent <name>    # create or update an agent definition
/provision skill <name>    # create or update a skill directory
/provision rule <name>     # create or update a rule file
```

**What it does**: Scaffolds `.claude/` artifacts following the ecosystem-conventions policy. Applies required YAML frontmatter, validates the `disallowedTools` / `isolation` mutual-exclusion rule for agents, ensures memory directories exist for agents declaring `memory: project`. The sanctioned entry point for ecosystem modification — only the architect agent invokes it.

---

#### /strategy-weekly [private]

**File**: `.claude/skills/strategy-weekly/SKILL.md`
**Auto-triggerable**: Yes — manager can invoke when an active sprint has run >7 days without a pulse.

```bash
/strategy-weekly           # quick health pulse (~5 minutes)
```

**What it does**: A lightweight weekly check — spawns one or two lead agents with a focused prompt to surface any sprint-level drift, acquisition-channel regressions, or dep-health warnings. Output written to `tasks/weekly/YYYY-WW.md`.

---

#### /strategy-monthly [private]

**File**: `.claude/skills/strategy-monthly/SKILL.md`
**Auto-triggerable**: Yes — manager invokes at monthly boundaries or when >30 days since last run.

```bash
/strategy-monthly          # full monthly assessment (~15 minutes)
```

**What it does**: Spawns product-lead + tech-lead + growth-lead in parallel. Each produces its domain assessment (product priorities, tech health, growth channels). Output synthesized into a monthly `tasks/monthly/YYYY-MM.md`. If the assessment flags "ecosystem refresh needed", manager spawns architect with the monthly report as input; architect then invokes `/provision` to update `.claude/` artifacts.

---

#### /session-learn [private]

**File**: `.claude/skills/session-learn/SKILL.md`
**Auto-triggerable**: No — user invokes at end of session.

```bash
/session-learn             # capture session findings and propose targeted improvements
```

**What it does**: Shipped in v1.7.0 (PR #92). Captures bug patterns, reviewer observations, quality gaps, and architectural observations from the current session. Proposes targeted improvements to `.claude/` agents, skills, and rules via the architect agent. The feedback loop that closes the self-improvement circle — lessons from a session become durable ecosystem changes, not just tasks/lessons.md notes.

---

## Workflows

### Full Autonomous Pipeline

The recommended workflow for complex features:

```bash
claude --agent manager
# Tell the manager what to research, design, and implement
# Manager orchestrates: strategist → developer → tester → reviewer → security-reviewer → PR
```

**What happens inside the manager session**:

```
You: "Research what other agent frameworks offer for health monitoring,
      then design and implement a sentinel health check command"

Manager:
  0. (Optional) Checks /strategy-monthly cadence — if >30 days, spawns
     product-lead + tech-lead + growth-lead in parallel before starting
  1. Spawns strategist:
     "Research competing agent frameworks' health monitoring features.
      Check LangChain, AutoGPT, CrewAI on GitHub. What do they monitor?
      What metrics matter? Design a sentinel health check approach."
  2. Strategist returns: "Analyzed 5 frameworks on GitHub. Key findings:
     - CrewAI monitors agent success rate + token usage
     - LangSmith tracks latency percentiles per chain
     - Recommendation: start with knowledge freshness + tool version drift.
     Proposed sentinel health command: check doc staleness, version consistency,
     template integrity. Output: traffic-light dashboard."
  3. Manager decomposes into implementation tasks based on strategy
  4. Spawns developer (worktree):
     "Implement sentinel health command per strategist's spec..."
  5. Developer looks up cobra docs via context7, implements
  6. Spawns tester (worktree):
     "Write tests for sentinel health. Cover: all-healthy, stale docs,
      version mismatch, missing templates."
  7. Tester looks up Go testing patterns via context7, writes tests
  8. Spawns reviewer (read-only):
     "Review sentinel health implementation."
  9. Reviewer searches GitHub for how similar tools structure health checks,
     uses sequentialthinking for systematic analysis
  10. Reviewer returns: "APPROVE WITH FIXES. 1 WARNING. Confidence: 96%..."
  11. Spawns developer again with the fix
  12. Reviewer passes — APPROVE, Confidence: 98%
  13. Spawns security-reviewer (the diff touches file I/O → scope warrants it):
      "Review for OWASP Top 10 + ASI exposure."
  14. Security-reviewer returns: SECURITY_APPROVED
  15. Runs /doublecheck — 95% confidence, SHIP
  16. Creates PR via GitHub MCP
```

**Confidence threshold protocol**: Every reviewer verdict is scored 0–100%. Manager accepts ≥ 95%; < 95% triggers a second pass; two consecutive < 95% passes escalate to the user. A low-confidence verdict is never acted on — the manager treats "Confidence absent" as < 95% too.

### Direct Agent Usage

For focused tasks where you manage the workflow yourself:

```bash
# Implement something specific
claude --agent developer
# "Add --verbose flag to neo status"

# Write tests for existing code
claude --agent tester
# "Write tests for internal/neo/cmd_status.go"

# Review recent changes
claude --agent reviewer
# "Review the changes in the last 3 commits"

# Run an on-demand security audit
claude --agent security-reviewer
# "Audit cmd/morpheus/main.go subprocess invocations"

# Refresh the .claude/ ecosystem after a strategy shift
claude --agent architect
# "Read tasks/monthly/2026-04.md and provision any missing agents"
```

When running agents directly (not via manager), each agent has full autonomy within its scope. The developer can run `make build` and `make test` on its own. The reviewer runs its 3-pass review independently. Architect runs `/provision` directly.

### Strategy Loop

The periodic strategy review runs monthly (required) or weekly (optional pulse):

```bash
/strategy-monthly
# Spawns product-lead + tech-lead + growth-lead in parallel (~15 minutes)
# Each produces a domain assessment; output synthesized into
# tasks/monthly/YYYY-MM.md
# If strategy shifted, manager then spawns architect + /provision
```

The strategy loop is separate from the build/test/review execution loop. It runs against the accumulated state of the project (metrics, dep health, channel performance) rather than against a pending diff.

### Skill Invocation

Skills are invoked with `/` in any Claude Code session:

```bash
# Core quality gates
/audit                    # verify context files against ground truth
/doublecheck              # adversarial verification of recent work
/fix                      # apply safe fixes from last audit

# Release + workflow
/release neo              # release a tool [private]
/commit                   # conventional commits for pending changes
/issue                    # create a well-formed GitHub Issue

# Security suite
/owasp-review             # OWASP + ASI audit via security-reviewer
/secret-scan              # gitleaks sweep
/dep-audit                # govulncheck + trivy + npm audit
/security-scan            # gosec + semgrep SAST

# Provisioning + strategy [all private]
/provision                # architect-only: create/update agent/skill/rule
/strategy-weekly          # weekly pulse
/strategy-monthly         # monthly multi-agent assessment
/session-learn            # capture session findings → ecosystem improvements

# Auto-triggered by Claude when appropriate:
# /audit, /doublecheck, /strategy-weekly, /strategy-monthly
# All others require explicit invocation.
```

---

## Agent Memory

All 10 agents use `memory: project`, which stores persistent cross-session learning in `.claude/agent-memory/{agent-name}/`.

**How it works**:
- Each agent has a `MEMORY.md` file in its memory directory
- The first 200 lines of MEMORY.md are auto-injected into the agent's system prompt at startup
- Agents can read from and write to their memory directory using Read/Write tools (read-only agents still have Read access to their own memory — they just cannot edit repo files)
- Memory persists across sessions — the developer remembers codebase patterns, the reviewer remembers recurring issues

**Storage locations**:
```
.claude/agent-memory/
├── manager/MEMORY.md            # task decomposition patterns, coordination learnings
├── developer/MEMORY.md          # codebase patterns, past mistakes, Go idioms
├── tester/MEMORY.md             # test patterns, coverage gaps, known flaky areas
├── reviewer/MEMORY.md           # recurring review findings, common mistakes
├── strategist/MEMORY.md         # product decisions, rejected ideas, roadmap context
├── security-reviewer/MEMORY.md  # recurring OWASP/ASI patterns, threat-model notes
├── architect/MEMORY.md          # ecosystem provisioning decisions, design-pattern inventory
├── product-lead/MEMORY.md       # monthly CEO-level findings, kill-criteria history
├── tech-lead/MEMORY.md          # dep-health snapshots, architecture drift observations
└── growth-lead/MEMORY.md        # channel performance, content calendar history
```

**Git status**: Agent memory directories are gitignored (`.claude/agent-memory/` in `.gitignore`). Memory is per-developer, not shared via version control. Each agent creates its own directory on first write — the manager is responsible for scaffolding them when a new agent is provisioned (`/provision` handles this automatically).

**Important**: per `ecosystem-conventions.md`, if an agent declares `memory: project`, its memory directory MUST exist at `.claude/agent-memory/{name}/` with a starter `MEMORY.md`. The architect + `/provision` pipeline enforces this.

---

## How Agents Communicate

### Within a Single Session (Coordinator Pattern)

Agents communicate through the manager via return values:

```
Manager → Agent tool prompt → Developer
Developer → return value → Manager
Manager → Agent tool prompt (includes developer's output) → Tester
Tester → return value → Manager
```

Each agent receives context from the manager's prompt. They do NOT share conversation history. The manager explicitly passes relevant context between agents.

### Between Sessions

Agents don't communicate directly between sessions. Shared state lives in:
- **Context files** (`.claude/SERVICE_CONTEXT.md`, etc.) — updated on main branch
- **Task files** (`tasks/todo.md`, `tasks/session-summary.md`) — session-local, gitignored
- **Agent memory** (`.claude/agent-memory/`) — per-agent, persistent, gitignored

### Important Constraint: Nested Subagents — Limited

Most subagents **cannot spawn other subagents**. Exceptions: product-lead, growth-lead (both declare `Agent` in their tool list), and architect (via `/provision`, which can scaffold additional agents). Everything else is one level deep.

This means:
- Manager can spawn any of the 9 non-manager agents (developer, tester, reviewer, strategist, security-reviewer, architect, product-lead, tech-lead, growth-lead)
- Developer CANNOT spawn tester or reviewer from within its session
- Product-lead and growth-lead CAN spawn focused research subagents because they have the `Agent` tool (used for competitor analysis, channel research, etc.)
- Skills that use parallel subagents (like `/audit` with 4 Explore agents, `/strategy-monthly` with 3 lead agents) work when invoked from the main conversation or from the manager, but NOT from within a spawned developer/tester/reviewer

---

## Configuration Reference

### Agent Frontmatter Fields

| Field | Required | Description |
|-------|----------|-------------|
| `name` | Yes | Unique identifier (lowercase, hyphens) — matches the filename without `.md` |
| `description` | Yes | When Claude should delegate to this agent |
| `tools` | Yes | Explicit allowlist; no wildcards except `Bash(prefix:*)` |
| `disallowedTools` | No | Denylist — removes from inherited/specified tools. Read-only agents MUST include `Write, Edit, NotebookEdit` |
| `model` | Yes | `opus` (complex reasoning), `sonnet` (focused tasks), `haiku`, `inherit` |
| `memory` | No | `project` (gitignored per-developer), `local`, `user` (global). If set to `project`, memory directory MUST exist |
| `maxTurns` | Yes | Maximum agentic turns before stopping. Code-writing agents: 50–100; research/review agents: 30–60 |
| `permissionMode` | No | `default`, `acceptEdits`, `bypassPermissions`, `plan`. Autonomous agents use `bypassPermissions` |
| `isolation` | No | `worktree` — runs in isolated git worktree. Mutually exclusive with read-only (`disallowedTools: Write, Edit, NotebookEdit`) |
| `skills` | No | Skills to preload (full content injected at startup) |
| `background` | No | `true` to always run as background task |

> **Read-only vs code-writing mutual exclusion**: An agent with `disallowedTools: Write, Edit, NotebookEdit` (read-only) MUST NOT also have `isolation: worktree`. The two modes are mutually exclusive — code-writing agents need a worktree to isolate their changes; read-only agents don't modify files, so a worktree is meaningless.

### Skill Frontmatter Fields

| Field | Required | Description |
|-------|----------|-------------|
| `description` | Yes | What the skill does and when to use it |
| `argument-hint` | Yes | Autocomplete hint (e.g., `[tool-name]`); use `""` if none |
| `allowed-tools` | Yes | Explicit tool list; include `Agent` if the skill spawns subagents |
| `disable-model-invocation` | No | `true` = only user can invoke, not Claude |
| `model` | No | Model override |
| `context` | No | `fork` to run in isolated subagent context |
| `agent` | No | Which agent to use when `context: fork` |

### Special Syntax

| Syntax | Where | Meaning |
|--------|-------|---------|
| `Agent(dev, tester, ...)` | Agent `tools` field | Restrict which subagents can be spawned. Manager uses `Agent(strategist, developer, tester, reviewer, security-reviewer, product-lead, tech-lead, growth-lead, architect)` |
| `Bash(git:*)` | Agent `tools` or Skill `allowed-tools` | Restrict Bash to specific command prefixes. Security-reviewer, tech-lead use scoped Bash |
| `$ARGUMENTS` | Skill body | All arguments passed when invoking |
| `$0`, `$1` | Skill body | Specific argument by index |
| `` !`command` `` | Skill body | Dynamic context injection — runs command before prompt |

### Required Ecosystem Rules

Two rule files auto-load for all work in this repo and enforce conventions:

- `.claude/rules/ecosystem-conventions.md` — the quick-reference for agent/skill/rule frontmatter, read-only vs code-writing rules, memory directory requirements
- `.claude/rules/security-baseline.md` — auto-loaded when writing Go source; 6 Go-specific security patterns (path traversal, shell injection, file permissions, secret handling, template injection, crypto/rand)

---

## Troubleshooting

### MCP tools not working

**Symptom**: Agent reports "tool not found" or MCP tool calls fail.

**Cause**: The MCP server providing the tool isn't running or configured. Agents that use MCP tools:
- **Strategist**: needs `MCP_DOCKER` (fetch, sequentialthinking), `context7` (library docs), `plugin_github_github` (GitHub search)
- **Manager**: needs `plugin_github_github` (branches, PRs, issues)
- **Developer**: needs `context7` (library docs), `MCP_DOCKER` (sequentialthinking)
- **Tester**: needs `context7` (library docs)
- **Reviewer**: needs `MCP_DOCKER` (sequentialthinking), `plugin_github_github` (code search)
- **Security-Reviewer**: needs `MCP_DOCKER` (sequentialthinking), `plugin_github_github` (code search)
- **Tech-Lead**: needs `MCP_DOCKER` (sequentialthinking), `context7` (library docs)
- **Product-Lead / Growth-Lead**: need `MCP_DOCKER` (fetch, sequentialthinking)

**Fix**:
- Check MCP server status in Claude Code settings
- GitHub tools require a GitHub authentication token
- context7 and MCP_DOCKER require their respective servers to be running
- All agents are designed to degrade gracefully — they fall back to repo context when MCP tools are unavailable

---

### Agent won't spawn

**Symptom**: Manager tries to spawn developer but nothing happens.

**Check**:
- Is the agent file in `.claude/agents/`? Run `ls .claude/agents/`
- Is the YAML frontmatter valid? First line must be `---`
- Is the `name` field correct and matching what the manager references?

---

### Worktree creation fails

**Symptom**: Developer or tester fails to start with a git error.

**Fix**: Commit or stash uncommitted changes before spawning worktree agents:
```bash
git stash        # stash changes
# ... run agent ...
git stash pop    # restore changes
```

---

### Skill not auto-triggering

**Symptom**: You expect Claude to invoke `/audit` automatically, but it doesn't.

**Check**:
- Does the skill have `disable-model-invocation: true`? If so, only you can invoke it.
- Is the skill's description clear about when to use it? Claude uses the description to decide.
- Run `/context` to check if skills were excluded due to the description budget limit.

---

### Agent memory not persisting

**Symptom**: Agent doesn't remember context from previous sessions.

**Check**:
- Does `.claude/agent-memory/{agent-name}/MEMORY.md` exist?
- Is the agent's `memory` field set to `project`?
- Memory is auto-created on first use — it won't exist until the agent writes to it.

---

### Reviewer modifying files

**Symptom**: Reviewer uses Bash to write to a file.

**Note**: Reviewer has Bash for `make build/test/vet` and `git` commands. If it writes files via Bash, it's violating its prompt instructions. This is prompt-enforced, not tool-enforced. The reviewer's `disallowedTools: Write, Edit, NotebookEdit` blocks the dedicated file-modification tools but not Bash redirects.

---

### Stale worktrees accumulating

**Symptom**: `git worktree list` shows many old worktrees.

**Fix**:
```bash
git worktree list              # see all worktrees
git worktree prune             # clean up stale ones
```

---

## Future: Agent Teams

The current setup uses the **coordinator pattern** (single session, hub-and-spoke via manager). When Claude Code's Agent Teams feature graduates from experimental, the same agent definitions can be used in **peer-to-peer mode**:

```bash
# Enable (experimental)
export CLAUDE_CODE_EXPERIMENTAL_AGENT_TEAMS=1

# Agent Teams adds:
# - Direct agent-to-agent messaging (mailbox)
# - Shared task list with dependency tracking
# - Self-organizing swarm pattern
# - Multiple terminal panes (tmux/iterm2 split)
```

No changes to agent files needed — the same definitions work with both patterns.
