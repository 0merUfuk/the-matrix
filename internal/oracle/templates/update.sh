#!/bin/bash
set -euo pipefail

# ─── Oracle Update Loop ────────────────────────────────────────
# Incremental refresh: re-research and re-synthesize specific stale topics.
# Skips the human gate — topics were confirmed by the user in oracle update.
#
# Usage: Called automatically by oracle update. Do not run directly.
# ─────────────────────────────────────────────────────────────────

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

source "${SCRIPT_DIR}/config.sh"

# Override research cycles with update-specific default
RESEARCH_CYCLES="${UPDATE_RESEARCH_CYCLES:-2}"

# ─── Read Topics to Update ────────────────────────────────────
TOPICS_FILE="${PROJECT_DIR}/topics.txt"
if [ ! -f "$TOPICS_FILE" ]; then
    echo "FATAL: topics.txt not found in ${PROJECT_DIR}" >&2
    exit 1
fi
TOPICS_TO_UPDATE=$(cat "$TOPICS_FILE" | tr '\n' ',' | sed 's/,$//')

if [ -z "$TOPICS_TO_UPDATE" ]; then
    echo "FATAL: topics.txt is empty" >&2
    exit 1
fi

# ─── Logging ────────────────────────────────────────────────────
LOG_FILE="${TASKS_DIR}/loop.log"
TEMP_PROMPT_FILE="${TASKS_DIR}/.current-prompt.tmp"
UPDATE_CONTEXT_FILE="${TASKS_DIR}/update-context.txt"

log() {
  local msg="[$(date '+%Y-%m-%d %H:%M:%S')] $1"
  echo "$msg" | tee -a "$LOG_FILE"
}

# ─── Write Update Context ───────────────────────────────────────
# This context is prepended to researcher and synthesizer prompts
# to focus them on only the specified topics.
write_update_context() {
  cat > "$UPDATE_CONTEXT_FILE" <<UPDATEEOF
TOPICS BEING UPDATED: ${TOPICS_TO_UPDATE}

IMPORTANT: This is an incremental update run, NOT a full research loop.
Focus your research and synthesis ONLY on the above topics.
Do not produce output for other topics.
For research cycles, investigate ONLY the listed topics regardless of the cycle number.
UPDATEEOF
}

