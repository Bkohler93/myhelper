---
phase: 32-setup-wizard-hardening
plan: 01
subsystem: wizard
tags: [go, wizard, config, validation, tdd]

requires:
  - "31-01: config.Load() returns empty strings for Model/Endpoint when unconfigured"
  - "31-02: validateConfig() in cmd/helpers.go hard-fails when model or endpoint is unset"
provides:
  - "Stage 1.5 Ollama endpoint prompt loop in wizard.go — writes endpoint key to config"
  - "Stage 3 skip-model fallback in wizard.go — writes model key even when pull is skipped or fails"
  - "WIZ-01/WIZ-02/WIZ-03 test coverage in wizard_test.go (7 new tests + updated TestRun_SkipAll)"
affects:
  - any path through myhelper setup (now always writes both endpoint and model to config)
  - validateConfig() in cmd/helpers.go (no longer fails after a complete setup run)

tech-stack:
  added: []
  patterns:
    - "Required-field prompt loop: for { prompt; read; validate; break on valid } — used for endpoint; same pattern reusable for future required fields"
    - "pullSucceeded flag tracks whether to enter fallback path — clean separation of pull success vs skip/fail"
    - "var line string at function scope — single declaration shared across all prompt stages in Run()"

key-files:
  created: []
  modified:
    - internal/wizard/wizard.go
    - internal/wizard/wizard_test.go

key-decisions:
  - "Stage 1.5 placed after reachability check (uses ollamaBaseURL as the pre-fill default — the URL that just passed the probe)"
  - "Endpoint loop is unbounded (no retry limit) — user must supply a valid URL to proceed; matches T-32-03 DoS acceptance decision"
  - "Stage 3 fallback loops exactly once on empty model name then returns error — bounded behavior per T-32-03 acceptance"

requirements-completed: [WIZ-01, WIZ-02, WIZ-03]

duration: ~1min
completed: 2026-05-10
---

# Phase 32 Plan 01: Setup Wizard Hardening Summary

**Wizard Stage 1.5 endpoint prompt loop and Stage 3 skip-model fallback added — config always contains both endpoint and model after any complete setup run**

## Performance

- **Duration:** ~1 min
- **Started:** 2026-05-10T16:20:14Z
- **Completed:** 2026-05-10T16:20:54Z
- **Tasks:** 2 (TDD RED + GREEN per task)
- **Files modified:** 2

## Accomplishments

- Added Stage 1.5 to `wizard.go:Run()`: endpoint prompt loop immediately after the "Ollama is running" message; accepts empty input as the pre-filled `ollamaBaseURL` default; rejects values without `http://` or `https://` prefix and re-prompts; writes to config via `mergeHomeConfig`
- Rewrote Stage 3 pull path in `wizard.go:Run()`: introduced `pullSucceeded` flag; when pull is skipped or fails, enters fallback that prompts for an existing local model name with one retry on empty entry; returns `fmt.Errorf("no model name provided")` if both attempts are empty; writes model via `mergeHomeConfig` on success
- Declared `var line string` at function scope (was `:=` inside Stage 3 only); all stages now share a single `line` variable
- Updated `TestRun_SkipAll` in `wizard_test.go`: changed input from `"n\n\n\n"` to `"\nn\nmymodel\n\n\n"` to match the new Stage 1.5 + fallback prompt sequence
- Added 7 new tests: `TestRun_EndpointPrompt_AcceptDefault`, `TestRun_EndpointPrompt_CustomValue`, `TestRun_EndpointPrompt_InvalidThenValid`, `TestRun_SkipModel_FallbackWritesModel`, `TestRun_SkipModel_EmptyThenProvided`, `TestRun_SkipModel_EmptyTwice`, `TestRun_PullFail_FallbackWritesModel`
- All 15 wizard tests pass; `go test ./...` exits 0; `go build ./...` exits 0

## Task Commits

TDD RED/GREEN flow per task:

1. **RED: Failing tests for WIZ-01/WIZ-02/WIZ-03** - `d974d36` (test)
2. **GREEN: wizard.go implementation** - `9671dd4` (feat)

## Files Created/Modified

- `internal/wizard/wizard.go` — Added Stage 1.5 endpoint prompt loop (14 lines); rewrote Stage 3 pull block with `pullSucceeded` flag and skip-model fallback (23 lines); moved `line` declaration to function scope
- `internal/wizard/wizard_test.go` — Updated `TestRun_SkipAll` input string; added 7 new test functions (193 lines added)

## Decisions Made

- Stage 1.5 placed immediately after the "Ollama is running" print (before Stage 2 hardware detection) — `ollamaBaseURL` is the natural pre-fill since it just passed the reachability probe
- Endpoint loop has no retry limit — matches the T-32-03 DoS acceptance decision (user must provide valid input to proceed)
- `pullSucceeded` boolean flag chosen over nested if/else to keep the skip/fail path flat and readable
- Error message: `"no model name provided — setup incomplete"` — explicit about cause and consequence

## Deviations from Plan

None - plan executed exactly as written.

## TDD Gate Compliance

- RED gate: `d974d36` — `test(32-01): add failing tests for WIZ-01/WIZ-02/WIZ-03 (RED)` (6 tests failing confirmed before committing)
- GREEN gate: `9671dd4` — `feat(32-01): add Stage 1.5 endpoint prompt and Stage 3 skip-model fallback` (all 15 tests PASS)

## Threat Flags

None — T-32-01 (endpoint validation) is implemented as required by the plan threat model. T-32-02/03/04 were accepted per plan.

## Known Stubs

None.

## Self-Check: PASSED

- `internal/wizard/wizard.go` — FOUND: `Stage 1.5`, `pullSucceeded`, `Enter the name of a local model`
- `internal/wizard/wizard_test.go` — FOUND: `TestRun_EndpointPrompt_AcceptDefault`, `TestRun_SkipModel_EmptyTwice`
- Commits: d974d36, 9671dd4 — both present in git log
- `go test ./internal/wizard/... -count=1` — all 15 tests PASS
- `go build ./...` — exits 0

---
*Phase: 32-setup-wizard-hardening*
*Completed: 2026-05-10*
