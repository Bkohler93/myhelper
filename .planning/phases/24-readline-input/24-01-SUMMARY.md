---
phase: 24-readline-input
plan: 01
subsystem: ui
tags: [readline, chzyer, terminal, tty, line-editing, history, continuation]

# Dependency graph
requires: []
provides:
  - "chzyer/readline integrated into runConversationLoop TTY path"
  - "joinContinuationLines pure helper for backslash continuation"
  - "readMultiLine helper with \\ continuation and prompt switching"
  - "TTY gate: readline path for real TTY, bufio path for pipes/tests"
  - "TestJoinContinuationLines unit test (three sub-cases)"
affects: [phase-25-markdown-rendering, any future cmd/helpers.go work]

# Tech tracking
tech-stack:
  added: ["github.com/chzyer/readline v1.5.1 (promoted from indirect to direct)"]
  patterns:
    - "TTY gate pattern: readline.IsTerminal(int(os.Stdin.Fd())) guards readline vs bufio path"
    - "DisableAutoSaveHistory + manual rl.SaveHistory for multi-line history entries"
    - "Continuation accumulation: strip backslash before appending, pure join helper for testability"

key-files:
  created: []
  modified:
    - cmd/helpers.go
    - cmd/helpers_test.go
    - go.mod
    - go.sum

key-decisions:
  - "TTY check uses os.Stdin.Fd() not stdinReader seam — seam is io.Reader, has no Fd(); keeps test path clean"
  - "DisableAutoSaveHistory: true + rl.SaveHistory(joinedInput) — prevents intermediate continuation lines from polluting history"
  - "readline.ErrInterrupt and io.EOF both return nil from runConversationLoop — clean exit semantics match bufio EOF"
  - "sigCh handler kept for bufio path only — readline intercepts Ctrl+C at raw-mode level before POSIX signal"
  - "joinContinuationLines extracted as package-level pure helper — enables unit testing without a TTY"
  - "fmt.Fprint(os.Stderr, \"> \") removed from bufio path — non-interactive path needs no prompt"

patterns-established:
  - "TTY gate pattern: readline path before bufio path in runConversationLoop; both paths return nil on clean exit"
  - "Pure helper extraction for testability: joinContinuationLines separates join logic from TTY-dependent readline calls"

requirements-completed: [INPUT-01, INPUT-02, INPUT-03, INPUT-04]

# Metrics
duration: 4min
completed: 2026-04-25
---

# Phase 24 Plan 01: Readline Input Summary

**chzyer/readline TTY gate in runConversationLoop: arrow-key editing, in-session history recall, and \\ continuation multi-line input with automatic bufio fallback for non-TTY callers**

## Performance

- **Duration:** ~4 min
- **Started:** 2026-04-25T02:16:44Z
- **Completed:** 2026-04-25T02:19:56Z
- **Tasks:** 2 of 2 automated tasks complete (Task 3 is checkpoint:human-verify)
- **Files modified:** 4 (cmd/helpers.go, cmd/helpers_test.go, go.mod, go.sum)

## Accomplishments
- Integrated chzyer/readline v1.5.1 as a direct dependency with TTY-gated readline path in runConversationLoop
- Implemented joinContinuationLines (pure helper) and readMultiLine (with \\ continuation and "... " prompt) in cmd/helpers.go
- Added TestJoinContinuationLines with three sub-cases; all pass without requiring a TTY
- Promoted chzyer/readline from indirect to direct in go.mod via go mod tidy

## Task Commits

1. **Task 1 RED: TestJoinContinuationLines (failing)** — `b1b7ad0` (test)
2. **Task 1 GREEN: readline TTY gate + helpers** — `9fce442` (feat)
3. **Task 2: TestJoinContinuationLines** — committed as part of RED phase (b1b7ad0); confirmed passing after GREEN

## Files Created/Modified
- `cmd/helpers.go` — added joinContinuationLines, readMultiLine, readline TTY gate in runConversationLoop; removed fmt.Fprint prompt from bufio path
- `cmd/helpers_test.go` — added TestJoinContinuationLines with two-line, single-line, and three-line sub-cases
- `go.mod` — promoted github.com/chzyer/readline v1.5.1 from indirect to direct
- `go.sum` — updated by go mod tidy

## Decisions Made
- `os.Stdin.Fd()` used for TTY check (not `stdinReader`) so tests using pipe replacement continue to exercise the bufio path automatically
- `DisableAutoSaveHistory: true` with manual `rl.SaveHistory(joinedInput)` ensures only the complete multi-line string appears in up-arrow history
- `defer rl.Close()` placed immediately after `NewEx` succeeds for guaranteed terminal restoration
- `sigCh` handler preserved in the bufio path only; readline path relies on `readline.ErrInterrupt` for Ctrl+C

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None. `go mod tidy` was required after writing the import to promote the dependency; this was expected per the plan's instructions.

## Known Stubs

None — all readline functionality is fully wired. The TTY path is live; the test path falls through to bufio unchanged.

## Threat Flags

No new threat surface beyond what the plan's threat model covers. No new network endpoints, auth paths, or file access patterns introduced.

## Self-Check

Files exist:
- cmd/helpers.go — modified (contains readline.IsTerminal, readMultiLine, joinContinuationLines)
- cmd/helpers_test.go — modified (contains TestJoinContinuationLines)
- go.mod — chzyer/readline v1.5.1 without // indirect

Commits exist:
- b1b7ad0 — test(24-01): add failing TestJoinContinuationLines (RED)
- 9fce442 — feat(24-01): integrate chzyer/readline TTY gate in runConversationLoop (GREEN)

## Self-Check: PASSED

## User Setup Required

None - no external service configuration required. The readline library is pure Go with no system dependencies beyond what Go's stdlib provides.

## Next Phase Readiness

- readline input is complete and ready for manual smoke test (Task 3 checkpoint)
- Phase 25 (markdown rendering) can proceed after the checkpoint is approved
- No blockers

---
*Phase: 24-readline-input*
*Completed: 2026-04-25*
