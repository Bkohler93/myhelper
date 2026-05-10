# Phase 19: Search Gate & Injection - Context

**Gathered:** 2026-04-11
**Status:** Ready for planning
**Mode:** Auto-generated (discuss skipped via workflow.skip_discuss)

<domain>
## Phase Boundary

The chat path automatically fetches and injects web search results when the query needs current information, with user flags to override. Wires `internal/search` (Phase 18) into `cmd/helpers.go` conversation flow: LLM gate → SearXNG fetch → LLM re-rank → token-budget-aware injection → `--search`/`--no-search` flag overrides.

</domain>

<decisions>
## Implementation Decisions

### Claude's Discretion
All implementation choices are at Claude's discretion — discuss phase was skipped per user setting. Use ROADMAP phase goal, success criteria, and codebase conventions to guide decisions.

Key constraints from success criteria:
- Gate: LLM yes/no "does this query need current web information?" — mirrors existing relevance gate in `internal/retrieval/retrieval.go`
- `--search` flag forces gate=true; `--no-search` flag forces gate=false
- Injected block: clearly delimited (`[WEB RESULTS]`), contains title + URL + snippet per result, fits token budget
- Re-rank failure: fall back to all results (not zero results)
- Zero relevant results after re-rank: skip injection entirely, LLM answers from own knowledge

</decisions>

<code_context>
## Existing Code Insights

Codebase context will be gathered during plan-phase research.

</code_context>

<specifics>
## Specific Ideas

Requirements: GATE-01, GATE-02, GATE-03, GATE-04, RANK-01, RANK-02, RANK-03, INJ-01, INJ-02, INJ-03

Success Criteria:
1. "what is the latest Go release?" triggers search automatically, response cites fetched snippets
2. "what is a goroutine?" does NOT trigger search — gate returns false
3. `myhelper --search "what is a goroutine?"` forces search even when gate=false
4. `myhelper --no-search "what is the latest Go release?"` suppresses search even when gate=true
5. Injected block delimited `[WEB RESULTS]`, contains title+URL+snippet, fits token limit
6. Re-rank failure → use all results; zero relevant results → skip injection

</specifics>

<deferred>
## Deferred Ideas

None — discuss phase skipped.

</deferred>

---

*Phase: 19-search-gate-and-injection*
*Context gathered: 2026-04-11 via auto-generated (discuss skipped)*
