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
- ✅ **v3.3 Rich Chat UX** — Phases 24-25 (shipped 2026-04-25)
- **v4.0 Search-First Simplification** — Phases 26-27 (active)

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

<details>
<summary>✅ v3.2 Observability & Polish (Phases 21-23) — SHIPPED 2026-04-24</summary>

- [x] Phase 21: inspect Command (2/2 plans) — completed 2026-04-24
- [x] Phase 22: Search Pipeline Spinners (1/1 plan) — completed 2026-04-24
- [x] Phase 23: Cleanup & Correctness (2/2 plans) — completed 2026-04-24

Full archive: `.planning/milestones/v3.2-ROADMAP.md`

</details>

<details>
<summary>✅ v3.3 Rich Chat UX (Phases 24-25) — SHIPPED 2026-04-25</summary>

- [x] Phase 24: Readline Input (1/1 plan) — completed 2026-04-25
- [x] Phase 25: Markdown Rendering (1/1 plan) — completed 2026-04-25

</details>

### v4.0 Search-First Simplification (Phases 26-27)

- [ ] **Phase 26: Dead Code Purge** - Delete internal/context, internal/planner, internal/retrieval, internal/scanner and remove --no-context flag; verify clean build and tests
- [ ] **Phase 27: Inspect Rewrite** - Rewrite cmd/inspect.go as a web search diagnostic dry-run showing gate decision, fetched results, re-rank output, and injected block preview

## Phase Details

### Phase 24: Readline Input
**Goal**: Users interact with the chat loop through a proper line-editing experience with history and multi-line input
**Depends on**: Nothing (continuing from v3.2 completed state)
**Requirements**: INPUT-01, INPUT-02, INPUT-03, INPUT-04
**Plans**: 1 plan
**Success Criteria** (what must be TRUE):
  1. User can press left/right arrow keys to move the cursor within the current input and backspace to delete a character
  2. User can press Home/End to jump to the start or end of the current input line
  3. User can press up/down arrow keys to cycle through previously submitted messages in the current session
  4. User can type a line ending in \ to continue input on the next line; bare Enter submits
  5. User presses bare Enter to submit the complete input (including any embedded newlines) to the model

Plans:
- [x] 24-01-PLAN.md — readline TTY gate, readMultiLine helper, continuation test

**UI hint**: yes

### Phase 25: Markdown Rendering
**Goal**: Users read model responses rendered as formatted markdown rather than raw token output
**Depends on**: Phase 24
**Requirements**: RNDR-01, RNDR-02
**Plans**: 1 plan
**Success Criteria** (what must be TRUE):
  1. After a model response finishes streaming, the raw output is replaced by a formatted markdown rendering of the complete response
  2. Code blocks in the rendered response are visually distinct from prose text, with visible fence formatting

Plans:
- [x] 25-01-PLAN.md — renderMarkdown helper, startSpinner, StreamChat TTY gate + erase-and-replace (all tasks complete)

**UI hint**: yes

### Phase 26: Dead Code Purge
**Goal**: The codebase contains only live, used packages — dead retrieval infrastructure is gone and the build is clean
**Depends on**: Phase 25
**Requirements**: PURGE-01, PURGE-02, PURGE-03, PURGE-04, PURGE-05, PURGE-06
**Plans**: TBD
**Success Criteria** (what must be TRUE):
  1. `internal/context`, `internal/planner`, `internal/retrieval`, and `internal/scanner` directories no longer exist in the repository
  2. `cmd/root.go` contains no `--no-context` flag declaration or `noContextFlag` variable
  3. `go build ./...` completes with no errors and no warnings after all deletions
  4. `go test ./...` passes with no failures (tests referencing deleted packages are also removed)
  5. `go mod tidy` produces no changes to go.mod or go.sum

### Phase 27: Inspect Rewrite
**Goal**: `myhelper inspect <query>` is a useful web search diagnostic that shows exactly what the search pipeline would do for a given query
**Depends on**: Phase 26
**Requirements**: INSP-01, INSP-02, INSP-03, INSP-04, INSP-05, INSP-06, INSP-07
**Plans**: TBD
**Success Criteria** (what must be TRUE):
  1. Running `myhelper inspect <query>` prints the gate LLM's YES/NO decision and its raw answer text
  2. When the gate says NO (search not needed), inspect prints a "search not needed" message and exits without fetching results
  3. When the gate says YES, inspect prints every fetched SearXNG result with its title, URL, and snippet
  4. Inspect prints re-rank output as two labeled groups: survivors (kept) and dropped results, so the user can see what was filtered
  5. Inspect prints the full `[WEB RESULTS]` block that would be injected into the chat context, followed by its token count
  6. `--search` forces inspect to bypass the gate and run the complete fetch → re-rank → preview pipeline regardless of the gate decision
  7. `--no-search` causes inspect to print "search suppressed" and exit immediately without calling the gate or fetching results

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
| 24. Readline Input | v3.3 | 1/1 | Complete | 2026-04-25 |
| 25. Markdown Rendering | v3.3 | 1/1 | Complete | 2026-04-25 |
| 26. Dead Code Purge | v4.0 | 0/TBD | Not started | - |
| 27. Inspect Rewrite | v4.0 | 0/TBD | Not started | - |
