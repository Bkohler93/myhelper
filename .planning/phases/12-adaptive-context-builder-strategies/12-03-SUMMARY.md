---
phase: 12-adaptive-context-builder-strategies
plan: "03"
subsystem: cmd
tags: [wiring, cleanup, retrieval, dead-code-removal]
dependency_graph:
  requires: [12-02]
  provides: [CTX-03, CTX-04]
  affects: [cmd/plan.go, cmd/starter.go, cmd/lookup.go, cmd/pattern.go, cmd/helpers.go, cmd/helpers_test.go]
tech_stack:
  added: []
  patterns: [per-command strategy injection, retrieval.BuildContext call pattern]
key_files:
  created: []
  modified:
    - cmd/plan.go
    - cmd/starter.go
    - cmd/lookup.go
    - cmd/pattern.go
    - cmd/helpers.go
    - cmd/helpers_test.go
decisions:
  - "Keep readIndexFile in helpers.go — still called by cmd/sync_test.go (not dead)"
  - "Keep TestReadIndexFile_StaleFlatIndex — tests readIndexFile which is preserved"
  - "Keep writeIndexFile and writeSummaryFile helpers — used by cmd/sync_test.go"
  - "Delete writeTempGoFile — only used by deleted TestMicroPassFile tests"
metrics:
  duration: "~15 minutes"
  completed: "2026-04-09"
  tasks_completed: 3
  files_modified: 6
---

# Phase 12 Plan 03: Wire Commands to retrieval.BuildContext and Delete Dead Code Summary

Wire all four query commands to call `retrieval.BuildContext` with the correct per-command Strategy, then delete `buildInjectedMessages` and all dead helpers from `cmd/helpers.go`, cleaning up `cmd/helpers_test.go` to remove tests for deleted functions.

## What Was Built

Four command files (`cmd/plan.go`, `cmd/starter.go`, `cmd/lookup.go`, `cmd/pattern.go`) now call `retrieval.BuildContext` with per-command strategies (`PlanStrategy`, `StarterStrategy`, `LookupStrategy`, `PatternStrategy`) instead of the old `buildInjectedMessages` helper. The dead helper and its dependencies (`pass1*` constants, `injectSummaries`, `microPassRe`, `microPassFile`, `buildInjectedMessages`) were deleted from `cmd/helpers.go`. Test functions covering the deleted code were removed from `cmd/helpers_test.go`.

## Tasks Completed

| Task | Name | Commit |
|------|------|--------|
| 1 | Audit dead code callers before deletion | (no files changed — audit only) |
| 2 | Rewire four commands to retrieval.BuildContext | 34658ff |
| 3 | Delete dead code from helpers.go and clean up helpers_test.go | 848c85d |

## Key Changes

**cmd/plan.go, cmd/starter.go, cmd/lookup.go, cmd/pattern.go:**
- Added import `"github.com/bkohler93/myhelper/internal/retrieval"`
- Replaced `buildInjectedMessages(root, input, cfg, ollama.Chat, pass1*Focus)` with `retrieval.BuildContext(root, input, retrieval.*Strategy, cfg, ollama.Chat)`
- Updated message assembly from `injected...` to `rctx.Messages...`

**cmd/helpers.go:**
- Deleted: `pass1BaseSystemPrompt`, `pass1PlanFocus`, `pass1LookupFocus`, `pass1StarterFocus`, `pass1PatternFocus` (5 constants)
- Deleted: `injectSummaries` function
- Deleted: `microPassRe` var and `microPassFile` function
- Deleted: `buildInjectedMessages` function
- Kept: `readIndexFile` (still used by `cmd/sync_test.go`)
- Removed unused imports: `"regexp"`, `"errors"` (after confirming `errors` was only used in `buildInjectedMessages`)

**cmd/helpers_test.go:**
- Deleted: `TestBuildInjectedMessages` (7 subtests)
- Deleted: `TestMicroPassFile` (8 subtests)
- Deleted: `TestBuildInjectedMessages_NoSymbolBlock`
- Deleted: `writeTempGoFile` helper (exclusively used by deleted tests)
- Kept: `TestReadIndexFile_StaleFlatIndex` (tests preserved `readIndexFile`)
- Kept: `writeIndexFile`, `writeSummaryFile` (used by `cmd/sync_test.go`)
- Kept: `TestRunConversationLoop`, `TestRunConversationLoop_Summarization`

## Decisions Made

| Decision | Rationale |
|----------|-----------|
| Keep `readIndexFile` in helpers.go | Called from `cmd/sync_test.go` lines 368, 405, 459, 491 — not dead code |
| Keep `TestReadIndexFile_StaleFlatIndex` | Tests `readIndexFile` which remains; behavior still needs test coverage |
| Keep `writeIndexFile` and `writeSummaryFile` | Both used extensively in `cmd/sync_test.go` |
| Delete `writeTempGoFile` | Only used by `TestMicroPassFile` which was deleted |

## Deviations from Plan

**1. [Rule 1 - Bug] readIndexFile kept despite plan listing it for deletion**

- **Found during:** Task 1 (audit)
- **Issue:** Plan listed `readIndexFile` as safe to delete if "not called from cmd/init.go or cmd/sync.go". Audit found it called in `cmd/sync_test.go` (4 call sites). Plan's condition was "if called from init.go or sync.go — DO NOT delete" but did not explicitly cover test files.
- **Fix:** Kept `readIndexFile` in `cmd/helpers.go`. Also kept `TestReadIndexFile_StaleFlatIndex` in `helpers_test.go` since it tests a function that still exists.
- **Files modified:** No extra changes — simply did not delete the function.
- **Commit:** Included in 848c85d

## Verification Results

```
go build ./...  → EXIT 0
go test ./...   → all packages PASS (cmd, config, history, ollama, retrieval, scanner)
grep buildInjectedMessages cmd/ → 0 matches
grep retrieval.BuildContext cmd/plan.go cmd/starter.go cmd/lookup.go cmd/pattern.go → 4 matches
grep pass1 cmd/helpers.go → 0 matches
grep func microPassFile cmd/helpers.go → 0 matches
```

## Self-Check

**Files exist:**
- cmd/plan.go — modified, exists
- cmd/starter.go — modified, exists
- cmd/lookup.go — modified, exists
- cmd/pattern.go — modified, exists
- cmd/helpers.go — modified, exists
- cmd/helpers_test.go — modified, exists

**Commits exist:**
- 34658ff — feat(12-03): rewire four commands to retrieval.BuildContext
- 848c85d — feat(12-03): delete buildInjectedMessages and dead helpers from cmd/helpers.go

## Self-Check: PASSED