# ─── Pre-Flight Checks ─────────────────────────────────────────
preflight() {
  cd "$PROJECT_DIR"

  check_auth

  mkdir -p "$RESEARCH_DIR" "$REVIEW_DIR" "$CYCLES_DIR" "$OUTPUT_DIR"

  if [ ! -f "$CONTEXT_FILE" ]; then
    log "FATAL: ${CONTEXT_FILE} not found. Ensure the update workspace is properly scaffolded."
    exit 1
  fi

  # Write the update context file for prompt prepending
  write_update_context

  # Clean any leftover temp files
  rm -f "$TEMP_PROMPT_FILE"

  log "═══════════════════════════════════════════════════════"
  log "ORACLE UPDATE LOOP STARTING"
  log "═══════════════════════════════════════════════════════"
  log "Project:  ${PROJECT_DIR}"
  log "Stack:    ${STACK_NAME}"
  log "Model:    ${MODEL}"
  log "Cycles:   ${RESEARCH_CYCLES} research + 1 synthesis"
  log "Topics:   ${TOPICS_TO_UPDATE}"
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

# ─── Run an Agent Phase ────────────────────────────────────────
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

  # Build the temp prompt: update context + cycle info + original prompt
  {
    cat "$UPDATE_CONTEXT_FILE"
    echo ""
    echo "CYCLE NUMBER: ${cycle}"
    echo "CYCLE PADDED: ${cycle_padded}"
    echo ""
    cat "$prompt_file"
  } > "$TEMP_PROMPT_FILE"

  local prompt
  prompt=$(cat "$TEMP_PROMPT_FILE")

  local output_file="${CYCLES_DIR}/cycle-${cycle_padded}-${role}-output.json"

  set +e
  run_agent "$prompt" "$allowed_tools" "$max_turns" "$output_file"
  local exit_code=$?
  set -e

  set +e
  classify_exit "$exit_code"
  local classify_result=$?
  set -e

  if [ $classify_result -eq 0 ]; then
    log "Cycle ${cycle} | ${role_upper} | completed successfully"
  elif [ $classify_result -eq 1 ]; then
    log "Cycle ${cycle} | ${role_upper} | exited with code ${exit_code}"
    log "  (max-turns limit reached - not an error)"
  else
    log "Cycle ${cycle} | ${role_upper} | FAILED with code ${exit_code}"
    rm -f "$TEMP_PROMPT_FILE"
    return 1
  fi

  # Extract token usage
  if [ -f "$output_file" ]; then
    local tokens
    tokens=$(python3 -c "
import json, sys
try:
    raw = open('$output_file').read()
    try:
        d = json.loads(raw)
    except json.JSONDecodeError:
        events = []
        for line in raw.splitlines():
            try:
                events.append(json.loads(line))
            except json.JSONDecodeError:
                pass
        d = next(event for event in reversed(events) if event.get('type') == 'turn.completed')
    u = d.get('usage', {})
    cache_read = u.get('cache_read_input_tokens', u.get('cached_input_tokens', 0))
    cache_create = u.get('cache_creation_input_tokens', u.get('cache_write_input_tokens', 0))
    inp = u.get('input_tokens', 0) + cache_read + cache_create
    out = u.get('output_tokens', 0)
    print(f'{inp} in / {out} out')
except: print('unknown')
" 2>/dev/null || echo "unknown")
    log "Cycle ${cycle} | ${role_upper} | tokens: ${tokens}"
  fi

  rm -f "$TEMP_PROMPT_FILE"
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

def load_result(path):
    raw = open(path).read()
    try:
        d = json.loads(raw)
        return d, d.get('num_turns', 0)
    except json.JSONDecodeError:
        events = []
        for line in raw.splitlines():
            try:
                events.append(json.loads(line))
            except json.JSONDecodeError:
                pass
        completed = [event for event in events if event.get('type') == 'turn.completed']
        if not completed:
            raise ValueError('no completed runtime turn')
        return completed[-1], len(completed)

for f in sorted(glob.glob('${CYCLES_DIR}/cycle-*-output.json')):
    try:
        d, turns = load_result(f)
        u = d.get('usage', {})
        inp = u.get('input_tokens', 0)
        out = u.get('output_tokens', 0)
        cache_create = u.get('cache_creation_input_tokens', u.get('cache_write_input_tokens', 0))
        cache_read = u.get('cache_read_input_tokens', u.get('cached_input_tokens', 0))
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

  echo "UPDATE_RUNNING" > "${TASKS_DIR}/status.txt"

  # ── Phase 1: Research Cycles (focused on stale topics) ────────
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
    if ! run_phase "$cycle" "researcher" \
      "${PROMPTS_DIR}/researcher.md" \
      "$RESEARCHER_TOOLS" \
      "$RESEARCHER_MAX_TURNS"; then
      echo "RUNTIME_ERROR" > "${TASKS_DIR}/status.txt"
      exit 1
    fi

    if check_stop; then
      echo "STOPPED_BY_USER" > "${TASKS_DIR}/status.txt"
      exit 0
    fi

    # Reviewer agent — validates research quality
    if ! run_phase "$cycle" "reviewer" \
      "${PROMPTS_DIR}/reviewer.md" \
      "$REVIEWER_TOOLS" \
      "$REVIEWER_MAX_TURNS"; then
      echo "RUNTIME_ERROR" > "${TASKS_DIR}/status.txt"
      exit 1
    fi

    if check_stop; then
      echo "STOPPED_BY_USER" > "${TASKS_DIR}/status.txt"
      exit 0
    fi

    log "Research cycle ${cycle} complete."
  done

  # ── Phase 2: Synthesis (no human gate) ──────────────────────
  log ""
  log "═══════════════════════════════════════════════════════"
  log "PHASE 2: SYNTHESIS (no human gate) (topics: ${TOPICS_TO_UPDATE})"
  log "═══════════════════════════════════════════════════════"
  echo "SYNTHESIS_RUNNING" > "${TASKS_DIR}/status.txt"

  # Synthesizer runs as next cycle after research
  local SYNTHESIS_CYCLE=$((RESEARCH_CYCLES + 1))
  if ! run_phase "$SYNTHESIS_CYCLE" "synthesizer" \
    "${PROMPTS_DIR}/synthesizer.md" \
    "$SYNTHESIZER_TOOLS" \
    "$SYNTHESIZER_MAX_TURNS"; then
    echo "RUNTIME_ERROR" > "${TASKS_DIR}/status.txt"
    exit 1
  fi

  echo "SYNTHESIS_COMPLETE" > "${TASKS_DIR}/status.txt"

  # ── Summary ─────────────────────────────────────────────────
  log ""
  log "═══════════════════════════════════════════════════════"
  log "ORACLE UPDATE LOOP COMPLETE"
  log "═══════════════════════════════════════════════════════"
  log "Stack:         ${STACK_NAME}"
  log "Topics:        ${TOPICS_TO_UPDATE}"
  log "Research docs: $(find "${RESEARCH_DIR}" -name '*.md' 2>/dev/null | wc -l | tr -d ' ') files"
  log "Output docs:   $(find "${OUTPUT_DIR}/${STACK_NAME}" -name '*.md' 2>/dev/null | wc -l | tr -d ' ') files"
  log ""
  log "Output location: output/${STACK_NAME}/"

  print_token_summary

  log "═══════════════════════════════════════════════════════"
}

main "$@"
