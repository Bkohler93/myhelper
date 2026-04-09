---
phase: 09-extended-ast-symbol-extraction
plan: "02"
subsystem: scanner
tags: [ast, symbol-extraction, call-edges, type-refs, tdd]
dependency_graph:
  requires: [09-01]
  provides: [extractCallEdges, extractTypeRefs, Symbol.CallEdges, Symbol.TypeRefs]
  affects: [internal/scanner/ast.go, internal/scanner/ast_test.go]
tech_stack:
  added: []
  patterns: [go/ast.Inspect, body-walking, deduplication via seen map, TDD red-green]
key_files:
  created: []
  modified:
    - internal/scanner/ast.go
    - internal/scanner/ast_test.go
decisions:
  - "extractCallEdges resolves selector calls using importAliasMap last-path-segment as pkgname; raw 'obj.Method' for non-import selectors"
  - "extractTypeRefs returns false from ast.Inspect on SelectorExpr to avoid double-collecting child Ident nodes"
  - "knownTypes pre-pass runs over f.Decls before main symbol loop to collect exported type names for TypeRefs matching"
  - "importAliasMap built once at ExtractSymbolsFull entry and shared across all FuncDecl body walks"
  - "nil body guards on both helpers satisfy T-09-05 threat mitigation (assembly stubs with nil Body)"
metrics:
  duration_seconds: 180
  completed_date: "2026-04-09"
  tasks_completed: 2
  files_modified: 2
---

# Phase 09 Plan 02: Call Edge and Type Ref Extraction Summary

**One-liner:** extractCallEdges and extractTypeRefs body-walking helpers wired into ExtractSymbolsFull, populating Symbol.CallEdges and Symbol.TypeRefs with deduplicated, import-resolved identifiers from exported function bodies.

## What Was Built

Added two private helper functions to `internal/scanner/ast.go` and updated `ExtractSymbolsFull` to call them for every exported `FuncDecl`:

**`extractCallEdges(body *ast.BlockStmt, importAliasMap map[string]string) []string`**
- Uses `ast.Inspect` to walk the function body and collect all `*ast.CallExpr` targets
- `*ast.Ident` (direct call `foo()`) → stored as `"foo"`
- `*ast.SelectorExpr` where `X.Name` is a known import alias → resolved to `"<pkgname>.<Sel>"` using last path segment of the import path
- `*ast.SelectorExpr` where `X.Name` is NOT a known import alias → stored raw as `"obj.Method"`
- Deduplication via `seen map[string]bool`
- Nil body guard: returns `nil` if `body == nil` (handles assembly stubs — T-09-05)

**`extractTypeRefs(body *ast.BlockStmt, importAliasMap map[string]string, knownTypes map[string]bool) []string`**
- Uses `ast.Inspect` to walk the function body and collect exported type references
- `*ast.SelectorExpr` where `X.Name` is a known import alias and `Sel.Name` is exported → `"<pkgname>.<Type>"`; returns `false` from inspector to avoid re-visiting child `Ident` nodes
- `*ast.Ident` where `Name` is in `knownTypes` → collected (file-local exported types only)
- Primitives (`int`, `string`, `bool`, etc.) excluded: they are not capitalized, so not in `knownTypes` and not import aliases
- Deduplication via `seen map[string]bool`
- Nil body guard: returns `nil` if `body == nil`

**`ExtractSymbolsFull` updates:**
- `buildImportAliasMap(f)` called once at function entry
- `knownTypes` pre-pass over `f.Decls` before main symbol loop, collecting exported `*ast.TypeSpec` names
- FuncDecl branch refactored to use local `sym Symbol` variable; `extractCallEdges` and `extractTypeRefs` called before `append`
- GenDecl branch (structs/interfaces) unchanged — no body to walk

## Commits

| Task | Commit | Description |
|------|--------|-------------|
| Task 1 (RED) | df09665 | test(09-02): add failing call_edges and type_refs subtests — RED state |
| Task 2 (GREEN) | 0ba6f58 | feat(09-02): implement extractCallEdges, extractTypeRefs, wire into ExtractSymbolsFull |

## Verification Results

```
go build ./internal/scanner/...                               PASS (clean)
go vet ./internal/scanner/...                                 PASS (clean)
go test ./internal/scanner/... -run TestExtractSymbolsFull -v PASS (7 subtests)
  TestExtractSymbolsFull/kind                                 PASS
  TestExtractSymbolsFull/signature/func                       PASS
  TestExtractSymbolsFull/signature/method                     PASS
  TestExtractSymbolsFull/signature/struct                     PASS
  TestExtractSymbolsFull/signature/interface                  PASS
  TestExtractSymbolsFull/lines/one-liner_func_at_line_3       PASS
  TestExtractSymbolsFull/lines/multi-line_func_End_>_Start    PASS
  TestExtractSymbolsFull/imports/file_with_imports            PASS
  TestExtractSymbolsFull/imports/no_imports                   PASS
  TestExtractSymbolsFull/stable_id                            PASS
  TestExtractSymbolsFull/call_edges                           PASS
  TestExtractSymbolsFull/type_refs                            PASS
go test ./internal/scanner/... -v                             PASS (full suite, no regressions)
  TestExtractSymbols (all subtests)                           PASS
  TestExtractSymbolMap (all subtests)                         PASS
```

## Deviations from Plan

None — plan executed exactly as written.

## Known Stubs

None — CallEdges and TypeRefs are now fully populated for all exported function/method bodies. Struct and interface symbols have nil CallEdges/TypeRefs by design (no body to walk).

## Threat Flags

No new security-relevant surface introduced. Both helpers operate entirely on in-memory AST nodes produced by `go/parser` from local filesystem files. Nil body guards satisfy T-09-05 mitigation requirement.

## Self-Check: PASSED

- [x] `internal/scanner/ast.go` contains `func extractCallEdges(body *ast.BlockStmt,`
- [x] `internal/scanner/ast.go` contains `func extractTypeRefs(body *ast.BlockStmt,`
- [x] `internal/scanner/ast_test.go` contains `t.Run("call_edges",`
- [x] `internal/scanner/ast_test.go` contains `t.Run("type_refs",`
- [x] Commit df09665 exists (RED)
- [x] Commit 0ba6f58 exists (GREEN)
- [x] `go test ./internal/scanner/... -v` exits 0, no FAIL lines
- [x] `go vet ./internal/scanner/...` exits 0
