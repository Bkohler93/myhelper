---
plan: 20-01
phase: 20-fix-srch-04
title: Fix SRCH-04 — Result Count Param
status: complete
completed: 2026-04-11
duration: ~10m
tasks_completed: 4
tasks_total: 4
files_created: 0
files_modified: 4
key-decisions:
  - Use num_results=10 as the SearXNG count parameter (hardcoded constant, no security surface)
  - Fix searchGate comment to accurately say "fails open" not "Fails CLOSED" (behavior was always open)
requires: []
provides:
  - SRCH-04 satisfied: num_results=10 in all SearXNG requests
  - TestSearch_RequestParams/result_count_present subtest
affects:
  - internal/search/search.go
  - internal/search/search_test.go
  - cmd/search.go
  - .planning/phases/18-searxng-client/18-VERIFICATION.md
tech-stack:
  added: []
  patterns: []
key-files:
  modified:
    - internal/search/search.go
    - internal/search/search_test.go
    - cmd/search.go
    - .planning/phases/18-searxng-client/18-VERIFICATION.md
---

# Phase 20 Plan 01: Fix SRCH-04 — Result Count Param Summary

**One-liner:** Added `num_results=10` to SearXNG request URL, verified with dedicated subtest, fixed misleading GATE-02 comment, and updated Phase 18 VERIFICATION.md to 6/6.

## What Was Done

Closed the one remaining v3.1 audit gap (SRCH-04) with four targeted changes:

1. **`internal/search/search.go`** — Appended `&num_results=10` to the SearXNG request URL. The URL is now: `endpoint + "/search?q=" + url.QueryEscape(query) + "&format=json&pageno=1&num_results=10"`

2. **`internal/search/search_test.go`** — Added `TestSearch_RequestParams/result_count_present` subtest to `TestSearch_RequestParams`. The subtest spins up an httptest server, asserts `num_results` param is present and equals `"10"`, and passes cleanly.

3. **`cmd/search.go`** — Fixed the `searchGate` comment from `"Fails CLOSED: returns false on any LLM error (GATE-02)."` to `"fails open (search skipped on error) — GATE-02."` The comment was factually wrong: returning `false` on LLM error means search is skipped (open degradation), not blocked.

4. **`.planning/phases/18-searxng-client/18-VERIFICATION.md`** — Updated body status to `passed`, Truth #6 to `VERIFIED`, score to `6/6`, SRCH-04 requirement to `SATISFIED`, replaced Gaps Summary with a closure note, and added a num_results behavioral spot-check row.

## Verification Results

All search tests pass (12/12 including new subtest):
```
go test ./internal/search/... -v  -> PASS (all subtests)
go build ./...                    -> exit 0
grep "num_results" search.go      -> line found
grep "fails open" cmd/search.go   -> line found
```

## Commits

| Commit  | Description |
|---------|-------------|
| cb5d5bc | fix(search): add num_results=10 param, fix GATE-02 comment (SRCH-04) |
| 11f7d88 | docs(20-01): update Phase 18 VERIFICATION.md — SRCH-04 gap closed (6/6) |

## Deviations from Plan

None — plan executed exactly as written. All four tasks completed in order with no surprises.

## Known Stubs

None.

## Threat Flags

None — `num_results=10` is a hardcoded constant appended to an existing URL parameter string. No new network endpoints, auth paths, or trust boundaries introduced.

## Self-Check: PASSED

- internal/search/search.go contains `num_results`: confirmed
- internal/search/search_test.go contains `result_count_present`: confirmed
- cmd/search.go contains `fails open`: confirmed
- Commit cb5d5bc exists in worktree
- Commit 11f7d88 exists in main repo
