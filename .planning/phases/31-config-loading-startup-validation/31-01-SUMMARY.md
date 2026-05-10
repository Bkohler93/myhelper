---
phase: 31-config-loading-startup-validation
plan: 01
subsystem: config
tags: [go, config, validation]

requires: []
provides:
  - "config.Load() returns empty string for Model and Endpoint when no config file and no env vars are set"
  - "DefaultEndpoint and DefaultModel constants removed from config.go"
  - "CFG-01 and CFG-02 test coverage in config_test.go"
affects:
  - 31-02
  - setup-wizard
  - any command that calls config.Load() and validates cfg.Model/cfg.Endpoint

tech-stack:
  added: []
  patterns:
    - "Zero-value Config struct — Model and Endpoint are empty strings by default; callers validate after Load()"

key-files:
  created: []
  modified:
    - internal/config/config.go
    - internal/config/config_test.go

key-decisions:
  - "Remove DefaultEndpoint and DefaultModel constants rather than keep them for backward compat — allows Wave 2 validateConfig() to reliably detect absent config"
  - "Retain DefaultTokenThreshold = 4100 — token budget has a sensible default; model/endpoint do not"

patterns-established:
  - "Config zero-value pattern: Load() seeds only fields with stable defaults; callers validateConfig() after Load() to detect missing required fields"

requirements-completed: [CFG-01, CFG-02]

duration: 10min
completed: 2026-05-10
---

# Phase 31 Plan 01: Config Loading Startup Validation Summary

**config.Load() now returns empty strings for Model and Endpoint when no config file or env vars are present, enabling Wave 2 validateConfig() to detect missing config reliably**

## Performance

- **Duration:** ~10 min
- **Started:** 2026-05-10T14:01:00Z
- **Completed:** 2026-05-10T14:11:48Z
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments
- Deleted `DefaultEndpoint` and `DefaultModel` constants from config.go const block
- Removed `Endpoint: DefaultEndpoint` and `Model: DefaultModel` pre-seeds from `Load()` struct literal
- Added subtest "model and endpoint are empty when no config or env set" to `TestLoad`, covering CFG-01 and CFG-02
- All 6 `TestLoad` subtests pass; `go build ./...` exits cleanly

## Task Commits

Each task was committed atomically:

1. **Task 1: Remove hardcoded defaults from config.Load() and delete orphaned constants** - `d619ef8` (feat)
2. **Task 2: Add CFG-01 and CFG-02 test cases to config_test.go** - `4ca6611` (test)

## Files Created/Modified
- `internal/config/config.go` - Removed DefaultEndpoint/DefaultModel constants and pre-seeds from Load()
- `internal/config/config_test.go` - Added CFG-01/CFG-02 subtest inside TestLoad

## Decisions Made
- Retained `DefaultTokenThreshold = 4100` — token budget has a sensible default; model and endpoint do not (they are deployment-specific)
- Deleted constants entirely rather than retaining as doc-only references — no external callers exist, deletion is cleanest

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None

## Threat Flags

None — changes reduce surface area (fail-fast on missing config vs. silently connecting to a hardcoded address). No new network endpoints, auth paths, or schema changes introduced.

## Known Stubs

None.

## Next Phase Readiness
- Wave 2 (plan 31-02) can now implement `validateConfig(cfg)` — `cfg.Model == ""` and `cfg.Endpoint == ""` reliably indicate absent config
- No blockers

---
*Phase: 31-config-loading-startup-validation*
*Completed: 2026-05-10*
