---
phase: 10-hierarchical-index-artifacts
plan: "01"
subsystem: scanner
tags: [artifacts, tdd, json, schema]
dependency_graph:
  requires:
    - internal/scanner/ast.go (Symbol struct, ExtractSymbolsFull)
    - internal/scanner/meta.go (ProjectMeta, ReadMeta)
    - internal/scanner/walker.go (Walk)
    - internal/scanner/scanner.go (ChatFn type)
  provides:
    - BuildArtifacts function
    - ProjectArtifact, PackagesArtifact, FilesArtifact, SymbolsArtifact structs
    - PackageEntry, FileArtifactEntry structs
    - SchemaVersion constant ("1.0")
    - ErrStaleFlatIndex sentinel
    - Symbol.FilePath field
    - ProjectMeta.GoVersion field
  affects:
    - cmd/init.go (will call BuildArtifacts in Plan 02)
    - cmd/sync.go (will call BuildArtifacts in Plan 02)
tech_stack:
  added: []
  patterns:
    - TDD (RED then GREEN)
    - json.MarshalIndent + os.WriteFile for all artifact writes
    - Non-fatal chatFn error (log to stderr, summary = "")
    - Per-package responsibility read from existing summaries/*.md
key_files:
  created:
    - internal/scanner/artifacts.go
    - internal/scanner/artifacts_test.go
  modified:
    - internal/scanner/ast.go (FilePath field + ExtractSymbolsFull assignments)
    - internal/scanner/meta.go (GoVersion field + go.mod parsing)
decisions:
  - "FilePath set in struct literal (not post-assignment) in both FuncDecl and GenDecl cases of ExtractSymbolsFull"
  - "pkgShortName derived from first symbol's StableID prefix for FileArtifactEntry.Package"
  - "writeJSON helper extracted to avoid repeated marshal/write boilerplate"
  - "chatFn called once regardless of whether summaries exist (non-fatal on error)"
metrics:
  duration_minutes: 20
  completed_date: "2026-04-09"
  tasks_completed: 2
  files_changed: 4
requirements:
  - IDX-01
  - IDX-02
  - IDX-03
  - IDX-04
  - IDX-05
---

# Phase 10 Plan 01: Hierarchical Artifact Types and BuildArtifacts Summary

**One-liner:** Four structured JSON artifact files (project/packages/files/symbols) with schemaVersion "1.0" written by BuildArtifacts using ExtractSymbolsFull, Walk, and ReadMeta.

## What Was Built

Implemented the complete artifact layer for the hierarchical index. `BuildArtifacts` walks the project root, calls `ExtractSymbolsFull` per file, groups by package, reads per-package responsibility summaries, calls `chatFn` once for a project-level summary, and writes four JSON artifact files to `.myhelper/`.

Extended two existing structs:
- `Symbol.FilePath` (set in both FuncDecl and GenDecl cases of `ExtractSymbolsFull`)
- `ProjectMeta.GoVersion` (parsed from `go X.Y` directive in go.mod)

## Tasks

### Task 1 (RED) — commit d357a5a
Added `FilePath` to `Symbol` and `GoVersion` to `ProjectMeta`. Created `artifacts_test.go` with `TestBuildArtifacts` containing 6 subtests. Build failed with undefined: `BuildArtifacts` and artifact types (RED state confirmed).

### Task 2 (GREEN) — commit f154e07
Created `artifacts.go` with all six struct types, `SchemaVersion` constant, `ErrStaleFlatIndex` sentinel, `BuildArtifacts` function (269 lines), and `writeJSON` helper. Set `FilePath` in both symbol-emitting branches of `ExtractSymbolsFull`. All 6 subtests pass; `go test ./...` exits 0.

## Verification

```
go test ./internal/scanner/... -run TestBuildArtifacts -v  -> 6/6 PASS
go test ./...                                              -> all packages OK
grep -n "FilePath" internal/scanner/ast.go                -> 3 matches (field + 2 assignments)
grep -n "GoVersion" internal/scanner/meta.go              -> 2 matches (field + parse)
grep -n "ErrStaleFlatIndex" internal/scanner/artifacts.go -> found
```

## Deviations from Plan

### Auto-fixed Issues

None — plan executed exactly as written.

**Minor note:** The plan's acceptance criterion `grep -n "sym.FilePath = path"` expected a post-literal assignment style. The actual implementation uses struct literal field initialization (`FilePath: path`), which is idiomatic Go and equally correct. Both FuncDecl and GenDecl cases set FilePath.

## Known Stubs

None — all artifact fields are populated from real data sources (Walk, ExtractSymbolsFull, ReadMeta, chatFn).

## Threat Flags

No new threat surface introduced beyond what is documented in the plan's threat model (T-10-01 through T-10-05).

## Self-Check: PASSED

- internal/scanner/artifacts.go: FOUND
- internal/scanner/artifacts_test.go: FOUND
- internal/scanner/ast.go (FilePath field): FOUND
- internal/scanner/meta.go (GoVersion field): FOUND
- commit d357a5a: FOUND
- commit f154e07: FOUND
- go test ./internal/scanner/... -run TestBuildArtifacts: 6/6 PASS
- go test ./...: all packages PASS
