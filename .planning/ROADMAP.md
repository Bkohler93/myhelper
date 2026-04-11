# Roadmap: myhelper

## Milestones

- ✅ **v1.0 Initial Release** — Phase 1 (shipped 2026-04-07)
- ✅ **v1.1 Conversational Mode** — Phases 2-4 (shipped 2026-04-08)
- ✅ **v1.2 Smart Context** — Phases 5-8 (shipped 2026-04-08)
- ✅ **v1.3 Structured Code Intelligence** — Phases 9-13 (shipped 2026-04-10)
- ✅ **v2.0 GSD Plan Executor** — Phases 14-15 (partial; abandoned 2026-04-10)
- ✅ **v3.0 Simple Chat Wrapper** — Phases 16-17 (shipped 2026-04-11)
- 🚧 **v3.1 Web Search** — Phases 18-19 (in progress)

## Phases

<details>
<summary>✅ v1.0 Initial Release (Phase 1) — SHIPPED 2026-04-07</summary>

- [x] Phase 1: Full Implementation (4/4 plans) — completed 2026-04-07

Full archive: `.planning/milestones/v1.0-ROADMAP.md`

</details>

<details>
<summary>✅ v1.1 Conversational Mode (Phases 2-4) — SHIPPED 2026-04-08</summary>

- [x] Phase 2: History & Token Infrastructure (3/3 plans) — completed 2026-04-07
- [x] Phase 3: Conversation Loop (2/2 plans) — completed 2026-04-07
- [x] Phase 4: Summarization & Re-condensation (2/2 plans) — completed 2026-04-08

Full archive: `.planning/milestones/v1.1-ROADMAP.md`

</details>

<details>
<summary>✅ v1.2 Smart Context (Phases 5-8) — SHIPPED 2026-04-08</summary>

- [x] Phase 5: Scanner & Index Generation (6/6 plans) — completed 2026-04-08
- [x] Phase 6: init + sync Commands (3/3 plans) — completed 2026-04-08
- [x] Phase 7: Two-Pass Context Injection (2/2 plans) — completed 2026-04-08
- [x] Phase 8: Large File Micro-Pass (2/2 plans) — completed 2026-04-08

Full archive: `.planning/milestones/v1.2-ROADMAP.md`

</details>

<details>
<summary>✅ v1.3 Structured Code Intelligence (Phases 9-13) — SHIPPED 2026-04-10</summary>

- [x] Phase 9: Extended AST & Symbol Extraction (2/2 plans) — completed 2026-04-09
- [x] Phase 10: Hierarchical Index Artifacts (2/2 plans) — completed 2026-04-09
- [x] Phase 11: Retrieval Package (1/1 plan) — completed 2026-04-10
- [x] Phase 12: Adaptive Context Builder & Strategies (3/3 plans) — completed 2026-04-10
- [x] Phase 13: Commands & Flags (3/3 plans) — completed 2026-04-10

Full archive: `.planning/milestones/v1.3-ROADMAP.md`

</details>

<details>
<summary>✅ v2.0 GSD Plan Executor (Phases 14-15) — PARTIAL / ABANDONED 2026-04-10</summary>

- [x] Phase 14: Ollama Client Extension (1/1 plan) — completed 2026-04-11
- [x] Phase 15: Plan Parser (2/2 plans) — completed 2026-04-11
- [-] Phase 16: Contract Extractor — abandoned (never started)
- [-] Phase 17: Patch & Verify — abandoned (never started)
- [-] Phase 18: Execute Command — abandoned (never started)

Note: Phases 16-18 were not built. Internal packages from v2.0 (planner, scanner, retrieval) remain in the codebase but are not wired to any CLI commands in v3.0. Phase numbering for v3.0 continues at 16.

</details>

<details>
<summary>✅ v3.0 Simple Chat Wrapper (Phases 16-17) — SHIPPED 2026-04-11</summary>

- [x] Phase 16: CLI Cleanup (1/1 plan) — completed 2026-04-11
- [x] Phase 17: Chat Entry Point (1/1 plan) — completed 2026-04-11

</details>

### 🚧 v3.1 Web Search (In Progress)

**Milestone Goal:** Add internet search capability to the chat path via SearXNG, with an automatic detection gate and LLM-filtered result injection.

- [ ] **Phase 18: SearXNG Client** - Build `internal/search/` package with configurable endpoint, structured results, and clean error handling
- [ ] **Phase 19: Search Gate & Injection** - Auto-detect gate, LLM re-rank pass, token-budget-aware context injection, and `--search`/`--no-search` flags

