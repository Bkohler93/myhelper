---
phase: 05-scanner-index-generation
plan: 03
subsystem: internal/scanner
tags: [scanner, index, token-budget, tdd, history, json]
dependency_graph:
  requires: [FileEntry, Walk, ExtractSymbols, ProjectMeta, Config.TokenThreshold, history.New]
  provides: [BuildIndex]
  affects: [internal/scanner/index.go]
tech_stack:
  added: []
  patterns: [TDD, history.New for token counting, sort.SliceStable for priority dropping, json.MarshalIndent]
key_files:
  created:
    - internal/scanner/index.go
    - internal/scanner/index_test.go
  modified: []
decisions:
  - "Per-entry TokenCount computed from JSON-serialized entry wrapped in history.Message{Role:user}; consistent with rest of codebase"
  - "sort.SliceStable used to preserve relative order while moving test files to end before truncation"
  - "nil entries slice replaced with empty []FileEntry{} before MarshalIndent to produce '[]' not 'null'"
  - "token budget cutoff stops at first entry that would push total over budget (not after)"
metrics:
  duration: "~5 minutes"
  completed: "2026-04-08"
  tasks_completed: 1
  files_created: 2
  files_modified: 0
---

# Phase 05 Plan 03: Index Assembler Summary

**One-liner:** Token-budgeted index assembler using Walk+ExtractSymbols to build []FileEntry with cl100k_base token counts, dropping test files first when the 80% budget is exceeded, then writing a JSON array to .myhelper/index.json.

## What Was Built

Implemented `internal/scanner/index.go` with one exported function and one unexported helper:

- `BuildIndex(root string, cfg config.Config) ([]FileEntry, error)`: walks root via Walk(), extracts AST symbols via ExtractSymbols(), computes per-entry token counts using history.New, applies 80% token budget with test-file-first dropping, and writes .myhelper/index.json
- `isTestFile(path string) bool`: reports whether a path ends with `_test.go`
- `const tokenBudgetSafetyFactor = 0.80`: budget constant matching phase specification

Key behaviors:
- Per-entry token count uses the JSON-serialized entry wrapped in `history.Message{Role:"user"}` — consistent with existing codebase pattern
- `sort.SliceStable` sorts non-test files before test files, preserving relative order within each group
- Truncation loop stops at the first entry that would push accumulated total over budget (that entry and all following are dropped)
- `nil` entries slice replaced with `[]FileEntry{}` before marshaling to ensure `[]` output (not `null`) for empty projects
- Unparseable files skipped with `fmt.Fprintf(os.Stderr, ...)` warning per plan spec

## TDD Execution

**RED:** 7 test cases written and confirmed failing (build error — `BuildIndex` undefined).
**GREEN:** Implementation written; all 7 tests pass on first run.
**REFACTOR:** Not needed — code was clean on first pass.

Tests cover:
1. Basic entry (path, package, symbols, positive token count)
2. Per-entry token count matches re-encoded JSON representation
3. Budget cap causes entries to be dropped
4. Test files dropped before non-test files
5. Output file exists after BuildIndex
6. Output file contains valid JSON array
7. No .go files produces empty array `[]`

## Verification Results

- `go test ./internal/scanner/ -run TestBuildIndex -v` — 7/7 PASS
- `go vet ./internal/scanner/` — no issues
- `go build ./...` — project compiles
- Grep checks: `tokenBudgetSafetyFactor`, `0.80`, `history.New(`, `_test.go`, `json.MarshalIndent`, `index.json` — all FOUND in index.go

## Deviations from Plan

None - plan executed exactly as written.

## Known Stubs

None.

## Threat Surface Scan

T-05-07 (DoS via large project) mitigated as required — `tokenBudgetSafetyFactor = 0.80` caps total index tokens at `cfg.TokenThreshold * 0.80`. Tests 3 and 4 verify this behavior. No new threat surface introduced beyond what was in the plan's threat model.

## Commits

| Hash | Message |
|------|---------|
| cdc2d71 | feat(05-03): implement BuildIndex() with token budgeting |

## Self-Check: PASSED

- `internal/scanner/index.go` — FOUND
- `internal/scanner/index_test.go` — FOUND
- Commit cdc2d71 — FOUND
