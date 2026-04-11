---
phase: 18-searxng-client
plan: 01
subsystem: api
tags: [searxng, http-client, search, go, stdlib, httptest]

# Dependency graph
requires: []
provides:
  - internal/search package with Config, LoadConfig(), Result, Search(), DefaultSearchEndpoint
  - SearXNG HTTP client with scheme normalization and url.QueryEscape query injection
  - Config resolution: default → home config → local config → MYHELPER_SEARCH_ENDPOINT env var
affects:
  - 19-search-gate (consumes search.Search and search.LoadConfig)

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "SearXNG JSON client mirrors ollama client pattern: scheme normalization, non-200 error with body, io.ReadAll"
    - "External test package (package search_test) for black-box testing via httptest.NewServer per subtest"
    - "Config resolution: default constant → home config file → local config file → env var (highest wins)"

key-files:
  created:
    - internal/search/search.go
    - internal/search/search_test.go
  modified: []

key-decisions:
  - "Used url.QueryEscape for q= parameter (T-18-01 mitigation) rather than url.Values to keep URL construction as a single readable string per research spec"
  - "Added minimal LoadConfig stub in Task 2 to allow test file compilation; replaced with full implementation in Task 3"
  - "Results with empty Title or URL are silently dropped; empty Snippet (content) is allowed and mapped to empty string"
  - "DefaultSearchEndpoint includes http:// scheme prefix (unlike DefaultEndpoint in config package which is bare host:port)"

patterns-established:
  - "Scheme normalization: check HasPrefix http:// or https://, prepend http:// if absent — mirrors chatURL() in ollama/client.go"
  - "Non-200 errors: io.ReadAll body, strings.TrimSpace, fmt.Errorf with status code in message"
  - "TDD Wave 0: external test package referencing undefined symbols; Wave 1: production code making tests green"

requirements-completed: [SRCH-01, SRCH-02, SRCH-03, SRCH-04, SRCH-05]

# Metrics
duration: 15min
completed: 2026-04-11
---

# Phase 18 Plan 01: SearXNG Client Summary

**Standalone SearXNG HTTP client package (internal/search) with url.QueryEscape injection, result filtering, and env/file config resolution — all five SRCH requirements satisfied via TDD**

## Performance

- **Duration:** ~15 min
- **Started:** 2026-04-11T00:00:00Z
- **Completed:** 2026-04-11T00:15:00Z
- **Tasks:** 3
- **Files modified:** 2

## Accomplishments

- Created `internal/search` package exporting Config, LoadConfig, Result, Search, DefaultSearchEndpoint
- Search() builds GET request with url.QueryEscape(query) and format=json; filters results with empty title or URL
- LoadConfig() resolves endpoint via four-layer precedence matching config.Load() pattern
- 11 test subtests covering result fields, request parameters, network errors, non-200 errors, and config resolution

## Task Commits

Each task was committed atomically:

1. **Task 1: Write failing test stubs (Wave 0 RED)** - `8aea7ae` (test)
2. **Task 2: Implement Search() and HTTP types (Wave 1 GREEN)** - `6f6adf7` (feat)
3. **Task 3: Implement LoadConfig() (Wave 1 GREEN)** - `52f8817` (feat)

## Files Created/Modified

- `internal/search/search.go` - Config, Result, DefaultSearchEndpoint, Search(), LoadConfig(), internal SearXNG JSON types, config file helpers
- `internal/search/search_test.go` - TestSearch, TestSearch_ResultFields, TestSearch_RequestParams, TestSearch_Errors, TestLoadConfig (external package search_test)

## Decisions Made

- Used `url.QueryEscape` and string concatenation for URL building (not `url.Values`) — matches research spec, easy to verify
- `DefaultSearchEndpoint = "http://192.168.0.9:8083"` includes scheme, unlike the Ollama `DefaultEndpoint` which is bare host:port
- Added minimal `LoadConfig` stub in Task 2 to unblock test compilation; full implementation in Task 3
- Config JSON key is `search_endpoint` (not `endpoint`) to avoid colliding with Ollama config keys in shared config files

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Added LoadConfig stub in Task 2 to allow test compilation**
- **Found during:** Task 2 (implement Search())
- **Issue:** `search_test.go` references `search.LoadConfig` in `TestLoadConfig`; the entire test file fails to compile without it, preventing any Task 2 tests from running
- **Fix:** Added a one-line stub `func LoadConfig() Config { return Config{Endpoint: DefaultSearchEndpoint} }` in Task 2; replaced with full implementation in Task 3
- **Files modified:** internal/search/search.go
- **Verification:** Task 2 targeted tests ran and passed; Task 3 completed the implementation and all tests pass
- **Committed in:** `6f6adf7` (Task 2 commit)

---

**Total deviations:** 1 auto-fixed (1 blocking)
**Impact on plan:** Necessary for test compilation. No scope creep — Task 3 completed LoadConfig as planned.

## Issues Encountered

- Pre-existing `TestParsePlan` failures in `internal/planner` package (missing fixture file `../../.planning/phases/14-ollama-client-extension/14-01-PLAN.md`). Confirmed pre-existing before my changes, out of scope, logged to deferred-items.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- `internal/search` package is complete and ready for Phase 19 (Search Gate & Injection)
- Phase 19 imports `search.Search(query, cfg)` and `search.LoadConfig()` directly
- No blockers or concerns

---
*Phase: 18-searxng-client*
*Completed: 2026-04-11*
