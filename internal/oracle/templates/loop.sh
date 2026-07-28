#!/bin/bash
set -euo pipefail

# ─── Oracle Research Loop ────────────────────────────────────────
# Three-phase autonomous knowledge collection loop:
#
#   Phase 1: Research (4 cycles)
#     Each cycle: Researcher agent → Reviewer agent
#     Output: tasks/research/*.md  (raw extracted knowledge)
#
#   Phase 2: Human Review Gate
#     Loop pauses automatically after all 4 research cycles.
#     Human reviews tasks/research/, optionally annotates files,
#     then: touch tasks/SYNTHESIZE_APPROVED
#
#   Phase 3: Synthesis (single pass)
#     Synthesizer agent reads all research + gold-standard examples
#     Output: output/{stack-name}/*.md  (19 knowledge docs)
#
# Usage:
#   .autonomous/loop.sh              # Run in foreground
#   .autonomous/loop.sh &            # Run in background
#   touch tasks/STOP                 # Graceful stop from anywhere
#   rm tasks/STOP                    # Clear stop signal to resume
#   touch tasks/SYNTHESIZE_APPROVED  # Approve human gate and continue
# ─────────────────────────────────────────────────────────────────

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Unset CLAUDECODE env var — prevents "nested session" errors when launched
# from inside a Claude Code terminal. Safe for -p (headless) mode.
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

  if ! command -v claude &>/dev/null; then
    log "FATAL: 'claude' CLI not found in PATH. Install: https://code.claude.com"
    exit 1
  fi

  if [ -z "${ANTHROPIC_API_KEY:-}" ]; then
    if ! claude auth status &>/dev/null 2>&1; then
      log "FATAL: Not authenticated. Run 'claude auth login' first."
      exit 1
    fi
  fi

  mkdir -p "$RESEARCH_DIR" "$REVIEW_DIR" "$CYCLES_DIR" "$OUTPUT_DIR"

  if [ ! -f "CLAUDE.md" ]; then
    log "FATAL: CLAUDE.md not found. Run oracle research to scaffold first."
    exit 1
  fi

  # Clean any leftover temp files
  rm -f "$TEMP_PROMPT_FILE"

  log "═══════════════════════════════════════════════════════"
  log "ORACLE LOOP STARTING"
  log "═══════════════════════════════════════════════════════"
  log "Project:  ${PROJECT_DIR}"
  log "Stack:    ${STACK_NAME}"
  log "Model:    ${MODEL}"
  log "Cycles:   ${RESEARCH_CYCLES} research + 1 synthesis"
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

# ─── Git Tag for Rollback ──────────────────────────────────────
tag_cycle() {
  local cycle=$1
  local role=$2
  local tag="oracle/cycle-$(printf '%03d' "$cycle")-${role}"
  git tag -f "$tag" HEAD 2>/dev/null || true
}

# ─── Run a Claude Phase ────────────────────────────────────────
run_phase() {
  local cycle=$1
  local role=$2           # researcher | reviewer | synthesizer
  local prompt_file=$3
  local allowed_tools=$4
  local max_turns=$5
  local cycle_padded
  cycle_padded=$(printf '%03d' "$cycle")

  local role_upper
  role_upper=$(echo "$role" | tr '[:lower:]' '[:upper:]')

  log "─── Cycle ${cycle} | ${role_upper} | max-turns=${max_turns} ───"

  tag_cycle "$cycle" "pre-${role}"

  {
    echo "CYCLE NUMBER: ${cycle}"
    echo "CYCLE PADDED: ${cycle_padded}"
    echo ""
    cat "$prompt_file"
  } > "$TEMP_PROMPT_FILE"

  local prompt
  prompt=$(cat "$TEMP_PROMPT_FILE")

  local output_file="${CYCLES_DIR}/cycle-${cycle_padded}-${role}-output.json"

  set +e
  claude -p "$prompt" \
    --allowedTools "$allowed_tools" \
    --max-turns "$max_turns" \
    --model "$MODEL" \
    --output-format json \
    --no-session-persistence \
    > "$output_file" 2>&1
  local exit_code=$?
  set -e

  tag_cycle "$cycle" "post-${role}"

  if [ $exit_code -eq 0 ]; then
    log "Cycle ${cycle} | ${role_upper} | completed successfully"
  else
    log "Cycle ${cycle} | ${role_upper} | exited with code ${exit_code}"
    if [ $exit_code -eq 1 ]; then
      log "  (likely: max-turns limit reached — not an error)"
    fi
  fi

  # Extract token usage
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

  rm -f "$TEMP_PROMPT_FILE"
}

