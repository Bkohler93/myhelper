---
phase: 09-extended-ast-symbol-extraction
plan: "01"
subsystem: scanner
tags: [ast, symbol-extraction, tdd]
dependency_graph:
  requires: []
  provides: [Symbol, ExtractSymbolsFull]
  affects: [internal/scanner/ast.go, internal/scanner/ast_test.go]
tech_stack:
  added: []
  patterns: [go/ast, go/parser, go/token, TDD red-green]
key_files:
  created: []
  modified:
    - internal/scanner/ast.go
    - internal/scanner/ast_test.go
decisions:
  - "Reuse buildFuncSig for method signatures — receiver NOT included in Signature string, stored in Receiver field"
  - "extractImportPaths includes blank and dot imports in Imports slice; buildImportAliasMap skips them for identifier lookup"
  - "nil imports when file has no import declarations (len(imports)==0 -> nil), not empty slice"
  - "Type aliases and non-struct/interface types skipped in GenDecl per RESEARCH.md open question 3"
  - "CallEdges and TypeRefs initialized as nil — Plan 02 adds body-walking pass"
metrics:
  duration_seconds: 105
  completed_date: "2026-04-09"
  tasks_completed: 2
  files_modified: 2
---

# Phase 09 Plan 01: Symbol Struct and ExtractSymbolsFull Summary

**One-liner:** Symbol struct and ExtractSymbolsFull function providing kind/signature/lines/imports/stableID for all exported Go symbols using go/ast.

## What Was Built

Added `Symbol` struct and `ExtractSymbolsFull` to `internal/scanner/ast.go`, plus a comprehensive `TestExtractSymbolsFull` test (TDD red-green). The function parses a Go source file and returns a rich profile for each exported symbol:

- **kind** — "func", "method", "struct", "interface"
- **signature** — `buildFuncSig` output for funcs/methods; "Name type; Name type" for structs; "Method(params) result" for interfaces
- **lines** — 1-indexed Start/End from `fset.Position()`
- **imports** — all file-level import paths on every symbol (nil when no imports)
- **stableID** — `pkg.Name` for funcs/types, `pkg.Receiver.Name` for methods
- **receiver** — unwrapped receiver type name (empty for non-methods)
- **CallEdges/TypeRefs** — nil (reserved for Plan 02 body-walking pass)

Six private helpers were added: `extractImportPaths`, `buildImportAliasMap`, `receiverTypeName`, `buildStructSig`, `buildInterfaceSig`, `buildMethodSigFromFuncType`.

## Commits

| Task | Commit | Description |
|------|--------|-------------|
| Task 1 (RED) | db0b969 | test(09-01): add failing TestExtractSymbolsFull — RED state |
| Task 2 (GREEN) | a85abb8 | feat(09-01): implement Symbol struct and ExtractSymbolsFull — GREEN state |

## Verification Results

```
go build ./internal/scanner/...           PASS (clean)
go test ./internal/scanner/... -v         PASS (42 tests, 0 failures)
  TestExtractSymbolsFull/kind             PASS
  TestExtractSymbolsFull/signature/func   PASS
  TestExtractSymbolsFull/signature/method PASS
  TestExtractSymbolsFull/signature/struct PASS
  TestExtractSymbolsFull/signature/interface PASS
  TestExtractSymbolsFull/lines/...        PASS
  TestExtractSymbolsFull/imports/...      PASS
  TestExtractSymbolsFull/stable_id        PASS
  TestExtractSymbols (all subtests)       PASS (no regression)
  TestExtractSymbolMap (all subtests)     PASS (no regression)
```

## Deviations from Plan

None — plan executed exactly as written.

## Known Stubs

None — all fields are fully implemented. CallEdges and TypeRefs are intentionally nil; Plan 02 will populate them via a body-walking pass.

## Threat Flags

No new security-relevant surface introduced. `ExtractSymbolsFull` reads local filesystem files via `go/parser` — same trust boundary as existing `ExtractSymbols` and `ExtractSymbolMap`. All threats accepted per plan threat model (T-09-01, T-09-02, T-09-03).

## Self-Check: PASSED

- [x] `internal/scanner/ast.go` — modified, contains `type Symbol struct`, `func ExtractSymbolsFull`
- [x] `internal/scanner/ast_test.go` — modified, contains `func TestExtractSymbolsFull`
- [x] Commit db0b969 exists (RED)
- [x] Commit a85abb8 exists (GREEN)
- [x] `go test ./internal/scanner/... -v` exits 0, no FAIL lines
