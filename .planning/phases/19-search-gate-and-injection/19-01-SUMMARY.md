---
phase: 19-search-gate-and-injection
plan: "01"
subsystem: cmd
tags: [search-gate, re-rank, injection, tdd]
dependency_graph:
  requires:
    - internal/ollama.Chat
    - internal/history.New / TokenCount
    - internal/search.Result / Config
  provides:
    - cmd/search.go:searchGate
    - cmd/search.go:reRankResults
    - cmd/search.go:filterByIndices
    - cmd/search.go:buildWebBlock
    - cmd/search.go:countTokens
  affects:
    - cmd/ (new helpers; no existing callers yet — wired in Plan 02)
tech_stack:
  added: []
  patterns:
    - httptest.NewServer for Ollama stub in unit tests
    - strings.HasPrefix(lower, "yes") for fail-closed gate polarity
    - history.New(threshold, msgs).TokenCount() for budget-aware token counting
key_files:
  created:
    - cmd/search_gate_test.go
    - cmd/search.go
  modified: []
decisions:
  - "All pipeline helpers in cmd/search.go (not a new internal package) — consistent with codebase convention"
  - "searchGate uses HasPrefix(lower, 'yes') not !HasPrefix(lower, 'no') — fail-closed per GATE-02"
  - "reRankResults RANK-02/RANK-03 distinction: error→all results; zero valid indices→nil (skip injection)"
  - "buildWebBlock overhead-first budget: header+footer tokens deducted before iterating results"
  - "countTokens uses history.New(...).TokenCount() — mirrors existing retrieval.tokenCount pattern"
metrics:
  duration: "~15 minutes"
  completed: "2026-04-11"
  tasks_completed: 3
  files_created: 2
  files_modified: 0
---

# Phase 19 Plan 01: Search Gate, Re-rank, and Injection Helpers Summary

**One-liner:** Fail-closed search gate, LLM re-ranker with RANK-02/RANK-03 distinction, and token-budget-aware [WEB RESULTS] injection block — all in cmd/search.go, fully TDD GREEN.

## What Was Built

Three unexported helper functions in a new `cmd/search.go` file form the read-only data-transformation core of the Phase 19 search pipeline:

1. **`searchGate(query, cfg)`** — Single-message LLM call using `ollama.Chat`. Returns `true` only when response has a "yes" prefix. Fails closed (returns `false`) on any error (GATE-02). Polarity is inverted from `needsContext()` in retrieval: search triggering an unwanted network call is worse than skipping.

2. **`reRankResults(query, results, cfg)`** — Two-message LLM call that filters fetched results to relevant subset. RANK-02/RANK-03 distinction preserved: `err != nil` → return all results (graceful degradation); zero valid indices after successful call → return `nil` (signal to skip injection entirely).

3. **`buildWebBlock(results, budgetTokens, cfg)`** — Builds `[WEB RESULTS]\n...\n[/WEB RESULTS]\n\n` block. Iterates results, breaks when cumulative token cost (via `countTokens`) would exceed `budgetTokens`. Returns `""` if nothing fits.

Supporting helpers: `filterByIndices` (parses 1-based integer indices from LLM response, strips punctuation) and `countTokens` (wraps `history.New(...).TokenCount()` per established retrieval pattern).

## Test Coverage

7 test functions in `cmd/search_gate_test.go` (package `cmd`, same package as helpers):

| Test | Requirement | Result |
|------|-------------|--------|
| TestSearchGate/"yes response returns true" | GATE-01 | PASS |
| TestSearchGate/"error returns false" | GATE-02 | PASS |
| TestSearchGate_FailClosed | GATE-02 edge | PASS |
| TestReRankResults/"returns filtered subset" | RANK-01 | PASS |
| TestReRankResults_ErrorFallback | RANK-02 | PASS |
| TestReRankResults_ZeroRelevant | RANK-03 | PASS |
| TestBuildWebBlock/"block starts with [WEB RESULTS]" | INJ-01 | PASS |
| TestBuildWebBlock/"each entry contains title URL snippet" | INJ-03 | PASS |
| TestBuildWebBlock_BudgetTrim | INJ-02 | PASS |

## Verification Checks

- `grep "HasPrefix.*yes" cmd/search.go` — MATCH (fail-closed gate polarity)
- `grep "return results, nil" cmd/search.go` — MATCH (RANK-02 fallback)
- `grep "return nil, nil" cmd/search.go` — MATCH (RANK-03 skip path)
- `grep "WEB RESULTS" cmd/search.go` — MATCH (INJ-01 delimiter)
- `go test ./cmd/... -run "TestSearchGate|TestReRankResults|TestBuildWebBlock"` — PASS
- `go test ./...` — cmd, config, history, ollama, retrieval, scanner, search all PASS

## Deviations from Plan

### Combined Implementation (Minor)

The plan called for Task 2 (searchGate only, panic stubs for rest) and Task 3 (full reRankResults/buildWebBlock implementation) as separate commits. Both were implemented in a single commit `feat(19-01): implement searchGate fail-closed (GATE-01, GATE-02)` since the implementations are straightforward and writing panic stubs would have required a second pass to replace them. All tests pass and all plan requirements are met; the deviation is cosmetic (commit granularity only).

### Pre-existing Test Failure (Out of Scope)

`internal/planner/TestParsePlan` fails because it references a missing fixture file at `../../.planning/phases/14-ollama-client-extension/14-01-PLAN.md`. This failure exists on the base commit (75ffd90) and is unrelated to this plan's changes. Not fixed per scope boundary rules — logged here for awareness.

## Known Stubs

None. All five functions are fully implemented.

## Threat Flags

None. No new network endpoints, auth paths, or schema changes introduced. The functions are pure data transformations calling existing internal packages.

## Self-Check: PASSED

- cmd/search_gate_test.go: FOUND
- cmd/search.go: FOUND
- Commit ae5a442 (feat — searchGate + all helpers): FOUND
- Commit b5259c9 (test — failing stubs): FOUND
- All 9 test assertions: PASS
