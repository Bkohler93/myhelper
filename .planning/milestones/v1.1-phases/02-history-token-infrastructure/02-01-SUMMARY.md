---
phase: 02-history-token-infrastructure
plan: "01"
subsystem: config
tags: [config, cli, cobra, token-threshold, env-var]

requires: []
provides:
  - "Config.TokenThreshold field with default 4100"
  - "MYHELPER_TOKEN_LIMIT env var support in Load()"
  - "localConfigPath() returning .myhelper/config.json"
  - "--token-limit persistent CLI flag on rootCmd"
  - "ApplyFlagOverrides() helper for command RunE functions"
affects:
  - "03-conversation-loop"
  - "04-summarization"

tech-stack:
  added: []
  patterns:
    - "CLI flag override applied post-Load() via ApplyFlagOverrides() — keeps Load() side-effect free"
    - "0-as-sentinel for int flags — callers check != 0 before applying override"

key-files:
  created:
    - internal/config/config_test.go
  modified:
    - internal/config/config.go
    - cmd/root.go

key-decisions:
  - "tokenLimitFlag uses 0 as sentinel (not set) — allows CLI flag default to differ from actual default 4100"
  - "ApplyFlagOverrides in cmd package, not config package — keeps config.Load() free of cobra dependency"
  - "TDD approach: wrote all 5 tests before implementing, confirmed RED state, then GREEN"

patterns-established:
  - "Config file at .myhelper/config.json (not .myhelper.json) — directory-based for future extensibility"
  - "Precedence: CLI flag > env var > .myhelper/config.json > ~/.config/myhelper/config.json > default"

requirements-completed: [CONF-01]

duration: 12min
completed: 2026-04-07
---

# Phase 02 Plan 01: Token Threshold Configuration Summary

**Config.TokenThreshold field (default 4100) with MYHELPER_TOKEN_LIMIT env var, .myhelper/config.json local path, and --token-limit persistent cobra flag**

## Performance

- **Duration:** ~12 min
- **Started:** 2026-04-07T00:00:00Z
- **Completed:** 2026-04-07
- **Tasks:** 2 (+ TDD RED commit)
- **Files modified:** 3

## Accomplishments

- Extended Config struct with TokenThreshold int field and DefaultTokenThreshold = 4100 constant
- Updated localConfigPath() from .myhelper.json to .myhelper/config.json
- Added MYHELPER_TOKEN_LIMIT env var handling in Load() with strconv.Atoi parsing
- Registered --token-limit persistent flag on rootCmd (default 0 = not set sentinel)
- Added ApplyFlagOverrides() helper for Phase 3 callers to apply CLI override post-Load()
- All 5 table-driven tests pass covering: default, env var, config file, path, env overrides file

## Task Commits

Each task was committed atomically:

1. **TDD RED: Failing tests for TokenThreshold** - `91e2541` (test)
2. **Task 1: Extend Config struct and Load() with TokenThreshold** - `d5fbaba` (feat)
3. **Task 2: Add --token-limit persistent CLI flag to rootCmd** - `4354c12` (feat)

## Files Created/Modified

- `internal/config/config.go` - Added DefaultTokenThreshold constant, TokenThreshold field, localConfigPath update, MYHELPER_TOKEN_LIMIT env handling, strconv import
- `internal/config/config_test.go` - New: 5 table-driven tests for Load() TokenThreshold behavior
- `cmd/root.go` - Added tokenLimitFlag var, init() with PersistentFlags registration, ApplyFlagOverrides() helper, internal/config import

## Decisions Made

- tokenLimitFlag uses 0 as sentinel for "not set" — Phase 3 callers check `!= 0` before applying override. This keeps the actual default (4100) centralized in DefaultTokenThreshold, not duplicated in the flag registration.
- ApplyFlagOverrides lives in cmd package, not config package — avoids introducing a cobra dependency into the config package, which stays pure stdlib.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Config.TokenThreshold is available for Phase 3 (conversation loop) and Phase 4 (summarization) to read from the resolved config
- ApplyFlagOverrides() is available for any command RunE that needs the CLI override applied
- All existing commands still build and vet clean

---
*Phase: 02-history-token-infrastructure*
*Completed: 2026-04-07*
