---
phase: 17-chat-entry-point
plan: "01"
subsystem: cmd
tags: [chat, repl, one-shot, summarization, cobra]
dependency_graph:
  requires: []
  provides: [CHAT-01, CHAT-02, CHAT-03, CHAT-04, CHAT-05, CHAT-06]
  affects: [cmd/root.go, cmd/helpers.go]
tech_stack:
  added: []
  patterns: [no-system-prompt history, cobra MaximumNArgs, TDD Wave 0]
key_files:
  created:
    - cmd/chat_test.go
  modified:
    - cmd/helpers.go
    - cmd/helpers_test.go
    - cmd/root.go
decisions:
  - "No system prompt in history: history.New called with nil — history starts empty, only user/assistant messages accumulate (CHAT-04)"
  - "summarize guard changed from len(msgs)<5 to len(msgs)<4 — no system message at index 0 to account for"
  - "candidates slice starts at msgs[0] not msgs[1] — entire history before final pair is eligible for compression"
  - "One-shot mode uses cobra.MaximumNArgs(1) — single positional arg streams response and exits (CHAT-02)"
metrics:
  duration: "2m17s"
  completed_date: "2026-04-11"
  tasks_completed: 3
  files_modified: 4
requirements:
  - CHAT-01
  - CHAT-02
  - CHAT-03
  - CHAT-04
  - CHAT-05
  - CHAT-06
---

# Phase 17 Plan 01: Chat Entry Point Summary

**One-liner:** Wire `myhelper` as a stateful REPL and one-shot chat interface backed by local Ollama with no-system-prompt history and rewritten summarization.

## What Was Built

The binary now has two modes triggered from `rootCmd`:

1. **One-shot mode** (`myhelper "question"`) — `hist.Add("user", args[0])` then `initiateConversation` streams the response and exits (CHAT-02).
2. **REPL mode** (`myhelper`) — `runConversationLoop` reads from stdin in a multi-turn loop until "quit", EOF, or SIGINT (CHAT-01).

The `summarize` function was rewritten to operate on history with no system prompt at index 0:
- Guard changed from `len(msgs) < 5` to `len(msgs) < 4`
- Candidate slice starts at `msgs[0]` (not `msgs[1]`)
- Re-condensation scan covers all messages (not `msgs[1:]`)
- Output is 3 messages `[system(summary), last_user, last_assistant]` — no preserved original system message

The `summarizePrompt` and `recondensePrompt` constants were added as package-level constants in `helpers.go` and passed through the call chain.

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | Write failing test stubs (Wave 0 RED) | b00d032 | cmd/chat_test.go, cmd/helpers.go |
| 2 | Rewrite summarize for no-system-prompt history | 534206a | cmd/helpers.go, cmd/helpers_test.go |
| 3 | Wire root.go as chat entry point | d4e95c2 | cmd/root.go |

## Verification Results

- `go test ./...` — all packages pass, no regressions
- `go build ./...` — exits 0
- `grep -c "^func Test" cmd/chat_test.go` — returns 4
- `TestRootCmd_OneShot`, `TestRootCmd_REPL`, `TestRootCmd_NoSystemPrompt`, `TestSummarize_NoSystemPrompt` — all pass
- `cobra.MaximumNArgs(1)` in root.go — confirmed
- `history.New(cfg.TokenThreshold, nil)` in root.go — confirmed
- `if len(msgs) < 4` in helpers.go — confirmed
- `candidates := msgs[0 : len(msgs)-2]` in helpers.go — confirmed

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing Critical] Added summarizePrompt/recondensePrompt constants in Task 1**
- **Found during:** Task 1 (TDD RED phase)
- **Issue:** `cmd/chat_test.go` referenced `summarizePrompt` and `recondensePrompt` constants that didn't exist yet, preventing compilation. The plan specified adding constants in Task 2, but tests needed them in Task 1 to compile.
- **Fix:** Added both constants to `helpers.go` as part of Task 1 so tests could compile and achieve proper RED state.
- **Files modified:** cmd/helpers.go
- **Commit:** b00d032

## Known Stubs

None — all behavior is fully wired.

## Threat Flags

None — no new network endpoints, auth paths, or schema changes introduced beyond what the threat model already covers (stdin→hist.Add and positional arg→hist.Add boundaries accepted per T-17-01 and T-17-02).

## Self-Check: PASSED

- `cmd/chat_test.go` exists: FOUND
- `cmd/helpers.go` exists: FOUND
- `cmd/root.go` exists: FOUND
- Commit b00d032: FOUND
- Commit 534206a: FOUND
- Commit d4e95c2: FOUND
