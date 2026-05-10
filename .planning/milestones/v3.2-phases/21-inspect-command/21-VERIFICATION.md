---
phase: 21-inspect-command
verified: 2026-04-24T19:00:00Z
status: human_needed
score: 5/5 must-haves verified
overrides_applied: 0
human_verification:
  - test: "Run `myhelper inspect \"how does the retrieval pipeline work\"` against a project with .myhelper/ artifacts present"
    expected: "Prints Gate: PASS (raw: ...) or Gate: FAIL (raw: ...), then pre-filter candidates with score:N, then re-rank N survivors / N dropped, then stage metrics, then final token count. No crash, no blank output."
    why_human: "Requires a live Ollama endpoint and pre-built .myhelper/ artifacts; cannot verify multi-stage LLM output programmatically without a running server"
  - test: "Run `myhelper inspect \"some query\"` in a directory with no .myhelper/ directory"
    expected: "Exits non-zero with message containing 'myhelper init'"
    why_human: "Requires absence-of-artifacts environment condition; exit code and message content need interactive verification"
---

# Phase 21: inspect Command Verification Report

**Phase Goal:** Users can run `myhelper inspect <query>` to see exactly which symbols and files the retrieval pipeline would select, and why, without triggering a model response.
**Verified:** 2026-04-24T19:00:00Z
**Status:** human_needed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths (from ROADMAP Success Criteria)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | `myhelper inspect "some query"` executes without error and prints per-stage diagnostics to stdout | ✓ VERIFIED | `cmd/inspect.go` compiles; `go run . help` lists inspect; `runInspect` calls `printInspectResult` which prints gate, pre-filter, re-rank, metrics, and final token count |
| 2 | Output shows the relevance gate decision (pass/fail) and the raw LLM answer that produced it | ✓ VERIFIED | `printInspectResult` lines 62–67: `fmt.Printf("Gate: FAIL (raw: %q)", result.GateAnswer)` and `fmt.Printf("Gate: PASS (raw: %q)", result.GateAnswer)`. `InspectResult.GateAnswer` populated by inlined gate call in `BuildInspectContext` (retrieval.go lines 809–815) |
| 3 | Output lists pre-filter candidates with their keyword scores (INSP-03 requires PreFilterCandidates added to InspectResult) | ✓ VERIFIED | `PreFilterCandidate` struct at retrieval.go:760–763. `InspectResult.PreFilterCandidates` field at line 774. `BuildInspectContext` appends with `scoreSymbol` at lines 831–834. `printInspectResult` ranges over them at lines 71–74 printing `score:N` |
| 4 | Output distinguishes symbols that survived LLM re-ranking from those that were dropped | ✓ VERIFIED | `printInspectResult` lines 77–89: computes `survivorCount = len(result.Symbols)`, `droppedCount = len(result.PreFilterCandidates) - survivorCount`, prints `Re-rank: N survivors / N dropped` |
| 5 | `myhelper inspect --no-context "some query"` exits with a message indicating context was bypassed and skips all retrieval stages | ✓ VERIFIED | Spot-check confirmed: `go run . inspect --no-context "test query"` outputs `Context bypassed (--no-context flag set)` and exits 0. `noContextFlag` check at inspect.go:38–41 precedes `BuildInspectContext` call |

