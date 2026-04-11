# Roadmap: myhelper

## Milestones

- ✅ **v1.0 Initial Release** — Phase 1 (shipped 2026-04-07)
- ✅ **v1.1 Conversational Mode** — Phases 2-4 (shipped 2026-04-08)
- ✅ **v1.2 Smart Context** — Phases 5-8 (shipped 2026-04-08)
- ✅ **v1.3 Structured Code Intelligence** — Phases 9-13 (shipped 2026-04-10)
- 🚧 **v2.0 GSD Plan Executor** — Phases 14-18 (in progress)

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

### 🚧 v2.0 GSD Plan Executor (In Progress)

**Milestone Goal:** Transform myhelper into a GSD-integrated code executor — reading structured PLAN.md files, injecting targeted retrieval context per task, and driving the local 7B model through atomic code changes step-by-step with patch application and compile verification.

- [x] **Phase 14: Ollama Client Extension** - Add structured JSON output support to the Ollama client (completed 2026-04-11)
- [ ] **Phase 15: Plan Parser** - Parse GSD PLAN.md files and auto-discover the active phase plan
- [ ] **Phase 16: Contract Extractor** - Extract and accumulate exported contracts across sequential tasks
- [ ] **Phase 17: Patch & Verify** - Generate display diffs, apply file writes, and verify compilation
- [ ] **Phase 18: Execute Command** - Integrate all prior phases into the `execute` command; remove `plan`

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
- [ ] 15-01-PLAN.md — Create internal/planner package: Plan/Task structs, ParsePlan with bufio frontmatter + XML task extraction, TestParsePlan suite
- [ ] 15-02-PLAN.md — Add FindActivePlan directory scanner and TestFindActivePlan suite

### Phase 16: Contract Extractor
**Goal**: Exported types and signatures accumulate across tasks and are available for injection into subsequent task context
**Depends on**: Phase 15
**Requirements**: CONTRACT-01, CONTRACT-02, CONTRACT-03
**Success Criteria** (what must be TRUE):
  1. After a task applies changes, exported Go declarations are extracted from the modified file via `go/ast` and stored in a `ContractAccumulator`
  2. The accumulator's contents are injectable as a formatted context block into a task's message slice
  3. When accumulated contracts would exceed 820 tokens, the oldest entries are summarized rather than dropped raw
**Plans**: TBD

### Phase 17: Patch & Verify
**Goal**: File changes are shown as a visual diff, written to disk on confirmation, and verified to compile
**Depends on**: Phase 15
**Requirements**: PATCH-01, PATCH-02, PATCH-03, VERIFY-01, VERIFY-02, VERIFY-03
**Success Criteria** (what must be TRUE):
  1. A human-readable unified diff is displayed to the user before any file is written
  2. Confirming a task writes the complete generated file content directly to disk (no patch application)
  3. `go build ./...` runs automatically after each applied task and halts progression on failure with full error output shown
  4. `go test ./...` runs after each applied task; skippable with `--no-verify`; failure halts progression with full output shown
**Plans**: TBD
**UI hint**: yes

### Phase 18: Execute Command
**Goal**: Users can run `myhelper execute` to step through all tasks in the active GSD phase plan with full retrieval context, contract accumulation, and compile verification
**Depends on**: Phases 14, 15, 16, 17
**Requirements**: EXEC-01, EXEC-02, EXEC-03, EXEC-04, EXEC-05, EXEC-06, CLEANUP-01
**Success Criteria** (what must be TRUE):
  1. `myhelper execute` discovers and loads the active phase plan without arguments, presenting each task's description and target file before any action
  2. User can confirm (y), skip (s), or quit (q) at each task gate; no file is written until confirmed
  3. An interrupted session resumes from the last incomplete task when `execute` is run again in the same project
  4. A Bubble Tea spinner appears on stderr during LLM generation for each task
  5. The `plan` command is gone from the binary; `myhelper plan` returns an unknown command error
**Plans**: TBD
**UI hint**: yes

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
| 14. Ollama Client Extension | v2.0 | 1/1 | Complete    | 2026-04-11 |
| 15. Plan Parser | v2.0 | 0/2 | Not started | - |
| 16. Contract Extractor | v2.0 | 0/? | Not started | - |
| 17. Patch & Verify | v2.0 | 0/? | Not started | - |
| 18. Execute Command | v2.0 | 0/? | Not started | - |
