---
phase: 12-adaptive-context-builder-strategies
plan: "01"
subsystem: retrieval
tags: [tdd, red-phase, context-assembly, strategy]
dependency_graph:
  requires: []
  provides: [failing-tests-for-12-02]
  affects: [internal/retrieval/retrieval_test.go]
tech_stack:
  added: []
  patterns: [TDD red-green-refactor, white-box package tests]
key_files:
  created: []
  modified:
    - internal/retrieval/retrieval_test.go
decisions:
  - "Tests use white-box package access (package retrieval) to call assembleMessages directly"
  - "strings import added to support Index/Contains/Repeat calls in new tests"
  - "Compile failure is the correct RED signal — no partial test run needed"
metrics:
  duration: "4 minutes"
  completed: "2026-04-10"
  tasks_completed: 1
  tasks_total: 1
  files_changed: 1
---

# Phase 12 Plan 01: TDD RED — Staged assembleMessages and Strategy Variables Summary

Failing tests (TDD RED phase) for the stage-aware `assembleMessages` function and the four per-command `Strategy` variables (`PlanStrategy`, `StarterStrategy`, `LookupStrategy`, `PatternStrategy`) appended to `internal/retrieval/retrieval_test.go`. All 7 new tests fail to compile because `assembleMessages` has the wrong signature and the four Strategy package-level variables do not yet exist.

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | Write failing tests for staged assembleMessages and Strategy variables | da923f4 | internal/retrieval/retrieval_test.go |

## What Was Built

7 new test functions added to `internal/retrieval/retrieval_test.go`:

1. **TestAssembleMessages_StageOrder** — verifies `## Project` section appears before `### Relevant Symbols` before `### Selected Files` in assembled content
2. **TestAssembleMessages_BudgetStop_Symbols** — verifies budget gate stops symbol inclusion before all 3 symbols fit under a 10-token threshold
3. **TestAssembleMessages_BudgetStop_ProjectSummary** — verifies project summary excluded when budget (80% of 10 tokens) is too small for the ~100-token summary
4. **TestStrategy_Plan** — asserts `PlanStrategy.Name="plan"`, `UseSymbols=false`, `UseFiles=false`, `MaxTokenRatio=0.50`
5. **TestStrategy_Starter** — asserts `StarterStrategy.Name="starter"`, `UseSymbols=true`, `UseFiles=true`, `MaxTokenRatio=0.80`
6. **TestStrategy_Lookup** — asserts `LookupStrategy.Name="lookup"`, `UseSymbols=true`, `UseFiles=false`, `MaxTokenRatio=0.30`
7. **TestStrategy_Pattern** — asserts `PatternStrategy.Name="pattern"`, `UseSymbols=false`, `UseFiles=false`, `MaxTokenRatio=0.10`

## RED State Confirmation

```
# github.com/bkohler93/myhelper/internal/retrieval [build failed]
internal/retrieval/retrieval_test.go:366:62: too many arguments in call to assembleMessages
internal/retrieval/retrieval_test.go:430:5: undefined: PlanStrategy
internal/retrieval/retrieval_test.go:445:5: undefined: StarterStrategy
...
FAIL github.com/bkohler93/myhelper/internal/retrieval [build failed]
```

Compile failure confirms RED state. Plan 02 will make these tests green by:
- Changing `assembleMessages` signature to accept `proj scanner.ProjectArtifact`, `strategy Strategy`, `cfg config.Config`, `chatFn scanner.ChatFn`
- Adding `PlanStrategy`, `StarterStrategy`, `LookupStrategy`, `PatternStrategy` package-level variables

## Deviations from Plan

None - plan executed exactly as written.

## Known Stubs

None - this is a test-only plan; no production stubs introduced.

## Threat Flags

No new security surface introduced. Test files are excluded from production binary.

## Self-Check: PASSED

- internal/retrieval/retrieval_test.go: FOUND
- Commit da923f4: FOUND (git log confirms)
- 7 test function matches confirmed via grep
