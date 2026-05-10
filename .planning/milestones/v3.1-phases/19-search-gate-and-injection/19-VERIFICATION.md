---
phase: 19-search-gate-and-injection
verified: 2026-04-11T00:00:00Z
status: passed
score: 13/13 must-haves verified
overrides_applied: 0
---

# Phase 19: Search Gate & Injection Verification Report

**Phase Goal:** The chat path automatically fetches and injects web search results when the query needs current information, with user flags to override
**Verified:** 2026-04-11
**Status:** passed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| #  | Truth                                                                                           | Status     | Evidence                                                                  |
|----|-------------------------------------------------------------------------------------------------|------------|---------------------------------------------------------------------------|
| 1  | searchGate() returns false on LLM error (fail-closed per GATE-02)                              | ✓ VERIFIED | TestSearchGate/error_returns_false + TestSearchGate_FailClosed PASS        |
| 2  | searchGate() returns true only when LLM responds with a 'yes'-prefixed string (GATE-01)        | ✓ VERIFIED | TestSearchGate/yes_response_returns_true PASS; `strings.HasPrefix(lower, "yes")` confirmed |
| 3  | reRankResults() returns all results when the LLM call errors (RANK-02 fallback)                | ✓ VERIFIED | TestReRankResults_ErrorFallback PASS; `return results, nil` present        |
| 4  | reRankResults() returns nil when re-rank succeeds but selects zero results (RANK-03)           | ✓ VERIFIED | TestReRankResults_ZeroRelevant PASS; `return nil, nil` path confirmed      |
| 5  | buildWebBlock() produces a block starting with [WEB RESULTS] (INJ-01)                          | ✓ VERIFIED | TestBuildWebBlock/block_starts_with_[WEB_RESULTS] PASS; header literal verified |
| 6  | buildWebBlock() drops trailing results when budget would be exceeded (INJ-02)                  | ✓ VERIFIED | TestBuildWebBlock_BudgetTrim PASS; break-on-budget logic in search.go:90  |
| 7  | buildWebBlock() includes title, URL, and snippet per entry (INJ-03)                            | ✓ VERIFIED | TestBuildWebBlock/each_entry_contains_title_URL_snippet PASS; fmt.Sprintf format confirmed |
| 8  | One-shot mode augments the query with web results when gate returns true                        | ✓ VERIFIED | root.go:37 calls buildUserMessage before hist.Add; wired to one-shot path |
| 9  | REPL mode augments each turn via preprocessor closure                                          | ✓ VERIFIED | root.go:43-45 builds preprocessor closure; passed to runConversationLoop  |
| 10 | --search flag bypasses the gate and forces search                                              | ✓ VERIFIED | TestBuildUserMessage_ForceSearch PASS; searchForce registered in init()   |
| 11 | --no-search flag suppresses search entirely                                                    | ✓ VERIFIED | TestBuildUserMessage_NoSearch PASS; searchSuppress registered in init()   |
| 12 | runConversationLoop accepts a nil preprocessor without panicking                               | ✓ VERIFIED | helpers.go:139 nil-guard confirmed; all existing tests pass nil           |
| 13 | All pre-existing tests compile and pass after signature change                                  | ✓ VERIFIED | `go test ./cmd/...` all PASS; planner failure is pre-existing, unrelated  |

**Score:** 13/13 truths verified

### Required Artifacts

| Artifact                   | Expected                                              | Status     | Details                                                          |
|----------------------------|-------------------------------------------------------|------------|------------------------------------------------------------------|
| `cmd/search_gate_test.go`  | Failing test stubs for all 10 requirements            | ✓ VERIFIED | 9 test functions present (7 original + 2 buildUserMessage); all PASS |
| `cmd/search.go`            | searchGate, reRankResults, filterByIndices, buildWebBlock, countTokens, buildUserMessage | ✓ VERIFIED | All 6 functions present, fully implemented, no stubs |
| `cmd/root.go`              | --search and --no-search flag registration; wired call sites | ✓ VERIFIED | searchForce/searchSuppress vars, init(), buildUserMessage calls present |
| `cmd/helpers.go`           | preprocessor func(string) string parameter on runConversationLoop | ✓ VERIFIED | 6th parameter present, nil-safe call at line 139 |

