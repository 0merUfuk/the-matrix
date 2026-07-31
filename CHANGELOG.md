# Changelog

All notable changes to this project will be documented in this file.
Format: [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), [SemVer](https://semver.org/).

## [1.8.2] - 2026-07-31

### Changed
- **morp**: renamed the CLI binary/command from `morpheus` to `morp` to avoid collision with the `morpheus` systems-biology tool in Homebrew-core. Internal Go package paths (`internal/morpheus`), import paths, YAML config keys (`morpheus:`), and type/function names remain unchanged.

## [Unreleased]

### Added
- `/self-state` skill: produces structured state.v1 JSON snapshots for autonomous self-improvement decisions
- `trinity self-state` subcommand: deterministic input collector, supports `--json`/`--out`/`--prune` flags

## [1.7.1] - 2026-04-29

### Added
- MIT LICENSE (#103)
- D-010 feature freeze ADR — scope commitment through Day 60 of the public launch window (#103)
- Non-blocking CI security gates workflow (`.github/workflows/security.yml`) — `govulncheck`, `gosec@v2.25.0`, `gitleaks`, `golangci-lint`; each job runs `continue-on-error: true` so findings are reported without blocking merges (#106)
- `.golangci.yml` — baseline lint configuration wired to the CI gate (#106)
- **cli**: `cobra.Example` blocks on 23 user-facing commands across neo (9), morpheus (5), oracle (6), and trinity (3) — `--help` output now shows idiomatic invocations (#108)
- **claude ecosystem**: seeded persistent memory directories for `developer`, `tester`, and `security-reviewer` agents — cross-session learning now starts from a curated baseline instead of an empty index (B1, #109)

### Changed
- **claude ecosystem**: post-v1.7.0 context reconciliation — `SERVICE_CONTEXT.md` and `NEXT_STEPS.md` updated to reflect tool versions v1.6.0 → v1.7.0, the v1.7.0 block moved from NEXT to RELEASED, auto-loaded rules count corrected (2 → 4), and Jira pipeline language updated from "cancelled" to "deferred" (#100)
- **docs**: resolved 87 stale-docs findings from the 2026-04-17 audit across `.claude/` context files, agent definitions, skills, and rule files — version drift eliminated across SERVICE_CONTEXT/CLAUDE.md/versioning.md/tool-actions, D-009 ADR added (Jira Pipeline cancellation + knowledge-layer pivot), `docs/AGENT_ECOSYSTEM.md` rewritten for the 10-agent / 14-skill ecosystem, README template counts corrected (#101)
- **README**: hero rewrite for public launch — dated pre-launch install banner, private-tap instructions first with GOPRIVATE, post-flip public quick start second; trinity correctly described as a CLI invoked on demand (not a background daemon); neo described as generating the ecosystem rather than orchestrating the others at runtime (#107)
- **claude ecosystem**: reconciled context files (`SERVICE_CONTEXT.md`, `KNOWN_ISSUES.md`, `NEXT_STEPS.md`) with the merged state of PRs #102–#108 (#110)
- **workflow**: `.gitignore` no longer excludes `.claude/agent-memory/` — tracked agent-memory seeds and the directory layout are now part of the repo (#111)
- **release pipeline**: `.goreleaser.yml` no longer ships the private-repo download strategy — homebrew formulas now consume public release assets directly, unblocking the public-flip moment (#114)

### Fixed
- **morpheus**: wizard silent error handling — cycle configuration now propagates errors with zero-value guards instead of returning empty configs on failure (#105)
- **neo**: `neo init` path synchronization prevents `.neo.json` state mismatch when the generated ecosystem is at a different absolute path than the wizard's CWD (#105)
- **oracle**: `oracle update` subprocess timeout now uses `Setpgid: true` + process-group-aware `cmd.Cancel` + `cmd.WaitDelay = 2s` — eliminates `WaitDelay expired before I/O complete` failures under `go test -race` (#105)
- **morpheus**: scaffold rollback cleanup now removes all partially written files on mid-generation failure (#105)
- **neo**: `neo init --preset` no longer requires a TTY — non-interactive preset runs (CI, scripts) work without a controlling terminal (#115)
- **trinity**: `trinity health` skill audit reconciled with the current 14-skill ecosystem; missing-`.claude/knowledge/` severity downgraded from CRITICAL to WARN since it is expected before the first oracle run (#117)
- **docs**: README agent count corrected — `neo` provisions 6 agents in generated ecosystems, not 10 (the-matrix repo itself has 10) (#118)

### Removed
- **docs**: stale `docs/templates/tool-claude-md.md` template (280 lines) — orphaned since the Go monorepo migration; tool-level CLAUDE.md authoring no longer references it (#119)

### Security
- **go**: Go toolchain 1.25.8 → 1.25.9 — CVE GO-2026-4870 (TLS handshake DoS) (#102)
- **morpheus**: autonomous-loop reviewer template (`config.sh.tmpl`) privilege downgraded — REVIEWER_TOOLS no longer grants Write; limited to Read/Grep/Glob/Bash (H-01, #104)
- **oracle**: `CopyFile` path traversal hardened with `filepath.Clean` + `filepath.EvalSymlinks` + base-directory prefix containment check (H-02, #104)
- **gitleaks**: allowlist tightened to whitelist documentation examples only — eliminates false positives on docs without weakening real-secret detection (#113)

## [1.7.0] - 2026-03-26

### Added
- **oracle**: `oracle update` command — incremental topic refresh; re-researches only stale or missing topics in an existing knowledge directory, skipping topics still within the staleness window (O-04)
- **neo**: `neo init --preset <name>` — skip the interactive wizard entirely with a pre-configured project type; three built-in presets: `originating-project-go-service` (chi v5, bun ORM, ginkgo+gomega, queue), `go-service` (chi v5, testing, no queue), `nextjs-solo` (Next.js, jest, prisma) (NEO-11)
- **morpheus**: Loop A report now includes knowledge gap suggestions — unknown packages and missing topics detected during the loop are surfaced in the report for follow-up oracle research (M-23)
- **neo/trinity**: `neo doctor` and `trinity health` now audit the agent/skill ecosystem — verifies all registered agents and skills have valid definition files and reports missing entries (PR #86)
- **claude ecosystem**: `/session-learn` skill — captures development session findings (bug patterns, reviewer observations, quality gaps) and proposes targeted improvements to `.claude/` agents, skills, and rules via the architect agent
- **ci**: PR-level CI workflow — runs `go test -race`, `go vet`, and `go build` on every pull request

### Fixed
- **neo**: `neo analyze` silently panicked with index-out-of-range when a detected stack had no entries in the `Stacks` slice; guarded with `len() > 0` before all `[0]` index accesses (fixes #88)
- **neo**: wizard service loop panicked with index-out-of-range when a stack type was not recognized and `Stacks` remained empty; guarded with `len() > 0` before `Stacks[0]` access (fixes #90)
- **neo**: `neo init --preset` returned exit code 0 on unknown preset name and OS errors — only `Ctrl+C` (huh.ErrUserAborted) now exits 0; all other errors print a red message to stderr and exit 1
- **neo**: `filepath.Abs` error was silently discarded when `--path` was provided in non-preset mode; now propagated and reported
- **morpheus**: `morpheus doctor` refactored for testability — behaviour extracted from `cobra.Command` handler into pure functions; 20 integration tests added
- **morpheus**: 20 additional IsInternal rendering tests for previously uncovered templates
- **morpheus**: security-reviewer agent allowlist was too broad; scoped Bash tool to specific prefixes; `MAX_TURNS` increased to match actual loop requirements
- **monorepo**: resolved 3 code issues (hasReviewerApproval check, error handling, agent parity — fixes #38, #40, #80)
- **monorepo**: resolved 9 documentation staleness issues across context files (fixes #5, #7, #19, #20, #31, #32, #33, #34, #39)

## [1.6.0] - 2026-03-23

### Added
- `morpheus loop --non-interactive`: skip all interactive prompts for CI/CD and automation pipelines; accepts `--goal`, `--max-cycles`, `--dev-turns`, `--test-turns`, `--review-turns` flags
- **morpheus**: `GoVersion` field added to `ProjectContext`; Dockerfile templates now use `{{.GoVersion}}` — no more hardcoded `golang:1.24` (closes #67)
- **morpheus**: mandatory "Coverage Gap Detection" step added as Step 1 to all 7 tester agent templates (morpheus go/node/loop + neo) — coverage gaps are now enumerated before any tests are written (closes #68)
- **claude ecosystem**: two new auto-loaded rules — `security-baseline.md` (6 Go security patterns: path traversal, shell injection, file permissions, secret handling, template injection, crypto/rand) and `ecosystem-conventions.md` (agent/skill/rule convention quick reference, condensed from /provision checklist)
- **claude ecosystem**: manager agent Pre-Work strategy section — defines when to invoke `/strategy-monthly` (>30 days or new Phase) and `/strategy-weekly` (active sprint >7 days), with fresh-clone fallback and explicit skip conditions

### Fixed
- **morpheus**: `DECISIONS.md.tmpl` unconditionally rendered Bun ORM, Google Wire, Supabase, and originating project-specific ADRs for all projects; originating project ADR blocks now gated behind `{{if .IsInternal}}` — non-originating project projects receive generic ADR stubs (closes #70)
- **morpheus**: config templates (`api-local.yaml.tmpl`, `worker-local.yaml.tmpl`) used bare `your-*` placeholder credentials; replaced with `${ENV_VAR}` expansion syntax (closes #69)
- **morpheus**: `Makefile.tmpl` `generate-wire`, `generate`, and `install-tools` targets (Wire, Ginkgo, Gomock) leaked into non-originating project projects; now gated behind `{{if .IsInternal}}`
- **claude ecosystem**: `/audit` and `/doublecheck` skills were missing `Agent` from `allowed-tools` — subagents were silently skipped, making quality gates non-functional
- **claude ecosystem**: `/owasp-review` skill was missing `Agent` from `allowed-tools` — could never invoke `security-reviewer`
- **claude ecosystem**: `security-reviewer` agent had broken Bash tool syntax (`Bash(make build)`) — scoped to `Bash(make:*)`, `Bash(go:*)`, `Bash(git:*)`, `Bash(npm:*)`, `Bash(ls:*)`, `Bash(find:*)`
- **claude ecosystem**: `tech-lead` agent had unrestricted `Bash`; scoped to match security-reviewer pattern
- **claude ecosystem**: 6 skills (`commit`, `fix`, `issue`, `release`, `owasp-review`, `secret-scan`) were orphaned — no agent had them in `skills:` frontmatter; now wired to manager, developer, tester, tech-lead
- **claude ecosystem**: 4 security skills (`dep-audit`, `owasp-review`, `secret-scan`, `security-scan`) had non-standard frontmatter (`name/version/maintainer`); standardized to `description/argument-hint/allowed-tools`
- monorepo: gate `docker-compose.yml.tmpl` and `CLAUDE.md.tmpl` originating project infrastructure names behind `{{ if .IsInternal }}` — non-originating project projects now get project-prefixed self-contained infra (fixes #55)
- morpheus: `RunGeneralWizardWithContext` silently discarded `strconv.Atoi` error on port input, producing `ApiPort = 0`; now falls back to 8080 with explicit error handling (fixes #56)
- morpheus: module path wizard accepted `https://github.com/...` URLs and wrote invalid `go.mod`; `normalizeModulePath()` now strips `https://` and `http://` prefixes in all four wizard functions (fixes #57)
- morpheus: `docker-compose.infra.yml.tmpl` hardcoded `originating-project-{{ .ServiceName }}` container names and `originating-project_main_db` for all Go projects; now gated behind `{{ if .IsInternal }}` — non-originating project projects get project-prefixed self-contained infra (fixes #58)
- oracle: `os.Chmod` result was silently discarded in `generator.go`; now captured and returned as an error (fixes #41)
- monorepo: `make release` porcelain check failed on gitignored untracked directories (`tasks/`, `bin/`); now filters `??` untracked entries — only tracked changes block a release (fixes #22)

## [1.5.0] - 2026-03-22

> **Tag note**: The `v1.5.0` git tag annotates commit `17ef470` (a post-release gitignore chore), not the release commit itself. This is cosmetic — the release artifacts are correct. Leaving the tag as-is to avoid a force-push.

### Added
- **morpheus**: expanded autonomous loop from 3-agent to 6-agent pipeline (manager, strategist, developer, tester, reviewer, security-reviewer) with dual-gate termination — both reviewer APPROVED and security SECURITY_APPROVED required (Phase 3a)
- **morpheus/neo**: security-reviewer agent template deployed to all morpheus template paths (go/, node/, loop/) and neo templates (Phase 3b)
- **morpheus/neo**: `/owasp-review` skill template in all morpheus and neo template paths (Phase 3b)
- **morpheus/neo**: `/secret-scan` skill (gitleaks integration) in all template paths and the-matrix `.claude/skills/` (Phase 3c)
- **morpheus/neo**: `/dep-audit` skill (govulncheck + trivy + npm audit) in all template paths and the-matrix `.claude/skills/` (Phase 3c)
- **morpheus/neo**: `/security-scan` skill (gosec + semgrep SAST) in all template paths and the-matrix `.claude/skills/` (Phase 3c)
- **morpheus**: full OWASP Top 10:2025 investigation protocol replaces placeholder `security.md.tmpl` in all 3 autonomous loop prompt paths (Phase 3c)
- **claude ecosystem**: `security-reviewer` agent — on-demand OWASP Top 10:2025 + ASI01-ASI10 auditor, read-only (Phase 3b)
- **claude ecosystem**: `/owasp-review`, `/secret-scan`, `/dep-audit`, `/security-scan` skills — full tool-backed security suite for the-matrix itself (Phase 3b/3c)
- **neo**: `neo analyze` command — reverse oracle: static analysis of existing codebases generates a populated `.claude/` ecosystem in under 2 seconds. Detects language, framework, architecture, testing, CI/CD. Supports Go, Node.js, Python, Flutter, Rust, Ruby (NEO-12)
- **neo**: `neo registry list/search/info/pull` — knowledge registry client for browsing and downloading pre-researched oracle packs from `0merUfuk/knowledge-registry`; local cache at `~/.the-matrix/registry/` (NEO-10)
- **neo**: `neo init` checks knowledge registry before running oracle research — instant pack download when a matching pack exists
- **neo**: `neo init` invokes `morpheus init --context --service-name --output-dir` per service for multi-repo microservice projects (NEO-06)
- **morpheus**: `morpheus init --context <path>` flag — reads `.neo.json` to pre-populate service name, tech stack, and IsInternal flag (M-22)
- **oracle**: `oracle research --gold-standards <dir>` flag — pluggable gold standard injection via explicit directory or `~/.the-matrix/gold-standards/<language>/` convention (O-06)
- **oracle**: `oracle inject` prompts to register injected docs as gold standard for future runs (O-06)

### Fixed
- **morpheus**: originating project-specific content (Stripe, RabbitMQ credential patterns, Go-specific tooling) now gated behind `{{ if .IsInternal }}` in all templates — non-originating project projects receive correct stack-agnostic output (M-21)
- **trinity**: `trinity sync` oracle-not-found error now includes install guidance (`brew tap 0merUfuk/thematrix && brew install oracle`) instead of only asking if oracle is in PATH
- **neo**: `neo init --dry-run` no longer contacts the registry or writes pack files
- **neo**: pack name path traversal guard — rejects names containing `../` or unsafe characters

## [1.4.0] - Not Released (ghost version)

No release was cut at v1.4.0. The version identifier was reserved in the roadmap (`.claude/rules/versioning.md`) for generalization + registry work (M-21, M-22, NEO-10, NEO-12), but these shipped together as part of v1.5.0. The v1.4.0 entry is intentionally empty to preserve SemVer continuity.

## [1.3.0] - 2026-03-18

### Added
- **oracle**: `oracle export` command — transform oracle knowledge docs to Cursor (.mdc), GitHub Copilot, AGENTS.md, and Windsurf formats
- **oracle**: format adapter architecture with `Format` interface and init()-based auto-registration
- **oracle**: language detection from directory names (11 languages supported) with format-appropriate file globs
- **oracle**: Cursor MDC auto-splitting for docs exceeding 6000 characters
- **monorepo**: `.matrix.yaml` unified configuration support for cross-tool defaults
- **neo/morpheus**: generated agent ecosystem upgraded to match the-matrix parity (GitHub MCP, ship workflow)

### Fixed
- **neo**: Flutter/Dart language value mismatch — wizard stored "flutter" but templates check "dart", silently stripping all Flutter-specific agent content
- **trinity**: `--path` flag ignored for `.matrix.yaml` config loading — `PersistentPreRunE` hardcoded `matrixcfg.Load(".")` instead of loading from target directory
- **oracle**: same `--path` config loading bug as trinity — moved config to point-of-use in each command handler
- **morpheus**: `hasReviewerApproval` checked ANY cycle's review verdict instead of latest — could approve finalize after a stale APPROVED even if latest cycle was NEEDS_FIX
- **morpheus**: `os.RemoveAll` and `WriteFileString` errors silently discarded in init and loop commands
- **oracle**: `filepath.Abs` and `os.RemoveAll` errors silently swallowed in inject and research commands
- **oracle**: dry-run tag showed `[ejs]` instead of `[tmpl]` (Node.js artifact)
- **oracle**: misleading "no docs found" error when all docs fail validation — now reports skipped doc names and validation criteria
- **oracle**: `extractTitle` guards against bare `# ` headings that produced empty titles and broken TOC anchors
- **oracle**: removed dead `"claude"` format skip in export all-format loop
- **oracle**: `inject.go` comment changed from "Atomic write" to "Sequential write" — writes are not atomic
- **trinity**: sync log showed misleading "unchanged" count identical to "updated" count — removed, now shows only "new" and "updated"

### Removed
- **neo**: dead trinity import and blank identifier workaround in `doctor.go`
- **trinity**: dead `checkOracleUpdate()` function in `refresh.go` — parsed oracle --help then printed "not yet implemented"

## [1.2.2] - 2026-03-17

### Fixed
- **release**: correct `require_relative` path in Homebrew formulae — strategy file is at tap root `lib/`, not `Formula/lib/` (PR #14)
- **release**: use GitHub-native changelog for clickable commit and PR links in releases (PR #13)

## [1.2.1] - 2026-03-17

### Fixed
- **release**: Homebrew formulae now use `GitHubPrivateRepositoryReleaseDownloadStrategy` for private repo binary downloads — fixes `brew install` failing with 404 on private repos (PR #11)

## [1.2.0] - 2026-03-17

### Added
- **claude ecosystem**: `/issue` skill for consistent GitHub Issue creation with enforced conventions (PR #9)
- **claude ecosystem**: 14-label taxonomy on GitHub (priority P0–P3, type, tool dimensions)
- **claude ecosystem**: `/doublecheck` and `/audit` skills for context file verification and ground-truth auditing

### Fixed
- **morpheus**: `morpheus doctor` false positives on loop-only projects — split `CRITICAL_FILES` into `CORE_FILES` (always checked) and `INIT_ONLY_FILES` (init-scaffolded only) (PR #4)
- **monorepo**: `.gitignore` `tasks/` pattern matched at any depth, masking 9 embedded template files under `internal/morpheus/templates/*/tasks/` — changed to root-only `/tasks/` (PR #4)
- **docs**: context file accuracy — corrected versions, template counts, file paths, phase priorities, and known issue statuses across 13+ documentation files (PR #4)
- **morpheus**: `AUTONOMOUS_LOOP.md` rendering — added frontmatter and code fences for proper GitHub display (PR #4)
- **docs**: stale Node.js path references in action files updated to Go monorepo paths (PR #4)
- **docs**: `KNOWN_ISSUES.md` section headers updated from `-cli` suffix to plain tool names (PR #4)
- **release**: pipeline hardened with workflow safety guards and `HOMEBREW_TAP_GITHUB_TOKEN` documentation (PR #3)

### Changed
- **monorepo**: Homebrew tap renamed from `homebrew-tap` to `homebrew-thematrix` with updated docs and README (PR #2)
- **cli**: unified CLI style system with theme, spinner, and styled components via lipgloss v2 (PR #1)

## [1.1.0] - 2026-03-17

### Added
- **claude ecosystem**: 5-agent roster (manager, developer, tester, reviewer, strategist) with MCP context7 integration
- **claude ecosystem**: agent orchestration — manager spawns dev/test/review in worktrees with handoff protocols
- **claude ecosystem**: `/commit` skill for the-matrix monorepo
- **claude ecosystem**: migrated commands/ to skills/ directory format

### Changed
- **morpheus**: loop termination architecture — reviewer APPROVED verdict is now the sole quality gate (replaces agent-written `ALL_TASKS_COMPLETE` / `status.txt` mechanism)
- **morpheus**: `morpheus finalize` and `morpheus status` now check reviewer verdict in cycle review files instead of `status.txt`
- **morpheus**: developer prompt no longer writes `ALL_TASKS_COMPLETE` — loop always runs full Dev → Test → Review pipeline per cycle
- **morpheus**: reviewer prompt now includes explicit termination contract (APPROVED stops loop, NEEDS_FIX continues)

### Fixed
- SemVer alignment: CHANGELOG headings use `[X.Y.Z]` (no v prefix per Keep a Changelog), git tags use `v` prefix per Go convention
- Resolved all `/doublecheck` findings — stale `the-matrix-go/` paths, version references, code warnings

## [1.0.0] - 2026-03-16

First stable release. Complete Go rewrite of the-matrix autonomous agent ecosystem (previously 3 separate Node.js CLIs). All 4 tools compiled into standalone binaries with embedded templates, zero runtime dependencies.

### Added

**neo** v1.0.0 — Meta-CLI Orchestrator
- `neo init` — 12-question wizard to provision complete `.claude/` agent ecosystem
- `neo doctor` — validate ecosystem integrity (read-only)
- `neo status` — ecosystem dashboard with knowledge freshness per stack
- `neo update` — detect stale knowledge, suggest oracle/trinity refresh workflow
- 14 embedded templates for ecosystem provisioning

**morpheus** v1.0.0 — Service Scaffolding + Autonomous Development Loops
- `morpheus init` — scaffold new services with full `.claude/` + `.autonomous/` + `tasks/` setup
- `morpheus loop` — add autonomous dev loop to ANY existing project with a CLAUDE.md
- `morpheus doctor` — validate scaffolded service before running loop
- `morpheus status` — loop progress at a glance
- `morpheus finalize` — post-loop `.claude/` context file population
- `morpheus report` — post-loop analytics from cycle data
- 42+ embedded templates (Go + Node.js + loop + shared)
- Generic loop prompts that work on any tech stack via CLAUDE.md context
- Claude-powered task decomposition for `tasks/todo.md`

**oracle** v1.0.0 — Automated Knowledge Synthesis
- `oracle research` — 3-phase autonomous loop (research → human gate → synthesis)
- `oracle research --config` — non-interactive mode from JSON config
- `oracle status` — loop progress or knowledge freshness check
- `oracle inject` — validate + copy oracle output to `.claude/knowledge/`
- 6 embedded templates for research workspace scaffolding

**trinity** v1.0.0 — Ecosystem Maintenance Runtime
- `trinity health` — read-only ecosystem validation
- `trinity sync` — push oracle output to `.claude/knowledge/` with validation
- `trinity refresh` — detect stale docs, suggest refresh workflow
- `.trinity.json` config for multi-stack sync

**Shared Infrastructure**
- `internal/cli` — lipgloss v2 terminal styling (colors, progress bars, status lines)
- `internal/config` — JSON/file I/O, config.sh parsing
- `internal/tmpl` — Go template engine with 17 custom helpers
- `internal/wizard` — charm.land/huh v2 interactive prompt wrappers
- `internal/subprocess` — `claude -p` wrapper for autonomous loops
- `internal/staleness` — document age calculation for freshness detection
- GoReleaser configuration for cross-platform binary distribution
