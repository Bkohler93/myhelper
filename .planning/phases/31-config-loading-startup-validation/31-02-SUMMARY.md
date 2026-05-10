---
phase: 31-config-loading-startup-validation
plan: 02
subsystem: cmd
tags: [go, config, validation, cobra]

requires:
  - "31-01: config.Load() returns empty strings for Model/Endpoint when unconfigured"
provides:
  - "validateConfig(cfg config.Config) error in cmd/helpers.go"
  - "Startup validation in rootCmd.RunE and runInspect — hard-fail with 'myhelper setup' hint"
  - "Cobra double-print suppressed via SilenceErrors=true + SilenceUsage=true"
  - "VAL-01 through VAL-05 test coverage in cmd/helpers_test.go"
affects:
  - 31-03
  - any command calling rootCmd (all subcommands inherit SilenceErrors/SilenceUsage)

tech-stack:
  added: []
  patterns:
    - "validateConfig pattern: pure string-empty check after Load()+ApplyFlagOverrides(); called before any Ollama/search work"
    - "SilenceErrors=true + fmt.Fprintln(os.Stderr, err) in Execute() = single-print error path"

key-files:
  created: []
  modified:
    - cmd/helpers.go
    - cmd/helpers_test.go
    - cmd/root.go
    - cmd/inspect.go

key-decisions:
  - "SilenceErrors=true + SilenceUsage=true set in init() — cobra no longer double-prints errors; Execute() remains the sole print site"
  - "validateConfig placed in helpers.go (not root.go) — keeps root.go focused on command wiring; validation is a shared helper"
  - "Error messages use newline formatting ('...\nRun myhelper setup...') to separate diagnosis from remediation hint"

patterns-established:
  - "validateConfig call site pattern: cfg := config.Load() → ApplyFlagOverrides(&cfg) → validateConfig(cfg) → proceed with Ollama/search"

requirements-completed: [VAL-01, VAL-02, VAL-03, VAL-04, VAL-05]

duration: ~2min
completed: 2026-05-10
---

# Phase 31 Plan 02: Startup Validation Summary

**validateConfig helper added to cmd/helpers.go; called in rootCmd.RunE and runInspect after config.Load() — hard-fails with 'myhelper setup' hint when model or endpoint is unset**

## Performance

- **Duration:** ~2 min
- **Started:** 2026-05-10T14:26:16Z
- **Completed:** 2026-05-10T14:28:16Z
- **Tasks:** 2
- **Files modified:** 4

## Accomplishments

- Added `validateConfig(cfg config.Config) error` to `cmd/helpers.go`; returns a descriptive error containing "myhelper setup" when Endpoint or Model is empty
- Added `strings` import to `cmd/helpers_test.go` and appended `TestValidateConfig` with four subtests (VAL-01/02/05 covered directly; VAL-03/04 covered by call sites)
- Set `rootCmd.SilenceErrors = true` and `rootCmd.SilenceUsage = true` in `init()` — cobra no longer double-prints errors; `Execute()` remains the sole print path
- Inserted `validateConfig(cfg)` call in `rootCmd.RunE` immediately after `ApplyFlagOverrides` (covers VAL-01, VAL-02, VAL-04)
- Inserted `validateConfig(cfg)` call in `runInspect` immediately after `ApplyFlagOverrides` (covers VAL-03)
- All existing tests continue to pass; `go test ./...` exits 0

## Task Commits

Each task was committed atomically following TDD RED/GREEN flow:

1. **RED: TestValidateConfig failing test** - `c15512a` (test)
2. **GREEN: validateConfig implementation in helpers.go** - `3298d71` (feat)
3. **Task 2: validateConfig call sites + silence flags** - `7ad0661` (feat)

## Files Created/Modified

- `cmd/helpers.go` - Added `validateConfig` function (17 lines)
- `cmd/helpers_test.go` - Added `strings` import; appended `TestValidateConfig` with 4 subtests
- `cmd/root.go` - Added `SilenceErrors=true`, `SilenceUsage=true` in `init()`; added `validateConfig(cfg)` call in `RunE`
- `cmd/inspect.go` - Added `validateConfig(cfg)` call in `runInspect`

## Decisions Made

- Placed `validateConfig` in `helpers.go` rather than `root.go` — keeps root.go focused on command wiring; the helper is shared by root and inspect
- Used `fmt.Errorf` with embedded newline to visually separate diagnosis from remediation hint in the error string
- Suppressed cobra usage printing (`SilenceUsage=true`) alongside errors (`SilenceErrors=true`) — a config validation error is not a usage mistake

## Deviations from Plan

None - plan executed exactly as written.

## TDD Gate Compliance

- RED gate: `c15512a` — `test(31-02): add failing TestValidateConfig` (compile failure confirmed before committing)
- GREEN gate: `3298d71` — `feat(31-02): add validateConfig helper to helpers.go` (all 4 subtests PASS)

## Threat Flags

None — changes are purely defensive (fail-fast validation with no I/O). Error messages contain only field names and a setup hint; no secrets or internal paths are disclosed (T-31-03 accepted per plan threat model).

## Known Stubs

None.

## Self-Check: PASSED

- `cmd/helpers.go` — FOUND: `func validateConfig`
- `cmd/helpers_test.go` — FOUND: `TestValidateConfig`
- `cmd/root.go` — FOUND: `validateConfig(cfg)`, `SilenceErrors`
- `cmd/inspect.go` — FOUND: `validateConfig(cfg)`
- Commits: c15512a, 3298d71, 7ad0661 — all present in git log

---
*Phase: 31-config-loading-startup-validation*
*Completed: 2026-05-10*