**Score:** 5/5 truths verified

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|---------|--------|---------|
| `internal/retrieval/retrieval.go` | InspectResult with PreFilterCandidates and GateAnswer fields; PreFilterCandidate type | ✓ VERIFIED | `type PreFilterCandidate struct` at line 760; `GateAnswer string` at line 773; `PreFilterCandidates []PreFilterCandidate` at line 774; all populated in `BuildInspectContext` |
| `cmd/inspect.go` | inspect cobra subcommand with runInspect and printInspectResult | ✓ VERIFIED | File exists and compiles; `inspectCmd` defined; `init()` calls `rootCmd.AddCommand(inspectCmd)`; `runInspect` and `printInspectResult` both present |
| `cmd/root.go` | noContextFlag, tokenLimitFlag, ApplyFlagOverrides | ✓ VERIFIED | `noContextFlag bool` at line 17; `tokenLimitFlag int` at line 18; `--no-context` and `--token-limit` registered at lines 24–25; `ApplyFlagOverrides` at line 63 |
| `cmd/helpers.go` | resolveInput, readInteractive helpers | ✓ VERIFIED | `func resolveInput` at line 153; `func readInteractive` at line 161 |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| `cmd/inspect.go:runInspect` | `retrieval.BuildInspectContext` | direct call with StarterStrategy | ✓ WIRED | inspect.go line 43: `retrieval.BuildInspectContext(root, input, retrieval.StarterStrategy, cfg, ollama.Chat)` |
| `cmd/inspect.go:init` | `rootCmd` | `rootCmd.AddCommand(inspectCmd)` | ✓ WIRED | inspect.go line 21 |
| `cmd/inspect.go:printInspectResult` | `result.PreFilterCandidates` | range loop printing stableID, kind, and score | ✓ WIRED | lines 71–75: `for _, c := range result.PreFilterCandidates` printing `c.Symbol.StableID`, `c.Symbol.Name`, `c.Symbol.Kind`, `c.Score` |
| `cmd/inspect.go:printInspectResult` | `result.GateAnswer` | fmt.Printf Gate: PASS/FAIL with raw answer | ✓ WIRED | lines 63, 67: GateAnswer used in both Gate: FAIL and Gate: PASS format strings |
| `retrieval.go:BuildInspectContext` | `InspectResult.GateAnswer` | inline gate call; raw response captured before bool-parsing | ✓ WIRED | lines 809–815: `rawAnswer` assigned from `chatFn`, trimmed into `result.GateAnswer`, then bool-parsed separately |
| `retrieval.go:BuildInspectContext` | `InspectResult.PreFilterCandidates` | append loop after applyTokenCap using scoreSymbol | ✓ WIRED | lines 827–834: `queryTerms` extracted, loop appends `PreFilterCandidate{Symbol: c, Score: scoreSymbol(c, queryTerms)}` |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
|----------|--------------|--------|--------------------|--------|
| `cmd/inspect.go:printInspectResult` | `result.PreFilterCandidates` | `BuildInspectContext` Stage 2 loop from `preFilter()` + `applyTokenCap()` + `scoreSymbol()` | Yes — derives from live artifact files via `loadArtifacts` | ✓ FLOWING |
| `cmd/inspect.go:printInspectResult` | `result.GateAnswer` | `BuildInspectContext` inlined gate call via `chatFn` (ollama.Chat) | Yes — raw LLM response string from live Ollama call | ✓ FLOWING |
| `cmd/inspect.go:printInspectResult` | `result.Symbols` | `BuildInspectContext` Stage 3 `llmReRank` result | Yes — re-ranked symbol list from live LLM call | ✓ FLOWING |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| `--no-context` bypass prints message and exits 0 | `go run . inspect --no-context "test query"` | `Context bypassed (--no-context flag set)` + exit 0 | ✓ PASS |
| `inspect` appears in CLI help | `go run . help` | `inspect     Dry-run retrieval and print per-stage context selection details` | ✓ PASS |
| Full build passes | `go build ./...` | exit 0, no output | ✓ PASS |
| Per-stage LLM diagnostic output with live artifacts | Requires running Ollama + .myhelper/ | Not run | ? SKIP (needs live server) |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|------------|-------------|--------|----------|
| INSP-01 | 21-02-PLAN.md | User can run `myhelper inspect <query>` to see per-stage retrieval diagnostics without sending a model response | ✓ SATISFIED | `inspectCmd` registered; `runInspect` calls `BuildInspectContext` (dry-run); no streaming call made; `go run . help` confirms command present |
| INSP-02 | 21-01-PLAN.md, 21-02-PLAN.md | `inspect` output shows relevance gate decision (pass / fail + raw LLM answer) | ✓ SATISFIED | `GateAnswer` field populated via inlined gate call; `printInspectResult` prints `Gate: PASS (raw: %q)` or `Gate: FAIL (raw: %q)` |
| INSP-03 | 21-01-PLAN.md, 21-02-PLAN.md | `inspect` output shows how many symbols passed the pre-filter stage and lists them with scores | ✓ SATISFIED | `PreFilterCandidate{Symbol, Score}` appended in `BuildInspectContext`; `printInspectResult` prints `Pre-filter: N candidates` then each with `score:N` |
| INSP-04 | 21-02-PLAN.md | `inspect` output shows which symbols survived LLM re-ranking vs which were dropped | ✓ SATISFIED | `printInspectResult` lines 77–89: `Re-rank: N survivors / N dropped` computed from `len(result.Symbols)` vs `len(result.PreFilterCandidates)` |
| INSP-05 | 21-01-PLAN.md, 21-02-PLAN.md | `inspect` respects `--no-context` flag (skips all retrieval stages, shows that context was bypassed) | ✓ SATISFIED | `noContextFlag` registered as PersistentFlag; inspect.go lines 38–41 check it before calling `BuildInspectContext`; spot-check confirmed bypass message and exit 0 |

