---
phase: 07-two-pass-context-injection
plan: "01"
subsystem: cmd
tags: [two-pass-injection, context, token-budget, tdd]
dependency_graph:
  requires: [scanner.Index, scanner.FileEntry, scanner.ChatFn, history.Message, history.New, config.Config]
  provides: [buildInjectedMessages, readIndexFile, injectSummaries, pass1BaseSystemPrompt]
  affects: [cmd/helpers.go, cmd/helpers_test.go]
tech_stack:
  added: []
  patterns: [TDD red-green, token budget 80% safety factor, os.Stat path validation, symbol fallback]
key_files:
  created: []
  modified:
    - cmd/helpers.go
    - cmd/helpers_test.go
decisions:
  - "Adapted FileEntry symbol fallback to use Symbols []string (not ExportedSymbols/UnexportedSymbols) — actual scanner.FileEntry has a single Symbols field; plan was written against a different design assumption"
  - "Token budget test uses threshold=9 (budget=7) so raw content (8 tokens) exceeds budget while symbol sig (7 tokens) fits — derived empirically to make the boundary test reliable"
metrics:
  duration: "~12 minutes"
  completed: "2026-04-08"
  tasks_completed: 1
  files_modified: 2
---

# Phase 7 Plan 01: buildInjectedMessages Helper Summary

Two-pass context injection helper implemented with TDD: readIndexFile + injectSummaries + buildInjectedMessages with full token budget logic and all 7 behavioral branches tested.

## What Was Built

### Functions Added to cmd/helpers.go

**`readIndexFile(root string) (scanner.Index, error)`** — reads and unmarshals `.myhelper/index.json`; propagates `os.IsNotExist` for caller discrimination.

**`injectSummaries(root, query string) ([]history.Message, error)`** — fallback path when no valid file paths survive Pass-1 validation; reads all `.md` files from `.myhelper/summaries/`, prepends them to the user query; returns bare query message if directory is missing or empty (no error).

**`buildInjectedMessages(root, query string, cfg config.Config, chatFn scanner.ChatFn, focus string) ([]history.Message, error)`** — full two-pass injection:
1. Reads index.json; falls back to bare query + stderr warning if missing
2. Calls `chatFn` with Pass-1 system prompt + index JSON
3. Validates each returned path with `os.Stat`; discards invalid paths
4. Falls back to `injectSummaries` when no valid paths survive
5. Applies 80% token budget: raw file content first, symbol list fallback, skip once budget exhausted
6. Returns `[]history.Message` with Role "user" only (never system)

**5 focus constants**: `pass1BaseSystemPrompt`, `pass1PlanFocus`, `pass1LookupFocus`, `pass1StarterFocus`, `pass1PatternFocus`

## Test Coverage (7 subtests, all pass)

| Subtest | Behavior verified |
|---------|-------------------|
| no index.json | Returns bare user query + stderr warning "No index found" |
| valid path from Pass-1 | Message has "relevant source code" header + original query |
| invalid path from Pass-1 | Falls back to summaries injection |
| zero parseable paths | Falls back to summaries injection |
| file exceeds budget | Symbol fallback used (raw skipped, symbols injected) |
| budget exhausted after first file | Second file excluded from message |
| no summaries dir on fallback | Returns bare user query, no error |

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] FileEntry.Symbols field name mismatch**
- **Found during:** Task 1 (reading scanner.go)
- **Issue:** Plan referenced `ExportedSymbols` and `UnexportedSymbols` fields. Actual `scanner.FileEntry` has a single `Symbols []string` field (exported symbols only, per D-04 in ast.go). The struct was simplified in Phase 6 relative to the plan's design assumption.
- **Fix:** Used `fe.Symbols` in symbol fallback block; updated sigContent format to `"// Symbols: <comma-joined>"` instead of the two-field format from the plan.
- **Files modified:** cmd/helpers.go
- **Commit:** 4bd1dca

**2. [Rule 1 - Bug] Token threshold in budget-exceeded test too small for symbol fallback**
- **Found during:** Task 1, GREEN phase first run
- **Issue:** Plan specified `tinyThreshold := 5` (budget=4). Both raw content (8 tokens) and symbol sig (7 tokens) exceeded budget=4, causing the loop to break with no content injected — test assertion failed.
- **Fix:** Changed threshold to 9 (budget=7); raw=8 exceeds, sig=7 fits exactly. Derived empirically from actual tiktoken counts.
- **Files modified:** cmd/helpers_test.go
- **Commit:** 4bd1dca

## Known Stubs

None — all functions are fully wired. `buildInjectedMessages` is ready to be called by query commands in plan 07-02.

## Threat Flags

None — no new network endpoints, auth paths, or trust boundaries introduced. Path validation (T-07-01, T-07-04) is implemented as specified: `os.Stat(filepath.Join(root, p))` keeps reads within project root; absolute paths from LLM will fail the relative lookup.

## Self-Check

- [x] cmd/helpers.go contains `func buildInjectedMessages(`
- [x] cmd/helpers.go contains `func readIndexFile(`
- [x] cmd/helpers.go contains `func injectSummaries(`
- [x] cmd/helpers.go contains all 5 focus constants
- [x] cmd/helpers_test.go contains `TestBuildInjectedMessages`
- [x] `go test ./cmd/ -run TestBuildInjectedMessages` exits 0 (7/7 subtests pass)
- [x] `go build ./...` exits 0
- [x] `go vet ./...` exits 0
- [x] Zero new go.mod dependencies (stdlib + existing imports only)
- [x] All returned messages from buildInjectedMessages have Role "user" (CTX-04 satisfied)

## Self-Check: PASSED
