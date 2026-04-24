# Requirements — v3.2 Observability & Polish

*Created: 2026-04-24*

---

## v3.2 Requirements

### INSP — inspect command

- [x] **INSP-01**: User can run `myhelper inspect <query>` to see per-stage retrieval diagnostics without sending a model response
- [x] **INSP-02**: `inspect` output shows relevance gate decision (pass / fail + raw LLM answer)
- [x] **INSP-03**: `inspect` output shows how many symbols passed the pre-filter stage and lists them with scores
- [x] **INSP-04**: `inspect` output shows which symbols survived LLM re-ranking vs which were dropped
- [x] **INSP-05**: `inspect` respects `--no-context` flag (skips all retrieval stages, shows that context was bypassed)

### UX — loading spinners

- [x] **UX-01**: A Bubble Tea spinner is shown while SearXNG fetches results; disappears when fetch completes
- [x] **UX-02**: A Bubble Tea spinner is shown while the LLM search-gate call runs; disappears when decision is made
- [x] **UX-03**: A Bubble Tea spinner is shown while the LLM re-rank call runs; disappears when re-ranking completes

### BUG — correctness fixes

- [ ] **BUG-01**: SearXNG URL construction tolerates a trailing slash on the configured endpoint (no double-slash in path)
- [ ] **BUG-02**: `BuildContext` and `BuildInspectContext` surface the `llmReRank` error instead of silently discarding it

### CLN — dead code removal

- [ ] **CLN-01**: `cmd/search.go:countTokens` removed; its callers redirected to the shared `retrieval` package helper
- [ ] **CLN-02**: `PackageEntry.Responsibility` either wired into `llmReRank` as context or removed from the re-rank pass; no unused field silently ignored
- [ ] **CLN-03**: `Symbol.CallEdges` and `Symbol.TypeRefs` documented as reserved-for-future-use in code; no active removal from schema (avoids breaking existing `.myhelper/` directories)

### CTX — context injection fix

- [ ] **CTX-03**: Dual context injection fixed — `context.md` content and `proj.Summary` not both injected when they are the same data; one source of truth used per query

### PERF — performance

- [ ] **PERF-01**: `microPassFile` uses the stored `Symbol.Start` / `Symbol.End` line numbers from the artifact instead of re-parsing AST via `ExtractSymbolMap` at runtime

---

## Future Requirements

- `Symbol.CallEdges` / `Symbol.TypeRefs` actively consumed by retrieval pipeline for call-graph-aware ranking (SYM-05, SYM-06) — deferred until retrieval quality warrants the complexity
- Phase 11 VERIFICATION.md — formal verification of RET-01–06 — deferred (covered by downstream phases and 13 unit tests)

## Out of Scope

- New commands beyond `inspect` — no additional commands in this milestone
- Changing the `.myhelper/` artifact schema in a breaking way — CLN-03 explicitly avoids this
- Persistent logging / log files — inline spinners and `inspect` output cover observability needs

---

## Traceability

| Phase | Requirements |
|-------|-------------|
| 21 — inspect Command | INSP-01, INSP-02, INSP-03, INSP-04, INSP-05 |
| 22 — Search Pipeline Spinners | UX-01, UX-02, UX-03 |
| 23 — Cleanup & Correctness | BUG-01, BUG-02, CLN-01, CLN-02, CLN-03, CTX-03, PERF-01 |
