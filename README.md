**Version**: 2.2
**Created**: 2026-03-12
**Last Updated**: 2026-08-10
**Authors:** Ömer Ufuk

---

# the-matrix

> Four Go CLIs that provision and maintain autonomous agent ecosystems for Claude Code and OpenAI Codex.

Most agent setups are hand-crafted, generic, and stale within weeks. The-matrix replaces that ritual with four small tools that interview your repo, synthesize real knowledge from Tier-1 sources, scaffold services with the autonomous loop pre-wired, and keep everything fresh as your stack evolves.

- **Setup-time rigor over runtime cleverness.** Permissions, tool allowlists, agent roles, and ADRs are pre-filled at provisioning time — not negotiated mid-session.
- **Knowledge is first-class.** Oracle synthesizes best-practice docs from official sources; trinity flags when they go stale and prints the re-research commands to run.
- **Multi-repo aware.** Got six services? Neo treats it that way, with per-service agent configs that share a common knowledge base.

## Project film

https://github.com/user-attachments/assets/f4228e7d-0437-485a-b958-9c60e171406c

An illustrative project film connecting neo, oracle, morp, and trinity: explicit setup, invoked research and development loops, review gates, and manual freshness checks. The workflow and artifacts are constructed examples, not a recording of executed commands.

## Install

> **Current public release**: v1.9.0 (released 2026-08-11). Source builds currently report `1.9.0-dev`. The `v1.8.0` tag marks the clean root commit; `main` continues from that public baseline.

### Homebrew

```bash
brew tap 0merUfuk/thematrix
brew install neo morp oracle trinity
```

Prefer not to use the command line? Install The Matrix app instead. It brings the four tools with it and runs in your browser:

```bash
brew install 0merUfuk/thematrix/matrix
brew services start matrix       # keeps it running; then open http://127.0.0.1:7474
```

Then in any project:

```bash
cd ~/my-project
neo init                         # defaults to Claude Code
neo init --runtime codex         # provision an OpenAI Codex ecosystem
```

### From source

```bash
git clone https://github.com/0merUfuk/the-matrix.git
cd the-matrix
make build
```

Binaries land in `bin/`. Add it to your `PATH` or copy the binaries elsewhere on `PATH`.

### With `go install`

Requires Go 1.26.10+:

```bash
go install github.com/0merUfuk/the-matrix/cmd/neo@latest
go install github.com/0merUfuk/the-matrix/cmd/morp@latest
go install github.com/0merUfuk/the-matrix/cmd/oracle@latest
go install github.com/0merUfuk/the-matrix/cmd/trinity@latest
```

## Tool Inventory

| Tool | What it does for you |
|------|----------------------|
| **neo** | One command (`neo init`) to stand up a runtime-specific agent ecosystem tailored to your stack. |
| **morp** | Scaffolds new services with an autonomous development loop already wired in, or drops the same loop into an existing repo via `morp loop`. |
| **oracle** | Researches any tech stack from official docs and maintained repositories, producing 17 durable best-practice knowledge docs your agents can read. |
| **trinity** | A CLI you invoke periodically — detects stale knowledge docs, syncs fresh oracle output into your project, and flags when re-research is due by printing the commands to run. |

All four compile to standalone binaries with template assets embedded via `go:embed`. Template rendering needs no external template files. Agent-powered research, decomposition, and loop commands require an authenticated CLI for the selected runtime; script-based loops also require Bash.

## System Topology

Neo is the single entry point. It interviews your repo, provisions context for the selected agent runtime, and prints the oracle and trinity commands to run for the detected stack. `neo init` invokes `morp` directly for multi-repo scaffolds, but oracle and trinity remain independent commands. Each tool is also usable standalone: `oracle research` creates a knowledge workspace, while `morp loop` adds the autonomous loop to an existing project without neo.

