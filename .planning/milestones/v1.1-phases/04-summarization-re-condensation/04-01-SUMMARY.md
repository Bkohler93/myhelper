---
phase: 04-summarization-re-condensation
plan: 01
subsystem: api
tags: [go, ollama, history, tdd, testing]

# Dependency graph
requires:
  - phase: 02-history-token-infrastructure
    provides: History struct, Message type, Add/Messages methods
  - phase: 03-conversation-loop
    provides: runConversationLoop, StreamChat usage pattern
provides:
  - history.Replace method — replaces internal message slice with copy of provided slice
  - ollama.Chat function — non-streaming POST to /api/chat returning full response string
affects:
  - 04-02 (summarization logic will call hist.Replace and ollama.Chat)

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "TDD RED-GREEN cycle for each infrastructure primitive"
    - "Non-streaming Ollama client reuses chatRequest/chatResponse types from StreamChat"
    - "Replace uses make+copy for copy-safe internal slice replacement"

key-files:
  created:
    - internal/ollama/client_test.go
  modified:
    - internal/history/history.go
    - internal/history/history_test.go
    - internal/ollama/client.go

key-decisions:
  - "Chat uses Stream: false in chatRequest — single JSON response rather than NDJSON stream"
  - "Replace uses make+copy pattern matching the same pattern as Messages() — consistent copy semantics"
  - "Chat decodes directly into chatResponse struct — same type as StreamChat, no new types needed"

patterns-established:
  - "Non-streaming Chat: POST with Stream:false, decode single chatResponse, return Message.Content"
  - "Replace: make([]Message, len) + copy — caller controls system message preservation"

requirements-completed:
  - SUMM-01
  - SUMM-02

# Metrics
duration: 10min
completed: 2026-04-07
---

# Phase 04 Plan 01: Summarization Infrastructure Summary

**history.Replace and ollama.Chat(non-streaming) added — the two primitives that runConversationLoop needs to summarize and replace conversation history**

## Performance

- **Duration:** ~10 min
- **Started:** 2026-04-07
- **Completed:** 2026-04-07
- **Tasks:** 2
- **Files modified:** 4

## Accomplishments

- Added `Replace(messages []Message)` to `History` struct — replaces internal slice with copy, enabling summarization to swap full history
- Added `Chat(cfg, messages) (string, error)` to ollama package — non-streaming POST to /api/chat, returns full response content without writing to stdout
- 8 new tests total (4 per function), all green, no pre-existing tests broken

## Task Commits

Each task was committed atomically:

1. **Task 1: Add history.Replace and tests (RED-GREEN)** - `9a4bfed` (feat)
2. **Task 2: Add ollama.Chat (non-streaming) and tests (RED-GREEN)** - `2ac1395` (feat)

**Plan metadata:** (docs commit below)

_Note: Both tasks followed strict TDD — RED (compile failure) verified before GREEN (implementation)._

## Files Created/Modified

- `internal/history/history.go` - Added Replace method after Messages()
- `internal/history/history_test.go` - Added TestHistory_Replace with 4 sub-tests
- `internal/ollama/client.go` - Added Chat function after StreamChat
- `internal/ollama/client_test.go` - Created with TestChat covering 4 sub-tests

## Decisions Made

- Chat reuses existing `chatRequest` and `chatResponse` types from StreamChat — no new types needed, Stream field set to false
- Replace uses `make([]Message, len(messages)) + copy(...)` — same copy-safe pattern as Messages(), consistent across the package

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- `history.Replace` and `ollama.Chat` are available for Phase 04 Plan 02 (summarization logic)
- `runConversationLoop` in Plan 02 can now call `ollama.Chat(cfg, hist.Messages())` to get summary text and `hist.Replace(newMessages)` to install it
- No blockers

---
*Phase: 04-summarization-re-condensation*
*Completed: 2026-04-07*
