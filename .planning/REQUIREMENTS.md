# Requirements: myhelper

**Defined:** 2026-04-10
**Core Value:** Fast, language-agnostic chat with a local 7B model — ask anything, get an answer, with optional web search for current information.

## v3.1 Requirements

Requirements for the Web Search milestone.

### Search Client

- [ ] **SRCH-01**: `internal/search` package exposes a `Search(query string, cfg Config) ([]Result, error)` function that queries the SearXNG JSON API and returns structured results
- [ ] **SRCH-02**: Each `Result` contains at minimum: `Title`, `URL`, `Snippet` (content excerpt)
- [ ] **SRCH-03**: SearXNG endpoint is configurable via `MYHELPER_SEARCH_ENDPOINT` env var, `.myhelper/config.json`, and `~/.config/myhelper/config.json` (default: `http://192.168.0.9:8083`)
- [ ] **SRCH-04**: `Search` fetches 8–10 results per query from SearXNG's `/search?q=...&format=json` endpoint
- [ ] **SRCH-05**: Network errors and non-200 responses return an error; the caller decides whether to proceed without search results

### Search Gate

- [ ] **GATE-01**: Before responding, a yes/no LLM call determines whether the query needs current or real-time information — returns `true` (search needed) or `false` (skip search)
- [ ] **GATE-02**: Gate fails open — if the gate call errors, search is skipped (not triggered)
- [ ] **GATE-03**: `--search` flag forces search on, bypassing the gate
- [ ] **GATE-04**: `--no-search` flag suppresses search entirely, bypassing the gate

### Result Re-ranking

- [ ] **RANK-01**: When search is triggered, a second LLM call filters the fetched results — model returns the indices/IDs of results that are genuinely relevant to the query
- [ ] **RANK-02**: Re-rank fails gracefully — if the call errors or returns no valid indices, all fetched results are injected (unfiltered fallback)
- [ ] **RANK-03**: If re-rank returns zero relevant results, search context is omitted entirely and the model answers from its own knowledge

### Context Injection

- [ ] **INJ-01**: Surviving result snippets are injected as a clearly delimited block (e.g., `[WEB RESULTS]`) in the user message before the model responds
- [ ] **INJ-02**: Injected results are token-budget-aware — snippets are truncated or dropped to fit within the configured token limit
- [ ] **INJ-03**: The injected block includes title and URL alongside each snippet so the model can attribute sources

## Deferred

| Feature | Status |
|---------|--------|
| `qwen2.5vl:7b` vision model integration | Deferred to later milestone |
| `execute` command — GSD plan executor | Deferred indefinitely |
| Contract accumulation, patch generation | Deferred indefinitely |
| Project-aware retrieval pipeline | Dormant (code exists, not wired) |

## Out of Scope

| Feature | Reason |
|---------|--------|
| Caching search results across sessions | Adds complexity; freshness is the point of web search |
| Showing search results to user before model responds | Adds latency and UI complexity; results are context, not output |
| Configuring number of results per query at runtime | Default 8–10 is sufficient; keep CLI surface small |
| Image search / VL model | Deferred milestone |
| Multiple search engine backends | SearXNG aggregates already; no need for additional backends |

## Traceability

| Requirement | Phase | Status |
|-------------|-------|--------|
| SRCH-01 | Phase 18 | Pending |
| SRCH-02 | Phase 18 | Pending |
| SRCH-03 | Phase 18 | Pending |
| SRCH-04 | Phase 20 | Pending |
| SRCH-05 | Phase 18 | Pending |
| GATE-01 | Phase 19 | Pending |
| GATE-02 | Phase 19 | Pending |
| GATE-03 | Phase 19 | Pending |
| GATE-04 | Phase 19 | Pending |
| RANK-01 | Phase 19 | Pending |
| RANK-02 | Phase 19 | Pending |
| RANK-03 | Phase 19 | Pending |
| INJ-01 | Phase 19 | Pending |
| INJ-02 | Phase 19 | Pending |
| INJ-03 | Phase 19 | Pending |

**Coverage:**
- v3.1 requirements: 15 total
- Mapped to phases: 15
- Unmapped: 0

---
*Requirements defined: 2026-04-11*
*Last updated: 2026-04-11 — v3.1 Web Search requirements*

## v3.0 Requirements (Completed)

### Chat Interface

- [x] **CHAT-01**: `myhelper` with no args starts a multi-turn interactive REPL — user types questions, model streams responses, session continues until "quit" or Ctrl+C
- [x] **CHAT-02**: `myhelper "question"` sends a single one-shot question, streams the response to stdout, and exits
- [x] **CHAT-03**: REPL maintains conversation history across turns (no re-sending the full transcript — uses existing `history.Message` slice)
- [x] **CHAT-04**: No system prompt by default — model responds to whatever the user types, language-agnostic
- [x] **CHAT-05**: History is summarized via non-streaming Ollama call when token count exceeds threshold; session continues uninterrupted after summarization
- [x] **CHAT-06**: Ollama endpoint and model remain configurable via `MYHELPER_ENDPOINT`, `MYHELPER_MODEL` env vars and `.myhelper/config.json` / `~/.config/myhelper/config.json`

### CLI Cleanup

- [x] **CLEANUP-01**: All existing subcommands (`starter`, `plan`, `lookup`, `pattern`, `inspect`, `init`, `sync`) removed from the cobra command tree — internal packages remain in the codebase
