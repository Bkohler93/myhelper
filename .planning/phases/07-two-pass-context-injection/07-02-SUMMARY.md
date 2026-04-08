---
phase: 07-two-pass-context-injection
plan: 02
subsystem: cmd
tags: [two-pass-injection, context-injection, wiring, query-commands]
dependency_graph:
  requires: [07-01]
  provides: [two-pass-context-injection-active]
  affects: [cmd/plan.go, cmd/lookup.go, cmd/starter.go, cmd/pattern.go, cmd/helpers.go]
tech_stack:
  added: []
  patterns: [two-pass-context-injection, os.Getwd-for-root, injected-messages-append]
key_files:
  created: []
  modified:
    - cmd/plan.go
    - cmd/lookup.go
    - cmd/starter.go
    - cmd/pattern.go
    - cmd/helpers.go
    - cmd/helpers_test.go
decisions:
  - "Symbols field ([]string) used in symbol fallback path — FileEntry has single Symbols field, not ExportedSymbols/UnexportedSymbols"
metrics:
  duration: ~8m
  completed_date: "2026-04-08"
  tasks_completed: 1
  files_modified: 6
---

# Phase 07 Plan 02: Wire buildInjectedMessages into All 4 Query Commands Summary

**One-liner:** Two-pass context injection activated in plan, lookup, starter, and pattern commands by inserting buildInjectedMessages call between appctx.LoadContext() and history.New().

## What Was Built

All 4 query commands now perform a Pass-1 LLM call before the main streaming response. Each command:

1. Calls `os.Getwd()` to get the project root
2. Calls `buildInjectedMessages(root, input, cfg, ollama.Chat, pass1XxxFocus)` with the correct per-command focus constant
3. Appends the returned `[]history.Message` slice to the messages slice (instead of the previous bare `{Role: "user", Content: input}` entry)
4. Passes the enriched message slice to `history.New()`

The system message construction via `buildSystemMessage` is unchanged. `initiateConversation` and `runConversationLoop` are unchanged.

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | Wire buildInjectedMessages into all 4 query commands | a24454d | cmd/plan.go, cmd/lookup.go, cmd/starter.go, cmd/pattern.go, cmd/helpers.go, cmd/helpers_test.go |

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed ExportedSymbols/UnexportedSymbols field mismatch in helpers.go**
- **Found during:** Task 1 verification (go build ./...)
- **Issue:** `buildInjectedMessages` in `cmd/helpers.go` referenced `fe.ExportedSymbols` and `fe.UnexportedSymbols` on `scanner.FileEntry`, but the struct only has a single `Symbols []string` field. This caused 6 compile errors.
- **Fix:** Replaced the two-branch symbol block with a single `"// Symbols: " + strings.Join(fe.Symbols, ", ")` format. Updated `cmd/helpers_test.go` struct literals to use `Symbols` instead of `ExportedSymbols`.
- **Files modified:** cmd/helpers.go, cmd/helpers_test.go
- **Commit:** a24454d (included in same task commit)

## Verification Results

- `go build ./...` exits 0
- `go vet ./...` exits 0
- `go test ./cmd/ -v` — all 13 tests pass (TestRunConversationLoop x5, TestBuildInjectedMessages x7, TestRunConversationLoop_Summarization x1)
- `grep -n "buildInjectedMessages" cmd/plan.go` — match with pass1PlanFocus
- `grep -n "buildInjectedMessages" cmd/lookup.go` — match with pass1LookupFocus
- `grep -n "buildInjectedMessages" cmd/starter.go` — match with pass1StarterFocus
- `grep -n "buildInjectedMessages" cmd/pattern.go` — match with pass1PatternFocus
- No bare `{Role:"user", Content: input}` entries remain in any of the 4 command files

## Known Stubs

None.

## Threat Flags

None — os.Getwd() errors are wrapped and returned to cobra cleanly (T-07-07 mitigated). No new network endpoints or trust boundaries introduced.

## Self-Check: PASSED

- cmd/plan.go: exists and contains buildInjectedMessages with pass1PlanFocus
- cmd/lookup.go: exists and contains buildInjectedMessages with pass1LookupFocus
- cmd/starter.go: exists and contains buildInjectedMessages with pass1StarterFocus
- cmd/pattern.go: exists and contains buildInjectedMessages with pass1PatternFocus
- Commit a24454d: verified in git log
