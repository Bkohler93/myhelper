---
phase: 23-cleanup-and-correctness
plan: 01
subsystem: retrieval
tags: [search, retrieval, dead-code, bugfix, go]

# Dependency graph
requires:
  - phase: 20-gap-closure-srch-04
    provides: SearXNG search pipeline with num_results=10 param
provides:
  - BUG-01 fix: trailing slash stripped before SearXNG URL concat
  - BUG-02 fix: llmReRank errors surfaced via named variable with explicit fallback
  - CLN-01: countTokens duplicate deleted from cmd/search.go
  - CLN-02: unused pkgs parameter removed from llmReRank signature
  - CLN-03: CallEdges/TypeRefs comments updated to "reserved for future" wording
affects: [retrieval, search, scanner]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Inline token counting via history.New(...).TokenCount() — no local helper aliases"
    - "Named error variables for all internal LLM calls — no silent _ discard"

key-files:
  created: []
  modified:
    - internal/search/search.go
    - internal/retrieval/retrieval.go
    - internal/retrieval/retrieval_test.go
    - internal/scanner/ast.go
    - cmd/search.go

key-decisions:
  - "pkgs variable in loadArtifacts replaced with _ in both BuildContext and BuildInspectContext — it was only passed to llmReRank which no longer accepts it"
  - "Test call sites for llmReRank updated to match new signature — pre-existing test coverage preserved"

patterns-established:
  - "All llmReRank callers use named reRankErr with explicit fallback to candidates"

requirements-completed: [BUG-01, BUG-02, CLN-01, CLN-02, CLN-03]

# Metrics
duration: 2min
completed: 2026-04-24
---

# Phase 23 Plan 01: Cleanup and Correctness Summary

**Two bug fixes and three dead-code removals: double-slash URL prevention, silent llmReRank error surfacing, and deletion of duplicate countTokens helper**

## Performance

- **Duration:** 2 min
- **Started:** 2026-04-24T00:46:18Z
- **Completed:** 2026-04-24T00:48:34Z
- **Tasks:** 3
- **Files modified:** 5

## Accomplishments
- BUG-01: `strings.TrimRight(endpoint, "/")` prevents double-slash path when SearXNG endpoint has trailing slash
- BUG-02: Both `llmReRank` call sites now use `reRankErr` named variable with explicit `selected = candidates` fallback
- CLN-01/CLN-02: `countTokens` deleted; `pkgs []scanner.PackageEntry` removed from `llmReRank` signature; both callers updated
- CLN-03: `CallEdges`/`TypeRefs` Symbol struct comments updated from "populated by Phase 10 body-walking pass" to "reserved for future ... ranking"

## Task Commits

Each task was committed atomically:

1. **Task 1: BUG-01 — strip trailing slash in search.go** - `104eb4a` (fix)
2. **Task 2: BUG-02 + CLN-02 — surface llmReRank errors, remove pkgs param** - `572fc79` (fix)
3. **Task 3: CLN-01 + CLN-03 — delete countTokens, update reserved comments** - `25cfda4` (fix)

**Plan metadata:** (docs commit)

## Files Created/Modified
- `internal/search/search.go` - BUG-01: TrimRight before URL concat
- `internal/retrieval/retrieval.go` - BUG-02 + CLN-02: named error var, removed pkgs param from llmReRank, _ in loadArtifacts returns
- `internal/retrieval/retrieval_test.go` - Updated llmReRank test call sites to match new signature
- `internal/scanner/ast.go` - CLN-03: reserved comments on CallEdges/TypeRefs
- `cmd/search.go` - CLN-01: deleted countTokens, inlined history.New(...).TokenCount()

## Decisions Made
- Used `_` for `pkgs` in `loadArtifacts` return assignment in both `BuildContext` and `BuildInspectContext` — the variable was only used as an argument to `llmReRank` which no longer accepts it; it is not referenced elsewhere in those functions

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] pkgs variable became unused after llmReRank signature change**
- **Found during:** Task 2 (BUG-02 + CLN-02)
- **Issue:** After removing `pkgs []scanner.PackageEntry` from `llmReRank`, the `pkgs` variable from `loadArtifacts` was declared but not used, causing a compile error. The plan noted "The pkgs variable is still used elsewhere in BuildContext (e.g. expandDeps call)" — but `expandDeps` only takes `files`, not `pkgs`. `pkgs` had no remaining consumers.
- **Fix:** Replaced `pkgs` with `_` in both `loadArtifacts` destructuring assignments (BuildContext line 98, BuildInspectContext line 794)
- **Files modified:** `internal/retrieval/retrieval.go`
- **Verification:** `go build ./...` passes
- **Committed in:** `572fc79` (Task 2 commit)

**2. [Rule 1 - Bug] retrieval_test.go call sites used old llmReRank signature**
- **Found during:** Task 3 (CLN-01 + CLN-03) — discovered during `go test ./...`
- **Issue:** Three test calls to `llmReRank` still passed a `nil` pkgs argument matching the old 5-param signature
- **Fix:** Removed the `nil` pkgs argument from all three test call sites
- **Files modified:** `internal/retrieval/retrieval_test.go`
- **Verification:** `go test ./internal/retrieval/...` passes
- **Committed in:** `25cfda4` (Task 3 commit)

---

**Total deviations:** 2 auto-fixed (2 Rule 1 bugs)
**Impact on plan:** Both auto-fixes were direct consequences of the planned CLN-02 signature change. No scope creep.

## Issues Encountered
- `internal/planner` TestParsePlan failure (references missing fixture file `14-01-PLAN.md`) — pre-existing, unrelated to this plan's changes. Logged as out-of-scope.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- All five requirements (BUG-01, BUG-02, CLN-01, CLN-02, CLN-03) satisfied
- `go build ./...` clean
- Retrieval tests pass
- Ready for remaining v3.2 work (inspect command wiring, spinners, dual context injection fix)

---
*Phase: 23-cleanup-and-correctness*
*Completed: 2026-04-24*
