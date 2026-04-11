# Roadmap: myhelper

## Milestones

- ✅ **v1.0 Initial Release** — Phase 1 (shipped 2026-04-07)
- ✅ **v1.1 Conversational Mode** — Phases 2-4 (shipped 2026-04-08)
- ✅ **v1.2 Smart Context** — Phases 5-8 (shipped 2026-04-08)
- ✅ **v1.3 Structured Code Intelligence** — Phases 9-13 (shipped 2026-04-10)
- ✅ **v2.0 GSD Plan Executor** — Phases 14-15 (partial; abandoned 2026-04-10)
- 🚧 **v3.0 Simple Chat Wrapper** — Phases 16-17 (in progress)

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

### 🚧 v3.0 Simple Chat Wrapper (In Progress)

**Milestone Goal:** Strip myhelper down to a language-agnostic interactive chat interface over the local Ollama model — fast, minimal, useful for any quick question.

- [ ] **Phase 16: CLI Cleanup** - Remove all subcommands from the cobra tree, leaving root command as sole entry point
- [ ] **Phase 17: Chat Entry Point** - Wire root command as multi-turn REPL and one-shot responder with history summarization

## Phase Details

### Phase 14: Ollama Client Extension
**Goal**: The Ollama client can return structured JSON output for internal pipeline calls
**Depends on**: Nothing (extends existing `internal/ollama`)
**Requirements**: OLLAMA-01, OLLAMA-02
**Success Criteria** (what must be TRUE):
  1. `ChatWithFormat` accepts a JSON schema and returns a parsed response without streaming
  2. Existing `Chat` and `StreamChat` functions are unchanged and all existing tests pass
  3. The `chatRequest` struct serializes with `format` omitted when no schema is provided
**Plans**: 1 plan

Plans:
- [x] 14-01-PLAN.md — Add Format field to chatRequest struct and implement ChatWithFormat with httptest suite

### Phase 15: Plan Parser
**Goal**: Active GSD phase plans are parsed from disk into typed structs ready for execution
**Depends on**: Phase 14
**Requirements**: PLAN-01, PLAN-02, PLAN-03
**Success Criteria** (what must be TRUE):
  1. `internal/planner` parses a GSD PLAN.md file (YAML frontmatter + XML task blocks) into a `Plan` with a slice of `Task` structs
  2. The active phase directory is discovered automatically from `.planning/phases/` without any argument — finding the highest-numbered directory missing a SUMMARY.md
  3. A PLAN.md with missing or malformed task fields returns a parse error rather than silently dropping tasks
**Plans**: 2 plans

Plans:
- [x] 15-01-PLAN.md — Create internal/planner package: Plan/Task structs, ParsePlan with bufio frontmatter + XML task extraction, TestParsePlan suite
- [x] 15-02-PLAN.md — Add FindActivePlan directory scanner and TestFindActivePlan suite

### Phase 16: CLI Cleanup
**Goal**: The binary has no subcommands — `myhelper` is the only entry point and unrecognized subcommands are gone
**Depends on**: Nothing (standalone surgery on cmd/)
**Requirements**: CLEANUP-01
**Success Criteria** (what must be TRUE):
  1. Running `myhelper starter`, `myhelper plan`, `myhelper lookup`, `myhelper pattern`, `myhelper inspect`, `myhelper init`, or `myhelper sync` returns an "unknown command" error
  2. `go build ./...` passes with no errors after removal
  3. All existing tests in internal packages (ollama, history, planner, scanner, retrieval) continue to pass
**Plans**: 1 plan

Plans:
- [ ] 16-01-PLAN.md — Delete 7 subcommand files, tui.go, coupled tests; trim root.go and helpers.go; run go mod tidy

### Phase 17: Chat Entry Point
**Goal**: Users can chat with the local Ollama model by running `myhelper` or `myhelper "question"` with no other setup
**Depends on**: Phase 16
**Requirements**: CHAT-01, CHAT-02, CHAT-03, CHAT-04, CHAT-05, CHAT-06
**Success Criteria** (what must be TRUE):
  1. `myhelper` with no arguments starts an interactive REPL — user types a question, model streams a response, session continues until "quit" or Ctrl+C
  2. `myhelper "what is a mutex?"` streams a response to stdout and exits with code 0
  3. In a REPL session, a follow-up question receives a response that reflects the prior exchange (history is maintained across turns)
  4. No system prompt is sent — the model receives only the user's messages and accumulated history
  5. When history exceeds the token threshold, the session automatically summarizes silently and continues without interruption
  6. Endpoint and model are picked up from `MYHELPER_ENDPOINT` / `MYHELPER_MODEL` env vars or `.myhelper/config.json` without any code changes
**Plans**: TBD

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
| 16. CLI Cleanup | v3.0 | 0/? | Not started | - |
| 17. Chat Entry Point | v3.0 | 0/? | Not started | - |
