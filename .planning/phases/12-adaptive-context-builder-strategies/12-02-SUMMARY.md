---
phase: 12-adaptive-context-builder-strategies
plan: "02"
subsystem: retrieval
tags: [context-assembly, strategy, tdd-green, budget-enforcement]
dependency_graph:
  requires: [12-01]
  provides: [stage-aware-assembleMessages, Strategy-vars, microPassFile-in-retrieval]
  affects: [internal/retrieval/retrieval.go, cmd/helpers.go]
tech_stack:
  added: []
  patterns: [stage-ordered-context-assembly, per-stage-budget-enforcement, strategy-pattern]
key_files:
  modified:
    - internal/retrieval/retrieval.go
decisions:
  - Tasks 1-3 committed as single unit because the package would not compile with the old assembleMessages signature while the new test calls expected the new signature — atomicity required
  - microPassFile left in cmd/helpers.go as dead code per plan; Plan 03 removes it
  - PatternStrategy short-circuit placed before relevance gate to avoid LLM call overhead for near-zero context strategies
metrics:
  duration: "2m 5s"
  completed: "2026-04-10T01:43:23Z"
  tasks_completed: 3
  files_modified: 1
---

# Phase 12 Plan 02: Stage-Aware assembleMessages and Strategy Variables Summary

TDD GREEN pass — implement stage-aware assembleMessages with per-stage budget enforcement, four exported Strategy variables, and microPassFile moved into the retrieval package.

## What Was Built

- **Four exported Strategy variables** (`PlanStrategy`, `StarterStrategy`, `LookupStrategy`, `PatternStrategy`) with exact field values per CTX-04
- **Stage-aware `assembleMessages`** (8-parameter signature): assembles context in order — project summary → symbol matches → file list → conditional file content expansion; single `usedTokens` counter enforces budget across all stages
- **microPassRe and microPassFile** copied verbatim from `cmd/helpers.go` into `internal/retrieval/retrieval.go` with `regexp` import added
- **PatternStrategy short-circuit** in `BuildContext` — skips all LLM calls for near-zero strategies (MaxTokenRatio ≤ 0.10, no symbols, no files)
- **BuildContext call site** updated to pass `proj`, `strategy`, `cfg`, `chatFn` to `assembleMessages`

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | Define four exported Strategy variables | 583c257 | internal/retrieval/retrieval.go |
| 2 | Move microPassFile + microPassRe into retrieval.go | 583c257 | internal/retrieval/retrieval.go |
| 3 | Implement stage-aware assembleMessages and update BuildContext | 583c257 | internal/retrieval/retrieval.go |

## Test Results

All 7 new tests from Plan 01 now GREEN:
- `TestStrategy_Plan` PASS
- `TestStrategy_Starter` PASS
- `TestStrategy_Lookup` PASS
- `TestStrategy_Pattern` PASS
- `TestAssembleMessages_StageOrder` PASS
- `TestAssembleMessages_BudgetStop_Symbols` PASS
- `TestAssembleMessages_BudgetStop_ProjectSummary` PASS

All pre-existing retrieval tests also PASS. Full project build (`go build ./...`) clean.

## Deviations from Plan

### Atomic commit for Tasks 1-3

Tasks 1, 2, and 3 were committed together in one commit (583c257) rather than three separate commits. The reason: the test file from Plan 01 already calls `assembleMessages` with the new 8-parameter signature. The package could not compile — and thus no tests could pass — until `assembleMessages` had the new signature. Task 1 (strategy vars) and Task 2 (microPassFile) needed to be in place before Task 3 could be verified. Committing all three together was the only way to produce a compilable, green state.

## Known Stubs

None.

## Threat Flags

None — no new network endpoints, auth paths, or trust boundary changes introduced. The T-12-02-02 mitigation (microPassRe digit-range parsing with line clamping) is present as specified.

## Self-Check: PASSED

- `internal/retrieval/retrieval.go` — FOUND (583c257 verified via `git log`)
- `go test ./internal/retrieval/... -count=1` exits 0
- All 4 Strategy var declarations confirmed at lines 46, 54, 62, 70
- `func microPassFile` confirmed at line 624
- `func assembleMessages` confirmed at line 442 with 8-parameter signature
- `assembleMessages(query, proj` call sites confirmed at lines 106 and 138
