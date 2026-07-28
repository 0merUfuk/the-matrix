#!/bin/bash
set -euo pipefail

# ─── Autonomous Development Loop ────────────────────────────────
# Orchestrates: [Manager] → [Strategist] → Developer → Tester → Reviewer → Security cycles
# Each phase runs as a separate `claude -p` call with role-specific prompts
# Shared memory lives in tasks/ directory
#
# Usage:
#   .autonomous/loop.sh              # Run in foreground
#   .autonomous/loop.sh &            # Run in background
#   touch tasks/STOP                 # Graceful stop from anywhere (SSH, phone, etc.)
#   rm tasks/STOP                    # Clear stop signal to resume
# ─────────────────────────────────────────────────────────────────

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Unset CLAUDECODE env var if set — this prevents "nested session" errors
# when the loop is launched from inside a Claude Code terminal.
# Safe for -p (headless) mode which doesn't use interactive TUI resources.
unset CLAUDECODE 2>/dev/null || true

source "${SCRIPT_DIR}/config.sh"

# ─── Logging ────────────────────────────────────────────────────
LOG_FILE="${TASKS_DIR}/loop.log"
TEMP_PROMPT_FILE="${TASKS_DIR}/.current-prompt.tmp"

log() {
  local msg="[$(date '+%Y-%m-%d %H:%M:%S')] $1"
  echo "$msg" | tee -a "$LOG_FILE"
}

# ─── Pre-Flight Checks ─────────────────────────────────────────
preflight() {
  cd "$PROJECT_DIR"

  # Check claude CLI exists
  if ! command -v claude &>/dev/null; then
    log "FATAL: 'claude' CLI not found in PATH. Install: https://code.claude.com"
    exit 1
  fi

  # Check authentication (Anthropic account or API key)
  if [ -z "${ANTHROPIC_API_KEY:-}" ]; then
    if ! claude auth status &>/dev/null 2>&1; then
      log "FATAL: Not authenticated. Run 'claude auth login' first."
      exit 1
    fi
  fi

  # Check git is installed
  if ! command -v git &>/dev/null; then
    log "FATAL: 'git' not found in PATH."
    exit 1
  fi

  # Check git repo (supports both regular repos and worktrees)
  if ! git rev-parse --git-dir > /dev/null 2>&1; then
    log "FATAL: Not a git repository. Run 'git init' first."
    exit 1
  fi

  # Ensure directory structure
  mkdir -p "$CYCLES_DIR"

  # Check required files
  if [ ! -f "tasks/todo.md" ]; then
    log "FATAL: tasks/todo.md not found. Create your task list first."
    exit 1
  fi

  if [ ! -f "CLAUDE.md" ]; then
    log "FATAL: CLAUDE.md not found. Run your blueprint to scaffold .claude/ first."
    exit 1
  fi

  if [ ! -d ".claude" ]; then
    log "FATAL: .claude/ directory not found. Run your blueprint first."
    exit 1
  fi

  if [ ! -f "tasks/TARGET_STATE.md" ] || [ ! -s "tasks/TARGET_STATE.md" ]; then
    log "WARNING: tasks/TARGET_STATE.md is empty. Agents won't have architectural guidance."
  fi

  if [ ! -f "tasks/DECISIONS.md" ] || [ ! -s "tasks/DECISIONS.md" ]; then
    log "WARNING: tasks/DECISIONS.md is empty. Agent may pause on first architectural decision."
  fi

  # Clean any leftover temp files
  rm -f "$TEMP_PROMPT_FILE"

  log "═══════════════════════════════════════════════════════"
  log "AUTONOMOUS LOOP STARTING"
  log "═══════════════════════════════════════════════════════"
  log "Project:     ${PROJECT_DIR}"
  log "Model:       ${MODEL}"
  log "Max cycles:  ${MAX_CYCLES}"
  log "Turns:       dev=${DEVELOPER_MAX_TURNS} test=${TESTER_MAX_TURNS} review=${REVIEWER_MAX_TURNS} security=${SECURITY_MAX_TURNS}"
  log "             manager=${MANAGER_MAX_TURNS} strategist=${STRATEGIST_MAX_TURNS}"
  log "═══════════════════════════════════════════════════════"
}

