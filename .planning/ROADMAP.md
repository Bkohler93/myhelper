# Roadmap: myhelper

## Milestones

- ✅ **v1.0 Initial Release** — Phase 1 (shipped 2026-04-07)
- ✅ **v1.1 Conversational Mode** — Phases 2-4 (shipped 2026-04-08)
- ✅ **v1.2 Smart Context** — Phases 5-8 (shipped 2026-04-08)
- ✅ **v1.3 Structured Code Intelligence** — Phases 9-13 (shipped 2026-04-10)
- ✅ **v2.0 GSD Plan Executor** — Phases 14-15 (partial; abandoned 2026-04-10)
- ✅ **v3.0 Simple Chat Wrapper** — Phases 16-17 (shipped 2026-04-11)
- ✅ **v3.1 Web Search** — Phases 18-20 (shipped 2026-04-11)
- ✅ **v3.2 Observability & Polish** — Phases 21-23 (shipped 2026-04-24)

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

- [x] Phase 9: Extended AST & Symbol Extension (2/2 plans) — completed 2026-04-09
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

<details>
<summary>✅ v3.1 Web Search (Phases 18-20) — SHIPPED 2026-04-11</summary>

- [x] Phase 18: SearXNG Client (1/1 plan) — completed 2026-04-11
- [x] Phase 19: Search Gate & Injection (2/2 plans) — completed 2026-04-11
- [x] Phase 20: Fix SRCH-04 — Result Count Param (1/1 plan) — completed 2026-04-11

Full archive: `.planning/milestones/v3.1-ROADMAP.md`

</details>

### v3.2 Observability & Polish

- [x] **Phase 21: inspect Command** — Wire cmd/inspect.go to BuildInspectContext with per-stage formatted output
- [x] **Phase 22: Search Pipeline Spinners** — Add goroutine-based spinners for SearXNG fetch, LLM gate, and LLM re-rank (no new dependencies)
- [x] **Phase 23: Cleanup & Correctness** — Fix bugs, eliminate dead code, and close dual context injection and microPassFile debt

## Phase Details

### Phase 21: inspect Command
**Goal**: Users can run `myhelper inspect <query>` to see exactly which symbols and files the retrieval pipeline would select, and why, without triggering a model response
**Depends on**: Phase 20 (v3.1 complete)
**Requirements**: INSP-01, INSP-02, INSP-03, INSP-04, INSP-05
**Success Criteria** (what must be TRUE):
  1. `myhelper inspect "some query"` executes without error and prints per-stage diagnostics to stdout
  2. Output shows the relevance gate decision (pass/fail) and the raw LLM answer that produced it
  3. Output lists pre-filter candidates with their keyword scores (INSP-03 requires PreFilterCandidates added to InspectResult)
  4. Output distinguishes symbols that survived LLM re-ranking from those that were dropped
  5. `myhelper inspect --no-context "some query"` exits with a message indicating context was bypassed and skips all retrieval stages
**Plans**: 2 plans
Plans:
- [x] 21-01-PLAN.md — Extend InspectResult + restore cmd helpers (retrieval.go, cmd/root.go, cmd/helpers.go)
- [x] 21-02-PLAN.md — Create cmd/inspect.go cobra subcommand with formatted diagnostics output

### Phase 22: Search Pipeline Spinners
**Goal**: Users see a loading spinner during each async wait in the search pipeline so the tool feels responsive instead of silently blocking
**Depends on**: Phase 21
**Requirements**: UX-01, UX-02, UX-03
**Success Criteria** (what must be TRUE):
  1. A spinner appears in the terminal while SearXNG is fetching results and clears when the fetch completes
  2. A spinner appears while the LLM gate call is running and clears when the yes/no decision is returned
  3. A spinner appears while the LLM re-rank call is running and clears when re-ranking completes
  4. Spinner is goroutine-based using only stdlib (os, time, fmt, strings) — go.mod is unchanged
**Plans**: 1 plan
Plans:
- [x] 22-01-PLAN.md — Add spinner type + wire 3 call sites in cmd/search.go (buildUserMessage)

### Phase 23: Cleanup & Correctness
**Goal**: All known v3.1 tech debt is eliminated — bugs fixed, duplicate code removed, and dormant fields either wired or documented — so the codebase is clean entering the next milestone
**Depends on**: Phase 22
**Requirements**: BUG-01, BUG-02, CLN-01, CLN-02, CLN-03, CTX-03, PERF-01
**Success Criteria** (what must be TRUE):
  1. A SearXNG endpoint configured with a trailing slash (e.g. `http://host/`) produces a valid URL with no double-slash in the path
  2. When `llmReRank` returns an error, `BuildContext` and `BuildInspectContext` surface it to the caller rather than silently discarding it
  3. `cmd/search.go` contains no `countTokens` function; all token-counting in the search path goes through the shared retrieval package helper
  4. `PackageEntry.Responsibility` is either passed into the `llmReRank` prompt or explicitly removed from the re-rank call; no field is silently populated and never read
  5. `microPassFile` reads `Symbol.Start`/`Symbol.End` from the stored artifact instead of calling `ExtractSymbolMap`; running `inspect` on a large file produces the same line range selection without re-parsing AST
  6. Queries no longer inject both `context.md` content and `proj.Summary` when they carry the same information; token usage for context-heavy queries measurably decreases
**Plans**: 2 plans
Plans:
- [x] 23-01-PLAN.md — Bug fixes + dead code removal (BUG-01, BUG-02, CLN-01, CLN-02, CLN-03)
- [x] 23-02-PLAN.md — microPassFile refactor + PROJECT.md Core Value update (PERF-01, CTX-03)

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
| 18. SearXNG Client | v3.1 | 1/1 | Complete | 2026-04-11 |
| 19. Search Gate & Injection | v3.1 | 2/2 | Complete | 2026-04-11 |
| 20. Fix SRCH-04 — Result Count Param | v3.1 | 1/1 | Complete | 2026-04-11 |
| 21. inspect Command | v3.2 | 2/2 | Complete | 2026-04-24 |
| 22. Search Pipeline Spinners | v3.2 | 1/1 | Complete | 2026-04-24 |
| 23. Cleanup & Correctness | v3.2 | 2/2 | Complete | 2026-04-24 |
