# Phase 29: Tavily Search Provider - Context

**Gathered:** 2026-05-09
**Status:** Ready for planning
**Mode:** Auto-generated (discuss skipped via workflow.skip_discuss)

<domain>
## Phase Boundary

Add Tavily as a search provider alongside SearXNG. Users with a Tavily API key (via env var `MYHELPER_TAVILY_KEY` or `search_provider`/`tavily_key` in config.json) get Tavily results by default. Users without a Tavily key continue to get SearXNG results unchanged. Provider selection is controlled by `search_provider` in config.

</domain>

<decisions>
## Implementation Decisions

### Claude's Discretion
All implementation choices are at Claude's discretion — discuss phase was skipped per user setting. Use ROADMAP phase goal, success criteria, and codebase conventions to guide decisions.

**Success criteria to satisfy:**
1. User with a Tavily API key in config gets Tavily search results instead of SearXNG by default
2. User can set `MYHELPER_TAVILY_KEY` env var to provide their Tavily key, overriding config
3. User can switch between Tavily and SearXNG by changing `search_provider` in config.json
4. User with no Tavily key and a SearXNG endpoint continues to get SearXNG results unchanged

</decisions>

<code_context>
## Existing Code Insights

Codebase context will be gathered during plan-phase research.

</code_context>

<specifics>
## Specific Ideas

No specific requirements — open to standard approaches.

</specifics>

<deferred>
## Deferred Ideas

None — discuss phase skipped.

</deferred>
