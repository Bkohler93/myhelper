---
phase: 16-cli-cleanup
plan: "01"
subsystem: cmd
tags: [cleanup, deletion, cobra, go-mod]
dependency_graph:
  requires: []
  provides: [minimal-rootCmd, conversation-helpers-only]
  affects: [cmd/root.go, cmd/helpers.go, cmd/helpers_test.go, go.mod, go.sum]
tech_stack:
  added: []
  patterns: [cobra-no-args-gate]
key_files:
  created: []
  modified:
    - cmd/root.go
    - cmd/helpers.go
    - cmd/helpers_test.go
    - go.mod
    - go.sum
decisions:
  - "Added Args: cobra.NoArgs + minimal RunE to rootCmd so cobra emits 'unknown command' for deleted subcommands without a RunE; Phase 17 will replace RunE with the chat entry point"
metrics:
  duration_minutes: 15
  completed_date: "2026-04-10"
  tasks_completed: 3
  files_changed: 15
---

# Phase 16 Plan 01: CLI Cleanup Summary

**One-liner:** Deleted all seven cobra subcommands plus tui.go, trimmed helpers.go and root.go to conversation-loop essentials only, and removed charmbracelet deps via go mod tidy — leaving a single-entry-point binary ready for Phase 17 chat wiring.

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | Delete seven command files, tui.go, and coupled tests | 715d55d | cmd/starter.go, plan.go, lookup.go, pattern.go, inspect.go, init.go, sync.go, tui.go, flags_test.go, sync_test.go (all deleted) |
| 2 | Trim root.go and helpers.go; remove orphaned tests | 7a307f3 | cmd/root.go, cmd/helpers.go, cmd/helpers_test.go |
| 3 | go mod tidy and smoke-test binary | 9255134 | go.mod, go.sum, cmd/root.go |

## Verification Results

- `go build ./...` exits 0
- All 7 removed subcommands produce "unknown command" output
- `go test ./cmd/... ./internal/ollama/... ./internal/history/... ./internal/scanner/... ./internal/retrieval/...` — all pass
- `grep "charmbracelet" go.mod` — no matches
- `cmd/` contains exactly: root.go, helpers.go, helpers_test.go

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] cobra.NoArgs alone does not produce "unknown command" without RunE**
- **Found during:** Task 3 smoke-test
- **Issue:** With no subcommands and no RunE on rootCmd, cobra treats positional args as accepted and prints help (exit 0) rather than emitting "unknown command". The plan's must-have truth required "unknown command" output.
- **Fix:** Added `Args: cobra.NoArgs` plus a minimal no-op `RunE` to rootCmd. Cobra now routes unknown positional args to its arg-validation path and emits `unknown command "starter" for "myhelper"`. Phase 17 will replace the no-op RunE with the chat entry point.
- **Files modified:** cmd/root.go
- **Commit:** 9255134

## Known Stubs

None — no UI rendering paths or placeholder data in this plan's scope. The no-op RunE is intentional scaffolding documented above, not a stub; Phase 17 replaces it with real behavior.

## Threat Flags

None — pure code-deletion phase. No new network endpoints, auth paths, file access patterns, or schema changes introduced.

## Self-Check: PASSED

- cmd/root.go exists: FOUND
- cmd/helpers.go exists: FOUND
- cmd/helpers_test.go exists: FOUND
- go.mod exists: FOUND
- Commit 715d55d exists: FOUND
- Commit 7a307f3 exists: FOUND
- Commit 9255134 exists: FOUND
