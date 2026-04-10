# Roadmap: myhelper

## Milestones

- ✅ **v1.0 Initial Release** — Phase 1 (shipped 2026-04-07)
- ✅ **v1.1 Conversational Mode** — Phases 2-4 (shipped 2026-04-08)
- ✅ **v1.2 Smart Context** — Phases 5-8 (shipped 2026-04-08)
- 🚧 **v1.3 Structured Code Intelligence** — Phases 9-13 (in progress)

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

### 🚧 v1.3 Structured Code Intelligence (In Progress)

**Milestone Goal:** Transform context handling from static injection into a structured, navigable representation of the codebase. The model retrieves incrementally (symbol → file → dependency expansion) rather than guessing context upfront.

- [x] **Phase 9: Extended AST & Symbol Extraction** — Enrich go/ast extraction with kind, signature, line range, imports, call edges, usage references, and stable identifiers (completed 2026-04-09)
- [x] **Phase 10: Hierarchical Index Artifacts** — Replace flat `index.json` with four layered artifact files: `project.json`, `packages.json`, `files.json`, `symbols.json` (completed 2026-04-09)
- [x] **Phase 11: Retrieval Package** — New `internal/retrieval/` package with deterministic pre-filter, LLM re-ranking, relevance gate, and dependency-aware expansion (completed 2026-04-10)
- [x] **Phase 12: Adaptive Context Builder & Strategies** — Staged context assembly replacing `buildInjectedMessages`, with per-command retrieval strategies (completed 2026-04-10)
- [ ] **Phase 13: Commands & Flags** — `--no-context` flag, `inspect` command, and `ApplyFlagOverrides` fix

## Phase Details

### Phase 9: Extended AST & Symbol Extraction
**Goal**: The go/ast extractor captures the full symbol profile needed to populate symbols.json and drive retrieval — kind, signature, line range, imports, call edges, struct/type references, and stable cross-package identifiers
**Depends on**: Phase 8
**Requirements**: SYM-01, SYM-02, SYM-03, SYM-04, SYM-05, SYM-06, SYM-07
**Success Criteria** (what must be TRUE):
  1. `ExtractSymbolsFull` returns kind (func/struct/interface/method) for every exported symbol in a parsed file
  2. `ExtractSymbolsFull` returns the full signature (params + returns for funcs; field list for structs) and exact start/end line numbers for each symbol
  3. `ExtractSymbolsFull` returns the list of import paths declared in each file
  4. `ExtractSymbolsFull` returns per-symbol call edges (direct and selector calls) and struct/type usage references within function bodies
  5. Call edge targets are stored as resolved identifiers (`<package>.<symbol>`) when the package can be determined from imports, otherwise as raw unqualified names
  6. Every symbol carries a stable identifier: functions and types use `<package>.<symbol>`; methods use `<package>.<receiver>.<method>`
**Plans**: 2 plans

Plans:
- [x] 09-01-PLAN.md — Symbol struct + ExtractSymbolsFull core (kind, signature, lines, imports, stableID)
- [x] 09-02-PLAN.md — Body walking: call edges (SYM-05) and type refs (SYM-06)

### Phase 10: Hierarchical Index Artifacts
**Goal**: `init` and `sync` produce four structured artifact files (`project.json`, `packages.json`, `files.json`, `symbols.json`) with a schema version field, replacing the flat `index.json` as the canonical index
**Depends on**: Phase 9
**Requirements**: IDX-01, IDX-02, IDX-03, IDX-04, IDX-05, IDX-06
**Success Criteria** (what must be TRUE):
  1. Running `myhelper init` on a Go project creates all four artifact files under `.myhelper/` with a `schemaVersion` field
  2. `project.json` contains module path, Go version, file/symbol counts, and an LLM-generated project summary paragraph
  3. `packages.json` and `files.json` contain correct package/file metadata and import data
  4. `symbols.json` contains the full symbol profile (kind, signature, line range, file path, call edges, references) for every exported symbol
  5. Any consumer that loads the old flat `index.json` receives a typed `ErrStaleFlatIndex` error instead of silently yielding zero values
**Plans**: 2 plans

Plans:
- [x] 10-01-PLAN.md — Artifact struct types + BuildArtifacts (Symbol.FilePath, ProjectMeta.GoVersion, TDD)
- [x] 10-02-PLAN.md — Wire BuildArtifacts into init/sync; ErrStaleFlatIndex in readIndexFile

