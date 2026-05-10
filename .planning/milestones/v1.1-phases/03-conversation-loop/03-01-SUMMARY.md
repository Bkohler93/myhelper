---
phase: 03-conversation-loop
plan: "01"
subsystem: cmd
tags: [conversation-loop, tdd, helpers, signal-handling]
dependency_graph:
  requires: [02-history-token-infrastructure, 02-stream-chat-integration]
  provides: [runConversationLoop]
  affects: [cmd/helpers.go, cmd/helpers_test.go]
tech_stack:
  added: [os/signal, syscall, io (stdinReader injection)]
  patterns: [injected-streamFn, package-var-reader-swap-for-tests]
key_files:
  created: [cmd/helpers_test.go]
  modified: [cmd/helpers.go]
decisions:
  - stdinReader package-level var enables test stdin injection without refactoring cmd files
  - streamFn injected as parameter so tests never call real ollama.StreamChat
  - SIGINT handled inside runConversationLoop (not cobra root) so signal lifetime matches loop lifetime
metrics:
  duration_minutes: 5
  completed_date: "2026-04-07"
  tasks_completed: 1
  tasks_total: 1
  files_changed: 2
---

# Phase 03 Plan 01: Implement runConversationLoop Summary

**One-liner:** Multi-turn conversation loop with SIGINT handling, empty-reprompt, quit exit, and injected streamFn for full testability.

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 (RED) | Failing tests for runConversationLoop | ba0c756 | cmd/helpers_test.go |
| 1 (GREEN) | Implement runConversationLoop | 7253c69 | cmd/helpers.go |

## Decisions Made

- **stdinReader package-level var:** Enables test stdin injection via `io.Pipe()` without modifying function signatures or using `os.Stdin` directly in the loop. Production code sets it to `os.Stdin` at package init.
- **streamFn injected as parameter:** Tests pass a fake stream function so the real `ollama.StreamChat` is never called in unit tests. This matches the plan's function signature.
- **SIGINT handler inside runConversationLoop:** Signal lifetime matches the loop lifetime. `defer signal.Stop(sigCh)` ensures cleanup. Goroutine scans stdin so `select` can unblock on SIGINT even while `bufio.Scanner.Scan()` blocks.

## Deviations from Plan

None — plan executed exactly as written.

## Known Stubs

None — `runConversationLoop` is fully wired. The 4 query commands do not yet call it (that is Plan 02 scope per the plan's success criteria).

## Self-Check

- [x] `cmd/helpers.go` contains `func runConversationLoop(`
- [x] `cmd/helpers.go` contains `signal.Notify(sigCh, syscall.SIGINT)`
- [x] `cmd/helpers.go` contains `fmt.Fprint(os.Stderr, "> ")`
- [x] `cmd/helpers_test.go` contains `TestRunConversationLoop`
- [x] `go test ./cmd/ -run TestRunConversationLoop` exits 0 (all 5 subtests pass)
- [x] `go build ./...` exits 0
