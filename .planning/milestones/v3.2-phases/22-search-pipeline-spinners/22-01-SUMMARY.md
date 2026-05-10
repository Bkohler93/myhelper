---
phase: 22-search-pipeline-spinners
plan: 01
subsystem: ui
tags: [spinner, stderr, goroutine, ticker, ux]

# Dependency graph
requires:
  - phase: 19-web-search-pipeline
    provides: buildUserMessage, searchGate, search.Search, reRankResults call sites in cmd/search.go
provides:
  - spinner type with goroutine-based 100ms ticker loop writing to stderr
  - startSpinner helper that clears the terminal line on done()
  - Three wired call sites in buildUserMessage: gate, fetch, re-rank
affects: [23-cleanup-correctness]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Inline spinner wrap: sp := startSpinner(...); result = fn(); sp.done() — no defer, clears at call site"

key-files:
  created: []
  modified:
    - cmd/search.go

key-decisions:
  - "No external dependencies — spinner uses only fmt, os, strings, time (stdlib)"
  - "Spinner placed in cmd/search.go alongside search helpers, not a separate file"
  - "sp.done() called immediately after blocking call, not via defer, to clear at call site"

patterns-established:
  - "startSpinner/done wrap pattern: start before call, done immediately after return, before any conditional on result"

requirements-completed: [UX-01, UX-02, UX-03]

# Metrics
duration: 1min
completed: 2026-04-24
---

# Phase 22 Plan 01: Search Pipeline Spinners Summary

**Goroutine-based terminal spinner wired at all three silent-wait points in the search pipeline (gate LLM call, SearXNG fetch, re-rank LLM call) using stdlib only**

## Performance

- **Duration:** 1 min
- **Started:** 2026-04-24T19:47:08Z
- **Completed:** 2026-04-24T19:48:21Z
- **Tasks:** 2
- **Files modified:** 1

## Accomplishments
- Added `spinner` struct, `startSpinner` goroutine (100ms ticker, 4-frame charset), and `done()` to cmd/search.go using only stdlib (`fmt`, `os`, `strings`, `time`)
- Wired sp1/sp2/sp3 at the three blocking wait points in `buildUserMessage` so users see visual feedback instead of a silent 1-3 second block
- Confirmed go.mod unchanged — zero new dependencies introduced

## Task Commits

Each task was committed atomically:

1. **Task 1: Add spinner type and startSpinner helper** - `6ba67bb` (feat)
2. **Task 2: Wire spinners at all three wait points** - `fe2a9eb` (feat)

**Plan metadata:** TBD (docs: complete plan)

## Files Created/Modified
- `cmd/search.go` - Added spinner type + startSpinner + done, wired sp1/sp2/sp3 in buildUserMessage

## Decisions Made
- Kept spinner implementation in cmd/search.go (not a new file) per plan guidance to keep search-layer helpers co-located
- Used inline `sp.done()` (not defer) so each spinner clears at its own call site rather than at function return
- No external packages — Bubble Tea listed in PROJECT.md as target but plan explicitly forbids it for this phase; stdlib ticker is sufficient

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

Pre-existing `internal/planner` test failures (`TestParsePlan`) reference a missing phase-14 plan file — confirmed pre-existing before this plan's changes, not introduced here.

## User Setup Required

None - no external service configuration required.

## Threat Surface Scan

No new network endpoints, auth paths, file access patterns, or schema changes introduced. Spinner writes only hardcoded label strings to stderr — T-22-01 and T-22-02 from plan's threat register remain accepted as specified.

## Self-Check: PASSED

- cmd/search.go: FOUND
- 22-01-SUMMARY.md: FOUND
- Commit 6ba67bb: FOUND
- Commit fe2a9eb: FOUND

## Next Phase Readiness
- Phase 22 complete; cmd/search.go spinners fully functional
- Phase 23 (Cleanup & Correctness) can proceed: BUG-01, BUG-02, CLN-01, CLN-02, CLN-03, CTX-03, PERF-01

---
*Phase: 22-search-pipeline-spinners*
*Completed: 2026-04-24*
