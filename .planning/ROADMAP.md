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
- ✅ **v4.0 Search-First Simplification** — Phases 26-27 (shipped 2026-04-26)
- [ ] **v5.0 Distribution & First-Run Setup** — Phases 28-30

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

<details>
<summary>✅ v4.0 Search-First Simplification (Phases 26-27) — SHIPPED 2026-04-26</summary>

- [x] Phase 26: Dead Code Purge (1/1 plan) — completed 2026-04-26
- [x] Phase 27: Inspect Rewrite (1/1 plan) — completed 2026-04-26

Full archive: `.planning/milestones/v4.0-ROADMAP.md`

</details>

### v5.0 Distribution & First-Run Setup

- [x] **Phase 28: Distribution** - goreleaser build pipeline, GitHub Actions release workflow, and curl install script — completed 2026-05-09
- [x] **Phase 29: Tavily Search Provider** - Tavily API integration as the default search provider with env var and config support — completed 2026-05-10
- [ ] **Phase 30: Setup Wizard** - `myhelper setup` interactive first-run wizard covering Ollama, hardware detection, model pull, and search config

## Phase Details

### Phase 28: Distribution
**Goal**: myhelper binaries are downloadable and installable without a Go toolchain
**Depends on**: Nothing (independent of feature work)
**Requirements**: DIST-01, DIST-02
**Success Criteria** (what must be TRUE):
  1. A WSL/Linux user can install myhelper with a single curl command — no Go install required
  2. Pushing a git tag triggers GitHub Actions to build darwin/amd64, darwin/arm64, linux/amd64, linux/arm64 binaries automatically
  3. Built binaries appear as downloadable assets on the GitHub Releases page
  4. The curl install script auto-detects OS and architecture and places the binary in PATH
**Plans**: TBD

### Phase 29: Tavily Search Provider
**Goal**: Users can use Tavily as a search provider in addition to SearXNG
**Depends on**: Nothing (independent — internal/search extension)
**Requirements**: SRCH-01, SRCH-02, SRCH-03
**Success Criteria** (what must be TRUE):
  1. User with a Tavily API key in config gets Tavily search results instead of SearXNG by default
  2. User can set `MYHELPER_TAVILY_KEY` env var to provide their Tavily key, overriding config
  3. User can switch between Tavily and SearXNG by changing `search_provider` in config.json
  4. User with no Tavily key and a SearXNG endpoint continues to get SearXNG results unchanged
**Plans**: 1 plan

Plans:
- [x] 29-01-PLAN.md — Extend search.go and search_test.go: Config fields, LoadConfig Tavily key resolution, tavilySearch client, provider dispatch — complete 2026-05-10

### Phase 30: Setup Wizard
**Goal**: A new user can go from zero to working chat in a single `myhelper setup` run
**Depends on**: Phase 29 (Tavily provider must exist before wizard configures it)
**Requirements**: SETUP-01, SETUP-02, SETUP-03, SETUP-04, SETUP-05, SETUP-06
**Success Criteria** (what must be TRUE):
  1. Running `myhelper setup` on a machine without Ollama shows platform-specific install instructions (brew on macOS, curl on Linux/WSL)
  2. Running `myhelper setup` on a machine with Ollama shows a model recommendation based on detected GPU VRAM or RAM
  3. User can confirm in-wizard to pull the recommended model without leaving the terminal
  4. User is prompted for a Tavily API key and the key is written to `~/.config/myhelper/config.json`
  5. User can optionally enter a SearXNG endpoint and it is written to config
**Plans**: 2 plans
**UI hint**: yes

Plans:
- [ ] 30-01-PLAN.md — Create internal/wizard/wizard.go: all wizard logic (checkOllama, installInstructions, detectMemoryMiB, recommendModel, pullModel, mergeHomeConfig, Run)
- [ ] 30-02-PLAN.md — Create internal/wizard/wizard_test.go and cmd/setup.go: unit tests and cobra subcommand wiring

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
| 26. Dead Code Purge | v4.0 | 1/1 | Complete | 2026-04-26 |
| 27. Inspect Rewrite | v4.0 | 1/1 | Complete | 2026-04-26 |
| 28. Distribution | v5.0 | 3/3 | Complete | 2026-05-09 |
| 29. Tavily Search Provider | v5.0 | 1/1 | Complete | 2026-05-10 |
| 30. Setup Wizard | v5.0 | 0/2 | Not started | - |