## Phase Details

### Phase 18: SearXNG Client
**Goal**: A standalone `internal/search/` package can query a SearXNG instance and return structured results ready for downstream consumption
**Depends on**: Phase 17
**Requirements**: SRCH-01, SRCH-02, SRCH-03, SRCH-04, SRCH-05
**Success Criteria** (what must be TRUE):
  1. Calling `search.Search("golang channels", cfg)` against a live SearXNG instance returns a slice of `Result` values each containing non-empty `Title`, `URL`, and `Snippet` fields
  2. The SearXNG endpoint is resolved from `MYHELPER_SEARCH_ENDPOINT` env var, then `.myhelper/config.json`, then `~/.config/myhelper/config.json`, defaulting to `http://192.168.0.9:8083`
  3. A network error or non-200 HTTP response returns an error from `Search` — the caller receives the error and the result slice is nil
  4. A successful call requests 8–10 results from SearXNG's `/search?q=...&format=json` endpoint (observable via request parameters in tests)
**Plans**: 1 plan
Plans:
- [ ] 18-01-PLAN.md — Build internal/search package: Config, LoadConfig(), Result, Search() with httptest-based unit tests

### Phase 19: Search Gate & Injection
**Goal**: The chat path automatically fetches and injects web search results when the query needs current information, with user flags to override
**Depends on**: Phase 18
**Requirements**: GATE-01, GATE-02, GATE-03, GATE-04, RANK-01, RANK-02, RANK-03, INJ-01, INJ-02, INJ-03
**Success Criteria** (what must be TRUE):
  1. Asking "what is the latest Go release?" triggers a search automatically and the model response cites fetched snippets — without any flags
  2. Asking "what is a goroutine?" does not trigger a search — the gate returns false and the model answers from its own knowledge
  3. Running `myhelper --search "what is a goroutine?"` forces a search even though the gate would have returned false
  4. Running `myhelper --no-search "what is the latest Go release?"` suppresses search even though the gate would have returned true
  5. When search is triggered, the injected message block is clearly delimited (e.g., `[WEB RESULTS]`), contains title and URL alongside each snippet, and fits within the configured token limit
  6. If the re-rank LLM call fails or returns no valid indices, the model still responds — either using all fetched results (re-rank error) or from its own knowledge (zero relevant results)
**Plans**: 1 plan
Plans:
- [ ] 18-01-PLAN.md — Build internal/search package: Config, LoadConfig(), Result, Search() with httptest-based unit tests

## Progress

| Phase | Milestone | Plans Complete | Status | Completed |
|-------|-----------|----------------|--------|-----------|
| 1. Full Implementation | v1.0 | 4/4 | Complete | 2026-04-07 |
| 2. History & Token Infrastructure | v1.1 | 3/3 | Complete | 2026-04-07 |
| 3. Conversation Loop | v1.1 | 2/2 | Complete | 2026-04-07 |
| 4. Summarization & Re-condensation | v1.1 | 2/2 | Complete | 2026-04-08 |
| 5. Scanner & Index Generation | v1.2 | 6/6 | Complete | 2026-04-08 |
| 6. init + sync Commands | v1.2 | 3/3 | Complete | 2026-04-08 |
| 7. Two-Pass Context Injection | v1.2 | 2/2 | Complete | 2026-04-08 |
| 8. Large File Micro-Pass | v1.2 | 2/2 | Complete | 2026-04-08 |
| 9. Extended AST & Symbol Extraction | v1.3 | 2/2 | Complete | 2026-04-09 |
| 10. Hierarchical Index Artifacts | v1.3 | 2/2 | Complete | 2026-04-09 |
| 11. Retrieval Package | v1.3 | 1/1 | Complete | 2026-04-10 |
| 12. Adaptive Context Builder & Strategies | v1.3 | 3/3 | Complete | 2026-04-10 |
| 13. Commands & Flags | v1.3 | 3/3 | Complete | 2026-04-10 |
| 14. Ollama Client Extension | v2.0 | 1/1 | Complete | 2026-04-11 |
| 15. Plan Parser | v2.0 | 2/2 | Complete | 2026-04-11 |
| 16. CLI Cleanup | v3.0 | 1/1 | Complete | 2026-04-11 |
| 17. Chat Entry Point | v3.0 | 1/1 | Complete | 2026-04-11 |
| 18. SearXNG Client | v3.1 | 0/? | Not started | - |
| 19. Search Gate & Injection | v3.1 | 0/? | Not started | - |
