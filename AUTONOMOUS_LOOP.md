**Version**: 1.2
**Created**: 2026-03-17
**Last Updated**: 2026-07-29
**Authors:** Ömer Ufuk

---

# Overall Summary: How the Autonomous Loop Works

## The Concept

A bash script (loop.sh) runs `claude -p` (headless Claude Code) in a loop. Each iteration has up to 6 phases —
Manager (review/reprioritize), Strategist (resolve pending decisions), Developer, Tester, Reviewer, and Security —
each getting a different role prompt. They communicate through files in `tasks/` (shared memory, gitignored).

Two quality gates must both pass before the loop finalizes: the Reviewer (code quality) and Security
(vulnerability audit). Either gate returning `NEEDS_FIX` / `SECURITY_BLOCKED` triggers another cycle.

## Architecture

```
.autonomous/                        # Loop infrastructure (gitignored)
├── loop.sh                         # The orchestrator — bash while loop
├── config.sh                       # Turns, model, allowedTools per role
├── SETUP_GUIDE.md                  # (optional, user-created)
└── prompts/
    ├── manager.md                  # "Review previous cycle, reprioritize tasks" (cycle > 1 only)
    ├── strategist.md               # "Resolve pending decisions" (only if decisions-pending.md has content)
    ├── developer.md                # "Implement the next task, commit, log"
    ├── tester.md                   # "Run tests, write tests, fix bugs"
    ├── reviewer.md                 # "Quality gate — APPROVED or NEEDS_FIX"
    └── security.md                 # "Security gate — SECURITY_APPROVED or SECURITY_BLOCKED"

tasks/                              # Shared memory between phases (gitignored)
├── todo.md                         # Master checklist — human writes, agent checks off
├── TARGET_STATE.md                 # Architectural north star — human writes
├── DECISIONS.md                    # Pre-filled decisions — human writes
├── current-state.md                # Handoff file — each agent rewrites for the next
├── lessons.md                      # Self-correcting memory — all agents append
├── decisions-pending.md            # Pause trigger — agent writes, human/strategist resolves
├── status.txt                      # "ALL_TASKS_COMPLETE" sentinel — breaks the loop
├── STOP                            # Touch to halt gracefully (SSH from phone)
├── loop.log                        # Timestamped log of everything
└── cycles/                         # Per-cycle output
    ├── cycle-001-developer-output.json
    ├── cycle-001-tester-output.json
    ├── cycle-001-reviewer-output.json
    ├── cycle-001-reviewer.md        # Reviewer verdict file (parsed by loop)
    ├── cycle-001-security-output.json
    └── cycle-001-security.md        # Security verdict file (parsed by loop)
```

## One Cycle Flow

```
    Manager              Strategist              Developer
   (cycle > 1)           (if decisions            (always)
      │                   pending)                   │
      ├─ Review prior     │                         ├─ Read todo.md
      ├─ Reprioritize     ├─ Read decisions-pending  ├─ Pick first [ ] task
      │                   ├─ Resolve ambiguity       ├─ Implement it
      │                   └─ Clear file ─────────────┤  ├─ git commit
      └─────────────────────┤                        ├─ Write current-state.md ──┐
                            │                        │                          │
                            ├────────────────────────┘                          │
                            │                                                   ▼
                                          ┌─────────────────────────────────────┐
                                          │              Tester                  │
                                          ├─ Read current-state.md              │
                                          ├─ Run tests                          │
                                          ├─ Write NEW tests                     │
                                          ├─ git commit                         │
                                          ├─ Write current-state.md ────────────┤
                                          └─────────────────────────────────────┘
                                                                                   │
                                                                                   ▼
                                          ┌─────────────────────────────────────┐
                                          │             Reviewer                 │
                                          ├─ Read current-state.md              │
                                          ├─ git diff since cycle start         │
                                          ├─ APPROVED or NEEDS_FIX              │
                                          ├─ Write review report (.md)          │
                                          └─ (if NEEDS_FIX → adds tasks) ───────┤
                                          └─────────────────────────────────────┘
                                                                                   │
                                                                                   ▼
                                          ┌─────────────────────────────────────┐
                                          │             Security                  │
                                          ├─ OWASP Top 10 + ASI01-10 audit      │
                                          ├─ SECURITY_APPROVED or BLOCKED        │
                                          ├─ Write security report (.md)        │
                                          └─────────────────────────────────────┘
                                                                                   │
                                                                                   ▼
                                          ┌─────────────────────────────────────┐
                                          │  Termination check                   │
                                          ├─ Reviewer APPROVED?                 │
                                          ├─ Security APPROVED?                 │
                                          ├─ BOTH → ALL_GATES_APPROVED → break  │
                                          └─ Either fails → next cycle ─────────┘
```

### Phase order and conditions

