---
phase: 08-large-file-micro-pass
plan: "02"
subsystem: cmd
tags: [micro-pass, large-file, tdd, context-injection, truncation]
dependency_graph:
  requires: [scanner.SymbolLine, scanner.ExtractSymbolMap]
  provides: [cmd.microPassFile, cmd.buildInjectedMessages (updated)]
  affects: [cmd/helpers.go, cmd/helpers_test.go]
tech_stack:
  added: [regexp (stdlib)]
  patterns: [TDD RED/GREEN, inner closure for fallback chain, package-level compiled regexp]
key_files:
  created: []
  modified:
    - cmd/helpers.go
    - cmd/helpers_test.go
decisions:
  - microPassFile uses an inner closure for the micro-pass attempt to keep the fallback chain readable without extracting a separate named function
  - Existing test "file content exceeds budget uses symbol fallback" renamed and updated to reflect micro-pass behavior; threshold=1 used to produce budget=0 which causes immediate skip
  - ExportedSymbols/UnexportedSymbols field references in existing tests fixed to use Symbols field (pre-existing build error resolved as part of this plan)
metrics:
  duration: ~25 minutes
  completed: "2026-04-08"
  tasks_completed: 1
  files_modified: 2
---

# Phase 08 Plan 02: microPassFile and buildInjectedMessages Update Summary

microPassFile function added to cmd/helpers.go implementing the full D-10 fallback chain (symbol map + chatFn line range, truncation, skip); symbol-block branch (ExportedSymbols/UnexportedSymbols) completely removed from buildInjectedMessages and replaced with microPassFile call.

## Tasks Completed

| Task | Description | Commit | Status |
|------|-------------|--------|--------|
| 1 (RED) | TestMicroPassFile (8 subtests) + TestBuildInjectedMessages_NoSymbolBlock + fix existing ExportedSymbols test references | 259b0f5 | PASS |
| 1 (GREEN) | microPassFile + regexp import + symbol-block removal + microPassFile wired into buildInjectedMessages | cff4054 | PASS |

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed ExportedSymbols/UnexportedSymbols field references in existing tests**
- **Found during:** RED phase
- **Issue:** Pre-existing build failure — `cmd/helpers_test.go` referenced `scanner.FileEntry.ExportedSymbols` and `scanner.FileEntry.UnexportedSymbols` which do not exist (the field is `Symbols`). This also blocked `cmd/helpers.go` from building.
- **Fix:** Updated 3 `scanner.FileEntry` literals in tests to use `Symbols: []string{...}` instead of `ExportedSymbols`/`UnexportedSymbols`. Renamed and updated the "file content exceeds token budget uses symbol fallback" test to reflect micro-pass behavior.
- **Files modified:** cmd/helpers_test.go
- **Commit:** 259b0f5

## Known Stubs

None.

## Threat Flags

None — microPassFile receives pre-validated paths from buildInjectedMessages (os.Stat gated). LLM-returned line range is parsed with regexp and clamped before array indexing. No new network endpoints or auth paths.

## Self-Check: PASSED

- cmd/helpers.go: FOUND
- cmd/helpers_test.go: FOUND
- Commit 259b0f5 (RED): FOUND
- Commit cff4054 (GREEN): FOUND
- `func microPassFile(` in cmd/helpers.go: FOUND (line 366)
- `var microPassRe = regexp.MustCompile(` in cmd/helpers.go: FOUND
- `microPassFile(` in buildInjectedMessages body: FOUND (line 527)
- `ExportedSymbols|UnexportedSymbols|entryByPath` in cmd/helpers.go: CLEAN (zero matches)
- `go test ./cmd/ -run TestMicroPassFile`: 8/8 PASS
- `go test ./cmd/`: all tests PASS (no regressions)
- `go test ./internal/scanner/`: all tests PASS
- `go build ./...`: clean