# ─── Human Review Gate ─────────────────────────────────────────
wait_for_human_approval() {
  local gate_file="${TASKS_DIR}/SYNTHESIZE_APPROVED"
  local timeout_minutes=60
  local poll_seconds=5
  local elapsed=0
  local max_elapsed=$((timeout_minutes * 60))

  log ""
  log "═══════════════════════════════════════════════════════"
  log "HUMAN REVIEW GATE"
  log "═══════════════════════════════════════════════════════"
  log "All 4 research cycles complete."
  log ""
  log "Review the research files in tasks/research/"
  log "You may annotate any file to guide the synthesizer."
  log ""
  log "When ready to synthesize, run:"
  log "  touch tasks/SYNTHESIZE_APPROVED"
  log ""
  log "Polling for approval (${timeout_minutes} min timeout)..."
  log "═══════════════════════════════════════════════════════"
  echo ""
  echo "  Waiting for tasks/SYNTHESIZE_APPROVED (Ctrl+C to abort)..."
  echo ""

  while [ ! -f "$gate_file" ]; do
    if check_stop; then
      return 1
    fi

    if [ $elapsed -ge $max_elapsed ]; then
      log "TIMEOUT: Human gate not approved within ${timeout_minutes} minutes."
      log "Re-run loop.sh after creating tasks/SYNTHESIZE_APPROVED."
      exit 1
    fi

    sleep $poll_seconds
    elapsed=$((elapsed + poll_seconds))
  done

  log "Human gate APPROVED. Proceeding to synthesis."
  return 0
}

# ─── Token Summary ─────────────────────────────────────────────
print_token_summary() {
  log ""
  log "─── Token Usage Summary ───────────────────────────────"
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

# ─── Main ───────────────────────────────────────────────────────
main() {
  preflight

  echo "RESEARCH_RUNNING" > "${TASKS_DIR}/status.txt"

  # ── Phase 1: Research Cycles ──────────────────────────────────
  for cycle in $(seq 1 "$RESEARCH_CYCLES"); do
    log ""
    log "═══════════════════════════════════════════════════════"
    log "RESEARCH CYCLE ${cycle} / ${RESEARCH_CYCLES}"
    log "═══════════════════════════════════════════════════════"
    echo "RESEARCH_CYCLE_${cycle}" > "${TASKS_DIR}/status.txt"

    if check_stop; then
      echo "STOPPED_BY_USER" > "${TASKS_DIR}/status.txt"
      exit 0
    fi

    # Researcher agent
    run_phase "$cycle" "researcher" \
      "${PROMPTS_DIR}/researcher.md" \
      "$RESEARCHER_TOOLS" \
      "$RESEARCHER_MAX_TURNS"

    if check_stop; then
      echo "STOPPED_BY_USER" > "${TASKS_DIR}/status.txt"
      exit 0
    fi

    # Reviewer agent — validates research quality
    run_phase "$cycle" "reviewer" \
      "${PROMPTS_DIR}/reviewer.md" \
      "$REVIEWER_TOOLS" \
      "$REVIEWER_MAX_TURNS"

    if check_stop; then
      echo "STOPPED_BY_USER" > "${TASKS_DIR}/status.txt"
      exit 0
    fi

    log "Research cycle ${cycle} complete."
  done

  echo "RESEARCH_COMPLETE" > "${TASKS_DIR}/status.txt"

  # ── Phase 2: Human Review Gate ────────────────────────────────
  if ! wait_for_human_approval; then
    echo "STOPPED_BY_USER" > "${TASKS_DIR}/status.txt"
    exit 0
  fi

  # ── Phase 3: Synthesis ────────────────────────────────────────
  log ""
  log "═══════════════════════════════════════════════════════"
  log "PHASE 3: SYNTHESIS"
  log "═══════════════════════════════════════════════════════"
  echo "SYNTHESIS_RUNNING" > "${TASKS_DIR}/status.txt"

  # Synthesizer runs as cycle 5 (after 4 research cycles)
  local SYNTHESIS_CYCLE=$((RESEARCH_CYCLES + 1))
  run_phase "$SYNTHESIS_CYCLE" "synthesizer" \
    "${PROMPTS_DIR}/synthesizer.md" \
    "$SYNTHESIZER_TOOLS" \
    "$SYNTHESIZER_MAX_TURNS"

  echo "SYNTHESIS_COMPLETE" > "${TASKS_DIR}/status.txt"

  # ── Summary ───────────────────────────────────────────────────
  log ""
  log "═══════════════════════════════════════════════════════"
  log "ORACLE LOOP COMPLETE"
  log "═══════════════════════════════════════════════════════"
  log "Stack:         ${STACK_NAME}"
  log "Research docs: $(find "${RESEARCH_DIR}" -name '*.md' 2>/dev/null | wc -l | tr -d ' ') files"
  log "Output docs:   $(find "${OUTPUT_DIR}/${STACK_NAME}" -name '*.md' 2>/dev/null | wc -l | tr -d ' ') files"
  log ""
  log "Output location: output/${STACK_NAME}/"
  log "Copy to .claude/knowledge/ when satisfied:"
  log "  cp -r output/${STACK_NAME}/ \$(git rev-parse --show-toplevel)/.claude/knowledge/${STACK_NAME}/"

  print_token_summary

  log "═══════════════════════════════════════════════════════"
}

main "$@"
