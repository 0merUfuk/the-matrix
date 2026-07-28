**Version**: 1.1
**Created**: 2026-03-17
**Last Updated**: 2026-03-24
**Authors:** Ömer Ufuk

---

# Overall Summary: How the Autonomous Loop Works

## The Concept

A bash script (loop.sh) runs claude -p (headless Claude Code) in a loop. Each iteration has 3 phases —
Developer, Tester, Reviewer — each getting a different role prompt. They communicate through files in
tasks/ (shared memory, gitignored).

## Architecture

```
.autonomous/                        # Loop infrastructure (gitignored)
├── loop.sh                         # The orchestrator — bash while loop
├── config.sh                       # Turns, model, allowedTools per role
├── SETUP_GUIDE.md                  # (optional, user-created)
└── prompts/
    ├── developer.md                # "Implement the next task, commit, log"
    ├── tester.md                   # "Run tests, write tests, fix bugs"
    └── reviewer.md                 # "Quality gate — APPROVED or NEEDS_FIX"

tasks/                              # Shared memory between phases (gitignored)
├── todo.md                         # Master checklist — human writes, agent checks off
├── TARGET_STATE.md                 # Architectural north star — human writes
├── DECISIONS.md                    # Pre-filled decisions — human writes
├── current-state.md                # Handoff file — each agent rewrites for the next
├── lessons.md                      # Self-correcting memory — all agents append
├── decisions-pending.md            # Pause trigger — agent writes, human resolves
├── status.txt                      # "ALL_TASKS_COMPLETE" sentinel — breaks the loop
├── STOP                            # Touch to halt gracefully (SSH from phone)
├── loop.log                        # Timestamped log of everything
└── cycles/                         # Per-cycle output
    ├── cycle-001-developer-output.json
    ├── cycle-001-tester-output.json
    └── cycle-001-reviewer-output.json
```

## One Cycle Flow

```
Developer                    Tester                      Reviewer
    │                           │                           │
    ├─ Read todo.md             │                           │
    ├─ Pick first [ ] task      │                           │
    ├─ Implement it             │                           │
    ├─ git commit               │                           │
    ├─ Write current-state.md ──┤                           │
    │                           ├─ Read current-state.md    │
    │                           ├─ Run tests                │
    │                           ├─ Write NEW tests          │
    │                           ├─ git commit               │
    │                           ├─ Write current-state.md ──┤
    │                           │                           ├─ Read current-state.md
    │                           │                           ├─ git diff since cycle start
    │                           │                           ├─ APPROVED or NEEDS_FIX
    │                           │                           ├─ Write review report
    │                           │                           └─ (if NEEDS_FIX → adds tasks)
    │                           │                           │
    └───────── next cycle ──────┴───────────────────────────┘
```

## Safety Mechanisms

```
┌────────────────────────────┬───────────────────────────────────────────────────────────────────────┐
│         Mechanism          │                                  How                                  │
├────────────────────────────┼───────────────────────────────────────────────────────────────────────┤
│ tasks/STOP                 │ Touch from anywhere to halt after current phase                       │
├────────────────────────────┼───────────────────────────────────────────────────────────────────────┤
│ tasks/decisions-pending.md │ Agent writes here when it needs a human decision → loop pauses        │
├────────────────────────────┼───────────────────────────────────────────────────────────────────────┤
│ MAX_CYCLES                 │ Hard cap on total iterations                                          │
├────────────────────────────┼───────────────────────────────────────────────────────────────────────┤
│ --max-turns                │ Per-phase turn limit (prevents infinite loops within a phase)         │
├────────────────────────────┼───────────────────────────────────────────────────────────────────────┤
│ Git tags                   │ auto/cycle-001-pre-developer etc. — rollback points before each phase │
├────────────────────────────┼───────────────────────────────────────────────────────────────────────┤
│ Dirty git check            │ Logs warning if previous phase left uncommitted code                  │
└────────────────────────────┴───────────────────────────────────────────────────────────────────────┘
```

## Setup Steps

1. Run `morpheus init` to scaffold a full service with .claude/ ecosystem
2. Run `morpheus loop` to add the autonomous loop to your project
3. Review & refine — check prompts match your tech stack, config has right tools
4. Fill human files — tasks/todo.md, tasks/TARGET_STATE.md, tasks/DECISIONS.md
5. Git init + commit — loop needs at least one commit for tagging
6. Dry run — single claude -p call with developer prompt to verify
7. Run loop — .autonomous/loop.sh (foreground) or nohup ... & (background)
8. Morning review — check tasks/loop.log, git log, tasks/current-state.md

## What Was Fixed After Test Run

```
┌──────────────────────┬────────────────────────────────┬────────────────────────────────────────────┐
│        Issue         │           Root Cause           │                    Fix                     │
├──────────────────────┼────────────────────────────────┼────────────────────────────────────────────┤
│ Script crashed       │ ${role^^} — bash 4+ on macOS   │ Replaced with tr '[:lower:]' '[:upper:]'   │
│ silently             │ bash 3.2                       │                                            │
├──────────────────────┼────────────────────────────────┼────────────────────────────────────────────┤
│ Nested session       │ CLAUDECODE env var set inside  │ unset CLAUDECODE at script start           │
│ blocked              │ Claude Code                    │                                            │
├──────────────────────┼────────────────────────────────┼────────────────────────────────────────────┤
│ Every phase hit      │ 15/10/8 too low                │ Bumped to 25/18/12 (test) and 30/20/15     │
│ max-turns            │                                │ (portal)                                   │
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