# ─── Stop Check ─────────────────────────────────────────────────
check_stop() {
  if [ -f "${TASKS_DIR}/STOP" ]; then
    log "STOP file detected. Halting loop gracefully."
    log "Remove tasks/STOP and re-run to resume."
    return 0
  fi
  return 1
}

# ─── Reviewer Verdict Check ────────────────────────────────────
# The reviewer is one of two quality gates for loop termination.
# Both reviewer APPROVED and security SECURITY_APPROVED are required.
# NEEDS_FIX triggers another cycle.
check_reviewer_approved() {
  local cycle_padded=$1
  local review_file="${CYCLES_DIR}/cycle-${cycle_padded}-review.md"

  if [ ! -f "$review_file" ]; then
    log "WARNING: No review file for cycle ${cycle_padded}. Continuing to next cycle."
    return 1
  fi

  if grep -q "^## Verdict:.*APPROVED" "$review_file" 2>/dev/null; then
    log "Reviewer verdict: APPROVED."
    return 0
  fi

  log "Reviewer verdict: NEEDS_FIX. Continuing to next cycle."
  return 1
}

# ─── Security Verdict Check ───────────────────────────────────
# Security is the second quality gate. Both reviewer and security
# must approve before the loop can finalize.
check_security_approved() {
  local cycle_padded=$1
  local security_file="${CYCLES_DIR}/cycle-${cycle_padded}-security.md"

  if [ ! -f "$security_file" ]; then
    log "WARNING: No security file for cycle ${cycle_padded}. Continuing to next cycle."
    return 1
  fi

  if grep -q "^## Verdict:.*SECURITY_APPROVED" "$security_file" 2>/dev/null; then
    log "Security verdict: SECURITY_APPROVED."
    return 0
  fi

  log "Security verdict: SECURITY_BLOCKED. Continuing to next cycle."
  return 1
}

# ─── Decision Pending Check ─────────────────────────────────────
check_decisions_pending() {
  # Detect agent-written questions (lines starting with "## ") only.
  # The file is pre-populated with instructional header text at init time,
  # so [ -s ] would always trigger — we must check for actual content.
  if [ -f "${TASKS_DIR}/decisions-pending.md" ] && grep -q "^## " "${TASKS_DIR}/decisions-pending.md"; then
    log "PAUSED: Decisions pending human review."
    log "  Review tasks/decisions-pending.md"
    log "  Make decisions, clear the file, then re-run."
    return 0
  fi
  return 1
}

# ─── Dirty Git Check ───────────────────────────────────────────
check_clean_git() {
  local phase=$1
  if [ -n "$(git status --porcelain -- ':!tasks/' ':!.autonomous/' 2>/dev/null)" ]; then
    log "WARNING: Dirty git state entering ${phase} phase."
    log "  Uncommitted changes from previous phase detected."
    log "  The ${phase} agent will inherit these uncommitted files."
  fi
}

# ─── Git Tag for Rollback ──────────────────────────────────────
tag_cycle() {
  local cycle=$1
  local phase=$2
  local tag="auto/cycle-$(printf '%03d' "$cycle")-${phase}"
  git tag -f "$tag" HEAD 2>/dev/null || true
}