### Key Link Verification

| From                         | To                          | Via                                            | Status     | Details                                                  |
|------------------------------|-----------------------------|------------------------------------------------|------------|----------------------------------------------------------|
| cmd/search.go:searchGate     | internal/ollama.Chat        | direct call with single-message slice          | ✓ WIRED    | ollama.Chat call confirmed at search.go:22               |
| cmd/search.go:buildWebBlock  | internal/history.New        | countTokens helper using TokenCount()          | ✓ WIRED    | history.New(...).TokenCount() at search.go:104           |
| cmd/root.go:RunE one-shot    | buildUserMessage            | direct call before hist.Add                    | ✓ WIRED    | root.go:37 `augmented := buildUserMessage(args[0], ...)`  |
| cmd/root.go:RunE REPL        | runConversationLoop         | preprocessor closure capturing searchCfg, flags | ✓ WIRED    | root.go:43-46 preprocessor closure passed                |
| cmd/helpers.go:loop          | preprocessor                | nil-safe call replacing hist.Add("user", input) | ✓ WIRED    | helpers.go:138-142 nil-safe preprocessor application     |

### Data-Flow Trace (Level 4)

| Artifact             | Data Variable | Source                        | Produces Real Data | Status      |
|----------------------|---------------|-------------------------------|-------------------|-------------|
| buildUserMessage     | ranked        | search.Search + reRankResults | Yes (live search) | ✓ FLOWING   |
| buildWebBlock        | results       | ranked from reRankResults     | Yes               | ✓ FLOWING   |
| runConversationLoop  | msg           | preprocessor(input)           | Yes               | ✓ FLOWING   |

Note: search.Search calls a live SearXNG instance at runtime. Tests use a closed-server to verify graceful degradation; live integration requires a running SearXNG instance.

### Behavioral Spot-Checks

| Behavior                                           | Command                                                                        | Result | Status  |
|----------------------------------------------------|--------------------------------------------------------------------------------|--------|---------|
| All search gate tests pass                         | `go test ./cmd/... -run TestSearchGate`                                        | PASS   | ✓ PASS  |
| All rerank tests pass                              | `go test ./cmd/... -run TestReRankResults`                                     | PASS   | ✓ PASS  |
| All injection block tests pass                     | `go test ./cmd/... -run TestBuildWebBlock`                                     | PASS   | ✓ PASS  |
| buildUserMessage flag tests pass                   | `go test ./cmd/... -run TestBuildUserMessage`                                  | PASS   | ✓ PASS  |
| Project builds cleanly                             | `go build ./...`                                                               | exit 0 | ✓ PASS  |
| HasPrefix(lower, "yes") gate polarity              | `grep "HasPrefix.*yes" cmd/search.go`                                          | MATCH  | ✓ PASS  |
| RANK-02 fallback return                            | `grep "return results, nil" cmd/search.go`                                     | MATCH  | ✓ PASS  |
| RANK-03 skip path                                  | `grep "return nil, nil" cmd/search.go`                                         | MATCH  | ✓ PASS  |
| INJ-01 delimiter                                   | `grep "\[WEB RESULTS\]" cmd/search.go`                                         | MATCH  | ✓ PASS  |
| preprocessor param on runConversationLoop          | `grep "preprocessor func(string) string" cmd/helpers.go`                       | MATCH  | ✓ PASS  |
| nil-safe preprocessor call                         | `grep "preprocessor != nil" cmd/helpers.go`                                    | MATCH  | ✓ PASS  |

### Requirements Coverage