### Phase 11: Retrieval Package
**Goal**: A standalone `internal/retrieval/` package provides the full retrieval pipeline — deterministic keyword/symbol pre-filter, LLM re-ranking, context relevance gate, and depth-1 dependency expansion — behind a single `BuildContext` entry point
**Depends on**: Phase 10
**Requirements**: RET-01, RET-02, RET-03, RET-04, RET-05, RET-06
**Success Criteria** (what must be TRUE):
  1. `retrieval.BuildContext(root, query, strategy, cfg, chatFn)` is the sole entry point; all retrieval logic lives in `internal/retrieval/` with none in `cmd/` or `helpers.go`
  2. `BuildContext` returns selected symbols, selected files, and final assembled messages
  3. A deterministic pre-filter narrows the symbol corpus before the LLM sees it; on small corpora (≤ ~40 files) it acts as an additive hint capped by token budget
  4. The LLM re-ranking pass evaluates pre-filtered candidates using summaries and signatures (not full file content)
  5. The relevance gate returns an empty context set when the query does not require repository context, and retrieval is skipped entirely
  6. Dependency-aware expansion adds depth-1 import-graph neighbors of selected files, bounded at ≤ 60% of the remaining token budget
**Plans**: 1 plan

Plans:
- [x] 11-01-PLAN.md — Full retrieval pipeline: package skeleton, all 4 stages (pre-filter, gate, re-ranking, expansion), BuildContext entry point, and test suite

### Phase 12: Adaptive Context Builder & Strategies
**Goal**: The adaptive context builder assembles context in token-bounded stages (project summary → symbol matches → file selection → conditional expansion), replacing `buildInjectedMessages`, with per-command strategies that calibrate retrieval depth to task needs
**Depends on**: Phase 11
**Requirements**: CTX-01, CTX-02, CTX-03, CTX-04
**Success Criteria** (what must be TRUE):
  1. Context assembly proceeds through stages: project summary → symbol matches → file selection → conditional expansion (file content or line-range via micro-pass)
  2. Each stage checks the token budget before appending and stops if the budget would be exceeded
  3. `buildInjectedMessages` is deleted from `helpers.go`; all four query commands call `retrieval.BuildContext` instead
  4. Per-command strategies deliver correct context depth: `plan` summaries-only; `starter` symbols + minimal file context; `lookup` minimal or none; `pattern` zero or near-zero
  5. `starter` injects matched symbol signatures and expands to file content only when the token budget allows and context is required
**Plans**: 3 plans

Plans:
- [x] 12-01-PLAN.md — TDD: Write failing tests for stage-aware assembleMessages and Strategy variables (CTX-01, CTX-02, CTX-04)
- [x] 12-02-PLAN.md — Implement stage-aware assembleMessages, move microPassFile, define Strategy variables (CTX-01, CTX-02, CTX-04)
- [x] 12-03-PLAN.md — Rewire four commands to retrieval.BuildContext; delete buildInjectedMessages and dead helpers (CTX-03, CTX-04)

### Phase 13: Commands & Flags
**Goal**: Users can bypass retrieval entirely with `--no-context`, debug retrieval decisions with `inspect`, and `--token-limit` correctly caps retrieval budget in all query commands
**Depends on**: Phase 12
**Requirements**: CMD-01, CMD-02, CMD-03
**Success Criteria** (what must be TRUE):
  1. `myhelper plan --no-context "..."` streams a response with no project context injected; the flag is available on all four query commands
  2. `myhelper inspect "..."` prints per-stage output: selected symbols and files, token usage per stage, final context size, and selection source (pre-filter, re-rank, or expansion) for each selected entry
  3. `myhelper plan --token-limit 3000 "..."` uses 3000 as the budget cap for retrieval; the flag's value is correctly applied via `ApplyFlagOverrides` before any retrieval math runs
**Plans**: 3 plans

Plans:
- [ ] 13-01-PLAN.md — Wave 0 TDD: failing tests for BuildInspectContext, SelectionSource, --no-context flag, ApplyFlagOverrides (CMD-01, CMD-02, CMD-03)
- [ ] 13-02-PLAN.md — ApplyFlagOverrides + --no-context flag: root.go + four query commands (CMD-01, CMD-03)
- [ ] 13-03-PLAN.md — BuildInspectContext + cmd/inspect.go (CMD-02)

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
| 9. Extended AST & Symbol Extraction | v1.3 | 2/2 | Complete   | 2026-04-09 |
| 10. Hierarchical Index Artifacts | v1.3 | 2/2 | Complete   | 2026-04-09 |
| 11. Retrieval Package | v1.3 | 1/1 | Complete   | 2026-04-10 |
| 12. Adaptive Context Builder & Strategies | v1.3 | 3/3 | Complete   | 2026-04-10 |
| 13. Commands & Flags | v1.3 | 0/3 | Not started | - |
