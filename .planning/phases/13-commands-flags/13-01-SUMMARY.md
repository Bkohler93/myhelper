---
phase: 13-commands-flags
plan: "01"
subsystem: retrieval, cmd
tags: [tdd, red-state, nyquist, cmd-flags, inspect-context]
dependency_graph:
  requires: []
  provides: [retrieval_test.BuildInspectContext_tests, cmd.flags_test]
  affects: [internal/retrieval/retrieval_test.go, cmd/flags_test.go]
tech_stack:
  added: []
  patterns: [TDD-RED, source-scan regression guard]
key_files:
  created: [cmd/flags_test.go]
  modified: [internal/retrieval/retrieval_test.go]
decisions:
  - "Wrote failing tests first (Nyquist/RED) before any implementation exists"
  - "Source-scan test for ApplyFlagOverrides guards against silent removal of CMD-03 fix"
  - "writeArtifacts helper appended to retrieval_test.go (not a separate file) to stay in same package"
metrics:
  duration: "~8 minutes"
  completed: "2026-04-10"
  tasks_completed: 2
  files_modified: 2
---

# Phase 13 Plan 01: Nyquist Failing Tests Summary

Wrote failing (RED) test stubs that define the contracts for Phase 13's three deliverables — `BuildInspectContext` with `SelectionSource`/`InspectResult` types, the `--no-context` flag, and `ApplyFlagOverrides` call sites — before any implementation exists.

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | Add failing BuildInspectContext tests to retrieval_test.go | 0c754eb | internal/retrieval/retrieval_test.go |
| 2 | Add failing flag tests to cmd/flags_test.go | ec654bf | cmd/flags_test.go |

## What Was Built

### Task 1 — retrieval_test.go additions

Appended to `internal/retrieval/retrieval_test.go`:

- `writeArtifacts(t, root, syms)` — helper that writes minimal four JSON artifact files to `root/.myhelper/` for use in BuildInspectContext tests
- `TestSelectionSource_String` — verifies `SourcePreFilter.String()=="pre-filter"`, `SourceReRank.String()=="re-rank"`, `SourceExpansion.String()=="expansion"`
- `TestBuildInspectContext_NoArtifacts` — no `.myhelper/` dir; expects `InspectResult{GatePassed: false}` with nil error
- `TestBuildInspectContext_GateBlocks` — artifacts present but `noChatFn` returns "no"; expects gate blocks with no stage metrics
- `TestBuildInspectContext_WithSymbols` — gate passes ("yes"), re-rank selects symbol ("retrieval.BuildContext"); expects `GatePassed==true`, `Symbols` non-empty with `Source==SourceReRank`, stage metrics for "pre-filter" and "re-rank", `FinalTokens>0`

RED state: `go test ./internal/retrieval/...` fails with 8 "undefined" errors for `BuildInspectContext`, `SelectionSource`, `SourcePreFilter`, `SourceReRank`, `SourceExpansion`.

### Task 2 — cmd/flags_test.go (new file)

Created `cmd/flags_test.go` in `package cmd`:

- `TestApplyFlagOverrides_QueryCommands` — source-scan regression guard; reads plan.go, lookup.go, starter.go, pattern.go and asserts each contains `"ApplyFlagOverrides(&cfg)"`. Fails for all four files (CMD-03 call not yet present).
- `TestNoContextFlag_Registered` — checks `rootCmd.PersistentFlags().Lookup("no-context") != nil` and default value is `"false"`. Fails because `noContextFlag` not yet registered (CMD-01).

RED state confirmed: both tests FAIL; `go build ./cmd/...` exits 0 (syntax clean).

## Verification Results

```
# retrieval RED state:
go test ./internal/retrieval/... → build failed (8 undefined symbols)

# cmd RED state:
go test ./cmd/... -run TestApplyFlagOverrides_QueryCommands → FAIL (4 commands missing call)
go test ./cmd/... -run TestNoContextFlag_Registered → FAIL (flag not registered)

# cmd build clean:
go build ./cmd/... → exit 0
```

## Deviations from Plan

None — plan executed exactly as written.

Note: The success criteria mentions existing retrieval tests should remain passing, but appending tests that reference undefined types to the same package file means `go test` cannot compile any tests in that package until Plan 02/03 provides the implementations. This is the expected and correct TDD RED state — the whole `internal/retrieval` test package is RED until `BuildInspectContext` and `SelectionSource` are implemented.

## Known Stubs

None — this plan only writes test stubs (intentionally stub-like by design as TDD RED state).

## Threat Flags

None — no new production code, network endpoints, or auth paths introduced. Test files only.

## Self-Check

- [x] `internal/retrieval/retrieval_test.go` contains `func TestSelectionSource_String` — FOUND
- [x] `internal/retrieval/retrieval_test.go` contains `func TestBuildInspectContext_NoArtifacts` — FOUND
- [x] `internal/retrieval/retrieval_test.go` contains `func writeArtifacts` — FOUND
- [x] `cmd/flags_test.go` exists — FOUND
- [x] `cmd/flags_test.go` contains `func TestApplyFlagOverrides_QueryCommands` — FOUND
- [x] `cmd/flags_test.go` contains `func TestNoContextFlag_Registered` — FOUND
- [x] Commit `0c754eb` exists — FOUND
- [x] Commit `ec654bf` exists — FOUND

## Self-Check: PASSED
