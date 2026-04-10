---
phase: 11-retrieval-package
plan: "01"
subsystem: retrieval
tags: [retrieval, pipeline, context-injection, symbols, reranking]
dependency_graph:
  requires:
    - internal/scanner (Symbol, FilesArtifact, SymbolsArtifact, PackagesArtifact, ProjectArtifact, ChatFn)
    - internal/history (Message, New, TokenCount)
    - internal/config (Config, TokenThreshold)
  provides:
    - internal/retrieval (BuildContext, Context, Strategy, DefaultStrategy)
  affects:
    - cmd/ (future plans will wire BuildContext into query commands)
tech_stack:
  added: []
  patterns:
    - Four-stage retrieval pipeline (relevance gate → pre-filter → re-rank → dep expansion)
    - Fail-open gate design (context omission is worse than extra tokens)
    - Small/large corpus bifurcation at 40-file threshold
    - Token budget tracked cumulatively across pipeline stages
key_files:
  created:
    - internal/retrieval/retrieval.go
    - internal/retrieval/retrieval_test.go
  modified: []
decisions:
  - "Package implemented as full functional code (not stubs): plan specified complete implementation for Task 1 then tests for Task 2"
  - "Test file uses package retrieval (white-box) for access to unexported pipeline functions"
  - "TestDependencyExpansion_BudgetCap uses budget=0 to deterministically verify cap without needing large file content"
metrics:
  duration: "2m33s"
  completed: "2026-04-10"
  tasks_completed: 2
  files_created: 2
  files_modified: 0
---

# Phase 11 Plan 01: Retrieval Package Skeleton Summary

Four-stage LLM retrieval pipeline (relevance gate, keyword pre-filter, LLM re-rank, dependency expansion) implemented as a complete, compilable `internal/retrieval` package with 13 passing tests.

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | Create retrieval.go package skeleton with all types, constants, and stub functions | 333d041 | internal/retrieval/retrieval.go |
| 2 | Create retrieval_test.go with tests for all 6 pipeline behaviors | 2519bef | internal/retrieval/retrieval_test.go |

## What Was Built

### retrieval.go (488 lines)

Full implementation of the four-stage context retrieval pipeline:

- **BuildContext**: Entry point with exact signature `func BuildContext(root, query string, strategy Strategy, cfg config.Config, chatFn scanner.ChatFn) (Context, error)`. Degrades gracefully when `.myhelper/` artifacts are absent.
- **Stage 1 — Relevance Gate** (`needsContext`): Asks LLM "yes/no" whether the query needs project context. Fails open on error.
- **Stage 2 — Pre-filter** (`preFilter`): Small corpus (≤40 files) returns all symbols sorted by relevance; large corpus returns only matching symbols with fallback to all if empty.
- **Stage 3 — Re-ranking** (`llmReRank`): LLM confirms which candidates are relevant by stableID. Falls back to all candidates on LLM error or empty response.
- **Stage 4 — Dependency expansion** (`expandDeps`): Adds depth-1 import neighbors up to 60% of remaining budget.
- **Message assembly** (`assembleMessages`): Formats selected symbols and file paths as markdown context preamble.
- **Budget tracking**: `contextBudgetFactor=0.80`, `expansionBudgetFactor=0.60`, `smallCorpusThreshold=40`.

Exported types: `Context` (Symbols, Files, Messages), `Strategy` (Name, UseSymbols, UseFiles, MaxTokenRatio), `DefaultStrategy`.

### retrieval_test.go (353 lines)

13 test cases covering all 6 pipeline behaviors:

- `TestPreFilter_LargeCorpus`: verifies top result is the query-matching symbol
- `TestPreFilter_SmallCorpus`: verifies all symbols returned as additive hints
- `TestPreFilter_EmptySymbols`: verifies no panic on empty input
- `TestRelevanceGate_FailsOpen`: chatFn error → returns true
- `TestRelevanceGate_NoResponse`: "no" response → returns false
- `TestRelevanceGate_YesResponse`: "yes" response → returns true
- `TestRerank_EmptyInput`: empty candidates → no LLM call, returns empty
- `TestRerank_Fallback`: LLM error → returns all candidates
- `TestRerank_FiltersByStableID`: LLM returns "pkg.Alpha" → only that symbol selected
- `TestDependencyExpansion_BudgetCap`: budget=0 → no expansion beyond selected
- `TestDependencyExpansion_NoOverlap`: already-selected files not re-added
- `TestBuildContext_NoArtifacts`: nonexistent root → bare user query, no error
- `TestBuildContext_Integration`: temp dir with real artifact JSON → non-empty Context returned

## Verification

```
go test ./internal/retrieval/... -v   → 13/13 PASS
go test ./...                         → all packages PASS, no regressions
go build ./...                        → exits 0
```

## Deviations from Plan

None — plan executed exactly as written. Task 1 specified a complete implementation (not a stub), and Task 2 wrote tests against it. All 13 acceptance criteria verified.

## Known Stubs

None — all pipeline stages are fully implemented.

## Threat Surface Scan

No new network endpoints, auth paths, or trust boundary changes. The `loadArtifacts` function reads from `root/.myhelper/` using stdlib `os.ReadFile` + `json.Unmarshal` — T-11-01 mitigation verified: malformed JSON returns error, `BuildContext` returns bare query on any artifact read error. T-11-03 mitigation verified: `applyTokenCap` enforced in Stage 2 before LLM call.

## Self-Check: PASSED

- `internal/retrieval/retrieval.go`: FOUND
- `internal/retrieval/retrieval_test.go`: FOUND
- Commit `333d041`: FOUND
- Commit `2519bef`: FOUND
