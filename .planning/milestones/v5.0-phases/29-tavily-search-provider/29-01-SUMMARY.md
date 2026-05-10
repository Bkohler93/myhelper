---
phase: 29-tavily-search-provider
plan: 01
subsystem: search
tags: [go, tavily, searxng, http-client, provider-dispatch, config-extension]

# Dependency graph
requires: []
provides:
  - Tavily HTTP client in internal/search using POST + Bearer token auth
  - Provider-dispatching Search function routing by cfg.Provider
  - Extended Config struct with Provider, TavilyKey, TavilyEndpoint fields
  - LoadConfig auto-selection: "tavily" when key present, "searxng" otherwise
  - MYHELPER_TAVILY_KEY env var support in LoadConfig
  - DefaultTavilyEndpoint constant (https://api.tavily.com/search)
  - httptest-based Tavily tests covering POST method, Bearer auth, result filtering, 401 error
  - Provider dispatch tests and env-var override tests
affects: [30-setup-wizard]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - Provider dispatch via cfg.Provider string field (no interface needed for two providers)
    - TavilyEndpoint config field as test-override URL (same pattern as Config.Endpoint for SearXNG)
    - POST + Bearer token auth using http.NewRequest + http.DefaultClient.Do
    - bytes.NewReader for JSON request body in HTTP POST

key-files:
  created: []
  modified:
    - internal/search/search.go
    - internal/search/search_test.go

key-decisions:
  - "Existing Search renamed to searxngSearch (unexported); new Search is pure dispatcher — no call-site changes needed"
  - "TavilyEndpoint added to Config for test overridability, defaults to DefaultTavilyEndpoint when empty"
  - "Auto-selection runs after env var application so MYHELPER_TAVILY_KEY still triggers tavily auto-select"
  - "No new dependencies — stdlib net/http handles POST + Bearer auth in ~30 lines"
  - "JSON tags search_provider and tavily_key match Phase 30 Setup Wizard config write conventions"

patterns-established:
  - "Provider dispatch: if cfg.Provider == 'tavily' { return tavilySearch } return searxngSearch"
  - "Config field as endpoint override for test isolation (TavilyEndpoint pattern)"

requirements-completed:
  - SRCH-01
  - SRCH-02
  - SRCH-03

# Metrics
duration: 3min
completed: 2026-05-10
---

# Phase 29 Plan 01: Tavily Search Provider Summary

**Tavily search provider added to internal/search via POST+Bearer dispatch; SearXNG unchanged and backward-compatible**

## Performance

- **Duration:** 3 min
- **Started:** 2026-05-10T04:52:30Z
- **Completed:** 2026-05-10T04:54:48Z
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments
- Extended `Config` struct with `Provider`, `TavilyKey`, `TavilyEndpoint` fields and correct JSON tags for Phase 30 wizard compatibility
- Added `tavilySearch` function using POST + `Authorization: Bearer` header; existing `Search` renamed to `searxngSearch` and a new dispatching `Search` added
- Extended `LoadConfig` to read `MYHELPER_TAVILY_KEY` env var and auto-select `"tavily"` provider when key is present
- Added 11 new test cases across `TestTavilySearch`, `TestSearch_ProviderDispatch`, and `TestLoadConfig_TavilyKeyEnvVar`; full test suite green

## Task Commits

Each task was committed atomically:

1. **Task 1: Extend search.go — Config, LoadConfig, tavilySearch, dispatcher** - `43a0a01` (feat)
2. **Task 2: Extend search_test.go — Tavily tests, provider dispatch, env-var override** - `928e7ff` (feat)

**Plan metadata:** _(committed with this SUMMARY)_

## Files Created/Modified
- `internal/search/search.go` - Added Provider/TavilyKey/TavilyEndpoint fields to Config, extended LoadConfig, added tavilySearch and dispatching Search, added bytes import and DefaultTavilyEndpoint constant
- `internal/search/search_test.go` - Added TestTavilySearch, TestSearch_ProviderDispatch, TestLoadConfig_TavilyKeyEnvVar; added "strings" import

## Decisions Made
- No new dependencies: stdlib `net/http` + `encoding/json` + `bytes` cover the Tavily POST pattern — consistent with project minimal-dependency philosophy
- `TavilyEndpoint` field in Config enables httptest override without package-level variables or function options
- Auto-selection logic placed after env var application so env-supplied key correctly triggers `"tavily"` provider
- JSON tags `search_provider` and `tavily_key` chosen to match Phase 30 Setup Wizard config conventions

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None.

## User Setup Required
None - no external service configuration required at this stage. Users configure Tavily via `MYHELPER_TAVILY_KEY` env var or `tavily_key` in `~/.config/myhelper/config.json`.

## Next Phase Readiness
- `search.Config` is ready for Phase 30 Setup Wizard to write `tavily_key` and `search_provider` to config
- `search.LoadConfig` correctly reads both fields from config files and env vars
- Backward compatible: users with no Tavily key continue to use SearXNG unchanged

## Self-Check: PASSED
- `internal/search/search.go` exists and compiles
- `internal/search/search_test.go` exists with all new test functions
- Task 1 commit `43a0a01` present in git log
- Task 2 commit `928e7ff` present in git log
- `go test ./...` exits 0

---
*Phase: 29-tavily-search-provider*
*Completed: 2026-05-10*