# ─── Run a Claude Phase ────────────────────────────────────────
run_phase() {
  local cycle=$1
  local role=$2           # developer | tester | reviewer | manager | strategist | security-reviewer
  local prompt_file=$3
  local allowed_tools=$4
  local max_turns=$5
  local cycle_padded
  cycle_padded=$(printf '%03d' "$cycle")

  local role_upper
  role_upper=$(echo "$role" | tr '[:lower:]' '[:upper:]')

  log "─── Cycle ${cycle} | ${role_upper} | max-turns=${max_turns} ───"

  # Check clean git state
  check_clean_git "$role"

  # Tag before phase starts (rollback point)
  tag_cycle "$cycle" "pre-${role}"

  # Write prompt to temp file to avoid bash variable expansion issues.
  # This is critical — prompts contain $, backticks, and special chars
  # that bash would interpret if passed as a variable.
  {
    echo "CYCLE NUMBER: ${cycle}"
    echo "CYCLE PADDED: ${cycle_padded}"
    echo ""
    cat "$prompt_file"
  } > "$TEMP_PROMPT_FILE"

  local prompt
  prompt=$(cat "$TEMP_PROMPT_FILE")

  # Run claude in headless mode
  local output_file="${CYCLES_DIR}/cycle-${cycle_padded}-${role}-output.json"

  set +e  # Don't exit on claude failure
  claude -p "$prompt" \
    --allowedTools "$allowed_tools" \
    --max-turns "$max_turns" \
    --model "$MODEL" \
    --output-format json \
    --no-session-persistence \
    > "$output_file" 2>&1
  local exit_code=$?
  set -e

  # Tag after phase completes
  tag_cycle "$cycle" "post-${role}"

  # Log result
  if [ $exit_code -eq 0 ]; then
    log "Cycle ${cycle} | ${role_upper} | completed successfully"
  else
    log "Cycle ${cycle} | ${role_upper} | exited with code ${exit_code}"
    # Exit code 1 from --max-turns means turns exhausted (not a failure)
    if [ $exit_code -eq 1 ]; then
      log "  (likely: max-turns limit reached — not an error)"
    fi
  fi

  # Extract token usage from output JSON (for fun stats)
  if [ -f "$output_file" ]; then
    local tokens
    tokens=$(python3 -c "
import json, sys
try:
    d = json.load(open('$output_file'))
    u = d.get('usage', {})
    inp = u.get('input_tokens', 0) + u.get('cache_read_input_tokens', 0) + u.get('cache_creation_input_tokens', 0)
    out = u.get('output_tokens', 0)
    print(f'{inp} in / {out} out')
except: print('unknown')
" 2>/dev/null || echo "unknown")
    log "Cycle ${cycle} | ${role_upper} | tokens: ${tokens}"
  fi

  # Clean up temp file
  rm -f "$TEMP_PROMPT_FILE"
}

# ─── Token Summary ─────────────────────────────────────────────
print_token_summary() {
  log ""
  log "─── Token Usage Summary (estimated API equivalent) ───"

  python3 -c "
import json, glob, os

total_input = 0
total_output = 0
total_cache_create = 0
total_cache_read = 0
phase_stats = []

for f in sorted(glob.glob('${CYCLES_DIR}/cycle-*-output.json')):
    try:
        d = json.load(open(f))
        u = d.get('usage', {})
        inp = u.get('input_tokens', 0)
        out = u.get('output_tokens', 0)
        cache_create = u.get('cache_creation_input_tokens', 0)
        cache_read = u.get('cache_read_input_tokens', 0)
        turns = d.get('num_turns', 0)
        name = os.path.basename(f).replace('-output.json', '')
        phase_stats.append((name, inp + cache_create + cache_read, out, turns))
        total_input += inp + cache_create + cache_read
        total_output += out
        total_cache_create += cache_create
        total_cache_read += cache_read
    except:
        pass

for name, inp, out, turns in phase_stats:
    print(f'  {name}: {inp:,} in / {out:,} out ({turns} turns)')

print(f'')
print(f'  TOTAL: {total_input:,} input / {total_output:,} output')
print(f'  Cache: {total_cache_create:,} created / {total_cache_read:,} read')
print(f'  Grand total: {total_input + total_output:,} tokens')
" 2>/dev/null | while IFS= read -r line; do log "$line"; done
}

# ─── Main Loop ──────────────────────────────────────────────────
main() {
  preflight

  local cycle=0
  local exit_reason="MAX_CYCLES_REACHED"

  while [ $cycle -lt "$MAX_CYCLES" ]; do
    cycle=$((cycle + 1))
    local cycle_padded
    cycle_padded=$(printf '%03d' "$cycle")

    log ""
    log "═══════════════════════════════════════════════════════"
    log "CYCLE ${cycle} / ${MAX_CYCLES}"
    log "═══════════════════════════════════════════════════════"

    # ── Pre-cycle checks ──
    if check_stop; then exit_reason="STOPPED_BY_USER"; break; fi

    # ── Manager: review previous cycle and reprioritize (cycle > 1 only) ──
    if [ "$cycle" -gt 1 ]; then
      run_phase "$cycle" "manager" \
        "${PROMPTS_DIR}/manager.md" \
        "$MANAGER_TOOLS" \
        "$MANAGER_MAX_TURNS"

      if check_stop; then exit_reason="STOPPED_BY_USER"; break; fi
    fi

    # ── Strategist: resolve pending decisions (only if decisions-pending.md has content) ──
    if [ -f "${TASKS_DIR}/decisions-pending.md" ] && grep -q "^## " "${TASKS_DIR}/decisions-pending.md"; then
      run_phase "$cycle" "strategist" \
        "${PROMPTS_DIR}/strategist.md" \
        "$STRATEGIST_TOOLS" \
        "$STRATEGIST_MAX_TURNS"

      if check_stop; then exit_reason="STOPPED_BY_USER"; break; fi
    fi

    # ── Phase 1: DEVELOPER ──
    run_phase "$cycle" "developer" \
      "${PROMPTS_DIR}/developer.md" \
      "$DEVELOPER_TOOLS" \
      "$DEVELOPER_MAX_TURNS"

    if check_stop; then exit_reason="STOPPED_BY_USER"; break; fi
    if check_decisions_pending; then exit_reason="DECISIONS_PENDING"; break; fi

    # ── Phase 2: TESTER ──
    run_phase "$cycle" "tester" \
      "${PROMPTS_DIR}/tester.md" \
      "$TESTER_TOOLS" \
      "$TESTER_MAX_TURNS"

    if check_stop; then exit_reason="STOPPED_BY_USER"; break; fi

    # ── Phase 3: REVIEWER (code quality gate) ──
    run_phase "$cycle" "reviewer" \
      "${PROMPTS_DIR}/reviewer.md" \
      "$REVIEWER_TOOLS" \
      "$REVIEWER_MAX_TURNS"

    if check_stop; then exit_reason="STOPPED_BY_USER"; break; fi

    # ── Phase 4: SECURITY (security quality gate) ──
    run_phase "$cycle" "security" \
      "${PROMPTS_DIR}/security.md" \
      "$SECURITY_TOOLS" \
      "$SECURITY_MAX_TURNS"

    if check_stop; then exit_reason="STOPPED_BY_USER"; break; fi

    # ── Termination: Both reviewer AND security must approve ──
    if check_reviewer_approved "$cycle_padded" && check_security_approved "$cycle_padded"; then
      log "Both quality gates passed. Finalizing."
      exit_reason="ALL_GATES_APPROVED"
      break
    fi

    log "Cycle ${cycle} complete. One or more quality gates require fixes — starting next cycle."
  done

  # ── Summary ──
  log ""
  log "═══════════════════════════════════════════════════════"
  log "AUTONOMOUS SESSION FINISHED"
  log "═══════════════════════════════════════════════════════"
  log "Cycles completed: ${cycle} / ${MAX_CYCLES}"
  log "Exit reason:      ${exit_reason}"
  log ""
  log "Git log (last 20 commits):"
  git log --oneline -20 2>/dev/null | while IFS= read -r line; do log "  $line"; done

  # Token summary
  print_token_summary

  log ""
  log "Check tasks/current-state.md for final state."
  log "Check tasks/cycles/ for per-cycle logs."
  log "═══════════════════════════════════════════════════════"
}

main "$@"
