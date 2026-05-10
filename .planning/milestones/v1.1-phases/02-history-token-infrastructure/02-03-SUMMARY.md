---
phase: 02-history-token-infrastructure
plan: "03"
subsystem: ollama
tags: [ollama, api-chat, streaming, history, messages, commands]

requires:
  - "02-01: Config.TokenThreshold"
  - "02-02: history.Message type"
provides:
  - "ollama.StreamChat(cfg, []history.Message) (string, error)"
  - "cmd.buildSystemMessage(projectContext, systemPrompt) string"
  - "All 4 query commands wired to /api/chat via StreamChat"
affects:
  - "03-conversation-loop"

tech-stack:
  added: []
  patterns:
    - "Two-element messages slice: system message (buildSystemMessage output) + user message"
    - "StreamChat returns full accumulated response text for history appending by caller"
    - "chatURL() helper normalizes endpoint to /api/chat regardless of http:// prefix"

key-files:
  created: []
  modified:
    - internal/ollama/client.go
    - cmd/helpers.go
    - cmd/plan.go
    - cmd/lookup.go
    - cmd/starter.go
    - cmd/pattern.go

key-decisions:
  - "StreamChat returns (string, error) not just error — caller needs response text to append to history in Phase 3"
  - "buildSystemMessage omits trailing double-newline after systemPrompt — /api/chat handles message separation by structure"

requirements-completed: [HIST-03]

duration: 6min
completed: 2026-04-07
---

# Phase 02 Plan 03: Replace StreamPrompt with StreamChat Summary

**Replaced /api/generate client (StreamPrompt) with /api/chat client (StreamChat); all 4 query commands updated to use two-element messages slice (system + user) with history.Message type**

## Performance

- **Duration:** ~6 min
- **Started:** 2026-04-07T22:36:18Z
- **Completed:** 2026-04-07
- **Tasks:** 2
- **Files modified:** 6

## Accomplishments

- Rewrote internal/ollama/client.go to target /api/chat: removed generateRequest, generateResponse, StreamPrompt, buildURL; added chatRequest, chatMessage, chatResponse types, StreamChat function, chatURL helper
- StreamChat accepts []history.Message and returns full accumulated response text (string, error) — ready for Phase 3 history appending
- Replaced buildPrompt() in helpers.go with buildSystemMessage() — drops the user input parameter since user content is now a separate message in the slice
- Updated all 4 query commands (plan, lookup, starter, pattern) to construct []history.Message{system, user} and call ollama.StreamChat
- go build ./..., go vet ./..., go test ./... all pass clean

## Task Commits

Each task was committed atomically:

1. **Task 1: Replace StreamPrompt with StreamChat** - `32b5bec` (feat)
2. **Task 2: Update helpers.go and all 4 command files** - `ae2313f` (feat)

## Files Created/Modified

- `internal/ollama/client.go` - Full rewrite: /api/chat types, StreamChat function, chatURL helper; StreamPrompt and /api/generate removed
- `cmd/helpers.go` - Replaced buildPrompt() with buildSystemMessage(); readInteractive() and resolveInput() unchanged
- `cmd/plan.go` - runPlan uses []history.Message + ollama.StreamChat; history import added
- `cmd/lookup.go` - runLookup uses []history.Message + ollama.StreamChat; history import added
- `cmd/starter.go` - runStarter uses []history.Message + ollama.StreamChat; history import added
- `cmd/pattern.go` - runPattern uses []history.Message + ollama.StreamChat; history import added

## Decisions Made

- StreamChat returns (string, error) rather than just error: Phase 3's conversation loop needs the response text to append to the History as an assistant message. Returning it here avoids a second call or buffering at the command layer.
- buildSystemMessage drops the userInput parameter from buildPrompt: in the /api/chat model the user content lives in its own message struct, not concatenated into the system message.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## Known Stubs

None - all 4 commands fully wired to StreamChat with correct message structure.

## Next Phase Readiness

- StreamChat is the stable interface Phase 3 (conversation loop) will call: pass a growing []history.Message, receive response text, append to History
- buildSystemMessage() is available for Phase 3 to assemble the system message content
- All commands produce streaming output identical to v1.0 behavior for single-turn use

## Self-Check: PASSED

- internal/ollama/client.go: FOUND, contains StreamChat, no StreamPrompt
- cmd/helpers.go: FOUND, contains buildSystemMessage, no buildPrompt
- cmd/plan.go: FOUND, contains ollama.StreamChat
- cmd/lookup.go: FOUND, contains ollama.StreamChat
- cmd/starter.go: FOUND, contains ollama.StreamChat
- cmd/pattern.go: FOUND, contains ollama.StreamChat
- Commit 32b5bec: FOUND
- Commit ae2313f: FOUND
- Note: .planning/ is gitignored (commit_docs=false) — SUMMARY.md not committed to git

---
*Phase: 02-history-token-infrastructure*
*Completed: 2026-04-07*
