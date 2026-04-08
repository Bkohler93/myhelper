# Roadmap: myhelper

## Milestones

- ✅ **v1.0 Initial Release** — Phase 1 (shipped 2026-04-07)
- ✅ **v1.1 Conversational Mode** — Phases 2-4 (shipped 2026-04-08)
- 🚧 **v1.2 Smart Context** — Phases 5-8 (in progress)

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

### 🚧 v1.2 Smart Context (In Progress)

**Milestone Goal:** Replace the blank init template with auto-generated project intelligence — an index and summaries the model uses to surgically inject only relevant code into each prompt.

- [ ] **Phase 5: Scanner & Index Generation** - Scan project files, extract AST symbols, generate token-budgeted index.json and per-package summaries
- [x] **Phase 6: init + sync Commands** - Wire scanner into init command and add sync command for full rescan (completed 2026-04-08)
- [ ] **Phase 7: Two-Pass Context Injection** - Pre-flight injection into all 4 query commands; model selects relevant files from index before answering
- [ ] **Phase 8: Large File Micro-Pass** - AST symbol-level line-range selection for oversized files; truncation as final fallback

## Phase Details

### Phase 5: Scanner & Index Generation
**Goal**: The project can be auto-scanned to produce a token-budgeted index.json and per-package LLM summaries stored in .myhelper/
**Depends on**: Phase 4 (v1.1 complete)
**Requirements**: INIT-01, INIT-02, INIT-03, INIT-04, INIT-05, INIT-06, INIT-07, INIT-08
**Success Criteria** (what must be TRUE):
  1. Running init in a Go project produces .myhelper/index.json with per-file entries capped at 80% of TokenThreshold
  2. index.json entries include exported symbols, function signatures, and package names extracted from .go files via go/ast
  3. index.json entries include module name, direct dependencies from go.mod, README content, and config file content
  4. .myhelper/summaries/ contains one LLM-generated design/pattern summary per package
  5. Files under .git/, vendor/, testdata/, .myhelper/, and those marked "// Code generated" are excluded from the scan
**Plans**: 5 plans

Plans:
- [x] 05-01-PLAN.md — Types, file walker, and AST extractor (FileEntry, ChatFn, Walk, ExtractSymbols)
- [x] 05-02-PLAN.md — Project metadata reader (go.mod, README, config files → ProjectMeta)
- [x] 05-03-PLAN.md — Index builder with token budgeting (BuildIndex → index.json)
- [x] 05-04-PLAN.md — Per-package summary generator (GenerateSummaries → summaries/{pkg}.md)
- [x] 05-05-PLAN.md — Scan() exported entry point + integration test

### Phase 6: init + sync Commands
**Goal**: Users can run init to generate project intelligence from scratch and sync to refresh it after project changes
**Depends on**: Phase 5
**Requirements**: SYNC-01, SYNC-02
**Success Criteria** (what must be TRUE):
  1. Running sync after init detects .go files changed since last_sync (via mtime) and re-indexes/re-summarizes only changed files; context.md is regenerated on every sync
  2. Running init always performs a full scan and overwrites all .myhelper/ artifacts unconditionally
**Plans**: 3 plans

Plans:
- [x] 06-01-PLAN.md — Bubble Tea TUI (RunWithSpinner) + shared helpers (generateContextMD, readLastSync, writeLastSync)
- [x] 06-02-PLAN.md — init command rewrite: full scan + context.md generation under spinner
- [x] 06-03-PLAN.md — sync command: delta rescan via mtime, index merge, selective summary regeneration

### Phase 7: Two-Pass Context Injection
**Goal**: All 4 query commands use index.json as a retrieval pre-flight so each prompt contains only the file content that is relevant to the query
**Depends on**: Phase 6
**Requirements**: CTX-01, CTX-02, CTX-04
**Success Criteria** (what must be TRUE):
  1. Running any query command (plan, lookup, starter, pattern) injects index.json into a Pass-1 message, receives a file list from the model, and then injects those files' summaries/content before the final answer
  2. File paths returned by the model in Pass 1 are validated with os.Stat; invalid paths are discarded silently; if no valid paths survive the entire summaries directory is injected as fallback
  3. Injected file content appears in user-role messages only — never in the system message — so existing summarization behavior is not disrupted
**Plans**: 2 plans

Plans:
- [ ] 07-01-PLAN.md — buildInjectedMessages helper + tests (Pass-1 logic, path validation, token budget, fallbacks)
- [ ] 07-02-PLAN.md — Wire buildInjectedMessages into all 4 query commands (plan, lookup, starter, pattern)

### Phase 8: Large File Micro-Pass
**Goal**: Files that exceed the context budget are handled gracefully via symbol-level line-range extraction rather than raw truncation
**Depends on**: Phase 7
**Requirements**: CTX-03
**Success Criteria** (what must be TRUE):
  1. When a selected file exceeds the context budget, go/ast generates a symbol map and the model is asked for a specific line range rather than receiving the full file
  2. When line-range extraction is not possible or the range still exceeds the budget, the content is truncated at a safe boundary as a final fallback with no panic or error surfaced to the user
**Plans**: TBD

## Progress

| Phase | Milestone | Plans Complete | Status | Completed |
|-------|-----------|----------------|--------|-----------|
| 1. Full Implementation | v1.0 | 4/4 | Complete | 2026-04-07 |
| 2. History & Token Infrastructure | v1.1 | 3/3 | Complete | 2026-04-07 |
| 3. Conversation Loop | v1.1 | 2/2 | Complete | 2026-04-07 |
| 4. Summarization & Re-condensation | v1.1 | 2/2 | Complete | 2026-04-08 |
| 5. Scanner & Index Generation | v1.2 | 0/5 | Not started | - |
| 6. init + sync Commands | v1.2 | 3/3 | Complete   | 2026-04-08 |
| 7. Two-Pass Context Injection | v1.2 | 0/2 | Not started | - |
| 8. Large File Micro-Pass | v1.2 | 0/? | Not started | - |