| Requirement | Source Plan | Description                                                 | Status       | Evidence                                                    |
|-------------|-------------|-------------------------------------------------------------|--------------|-------------------------------------------------------------|
| GATE-01     | 19-01       | searchGate() calls LLM, returns bool                        | ✓ SATISFIED  | searchGate uses ollama.Chat, returns bool based on "yes" prefix |
| GATE-02     | 19-01       | Gate returns false on LLM error or ambiguous response       | ✓ SATISFIED  | `return false` on err; HasPrefix("yes") not HasPrefix(!"no") |
| GATE-03     | 19-02       | --search flag forces search regardless of gate              | ✓ SATISFIED  | searchForce flag registered; buildUserMessage forceSearch=true skips gate |
| GATE-04     | 19-02       | --no-search flag suppresses search regardless of gate       | ✓ SATISFIED  | searchSuppress flag registered; noSearch check is first in buildUserMessage |
| RANK-01     | 19-01       | reRankResults() filters results via LLM                     | ✓ SATISFIED  | TestReRankResults/returns_filtered_subset PASS              |
| RANK-02     | 19-01       | reRankResults error → return all results (fallback)         | ✓ SATISFIED  | `return results, nil` on err confirmed                      |
| RANK-03     | 19-01       | reRankResults zero relevant → return nil (skip injection)   | ✓ SATISFIED  | `return nil, nil` when len(selected)==0 after successful call |
| INJ-01      | 19-01       | Injected block starts with [WEB RESULTS]                    | ✓ SATISFIED  | header = "[WEB RESULTS]\n"; block prepended before query    |
| INJ-02      | 19-01       | Results dropped when token budget exceeded                  | ✓ SATISFIED  | break when used+cost > budgetTokens; TestBuildWebBlock_BudgetTrim PASS |
| INJ-03      | 19-01       | Each entry contains title, URL, snippet                     | ✓ SATISFIED  | fmt.Sprintf("[%d] %s\n%s\n%s\n\n", i+1, r.Title, r.URL, r.Snippet) |

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| — | — | None found | — | — |

No TODOs, FIXMEs, placeholder returns, empty handlers, or stub implementations found in Phase 19 files.

### Human Verification Required

The following items require a live environment with SearXNG and Ollama running to fully verify the end-to-end roadmap success criteria:

1. **Live search trigger**
   - **Test:** Run `myhelper "what is the latest Go release?"` against a live Ollama + SearXNG instance
   - **Expected:** Model response cites fetched snippets; no `--search` flag needed
   - **Why human:** Requires live SearXNG at 192.168.0.9:8083 and Ollama — cannot stub in automated check

2. **Live gate suppression**
   - **Test:** Run `myhelper "what is a goroutine?"` against live services
   - **Expected:** Gate returns false; model answers from its own knowledge without web results injected
   - **Why human:** Gate LLM decision is non-deterministic; requires live Ollama

3. **--search flag forces search**
   - **Test:** Run `myhelper --search "what is a goroutine?"`
   - **Expected:** Search fires even though gate would return false; snippets appear in model context
   - **Why human:** Requires live SearXNG to return actual results

4. **--no-search flag suppresses search**
   - **Test:** Run `myhelper --no-search "what is the latest Go release?"`
   - **Expected:** No SearXNG call made; model answers from own knowledge
   - **Why human:** Requires observing network traffic or live SearXNG absence

Note: Items 3 and 4 are unit-tested (TestBuildUserMessage_ForceSearch, TestBuildUserMessage_NoSearch) with graceful-degradation paths confirmed. The live checks above confirm end-to-end wiring through the CLI flags to the running services.

### Gaps Summary

No gaps found. All 10 requirements are implemented, tested, and wired into the conversation path. The pre-existing `TestParsePlan` failure in `internal/planner` is unrelated to Phase 19 (it references a missing fixture file from the archived Phase 14 directory and was present on the base commit before any Phase 19 work).

---

_Verified: 2026-04-11_
_Verifier: Claude (gsd-verifier)_
