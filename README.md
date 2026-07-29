**Version**: 2.1
**Created**: 2026-03-12
**Last Updated**: 2026-07-26
**Authors:** Ömer Ufuk

---

# the-matrix

> Four Go CLIs that provision and maintain autonomous [Claude Code](https://docs.anthropic.com/en/docs/claude-code) agent ecosystems.

Most Claude Code setups are hand-crafted, generic, and stale within weeks. The-matrix replaces that ritual with four small tools that interview your repo, synthesize real knowledge from Tier-1 sources, scaffold services with the autonomous loop pre-wired, and keep everything fresh as your stack evolves.

- **Setup-time rigor over runtime cleverness.** Permissions, tool allowlists, agent roles, and ADRs are pre-filled at provisioning time — not negotiated mid-session.
- **Knowledge is first-class.** Oracle synthesizes best-practice docs from official sources; trinity flags when they go stale and prints the re-research commands to run.
- **Multi-repo aware.** Got six services? Neo treats it that way, with per-service agent configs that share a common knowledge base.

## Install

> **Current public release**: v1.8.0 (released 2026-07-28). The `v1.8.0` tag marks the clean root commit; `main` continues from that public baseline.

### Homebrew

```bash
brew tap 0merUfuk/thematrix
brew install neo morpheus oracle trinity
```

Then in any project:

```bash
cd ~/my-project
neo init          # interview your repo, provision .claude/ ecosystem
```

### From source

```bash
git clone https://github.com/0merUfuk/the-matrix.git
cd the-matrix
make build
```

Binaries land in `bin/`. Add it to your `PATH` or copy the binaries elsewhere on `PATH`.

### With `go install`

Requires Go 1.25.8+:

```bash
go install github.com/0merUfuk/the-matrix/cmd/neo@latest
go install github.com/0merUfuk/the-matrix/cmd/morpheus@latest
go install github.com/0merUfuk/the-matrix/cmd/oracle@latest
go install github.com/0merUfuk/the-matrix/cmd/trinity@latest
```

## Tool Inventory

| Tool | What it does for you |
|------|----------------------|
| **neo** | One command (`neo init`) to stand up a complete agent ecosystem — agents, skills, rules, and knowledge docs — tailored to your stack. |
| **morpheus** | Scaffolds new services with an autonomous development loop already wired in, or drops the same loop into an existing repo via `morpheus loop`. |
| **oracle** | Researches any tech stack from official docs and maintained repositories, producing 17 durable best-practice knowledge docs your agents can read. |
| **trinity** | A CLI you invoke periodically — detects stale knowledge docs, syncs fresh oracle output into your project, and flags when re-research is due by printing the commands to run. |

All four compile to standalone binaries with template assets embedded via `go:embed`. Template rendering needs no external template files; Claude-powered research and autonomous-loop commands require an authenticated Claude Code CLI, and the script-based loops require Bash.

## System Topology

Neo is the single entry point. It interviews your repo, provisions the `.claude/` ecosystem, and prints the oracle and trinity commands to run for the detected stack — `neo init` invokes morpheus directly for multi-repo scaffolds, but oracle and trinity run as independent commands you trigger when you need them. Each tool is also usable standalone — if you only want knowledge docs, `oracle research` is enough; if you only want the loop added to an existing project, `morpheus loop` does that without neo.

See the [Architecture](#architecture) section below for the monorepo layout and design decisions.

## Repository Hygiene

The-matrix still generates `.claude/` directories in projects that use it. This repository's own root `.claude/` directory is different: it is local/private maintainer state, ignored by Git, and absent from fresh public clones. This repository's public history starts from a clean root commit. Old history was rewritten to remove all former-project references; `v1.8.0` marks the clean root, and `main` continues from it.

## Key Commands

### neo — Provision an agent ecosystem

```bash
# Interactive wizard to scaffold a complete .claude/ ecosystem
neo init

# Auto-detect existing codebase and generate .claude/ ecosystem (no wizard)
neo analyze

# Check ecosystem health
neo doctor

# View ecosystem status
neo status

# Browse the knowledge registry for pre-built packs
neo registry list
neo registry info go-rules
```

### morpheus — Scaffold a new service

```bash
# Interactive wizard to generate a new Go or Node.js service
morpheus init

# Set up autonomous development loop on an existing project
morpheus loop

# Validate scaffolded output before running autonomous loop
morpheus doctor

# Check autonomous loop progress
morpheus status
```

### oracle — Research and synthesize knowledge

```bash
# Interactive wizard to research a tech stack
oracle research

# Non-interactive mode with a config file
oracle research --config stack.json

# Check research loop progress
oracle status

# Inject oracle output into a project's .claude/knowledge/
oracle inject --from output/go-fiber/ --to /path/to/project/.claude/knowledge/go-rules/
```

### trinity — Maintain an ecosystem

```bash
# Read-only health check of .claude/ ecosystem integrity
trinity health --path /path/to/project

# Sync oracle knowledge output into a project
trinity sync --from /path/to/oracle/output --to /path/to/project

# Detect stale documents and suggest refresh
trinity refresh --path /path/to/project
```

## Agent Roster

The ecosystems neo provisions are populated with 6 specialized Claude Code agents that coordinate on autonomous work — `manager`, `developer`, `tester`, `reviewer`, `strategist`, and `security-reviewer`. Each has a scoped tool allowlist, a role definition, and (for code-writing agents) isolated git worktrees. Their definitions are embedded as templates and written into each generated project's `.claude/agents/` directory.

The maintainer checkout may have additional private workflow agents and context under its ignored root `.claude/` directory. Those local files are not shipped in this public repo or in fresh clones.

## Releasing

Releases are automated via [GoReleaser](https://goreleaser.com/) and GitHub Actions. Pushing a `v*` tag triggers the pipeline: tests run, cross-platform binaries are built, a GitHub Release is created, and Homebrew formulae are pushed to the tap.

```bash
# Tag and push (runs tests before tagging)
make release VERSION=1.2.0 SUMMARY="knowledge registry and preset mode"
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
│   ├── subprocess/# Claude subprocess wrapper
│   ├── staleness/ # Document age calculation
│   ├── matrixcfg/ # .matrix.yaml unified config schema (Config + per-tool options)
│   ├── neo/       # neo implementation + 23 templates
│   ├── morpheus/  # morpheus implementation + 93 templates (92 .tmpl + loop.sh)
│   ├── oracle/    # oracle implementation + 7 templates (5 .tmpl + loop.sh + update.sh)
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
