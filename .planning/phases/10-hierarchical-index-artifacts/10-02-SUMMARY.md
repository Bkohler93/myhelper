---
phase: 10-hierarchical-index-artifacts
plan: "02"
subsystem: cmd
tags: [artifacts, init, sync, helpers, tdd, error-handling]
dependency_graph:
  requires:
    - internal/scanner/artifacts.go (BuildArtifacts, ErrStaleFlatIndex)
    - cmd/init.go (runInit flow)
    - cmd/sync.go (runSync flow)
    - cmd/helpers.go (readIndexFile, buildInjectedMessages)
  provides:
    - init calls BuildArtifacts after Scan
    - sync calls BuildArtifacts after deltaSummaries
    - readIndexFile returns ErrStaleFlatIndex when project.json exists
    - buildInjectedMessages treats ErrStaleFlatIndex as non-fatal fallback to injectSummaries
  affects:
    - Phase 11 (retrieval package) — can now rely on artifact files being present after init/sync
tech_stack:
  added: []
  patterns:
    - TDD (RED then GREEN — test added before implementation)
    - errors.Is sentinel pattern for ErrStaleFlatIndex
    - os.Stat check before read for stale-detection
key_files:
  created: []
  modified:
    - cmd/init.go (BuildArtifacts call after scanner.Scan)
    - cmd/sync.go (BuildArtifacts call after deltaSummaries)
    - cmd/helpers.go (ErrStaleFlatIndex detection in readIndexFile; fallback in buildInjectedMessages)
    - cmd/helpers_test.go (TestReadIndexFile_StaleFlatIndex with 2 subtests)
decisions:
  - "ErrStaleFlatIndex treated identically to os.ErrNotExist in buildInjectedMessages — silent fallback to injectSummaries, no user-visible error (T-10-07 mitigated)"
  - "Test written in RED state before implementation (TDD per plan requirement)"
  - "Task 2 test (TestReadIndexFile_StaleFlatIndex) written as part of Task 1 TDD RED/GREEN cycle — both committed together"
metrics:
  duration_minutes: 8
  completed_date: "2026-04-09"
  tasks_completed: 2
  files_changed: 4
requirements:
  - IDX-01
  - IDX-06
---

# Phase 10 Plan 02: Wire BuildArtifacts into init/sync and add ErrStaleFlatIndex Summary

**One-liner:** `myhelper init` and `myhelper sync` now produce all four artifact files on every run; query commands fall back to summaries gracefully via ErrStaleFlatIndex sentinel.

## What Was Built

Wired `scanner.BuildArtifacts` into the two mutation commands (`init` and `sync`) and updated `readIndexFile` / `buildInjectedMessages` in `helpers.go` to detect and handle the stale-flat-index condition gracefully.

**cmd/init.go:** Added Step 1b between scanner.Scan and generateContextMD — calls `scanner.BuildArtifacts(root, cfg, ollama.Chat)`. Progress label "Building artifact index..." shown in spinner.

**cmd/sync.go:** Added Step 3b between deltaSummaries and generateContextMD — calls `scanner.BuildArtifacts(root, cfg, ollama.Chat)`. Progress label "Updating artifact index..." shown in spinner.

**cmd/helpers.go readIndexFile:** Added os.Stat check for `project.json` at the top of the function. When the file exists (new artifact files present), returns `scanner.ErrStaleFlatIndex` immediately without reading index.json.

**cmd/helpers.go buildInjectedMessages:** Extended error handling to treat `errors.Is(err, scanner.ErrStaleFlatIndex)` as a non-fatal fallback to `injectSummaries` — identical behavior to `os.ErrNotExist`. No error message printed to user (T-10-07 mitigated).

**cmd/helpers_test.go:** Added `TestReadIndexFile_StaleFlatIndex` with two subtests:
- `returns_stale_error_when_project_json_exists` — creates tmpDir/.myhelper/project.json, asserts ErrStaleFlatIndex returned
- `returns_not_exist_when_no_files` — empty tmpDir, asserts error is NOT ErrStaleFlatIndex

## Tasks

### Task 1 (RED + GREEN) — commit 4b06303
Added `TestReadIndexFile_StaleFlatIndex` test (RED: first subtest failed, implementation not yet present). Implemented all four changes across init.go, sync.go, and helpers.go. GREEN: both subtests pass, `go test ./...` exits 0.

### Task 2 — no separate commit
`TestReadIndexFile_StaleFlatIndex` was written as part of Task 1's TDD RED/GREEN cycle. All acceptance criteria for Task 2 met within Task 1's commit.

## Verification

```
go test ./cmd/... -run TestReadIndexFile_StaleFlatIndex -v  -> 2/2 PASS
go test ./...                                              -> all packages OK
grep -n "BuildArtifacts" cmd/init.go                       -> 1 match (line 49)
grep -n "BuildArtifacts" cmd/sync.go                       -> 1 match (line 72)
grep -n "ErrStaleFlatIndex" cmd/helpers.go                 -> 3 matches (comment + readIndexFile + buildInjectedMessages)
go build ./...                                             -> exit 0
```

## Deviations from Plan

### Auto-fixed Issues

None — plan executed exactly as written.

**Note:** Tasks 1 and 2 were committed together since Task 2's test was written as part of Task 1's TDD RED/GREEN cycle. The test exists in the same commit as the implementation (4b06303). This is consistent with TDD — the test was written first (RED verified), then implementation made it GREEN.

## Known Stubs

None — all code paths connect to real implementations (scanner.BuildArtifacts, injectSummaries).

## Threat Flags

No new threat surface introduced. T-10-07 (ErrStaleFlatIndex exposed to user) mitigated: the error is handled silently in buildInjectedMessages with fallback to injectSummaries — no message printed to user.

## Self-Check: PASSED

- cmd/init.go BuildArtifacts call: FOUND (line 49)
- cmd/sync.go BuildArtifacts call: FOUND (line 72)
- cmd/helpers.go ErrStaleFlatIndex in readIndexFile: FOUND (line 315)
- cmd/helpers.go ErrStaleFlatIndex in buildInjectedMessages: FOUND (line 481)
- cmd/helpers_test.go TestReadIndexFile_StaleFlatIndex: FOUND (line 612)
- commit 4b06303: FOUND
- go test ./cmd/... -run TestReadIndexFile_StaleFlatIndex: 2/2 PASS
- go test ./...: all packages PASS
- go build ./...: exit 0
