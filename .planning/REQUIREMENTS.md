# Requirements: myhelper

**Defined:** 2026-04-24
**Core Value:** Fast, local chat with optional web search for current information — powered by a local Ollama model, no external API dependencies required.

## v4.0 Requirements (active)

### Dead Code Purge

- [ ] **PURGE-01**: The `internal/context` package is deleted from the codebase
- [ ] **PURGE-02**: The `internal/planner` package is deleted from the codebase
- [ ] **PURGE-03**: The `internal/retrieval` package is deleted from the codebase
- [ ] **PURGE-04**: The `internal/scanner` package is deleted from the codebase
- [ ] **PURGE-05**: The `--no-context` flag and `noContextFlag` var are removed from `cmd/root.go`
- [ ] **PURGE-06**: `go build ./...` and `go mod tidy` pass clean after all deletions

### Inspect Rewrite

- [ ] **INSP-01**: `myhelper inspect <query>` prints the gate decision (YES/NO) and raw LLM answer
- [ ] **INSP-02**: When gate says NO, `inspect` stops with a "search not needed" message
- [ ] **INSP-03**: When gate says YES, `inspect` prints all fetched results (title, URL, snippet) from SearXNG
- [ ] **INSP-04**: `inspect` prints re-rank output — survivors listed separately from dropped results
- [ ] **INSP-05**: `inspect` prints the full `[WEB RESULTS]` block that would have been injected, with token count
- [ ] **INSP-06**: `--search` flag forces `inspect` to skip the gate and run the full fetch → re-rank → preview pipeline
- [ ] **INSP-07**: `--no-search` flag on `inspect` prints "search suppressed" and exits immediately

---

## v3.3 Requirements (shipped 2026-04-25)

### Input

- [x] **INPUT-01**: User can edit the current input line with arrow keys, backspace, and home/end navigation
- [x] **INPUT-02**: User can recall previous messages in the session using up/down arrow key history
- [x] **INPUT-03**: User can type a multi-line message using `\`-continuation lines
- [x] **INPUT-04**: User submits the full message (including any embedded newlines) by pressing bare Enter

### Rendering

- [x] **RNDR-01**: User sees a formatted markdown rendering of the model's complete response after the stream ends
- [x] **RNDR-02**: Code blocks in responses are rendered with visible fence formatting (distinct from prose text)

---

## Future Requirements

### Input (deferred)

- **INPUT-05**: User can configure a custom system prompt per session
- **INPUT-06**: User can load a saved system prompt from a file

### Rendering (deferred)

- **RNDR-03**: Code blocks are syntax-highlighted with color
- **RNDR-04**: User can toggle markdown rendering off (raw output mode)

### Inspect (deferred)

- **INSP-08**: `inspect --json` machine-readable output for scripting

## Out of Scope

| Feature | Reason |
|---------|--------|
| Re-implementing .myhelper/ retrieval | Removed, not replaced — chat is now web-search-first |
| Keeping init/sync/starter/plan/lookup/pattern | Already removed in v3.x |
| Simulating search without network calls | inspect makes real calls to show real decisions |
| Real-time streaming markdown | Partial markdown renders incorrectly mid-stream; stream tokens first, render on completion |
| Session history persistence | Single-session only by design |

## Traceability

| Requirement | Phase | Status |
|-------------|-------|--------|
| PURGE-01 | TBD | Pending |
| PURGE-02 | TBD | Pending |
| PURGE-03 | TBD | Pending |
| PURGE-04 | TBD | Pending |
| PURGE-05 | TBD | Pending |
| PURGE-06 | TBD | Pending |
| INSP-01 | TBD | Pending |
| INSP-02 | TBD | Pending |
| INSP-03 | TBD | Pending |
| INSP-04 | TBD | Pending |
| INSP-05 | TBD | Pending |
| INSP-06 | TBD | Pending |
| INSP-07 | TBD | Pending |

**Coverage:**
- v4.0 requirements: 13 total
- Mapped to phases: TBD (roadmapper)
- Unmapped: 13

---
*Requirements defined: 2026-04-24*
*Last updated: 2026-04-25 — v4.0 requirements added; v3.3 requirements marked shipped*
