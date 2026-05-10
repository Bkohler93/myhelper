---
phase: 30-setup-wizard
plan: 01
subsystem: cli
tags: [go, wizard, ollama, hardware-detection, config-merge, http-streaming]

# Dependency graph
requires:
  - phase: 29-tavily-search
    provides: "search.Config JSON tags (tavily_key, search_provider, search_endpoint) that wizard writes"
provides:
  - "internal/wizard package with Run(r io.Reader, w io.Writer) error"
  - "checkOllama() — HTTP reachability check for Ollama service"
  - "installInstructions() — platform-specific Ollama install commands"
  - "detectMemoryMiB() — GPU VRAM (nvidia-smi) or RAM (system_profiler / /proc/meminfo)"
  - "recommendModel() — 4-tier model recommendation table by memory"
  - "pullModel() — NDJSON streaming pull via Ollama /api/pull"
  - "mergeHomeConfig() — non-destructive map-based config merge with 0600 permissions"
affects: [30-02-tests, 30-03-cobra-wiring]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Single bufio.Reader threaded through all wizard prompt steps (avoids multi-scanner buffering)"
    - "map[string]interface{} config merge (preserves keys owned by other packages)"
    - "NDJSON streaming via bufio.Scanner on HTTP response body"
    - "0600 file permissions for config files containing API keys"

key-files:
  created:
    - internal/wizard/wizard.go
  modified: []

key-decisions:
  - "Wizard is self-contained with no internal/* imports to avoid search.Config.MarshalJSON redaction of TavilyKey"
  - "Single *bufio.Reader in Run() threaded to all stages — no second Scanner or Reader over the same io.Reader"
  - "map[string]interface{} merge for config writes — typed struct unmarshal would drop unknown keys on re-marshal"
  - "T-30-03 mitigated: SearXNG endpoint validated for http:// or https:// prefix before writing to config"
  - "T-30-01 mitigated: config file written with 0600 permissions (owner-only read/write)"
  - "Wizard writes model field to config after successful pull so subsequent runs use the pulled model"

patterns-established:
  - "Wizard logic lives in internal/wizard/ (testable) separate from cobra wiring in cmd/setup.go"
  - "io.Reader/io.Writer injection for terminal wizard: Run(r io.Reader, w io.Writer) enables unit testing without OS I/O"

requirements-completed: [SETUP-01, SETUP-02, SETUP-03, SETUP-04, SETUP-05, SETUP-06]

# Metrics
duration: 12min
completed: 2026-05-10
---

# Phase 30 Plan 01: Setup Wizard Logic Summary

**Self-contained internal/wizard package implementing 5-stage interactive setup: Ollama check, hardware-aware model recommendation, streaming pull via /api/pull NDJSON, Tavily key capture, and non-destructive map-merge config write with 0600 permissions**

## Performance

- **Duration:** 12 min
- **Started:** 2026-05-10T00:00:00Z
- **Completed:** 2026-05-10T00:12:00Z
- **Tasks:** 1
- **Files modified:** 1

## Accomplishments
- Created `internal/wizard/wizard.go` with all wizard logic as a self-contained package (no internal/* imports)
- Implemented non-destructive config merge via `map[string]interface{}` — preserves all existing keys owned by config.Config and search.Config packages
- Implemented NDJSON streaming pull with percentage progress display; single bufio.Scanner on HTTP resp.Body only (separate from wizard's stdin Reader)
- Applied both threat model mitigations: 0600 file permissions (T-30-01) and SearXNG endpoint prefix validation (T-30-03)

## Task Commits

1. **Task 1: Create internal/wizard/wizard.go with all wizard logic** - `f48e97f` (feat)

**Plan metadata:** (see final commit below)

## Files Created/Modified
- `internal/wizard/wizard.go` - Complete wizard logic: Run, checkOllama, installInstructions, detectMemoryMiB, recommendModel, pullModel, mergeHomeConfig, homeConfigPath

## Decisions Made
- Wizard has no imports from `github.com/bkohler93/myhelper/*` — self-contained to prevent `search.Config.MarshalJSON` from redacting TavilyKey if config is ever serialized through that type
- Single `*bufio.Reader` at top of `Run()` threaded to all 5 stages — prevents the multi-Scanner stdin buffering pitfall documented in RESEARCH.md Pitfall 3
- Added SearXNG endpoint validation (http/https prefix check) per T-30-03 threat mitigation — not in the plan's `<action>` block but required by the `<threat_model>`

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing Critical] Added SearXNG endpoint prefix validation**
- **Found during:** Task 1 (creating wizard.go)
- **Issue:** Threat model entry T-30-03 (Tampering) required validating that the SearXNG endpoint input has `http://` or `https://` prefix. The `<action>` block did not include this validation step.
- **Fix:** Added `strings.HasPrefix` check before writing the endpoint to config; prints a warning and skips the write if the input is a bare hostname.
- **Files modified:** internal/wizard/wizard.go
- **Verification:** go build and go vet pass; logic reviewed inline
- **Committed in:** f48e97f (Task 1 commit)

---

**Total deviations:** 1 auto-fixed (Rule 2 — missing critical security mitigation from threat model)
**Impact on plan:** Security-only addition. No behavior change on valid input. Prevents bare hostname strings from being written to config as a search endpoint.

## Issues Encountered
None — plan executed cleanly. All acceptance criteria verified before commit.

## User Setup Required
None - no external service configuration required for this plan.

## Next Phase Readiness
- `internal/wizard/wizard.go` is ready for unit tests (Plan 02) and cobra wiring (Plan 03)
- All 7 exported/private functions have correct signatures for table-driven unit tests
- `Run(r io.Reader, w io.Writer)` injection pattern is test-friendly with `*strings.Reader` and `*bytes.Buffer`

## Self-Check: PASSED

- [x] `internal/wizard/wizard.go` exists and contains all required functions
- [x] `go build ./internal/wizard/...` exits 0
- [x] `go vet ./internal/wizard/...` exits 0
- [x] `bufio.NewReader` count = 1 (single reader in Run)
- [x] `bufio.NewScanner` count = 1 (HTTP resp.Body in pullModel only)
- [x] `os.WriteFile` with `0600` permissions in mergeHomeConfig
- [x] No `github.com/bkohler93/myhelper/*` imports
- [x] JSON keys: `tavily_key`, `search_provider`, `search_endpoint`, `model` all present
- [x] Commit f48e97f confirmed in git log

---
*Phase: 30-setup-wizard*
*Completed: 2026-05-10*
