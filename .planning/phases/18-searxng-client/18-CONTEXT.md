# Phase 18: SearXNG Client - Context

**Gathered:** 2026-04-11
**Status:** Ready for planning
**Mode:** Auto-generated (discuss skipped via workflow.skip_discuss)

<domain>
## Phase Boundary

A standalone `internal/search/` package can query a SearXNG instance and return structured results ready for downstream consumption. This phase builds only the client — no integration into the chat path yet (that is phase 19).

</domain>

<decisions>
## Implementation Decisions

### Claude's Discretion
All implementation choices are at Claude's discretion — discuss phase was skipped per user setting. Use ROADMAP phase goal, success criteria, and codebase conventions to guide decisions.

Key constraints from success criteria:
- Package path: `internal/search/`
- Public API: `search.Search(query string, cfg Config) ([]Result, error)`
- Result fields: `Title`, `URL`, `Snippet` (all non-empty for valid results)
- Endpoint config: `MYHELPER_SEARCH_ENDPOINT` env var → `.myhelper/config.json` → `~/.config/myhelper/config.json` → default `http://192.168.0.9:8083`
- HTTP: GET `/search?q=...&format=json`, request 8–10 results
- Error handling: network errors and non-200 responses return error, nil slice

</decisions>

<code_context>
## Existing Code Insights

### Reusable Assets
- `internal/config/config.go` — existing config resolution pattern (env var → local file → global file → default); can mirror this for search endpoint
- `internal/ollama/client.go` — example of an HTTP client package in this codebase (plain `net/http`, no external HTTP libraries)

### Established Patterns
- Packages use `net/http` directly — no external HTTP client libraries
- Config is passed as a struct, not globals
- Tests use `net/http/httptest` for server mocking (seen in `cmd/chat_test.go`, `internal/ollama/`)
- Error wrapping with `fmt.Errorf("...: %w", err)`

### Integration Points
- Will be consumed by phase 19 (`internal/search` imported into chat path)
- Config struct should be compatible with or derived from `internal/config.Config`

</code_context>

<specifics>
## Specific Ideas

No specific requirements — open to standard approaches within the patterns above.

</specifics>

<deferred>
## Deferred Ideas

None — discuss phase skipped.

</deferred>
