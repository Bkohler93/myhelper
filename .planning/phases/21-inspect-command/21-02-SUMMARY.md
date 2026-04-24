---
phase: 21-inspect-command
plan: "02"
subsystem: cmd
tags: [inspect, cobra, retrieval, diagnostics, dry-run]

dependency_graph:
  requires:
    - phase: 21-01
      provides: InspectResult.GateAnswer, InspectResult.PreFilterCandidates, PreFilterCandidate type, BuildInspectContext, resolveInput, noContextFlag, ApplyFlagOverrides
  provides:
    - cmd/inspect.go with inspectCmd cobra subcommand
    - runInspect: --no-context bypass, artifact-missing detection, BuildInspectContext call
    - printInspectResult: per-stage formatted output (gate, pre-filter, re-rank, metrics, files)
  affects:
    - user-visible myhelper CLI (inspect command now available)

tech_stack:
  added: []
  patterns:
    - No-artifacts detection via GatePassed:false + empty GateAnswer sentinel
    - Dry-run command pattern: call pipeline function, print diagnostics, no conversation loop

key_files:
  created:
    - cmd/inspect.go
  modified: []

key_decisions:
  - "Detect missing .myhelper/ artifacts via GatePassed:false + empty GateAnswer sentinel (BuildInspectContext returns this combination only when loadArtifacts fails)"
  - "Uses StarterStrategy (UseSymbols:true, UseFiles:true, MaxTokenRatio:0.80) — most comprehensive, consistent with phase context and plan spec"
  - "No-artifacts check placed in runInspect before printInspectResult to avoid printing Gate: FAIL for a case that isn't a real gate failure"

patterns_established:
  - "Inspect-style commands: call BuildInspectContext, check for no-artifacts sentinel, then printInspectResult"

requirements_completed: [INSP-01, INSP-02, INSP-03, INSP-04, INSP-05]

duration: ~1min
completed: 2026-04-24
---

# Phase 21 Plan 02: Inspect Command Summary

**`myhelper inspect` cobra subcommand wired to `BuildInspectContext`, printing gate pass/fail with raw LLM answer, pre-filter candidates with keyword scores, re-rank survivor/dropped counts, stage token metrics, and selected files**

## Performance

- **Duration:** ~1 min
- **Started:** 2026-04-24T18:40:35Z
- **Completed:** 2026-04-24T18:41:41Z
- **Tasks:** 1
- **Files modified:** 1

## Accomplishments

- Created `cmd/inspect.go` registering `inspectCmd` with `rootCmd`
- `runInspect` handles `--no-context` bypass, config load with flag overrides, and calls `BuildInspectContext` with `StarterStrategy`
- `printInspectResult` prints all per-stage diagnostic output satisfying INSP-01 through INSP-05
- Missing-artifacts case detected and prints actionable instructions to run `myhelper init` before exiting 1

## Task Commits

Each task was committed atomically:

1. **Task 1: Create cmd/inspect.go with inspectCmd, runInspect, and printInspectResult** - `c37cffb` (feat)

**Plan metadata:** (docs commit to follow)

## Files Created/Modified

- `cmd/inspect.go` - inspect cobra subcommand; runInspect + printInspectResult

## Decisions Made

- **Artifacts sentinel detection:** `BuildInspectContext` returns `InspectResult{GatePassed: false}` with `GateAnswer == ""` when `.myhelper/` is absent (no LLM call was made). Used this sentinel in `runInspect` to print actionable instructions and exit 1, satisfying the must_have truth "When no .myhelper/ artifacts exist, output prints instructions to run myhelper init and exits 1". Plan template did not include this check — added as Rule 2 (missing critical functionality).
- **StarterStrategy:** Used per phase context spec — most comprehensive strategy (UseSymbols:true, UseFiles:true, MaxTokenRatio:0.80). Consistent with CLAUDE.md architecture table.
- **`printInspectResult` as package-private helper:** Not exported since inspect diagnostics are only rendered by the inspect command.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing Critical] Added .myhelper/ artifact absence detection to runInspect**
- **Found during:** Task 1
- **Issue:** Plan template for `runInspect` called `printInspectResult` for all `!result.GatePassed` cases, but `BuildInspectContext` uses the same `GatePassed:false` return for both "gate said no" and "artifacts missing". Without distinguishing them, a missing `.myhelper/` directory would print `Gate: FAIL (raw: "")` instead of actionable instructions. The `must_haves` truth requires exiting 1 with instructions in this case.
- **Fix:** Added check `if !result.GatePassed && result.GateAnswer == ""` in `runInspect` before `printInspectResult`. Prints `No .myhelper/ artifacts found. Run 'myhelper init' first.` to stderr and calls `os.Exit(1)`.
- **Files modified:** cmd/inspect.go
- **Verification:** go build ./... and go test ./... both pass; logic matches BuildInspectContext sentinel contract (line 794 comment confirms caller handles this).
- **Committed in:** c37cffb (Task 1 commit)

---

**Total deviations:** 1 auto-fixed (1 missing critical)
**Impact on plan:** Auto-fix required for correctness per must_haves truth. No scope creep.

## Issues Encountered

Pre-existing `internal/planner.TestParsePlan` failure (references `.planning/phases/14-ollama-client-extension/14-01-PLAN.md` which no longer exists). Predates this plan — documented in 21-01-SUMMARY.md. Out of scope.

## Known Stubs

None. All diagnostic fields come directly from `InspectResult`; no placeholder values.

## Threat Flags

None. No new network endpoints, auth paths, or trust boundaries introduced. All threats accepted per plan threat model (T-21-03: information disclosure of symbols/files/scores to local CLI user is intended; T-21-04: local Ollama same trust as all other commands; T-21-05: local CPU/memory under user control).

## Next Phase Readiness

- `myhelper inspect` is fully functional; v3.2 inspect requirement complete
- Remaining v3.2 work: loading spinners (Bubble Tea), double-slash SearXNG URL bug, silent llmReRank error discard, dead code removal, dual context injection fix, microPassFile perf

---
*Phase: 21-inspect-command*
*Completed: 2026-04-24*