See the [Architecture](#architecture) section below for the monorepo layout and design decisions.

## Supported Agent Runtimes

| Runtime | Primary context | Generated integration |
|---------|-----------------|-----------------------|
| **Claude Code** | `CLAUDE.md` | `.claude/agents/`, `.claude/skills/`, rules, settings, and runtime shell config |
| **OpenAI Codex** | `AGENTS.md` | Consolidated role guidance and Codex runtime shell config |

Use `--runtime claude` or `--runtime codex` with `neo init`, `morp init`, `morp loop`, and `oracle research`. An explicit selection is recorded as `runtime:` in the target project's `.matrix.yaml`. With no flag, selection checks project-local Matrix config, then `CLAUDE.md`, then `AGENTS.md`, and finally defaults to Claude Code.

Claude projects require an authenticated `claude` CLI. Codex projects require an authenticated `codex` CLI (`codex login`). Both runtimes use `.claude/knowledge/` as the shared knowledge directory; that generated target-project path is independent of this repository's private root `.claude/` state.

## Repository Hygiene

The-matrix generates `.claude/` integration for Claude projects and uses `.claude/knowledge/` as the shared knowledge location for either runtime. This repository's own root `.claude/` directory is different: it is local/private maintainer state, ignored by Git, and absent from fresh public clones. This repository's public history starts from a clean root commit. Old history was rewritten to remove all former-project references; `v1.8.0` marks the clean root, and `main` continues from it.

## Key Commands

### neo — Provision an agent ecosystem

```bash
# Interactive wizard; choose claude or codex explicitly
neo init --runtime codex

# Auto-detect existing codebase and generate a Claude ecosystem (no wizard)
neo analyze

# Check ecosystem health
neo doctor

# View ecosystem status
neo status

# Browse the knowledge registry for pre-built packs
neo registry list
neo registry info go-rules
```

### morp — Scaffold a new service

```bash
# Interactive wizard to generate a new Go or Node.js service
morp init --runtime codex

# Set up autonomous development loop on an existing project
morp loop --runtime codex

# Validate Claude scaffold output before running the loop
morp doctor

# Check autonomous loop progress
morp status
```

### oracle — Research and synthesize knowledge

```bash
# Interactive wizard to research a tech stack with Codex
oracle research --runtime codex

# Non-interactive mode with a config file
oracle research --config stack.json --runtime claude

# Check research loop progress
oracle status

# Inject oracle output into a project's .claude/knowledge/
oracle inject --from output/go-fiber/ --to /path/to/project/.claude/knowledge/go-rules/
```

### trinity — Maintain an ecosystem

```bash
# Runtime-aware, read-only ecosystem integrity check
trinity health --path /path/to/project

# Sync oracle knowledge output into a project
trinity sync --from /path/to/oracle/output --to /path/to/project

# Detect stale documents and suggest refresh
trinity refresh --path /path/to/project
```

### The Matrix app — for people who don't use the CLI

```bash
matrix                     # opens The Matrix in your browser
```

The Matrix app runs the same four tools behind a simple, step-by-step screen: choose a project folder, set up its AI agents (two questions instead of a wizard), see how healthy the project is and which knowledge is out of date, research best practices for the project's stack (with a pause to read the findings before the docs are written), add the finished docs for the agents, and let the agents build a goal you describe while you watch their progress. It only listens on this computer (`127.0.0.1:7474`), keeps its project list in `~/.the-matrix/projects.json`, and asks before anything that changes files or uses your agent account. Research and building run on your own Claude Code or Codex account and can take an hour or more; they keep going if you close the page, and you can stop them at any time. Closing the app stops them too.

## Agent Roster

Claude ecosystems contain 6 specialized agent definitions that coordinate on autonomous work: `manager`, `developer`, `tester`, `reviewer`, `strategist`, and `security-reviewer`. Each has a scoped tool allowlist and role definition; code-writing Claude agents can use isolated git worktrees. Codex ecosystems consolidate the same role guidance into `AGENTS.md` and do not provision separate agent or skill files.

The maintainer checkout may have additional private workflow agents and context under its ignored root `.claude/` directory. Those local files are not shipped in this public repo or in fresh clones.

## Releasing

Releases are automated via [GoReleaser](https://goreleaser.com/) and GitHub Actions. Pushing a `v*` tag triggers the pipeline: tests run, cross-platform binaries are built, a GitHub Release is created, and Homebrew formulae are pushed to the tap.

```bash
# Tag and push (runs tests before tagging)
make release VERSION=X.Y.Z SUMMARY="one-line release summary"
```

If a release partially fails (e.g., token expiry mid-publish), delete the draft release on GitHub and re-run the workflow by re-pushing the tag: `git tag -af vX.Y.Z -m "Release vX.Y.Z — re-release" && git push origin vX.Y.Z --force`.

Two GitHub Actions secrets are required on the repository:

| Secret | Purpose |
|--------|---------|
| `GITHUB_TOKEN` | Built-in. Creates GitHub Releases and uploads assets. |
| `HOMEBREW_TAP_GITHUB_TOKEN` | PAT with `repo` scope. Pushes formulae to `0merUfuk/homebrew-thematrix`. |

## Architecture

```
the-matrix/
├── cmd/           # CLI entrypoints (one per tool)
├── internal/
│   ├── cli/       # Terminal formatting (lipgloss)
│   ├── config/    # JSON/file I/O
│   ├── tmpl/      # Template engine (text/template + 17 helpers)
│   ├── wizard/    # Interactive prompts (huh v2)
│   ├── runtime/   # ARAL interface, registry, detection, Claude/Codex adapters
│   ├── subprocess/# Runtime-neutral agent subprocess wrapper
│   ├── staleness/ # Document age calculation
│   ├── matrixcfg/ # .matrix.yaml schema and runtime persistence
│   ├── neo/       # neo implementation + embedded runtime templates
│   ├── morpheus/  # morp implementation + shared/runtime template layers
│   ├── oracle/    # oracle implementation + export/runtime templates
│   ├── registry/  # knowledge registry (pack catalog + validation)
│   └── trinity/   # trinity implementation (no templates)
├── Makefile
└── .goreleaser.yml
```

Key design decisions:

- **Monorepo**: All 4 tools share internal packages, reducing duplication.
- **go:embed**: Templates are compiled into binaries -- no external file dependencies.
- **cobra**: Standard Go CLI framework for commands, flags, and help text.
- **Charm.land**: `huh` for interactive wizards, `lipgloss` for styled terminal output.

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feat/my-feature`)
3. Make your changes and add tests
4. Run `make check` to validate (fmt-check + lint + test-race)
5. Commit and open a pull request

## License

MIT
