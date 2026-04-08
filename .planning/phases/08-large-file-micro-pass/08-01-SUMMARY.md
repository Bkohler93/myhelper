---
phase: 08-large-file-micro-pass
plan: "01"
subsystem: scanner
tags: [ast, symbols, line-numbers, tdd]
dependency_graph:
  requires: []
  provides: [scanner.SymbolLine, scanner.ExtractSymbolMap]
  affects: [internal/scanner/ast.go, internal/scanner/ast_test.go]
tech_stack:
  added: []
  patterns: [go/ast fset.Position() for 1-indexed line ranges, TDD RED/GREEN]
key_files:
  created: []
  modified:
    - internal/scanner/ast.go
    - internal/scanner/ast_test.go
decisions:
  - ExtractSymbolMap includes all funcs (exported+unexported) unlike ExtractSymbols which only includes exported
  - SymbolLine.Name uses short "func Name" format (no signature) to be compact for micro-pass line-range queries
metrics:
  duration: ~5 minutes
  completed: "2026-04-08"
  tasks_completed: 1
  files_modified: 2
---

# Phase 08 Plan 01: SymbolLine and ExtractSymbolMap Summary

SymbolLine type and ExtractSymbolMap function added to scanner package using go/ast fset.Position() for 1-indexed line ranges covering all funcs and exported struct/interface types.

## Tasks Completed

| Task | Description | Commit | Status |
|------|-------------|--------|--------|
| 1 (RED) | TestExtractSymbolMap — 8 failing subtests | 16918ee | PASS |
| 1 (GREEN) | SymbolLine type + ExtractSymbolMap implementation | 8d8a0df | PASS |

## Deviations from Plan

None — plan executed exactly as written.

## Known Stubs

None.

## Threat Flags

None — no new network endpoints, auth paths, file access patterns, or schema changes at trust boundaries introduced. ExtractSymbolMap reads local filesystem paths under caller control, consistent with existing pattern in ExtractSymbols.

## Out-of-Scope Build Failures (Deferred)

Pre-existing build failure in `cmd/helpers.go` references `fe.ExportedSymbols` and `fe.UnexportedSymbols` on `scanner.FileEntry`, which currently only has a `Symbols` field. This mismatch existed at base commit `aa627a6` and is not caused by plan 08-01 changes. Tracked for resolution in a subsequent plan.

## Self-Check: PASSED

- internal/scanner/ast.go: FOUND
- internal/scanner/ast_test.go: FOUND
- Commit 16918ee: FOUND (test RED phase)
- Commit 8d8a0df: FOUND (feat GREEN phase)
- `type SymbolLine struct` at ast.go:63: FOUND
- `func ExtractSymbolMap` at ast.go:81: FOUND
- `func ExtractSymbols` at ast.go:19: FOUND (unchanged)
- `go test ./internal/scanner/ -run TestExtractSymbolMap`: 8/8 PASS
- `go test ./internal/scanner/`: all tests PASS
