---
phase: 15-plan-parser
plan: "02"
subsystem: planner
tags: [discover, plan-finder, directory-scan, tdd]
dependency_graph:
  requires: [internal/planner.ParsePlan]
  provides: [internal/planner.FindActivePlan]
  affects: [phase-18-execute-command]
tech_stack:
  added: []
  patterns: [os.ReadDir directory scan, os.Chdir in tests for CWD-relative paths, integer sort via sort.Slice]
key_files:
  created:
    - internal/planner/discover.go
    - internal/planner/discover_test.go
  modified: []
decisions:
  - "Hardcoded phasesDir constant over parameter — FindActivePlan takes no args; callers always want .planning/phases/ (consistent with ParsePlan's path-in-arg approach)"
  - "Two-pass inner loop (hasSummary check then planFile search) — clear separation of concerns; avoids early-exit complexity"
  - "os.Chdir in tests over parameterized path — matches plan spec; avoids adding a path parameter to the exported function signature"
metrics:
  duration: "< 5 minutes"
  completed: "2026-04-10"
  tasks_completed: 2
  tasks_total: 2
  files_created: 2
  files_modified: 0
---

# Phase 15 Plan 02: FindActivePlan Directory Scanner Summary

## One-Liner

os.ReadDir-based active-plan discovery scanning .planning/phases/ with integer-sorted phase directory enumeration and SUMMARY.md completion detection.

## What Was Built

Added `FindActivePlan() (string, error)` to the `internal/planner` package in a new file `discover.go`:

- `const phasesDir = ".planning/phases"` — hardcoded relative path consumed by the execute command
- `FindActivePlan` scans the phases directory, filters to directories with numeric prefixes (via `strconv.Atoi`), sorts descending by integer value, then walks from highest to lowest looking for the first directory with no `*-SUMMARY.md` file
- Returns `filepath.Join(phasesDir, dirName, planFile)` — path directly usable as argument to `ParsePlan`
- Error messages match the exact format specified: "find active plan: read phases dir: %w", "no active phase found in .planning/phases/", "no PLAN.md in active phase dir %s"

`TestFindActivePlan` suite in `discover_test.go` with 5 subtests:
- Happy path: highest non-complete dir selected and path returned correctly
- All-complete: returns "no active phase found" error
- No phases dir: returns "read phases dir" error
- No PLAN.md in active dir: returns "no PLAN.md" error
- Numeric sort validation: `10-higher` chosen over `9-lower` (integer, not lexicographic)

## Tasks

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | Implement FindActivePlan directory scanner | d5e913c | internal/planner/discover.go |
| 2 | Write TestFindActivePlan suite with temp directory fixtures | 8a4a7bd | internal/planner/discover_test.go |

## Verification

```
go test ./internal/planner/... -v -run TestFindActivePlan
--- PASS: TestFindActivePlan (0.00s)
    --- PASS: TestFindActivePlan/returns_path_for_highest-numbered_dir_without_SUMMARY.md
    --- PASS: TestFindActivePlan/skips_completed_phases_(all_have_SUMMARY.md)
    --- PASS: TestFindActivePlan/returns_error_when_phases_dir_does_not_exist
    --- PASS: TestFindActivePlan/returns_error_when_active_dir_has_no_PLAN.md
    --- PASS: TestFindActivePlan/numeric_sort:_9-foo_comes_before_10-bar_(10_is_higher)

go test ./internal/planner/... → all planner tests PASS (TestParsePlan + TestFindActivePlan)
go build ./...  → exit 0
go test ./...   → all packages PASS
git diff go.mod → (no changes)
```

## Deviations from Plan

None — plan executed exactly as written.

## Known Stubs

None — FindActivePlan fully implemented with real filesystem I/O.

## Threat Flags

None — no new network endpoints, auth paths, or schema changes. FindActivePlan reads local filesystem only (name-only inspection, no file content opened).

## Self-Check: PASSED

- [x] internal/planner/discover.go exists and contains `func FindActivePlan() (string, error)`
- [x] internal/planner/discover.go contains `const phasesDir = ".planning/phases"`
- [x] discover.go has 2 `strings.HasSuffix` calls (SUMMARY.md and PLAN.md)
- [x] internal/planner/discover_test.go exists and contains `func TestFindActivePlan`
- [x] Commit d5e913c exists (Task 1)
- [x] Commit 8a4a7bd exists (Task 2)
- [x] All 5 TestFindActivePlan subtests PASS
- [x] All 7 TestParsePlan subtests still PASS (no regressions)
- [x] go build ./... exits 0
- [x] go test ./... exits 0
- [x] go.mod unchanged