| # | Phase | When | Role |
|---|-------|------|------|
| 0a | Manager | cycle > 1 | Review prior cycle output, reprioritize `tasks/todo.md` |
| 0b | Strategist | `decisions-pending.md` has content | Resolve pending architectural decisions |
| 1 | Developer | always | Implement next task, commit, write `current-state.md` |
| 2 | Tester | always | Run + write tests, commit, hand off |
| 3 | Reviewer | always | Code quality gate — APPROVED or NEEDS_FIX |
| 4 | Security | always | Security gate — SECURITY_APPROVED or SECURITY_BLOCKED |

Manager and Strategist are **conditional** phases that run before the core Developer → Tester → Reviewer → Security
pipeline. The loop checks `tasks/STOP` after every phase and halts immediately if present.

## Safety Mechanisms

```
┌──────────────────────┬─────────────────────┬────────────────────────────────────────────┐
│      Mechanism       │       Check         │                   How                       │
├──────────────────────┼─────────────────────┼────────────────────────────────────────────┤
│ tasks/STOP           │ check_stop()        │ Touch from anywhere to halt after current  │
│                      │                     │ phase                                      │
├──────────────────────┼─────────────────────┼────────────────────────────────────────────┤
│ decisions-pending.md │ check_decisions_    │ Agent writes questions here → loop pauses  │
│                      │   pending()         │ until human/strategist resolves them        │
├──────────────────────┼─────────────────────┼────────────────────────────────────────────┤
│ MAX_CYCLES           │ while loop bound    │ Hard cap on total iterations                │
├──────────────────────┼─────────────────────┼────────────────────────────────────────────┤
│ --max-turns          │ per-phase flag      │ Per-phase turn limit (prevents infinite     │
│                      │                     │ loops within a phase)                      │
├──────────────────────┼─────────────────────┼────────────────────────────────────────────┤
│ Git tags             │ tag_cycle()         │ auto/cycle-001-pre-developer etc. —         │
│                      │                     │ rollback points before each phase          │
├──────────────────────┼─────────────────────┼────────────────────────────────────────────┤
│ Dirty git check      │ check_clean_git()   │ Logs warning if previous phase left        │
│                      │                     │ uncommitted code                            │
├──────────────────────┼─────────────────────┼────────────────────────────────────────────┤
│ Dual quality gates   │ check_reviewer_     │ BOTH reviewer AND security must approve    │
│                      │   approved() +      │ before the loop finalizes — either gate    │
│                      │ check_security_     │ failing triggers another cycle             │
│                      │   approved()        │                                            │
└──────────────────────┴─────────────────────┴────────────────────────────────────────────┘
```

## Setup Steps

1. Run `morpheus init` to scaffold a full service with .claude/ ecosystem
2. Run `morpheus loop` to add the autonomous loop to your project
3. Review & refine — check prompts match your tech stack, config has right tools
4. Fill human files — tasks/todo.md, tasks/TARGET_STATE.md, tasks/DECISIONS.md
5. Git init + commit — loop needs at least one commit for tagging
6. Dry run — single `claude -p` call with developer prompt to verify
7. Run loop — `.autonomous/loop.sh` (foreground) or `nohup ... &` (background)
8. Morning review — check tasks/loop.log, git log, tasks/current-state.md

## What Was Fixed After Test Run

```
┌──────────────────────┬────────────────────────────────┬────────────────────────────────────────────┐
│        Issue          │           Root Cause           │                    Fix                     │
├──────────────────────┼────────────────────────────────┼────────────────────────────────────────────┤
│ Script crashed        │ ${role^^} — bash 4+ on macOS   │ Replaced with tr '[:lower:]' '[:upper:]'   │
│ silently              │ bash 3.2                       │                                            │
├──────────────────────┼────────────────────────────────┼────────────────────────────────────────────┤
│ Nested session       │ CLAUDECODE env var set inside  │ unset CLAUDECODE at script start           │
│ blocked              │ Claude Code                    │                                            │
├──────────────────────┼────────────────────────────────┼────────────────────────────────────────────┤
│ Every phase hit       │ 15/10/8 too low                │ Bumped to 25/18/12 (test) and 30/20/15     │
│ max-turns             │                                │ (portal)                                   │
├──────────────────────┼────────────────────────────────┼────────────────────────────────────────────┤
│ No git commits       │ Agent ran out of turns before  │ Reordered prompts: commit first, log       │
│                      │ committing                     │ second                                     │
├──────────────────────┼────────────────────────────────┼────────────────────────────────────────────┤
│ npm test 2>&1 denied │ Shell redirects don't match    │ Gave tester Bash(npm *) (broad prefix)     │
│                      │ glob patterns                  │                                            │
├──────────────────────┼────────────────────────────────┼────────────────────────────────────────────┤
│ mkdir -p denied      │ Narrow pattern issue           │ Added Bash(cat *) and kept Bash(mkdir *)   │
├──────────────────────┼────────────────────────────────┼────────────────────────────────────────────┤
│ Cost references      │ Using Anthropic account, not   │ Removed all cost/budget code; added token  │
│                      │ API                            │ counter for fun                            │
└──────────────────────┴────────────────────────────────┴────────────────────────────────────────────┘
```