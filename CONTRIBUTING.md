# Contributing to the-matrix

Thanks for your interest in contributing! the-matrix is a CLI-driven autonomous agent ecosystem built in Go. This guide covers the basics.

## Prerequisites

- Go 1.26.9+
- `make` (GNU Make)
- A GitHub account

## Getting Started

```bash
git clone https://github.com/0merUfuk/the-matrix.git
cd the-matrix
make build        # builds all 4 binaries to bin/
make test         # runs the full test suite
```

If `make build` and `make test` pass, you're ready to contribute.

## Development Workflow

1. **Fork** the repository
2. **Create a branch** from `main`: `git checkout -b feat/my-feature`
3. **Make your changes** — keep commits focused
4. **Add tests** for any new functionality
5. **Validate locally**:
   ```bash
   make vet        # go vet
   make test       # full test suite
   make build      # all 4 binaries compile
   ```
6. **Commit** with a clear message (see Commit Style below)
7. **Open a pull request** — describe what changed and why

## Commit Style

Use [Conventional Commits](https://www.conventionalcommits.org/):

```
feat(morp): add Python/FastAPI wizard support
fix(trinity): correct staleness threshold for quarterly stacks
docs(README): update agent roster count
chore(deps): bump cobra to v1.9.0
```

Scope: `neo`, `morp`, `oracle`, `trinity`, `runtime`, `subprocess`, `matrixcfg`, `cli`, `config`, `tmpl`, `wizard`, `docs`, `deps`, or `ci`.

## Project Structure

```
cmd/           # CLI entrypoints (one per tool)
internal/      # Shared packages + per-tool implementations
  cli/         # Terminal formatting (lipgloss)
  config/      # JSON/file I/O
  tmpl/        # Template engine (text/template + 17 helpers)
  wizard/      # Interactive prompts (huh v2)
  runtime/     # ARAL interface, registry, detection, Claude/Codex adapters
  subprocess/  # Runtime-neutral invocation + legacy Claude wrapper
  staleness/   # Document age calculation
  matrixcfg/   # .matrix.yaml schema and runtime persistence
  neo/         # neo implementation + embedded runtime templates
  morpheus/    # morp implementation + shared/runtime template layers
  oracle/      # oracle implementation + export/runtime templates
  registry/    # knowledge registry (pack catalog)
  trinity/     # trinity implementation
```

All templates are embedded via `go:embed` — no external file dependencies at runtime. Runtime-independent content belongs under `templates/shared/`; CLI wrappers, settings, and shell invocation belong under `templates/runtimes/<name>/`. Claude target projects generate `.claude/` integration, while Codex projects generate `AGENTS.md` and share `.claude/knowledge/`. This repo's own root `.claude/` directory is private/ignored maintainer state and is absent from fresh public clones.

## Runtime Changes

Runtime adapters implement `internal/runtime.Runtime`, register in `init()`, and are imported through `internal/runtime/builtin`. Preserve the no-flag Claude compatibility path. Changes to selection, generators, templates, subprocess invocation, doctor, or health checks should cover explicit Claude, explicit Codex, automatic detection, and generated fixture content. Keep runtime-specific context paths out of templates rendered for another runtime.

## Architecture Decisions

Use `ARCHITECTURE.md`, issues, and pull request discussion for public design context. Maintainer-local ADR notes may exist in private `.claude/` context, but they are not required for public contributions.

## What to Work On

- Issues labeled [`good first issue`](https://github.com/0merUfuk/the-matrix/labels/good%20first%20issue) are scoped for new contributors
- Issues labeled [`bug`](https://github.com/0merUfuk/the-matrix/labels/bug) are bugs that need fixing
- If you have a feature idea, open an issue first to discuss scope before implementing

## Code Style

- Follow standard `gofmt` / `go vet` conventions
- Match the existing code style in the file you're editing
- Keep functions focused — if a function exceeds ~80 lines, consider splitting
- Table-driven tests are preferred (see existing `*_test.go` files for patterns)

## Pull Request Checklist

- [ ] `make vet` passes
- [ ] `make test` passes (all packages green, 0 failures)
- [ ] `make build` passes (all 4 binaries)
- [ ] New code has tests
- [ ] Commit messages follow Conventional Commits
- [ ] No generated/private `.claude/` files are staged

## License

By contributing, you agree that your contributions will be licensed under the [MIT License](LICENSE).

## Questions?

Open an issue with the `question` label, or start a discussion in the PR.
