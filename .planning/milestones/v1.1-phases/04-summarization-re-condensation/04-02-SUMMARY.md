---
phase: 04-summarization-re-condensation
plan: 02
subsystem: cmd
tags: [go, summarization, history, tdd, conversation-loop]

# Dependency graph
requires:
  - phase: 04-01
    provides: history.Replace, ollama.Chat (non-streaming)
  - phase: 03-conversation-loop
    provides: runConversationLoop structure, test patterns (fakeStream, replaceStdin)
provides:
  - summarize() helper in cmd/helpers.go — compresses history via ollama.Chat and hist.Replace
  - runConversationLoop extended with summarizePrompt and recondensePrompt parameters
  - 8 command-specific summarization prompt constants across 4 command files
affects:
  - All 4 query commands (plan, lookup, starter, pattern) now pass prompts to runConversationLoop

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "TDD RED-GREEN: failing tests committed first, then implementation"
    - "Re-condensation detection via strings.HasPrefix on system message content"
    - "Summarize guard: len(msgs) < 5 returns nil (nothing meaningful to compress)"

key-files:
  created: []
  modified:
    - cmd/helpers.go
    - cmd/helpers_test.go
    - cmd/plan.go
    - cmd/lookup.go
    - cmd/starter.go
    - cmd/pattern.go

key-decisions:
  - "summarize() calls ollama.Chat directly (not through streamFn) — summarization is a non-streaming internal operation"
  - "len(msgs) < 5 guard prevents compression when only system+user+assistant exist (nothing to compress separately from final pair)"
  - "Re-condensation detected by 'Summary of previous conversation:' prefix on any system message after index 0"

requirements-completed:
  - SUMM-01
  - SUMM-02
  - SUMM-03

# Metrics
duration: 2min
completed: 2026-04-08
---

# Phase 04 Plan 02: Summarization Wired into Conversation Loop Summary

**Command-specific summarization prompts and summarize() helper integrated into runConversationLoop — all 4 query commands now automatically condense history when token threshold is exceeded**

## Performance

- **Duration:** ~2 min
- **Started:** 2026-04-08
- **Completed:** 2026-04-08
- **Tasks:** 2
- **Files modified:** 6

## Accomplishments

- Added 8 prompt constants (summerize + recondense per command) to plan.go, lookup.go, starter.go, pattern.go
- Extended `runConversationLoop` signature with `summarizePrompt` and `recondensePrompt` parameters
- Added `summarize()` helper function in cmd/helpers.go that:
  - Detects re-condensation by looking for an existing "Summary of previous conversation:" system message
  - Builds candidate messages (everything between system[0] and final user+assistant pair)
  - Calls ollama.Chat for summarization (non-streaming)
  - Replaces history with: system[0] + new summary system message + final user+assistant pair
  - Guards against compressing when msgs < 5 (nothing meaningful to compress)
- Prints "[Condensing history...]" to stderr before summarization
- Updated all 4 command call sites to pass command-specific prompt pairs
- Updated all existing tests (5 calls) to use updated signature with empty string args

## Task Commits

Each task committed atomically:

1. **Task 1: Add command-specific summarization prompt constants** - `f7629fa` (feat)
2. **Task 2 RED: Failing tests for summarization behavior** - `9ca90d0` (test)
3. **Task 2 GREEN: Wire summarization into runConversationLoop** - `e0e4136` (feat)

## Files Created/Modified

- `cmd/helpers.go` - Extended runConversationLoop signature; added summarize() helper; added ollama import
- `cmd/helpers_test.go` - Updated all 5 existing runConversationLoop calls; added TestRunConversationLoop_Summarization
- `cmd/plan.go` - Added planSummarizePrompt, planRecondensePrompt; updated call site
- `cmd/lookup.go` - Added lookupSummarizePrompt, lookupRecondensePrompt; updated call site
- `cmd/starter.go` - Added starterSummarizePrompt, starterRecondensePrompt; updated call site
- `cmd/pattern.go` - Added patternSummarizePrompt, patternRecondensePrompt; updated call site

## Decisions Made

- summarize() calls ollama.Chat directly rather than accepting a chatFn parameter — keeps the function signature minimal; summarization is an internal operation not needing test injection at this level
- len(msgs) < 5 guard: with only [system, user, assistant] (3 msgs) there is no content to compress separately from the final pair; threshold is 5 (system + at least one prior exchange + final pair)

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

None.

## Known Stubs

None. All summarization logic is fully wired. ollama.Chat is tested against the real network in internal/ollama/client_test.go (stubbed via httptest); the summarize() path that calls ollama.Chat is only exercised when ExceedsLimit() fires (requires real Ollama server).

---
*Phase: 04-summarization-re-condensation*
*Completed: 2026-04-08*