All 5 INSP requirements are satisfied. No orphaned requirements found for Phase 21 in REQUIREMENTS.md.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| `cmd/inspect.go` | 40, 55 | `return nil` | ℹ️ Info | Both are legitimate success returns: line 40 is the intended --no-context early return; line 55 is the normal success return after printing diagnostics. Not stubs. |

No blockers found. The `return nil` matches are intentional success paths, not empty implementations.

### Pre-existing Test Failure (Out of Scope)

`internal/planner.TestParsePlan` fails because it references `.planning/phases/14-ollama-client-extension/14-01-PLAN.md` which no longer exists on disk. This failure predates Phase 21 entirely (test added in commit `40a4c3d`, before any Phase 21 work). It is not caused by or related to this phase.

Post-plan fix commits WR-01, WR-02, WR-03 (commits `501dffb`, `ecd9771`, `fc7d80e`) applied minor correctness improvements after the SUMMARY was written: returning errors from `RunE` rather than calling `os.Exit`, surfacing gate LLM errors into `GateAnswer`, and calling `ApplyFlagOverrides` in `rootCmd` RunE. All three are present in the current codebase and verified above.

### Human Verification Required

#### 1. Full Diagnostic Output with Live Artifacts

**Test:** Navigate to a project directory that has `.myhelper/` artifacts (run `myhelper init` first if needed), then run `myhelper inspect "how does the retrieval pipeline work"`
**Expected:** Output contains:
- A line starting with `Gate: PASS (raw:` or `Gate: FAIL (raw:` with a non-empty raw LLM answer string
- A `Pre-filter: N candidates` section with at least one `- (stableID) Name [kind] score:N` line when gate passes
- A `Re-rank: N survivors / N dropped` line
- A `Stage metrics:` section with `pre-filter: N tokens`, `re-rank: N tokens`, and `total: N tokens`
- A `Final context size: N tokens` line
- No model conversation response (dry-run only)
**Why human:** Requires a live Ollama endpoint responding to chat requests, plus pre-built `.myhelper/` artifact files. Cannot run the full retrieval pipeline in this verification context.

#### 2. Missing Artifacts Error Path

**Test:** Run `myhelper inspect "test"` in a directory with no `.myhelper/` subdirectory
**Expected:** Exits non-zero (exit code 1) with an error message containing "myhelper init"
**Why human:** Requires a controlled filesystem environment without `.myhelper/`; exit code and stderr message need interactive confirmation. Code path is clear (`runInspect` lines 50–52 return `fmt.Errorf("no .myhelper/ artifacts found — run \`myhelper init\` first")`) but cannot be invoked in this working directory as `.myhelper/` exists here.

### Gaps Summary

No gaps. All 5 ROADMAP success criteria are met by substantive, wired, data-flowing code. The two human verification items are standard smoke-tests requiring a live Ollama server — they do not indicate missing implementation.

---

_Verified: 2026-04-24T19:00:00Z_
_Verifier: Claude (gsd-verifier)_
