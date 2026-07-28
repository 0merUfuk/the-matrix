# Contributing to the-matrix

Thanks for your interest in contributing! the-matrix is a CLI-driven autonomous agent ecosystem built in Go. This guide covers the basics.

## Prerequisites

- Go 1.25.8+
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
feat(morpheus): add Python/FastAPI wizard support
fix(trinity): correct staleness threshold for quarterly stacks
docs(README): update agent roster count
chore(deps): bump cobra to v1.9.0
```

Scope: `neo`, `morpheus`, `oracle`, `trinity`, `cli`, `config`, `tmpl`, `wizard`, `docs`, `deps`, or `ci`.

## Project Structure

```
cmd/           # CLI entrypoints (one per tool)
internal/      # Shared packages + per-tool implementations
  cli/         # Terminal formatting (lipgloss)
  config/      # JSON/file I/O
  tmpl/        # Template engine (text/template + 17 helpers)
  wizard/      # Interactive prompts (huh v2)
  subprocess/  # Claude subprocess wrapper
  staleness/   # Document age calculation
  matrixcfg/   # .matrix.yaml config schema
  neo/         # neo implementation + 23 templates
  morpheus/    # morpheus implementation + 93 templates
  oracle/      # oracle implementation + export formats
  registry/    # knowledge registry (pack catalog)
  trinity/     # trinity implementation
```

All templates are embedded via `go:embed` — no external file dependencies at runtime. The CLIs generate `.claude/` directories in target projects, but this repo's own root `.claude/` directory is private/ignored maintainer state and is absent from fresh public clones.

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
- [ ] `make test` passes (all 15 test packages green, 0 failures)
- [ ] `make build` passes (all 4 binaries)
- [ ] New code has tests
- [ ] Commit messages follow Conventional Commits
- [ ] No generated/private `.claude/` files are staged

## License

By contributing, you agree that your contributions will be licensed under the [MIT License](LICENSE).

## Questions?

Open an issue with the `question` label, or start a discussion in the PR.
