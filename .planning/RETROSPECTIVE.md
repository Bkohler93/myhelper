# Project Retrospective

*A living document updated after each milestone. Lessons feed forward into future planning.*

## Milestone: v1.0 — Initial Release

**Shipped:** 2026-04-07
**Phases:** 1 | **Plans:** 4 | **Sessions:** 1

### What Was Built
- Streaming Ollama client via NDJSON line-scan writing tokens to stdout as they arrive
- `context.md` system — loader + `init` subcommand with no-overwrite template write
- Five cobra subcommands with shared interactive-fallback input helpers
- Distinct focused system prompts per command (<230 chars each, single-shot)

### What Worked
- Single-phase approach: all v1 requirements landed in one phase, no inter-phase coordination overhead
- YOLO/budget mode: no research agents or plan checker — fast execution, low friction for a well-understood problem
- Keeping system prompts under 230 chars enforced during design, not retrofitted
- Writing interactive prompts to stderr (not stdout) avoided a subtle streaming bug

### What Was Inefficient
- Plan 1.4 (`04-SUMMARY.md`) one-liner was left as "One-liner:" — summary tool failed to extract it cleanly; worth double-checking summaries before archival
- Human UAT left incomplete (3 tests pending) — requires a live Ollama instance which wasn't available during session

### Patterns Established
- Interactive arg handling: `getInput(args, prompt)` helper shared across all commands — one place to change behavior
- stderr for prompts, stdout for model output — clean separation from the start
- Config precedence documented in STATE.md decisions for future reference

### Key Lessons
1. Single-phase milestones work well for small, well-defined tools — don't over-phase what you can see end-to-end
2. Human UAT that requires live external services should be flagged explicitly at plan time so it doesn't block milestone completion
3. Summary one-liner extraction is fragile if the summary file is malformed — verify before archiving

### Cost Observations
- Model mix: budget profile throughout
- Sessions: 1 focused session
- Notable: 488 LOC in ~30 minutes of execution across 4 plans

---

## Milestone: v1.1 — Conversational Mode

**Shipped:** 2026-04-08
**Phases:** 3 (2-4) | **Plans:** 7 | **Commits:** 16

### What Was Built
- `internal/history` package — Message type, History struct, tiktoken-based token counting, ExceedsLimit(), Replace()
- Config extended with TokenThreshold (env var, flag, default 4,100); local config path moved to `.myhelper/config.json`
- Ollama client migrated from `/api/generate` to `/api/chat` (StreamChat + non-streaming Chat)
- `runConversationLoop` with SIGINT handling, quit detection, stdin injection for tests
- Command-specific summarization and re-condensation prompts across all 4 query commands
- `summarize()` helper with re-condensation detection, history guard, and `hist.Replace` integration

### What Worked
- TDD throughout: every new function (Replace, Chat, runConversationLoop, summarize) had failing tests written before implementation — no regressions introduced
- Worktree isolation for executor agents: parallel Wave 1 and Wave 2 execution with clean fast-forward merges to main
- stdin injection via package-level `stdinReader` var — elegant solution that kept cobra out of tests without refactoring commands
- streamFn injection into runConversationLoop — tests never touch real Ollama; pure unit tests run fast
- SIGINT handler scoped to loop lifetime — no global signal state, clean cleanup

### What Was Inefficient
- ROADMAP.md and REQUIREMENTS.md had stale state after Phase 3 execution (CONV-04 unchecked, Phase 3 progress table showing 0/2) — required manual cleanup before milestone archive; worth verifying plan summary→roadmap updates after each wave
- Worktree branches (worktree-agent-*) needed manual `git merge --ff-only` after each wave — orchestrator should detect and merge automatically
- REQUIREMENTS.md CONV-04 checkbox was never updated during Phase 3 execution, discovered only at milestone completion

### Patterns Established
- Package-level `stdinReader` for cmd test injection — reusable pattern for any stdin-reading command
- streamFn as parameter to all loops that need to call Ollama — clean seam for testing
- `"Summary of previous conversation:"` prefix as the sole re-condensation signal — simple, no extra state
- len(msgs) < 5 guard in summarize() — prevents no-op summarization on nearly-empty history

### Key Lessons
1. Plan summaries should update ROADMAP.md plan checkboxes at task granularity — stale checkboxes accumulate and require cleanup at milestone completion
2. Worktree merge step should be orchestrated automatically — after each plan completes, merge the worktree branch before spawning the next wave
3. TDD pays off immediately in multi-plan phases: Wave 2 executor read Wave 1's tests as a spec and built directly to the interface — zero integration issues
4. summarize() calling ollama.Chat directly (not via streamFn) is intentional — document the decision clearly so future contributors don't try to inject it

### Cost Observations
- Model mix: budget profile throughout (sonnet for planner/verifier, haiku for executor)
- Sessions: 2 focused sessions across 1 day
- Notable: 883 LOC additions across 16 commits; 7 plans, 10 tasks, all verified

---

## Milestone: v1.2 — Smart Context

**Shipped:** 2026-04-08
**Phases:** 4 (5-8) | **Plans:** 13 | **Commits:** 40

### What Was Built
- AST-based Go file scanner: `Walk`, `ExtractSymbols`, `ExtractSymbolMap`, token-budgeted `BuildIndex` → `.myhelper/index.json`
- Project metadata reader: go.mod, README, config files → `ProjectMeta` included in index
- Per-package LLM summary generator via `ollama.Chat` → `.myhelper/summaries/{pkg}.md`
- `init` rewrite: full scan + `generateContextMD` + `writeLastSync` under Bubble Tea `RunWithSpinner`
- `sync` command: mtime-based delta rescan, index merge, selective package re-summarization
- `buildInjectedMessages`: two-pass context injection wired into all 4 query commands (plan, lookup, starter, pattern)
- `microPassFile`: AST symbol-map → LLM line-range selection → truncation fallback for oversized files

### What Worked
- TDD maintained throughout all 4 phases — red/green discipline caught `Symbols` vs `ExportedSymbols` field name issues at test time, not runtime
- Bubble Tea spinner gave clean UX for long init/sync scans with no extra architecture
- Two-pass retrieval design eliminated the need for vector/embedding infrastructure — model reads index.json natively
- Code review + auto-fix workflow (REVIEW.md → REVIEW-FIX.md) surfaced and resolved path-traversal vulnerability in `buildInjectedMessages` before shipping

### What Was Inefficient
- Milestone audit (`gaps_found`) was run before Phase 8 completed — audit was stale at milestone completion; audits should run after all phases are done
- Phase 06 and 07 VERIFICATION.md files were never created — REQUIREMENTS.md traceability table remained mostly unchecked despite code being shipped; verification step should be non-optional
- `ApplyFlagOverrides` not wired into query commands — `--token-limit` flag silently no-ops; discovered at audit, not during implementation
- Sync guard checks `meta.json` instead of `index.json` — overly strict on interrupted init; small but real correctness gap

### Patterns Established
- `RunWithSpinner` as the standard TUI wrapper for long-running CLI operations — Bubble Tea composable with Cobra
- `buildInjectedMessages` as the central retrieval seam — easy to test, easy to extend
- Code review → auto-fix pipeline as a standard post-execution step before milestone close
- Path-traversal guard in any LLM-returned file paths: `filepath.Clean` + prefix check against project root

### Key Lessons
1. Run the milestone audit **after all phases complete** — running it mid-milestone produces stale gap reports
2. VERIFICATION.md files should be created immediately after each phase executes, not deferred — verification gaps accumulate invisibly
3. Flag wiring (`ApplyFlagOverrides`) is easy to miss when adding new commands — add it to the phase checklist for any command that uses config flags
4. The two-pass LLM retrieval pattern (index → file selection → content injection) works well at 8k context; revisit if context windows grow significantly

### Cost Observations
- Model mix: balanced profile (sonnet for planner/verifier/reviewer, executor agents)
- Sessions: 1 intensive session (~12 hours)
- Notable: 5,723 insertions across 41 files; 13 plans, 40 commits; test LOC (2,331) now exceeds source LOC (2,147)

---

## Milestone: v1.3 — Structured Code Intelligence

**Shipped:** 2026-04-10
**Phases:** 5 (9-13) | **Plans:** 11 | **Commits:** 66

### What Was Built
- `ExtractSymbolsFull` capturing full symbol profile (kind, signature, line range, imports, call edges, type refs, stable IDs) for all exported Go symbols via go/ast (Phase 9)
- Four layered artifact files (`project.json`, `packages.json`, `files.json`, `symbols.json`) replacing flat `index.json`, produced by `init` and `sync` with schema version field (Phase 10)
- Four-stage LLM retrieval pipeline (relevance gate → keyword pre-filter → LLM re-rank → dep expansion) in `internal/retrieval/` behind `BuildContext` — 13 unit tests (Phase 11)
- Stage-aware token-bounded context assembly (`assembleMessages`) with per-command strategies replacing `buildInjectedMessages` (Phase 12)
- `--no-context` flag, `inspect` dry-run command, and `--token-limit` correctly wired via `ApplyFlagOverrides` across all query commands (Phase 13)

### What Worked
- Implementing Phase 11 as full functional code (not stubs) in a single plan worked well — the retrieval package is self-contained and well-tested before Phase 12 depended on it
- White-box tests (`package retrieval` not `package retrieval_test`) gave clean access to unexported pipeline functions without additional exports
- Fail-open gate design (treat anything not starting with "no" as "yes") is the right call for a small model — silent context omission is harder to debug than extra tokens
- Moving `microPassFile` from `cmd/helpers.go` to `internal/retrieval/` during Phase 12 was a clean refactor that enforced the retrieval-logic-in-retrieval-package rule with no regressions
- Code review pipeline caught WR-01, WR-02, WR-04 before shipping; REVIEW-FIX.md workflow continues to pay off

### What Was Inefficient
- Phase 11 never produced a VERIFICATION.md — RET-01–06 ended the milestone formally "partial" despite the implementation being correct; the verification step was skipped under time pressure and required downstream phases to serve as indirect confirmation
- `Symbol.CallEdges`/`TypeRefs` were extracted and stored (2 full plan's worth of AST walking) but the retrieval pipeline never reads them — the work shipped as tech debt rather than live functionality
- `PackageEntry.Responsibility` and `FileArtifactEntry.ExportedNames` are both written to artifacts but unused by the retrieval pipeline — similar story
- Dual context injection remained: both `appctx.LoadContext()` (context.md) and `proj.Summary` (from `BuildContext`) carry project descriptions derived from the same package summaries — discovered at audit, not during implementation

### Patterns Established
- `smallCorpusThreshold=40` bifurcation in pre-filter: small projects get all symbols as additive hints, large projects get keyword-gated candidates — avoids over-filtering on the primary use case (small personal projects)
- `expansionBudgetFactor=0.60` cap on dependency expansion: prevents expansion from consuming the budget needed for the actual user query
- `TestDependencyExpansion_BudgetCap` using `budget=0` as a deterministic test of the cap behavior — cleaner than building a large fake corpus
- Strategy struct as the retrieval configuration contract: `UseSymbols`, `UseFiles`, `MaxTokenRatio` fields give per-command control without scattered conditionals

### Key Lessons
1. If a phase produces data that nothing consumes, that's a design gap — wire it or defer the extraction, but don't do both halves of work and leave the integration as "tech debt"
2. VERIFICATION.md should be non-negotiable after each phase — Phase 11's missing verification propagated to the audit as 6 partial requirements that required manual cross-referencing to resolve
3. The audit status `tech_debt` is a legitimate milestone state — no need to chase `passed` if the gaps are understood and accepted
4. `microPassFile` moving packages mid-milestone (cmd → retrieval) was smooth because the function was pure (no global state, clear inputs/outputs) — keep retrieval functions pure for portability

### Cost Observations
- Model mix: balanced profile (sonnet for planning/review, executor agents for implementation)
- Sessions: 2 focused sessions over 2 days
- Notable: 10,687 insertions across 57 files; 11 plans, 66 commits; `internal/retrieval/` is now the largest package by code

---

## Milestone: v3.2 — Observability & Polish

**Shipped:** 2026-04-24
**Phases:** 3 (21-23) | **Plans:** 5 | **Sessions:** 1 (same day)

### What Was Built
- `myhelper inspect` dry-run command — per-stage gate/pre-filter/re-rank/metrics diagnostics with `--no-context` bypass and missing-artifacts detection
- Goroutine-based terminal spinners at all 3 search pipeline waits (gate, fetch, re-rank) using stdlib only — zero new dependencies
- SearXNG double-slash URL bug fixed; `llmReRank` error now named `reRankErr` with explicit `selected = candidates` fallback
- Dead code eliminated: `countTokens` duplicate, `pkgs` param from `llmReRank`, `CallEdges`/`TypeRefs` documented as reserved-for-future
- `microPassFile` refactored to use stored `Symbol.Start/End` — eliminates per-call AST re-parse; `ExtractSymbolMap` fallback for unindexed files

### What Worked
- Single-session, single-day execution — all 5 plans landed in ~2.5 hours
- Stdlib-only spinner: no external dependency for a pure UX feature; goroutine + channel pattern is clean and portable
- CTX-03 closed without a code change — investigation proved the dual injection path doesn't exist (`LoadContext` never called); correct outcome when a suspected bug turns out to be a ghost
- Dedicated cleanup phase (23) made each fix trivially targeted — no risk of scope creep when the requirements are explicit bugs/dead code items
- Post-plan correctness commit pattern (commit `1af3465`) — minor WR-style fixes after SUMMARY is written work cleanly as long as they're atomic and logged

### What Was Inefficient
- `myhelper inspect` adds observability for the `.myhelper/` retrieval pipeline specifically — but the project direction has shifted to chat+web-search as the primary use case; the pipeline is now secondary; mild mismatch worth acknowledging
- Deferred live UAT (inspect + spinners) is now a repeat pattern across every milestone — should document the Ollama test setup needed at plan time, not just note it at close

### Patterns Established
- Spinner wrap: `sp := startSpinner("label..."); result = blockingCall(); sp.done()` — no defer, clears at call site
- `microPassFile` stored-first pattern: filter stored symbols by FilePath → build line map; fall back to `ExtractSymbolMap` only when `len(relevantSyms) == 0`
- Named error var for all internal LLM calls: `result, err := llmFn(...); if err != nil { result = fallback }` — explicit over silent discard

### Key Lessons
1. Before writing a fix for a suspected dual-injection bug, grep callers first — CTX-03 took zero code changes because the caller didn't exist
2. A dedicated cleanup phase at the end of a milestone is a strong pattern — when requirements are explicit bugs/removals, execution is fast and risk is low
3. Live UAT deferred at close is now the third straight milestone with this pattern — if the tool direction shifts to make the retrieval pipeline fully secondary, these tests may become permanently moot

### Cost Observations
- Model mix: sonnet throughout (single-model profile)
- Sessions: 1 focused session
- Notable: 19 files, +1,811 / -149 lines across 5 plans; 7,781 total Go LOC; fastest milestone execution yet

---

## Milestone: v4.0 — Search-First Simplification

**Shipped:** 2026-04-26
**Phases:** 2 (26-27) | **Plans:** 2

### What Was Built
- Deleted `internal/context`, `internal/planner`, `internal/retrieval`, `internal/scanner` — 22 files, ~5,500 lines of dead retrieval infrastructure removed
- Removed `--no-context` flag from root.go (was only meaningful with the deleted retrieval pipeline)
- Rewrote `cmd/inspect.go` as a full web search diagnostic: gate → fetch → re-rank → block preview with token cost; `--search` bypasses gate; `--no-search` exits immediately

### What Worked
- Pre-deletion grep gate: confirmed zero live imports before deleting packages — clean execution, no surprises
- Stub-then-rewrite pattern for inspect: Phase 26 stubbed inspect.go with cobra registration intact; Phase 27 replaced the body entirely without touching wiring — clean two-phase handoff
- Live smoke test at checkpoint: user confirmed all three test cases in one session before committing
- Autonomous workflow end-to-end: discuss → plan → execute → audit → complete in one session

### What Was Inefficient
- CLAUDE.md documentation drift: architecture section still describes deleted packages after v4.0 — low-friction fix deferred as tech debt; should be caught at milestone close more reliably
- REQUIREMENTS.md traceability checkboxes remained unchecked throughout development — all Pending at archive time; actual status was satisfied but traceability table didn't reflect it

### Patterns Established
- Pre-deletion grep gate: always verify `grep -r 'internal/pkg' cmd/` returns zero before deleting — prevents broken build surprises
- Inspect as web diagnostic (not retrieval diagnostic): `myhelper inspect` now shows the search pipeline decisions that are actually relevant in v4.0+ architecture

### Key Lessons
1. When direction shifts (retrieval → web-search-first), clean up aggressively in the next milestone — 5,500 lines of dead code accumulated across 4 packages before v4.0 removed them; the sooner the cut, the cheaper the carry cost
2. The stub-then-rewrite pattern works cleanly for command rewrites that span phases: keep cobra registration in Phase N, replace body in Phase N+1
3. Live smoke test before commit is the right checkpoint for diagnostic commands — unit tests can't fully substitute for watching the actual pipeline output

### Cost Observations
- Model mix: sonnet throughout (autonomous mode)
- Sessions: 1 focused session
- Notable: net −5,466 lines; smallest-LOC milestone yet despite shipping two complete phases

---

## Milestone: v5.0 — Distribution & First-Run Setup

**Shipped:** 2026-05-10
**Phases:** 3 (28–30) | **Plans:** 6 | **Sessions:** 2

### What Was Built
- goreleaser v2 config + GitHub Actions release workflow: 4-platform binary releases on v* tag pushes
- curl-pipe install.sh with uname OS/arch detection, SHA256 verification, PATH management
- Tavily HTTP client with Bearer auth; provider dispatch in Search(); MYHELPER_TAVILY_KEY env var; SearXNG backward-compatible
- `myhelper setup` interactive wizard: Ollama check → platform install instructions → hardware detection (nvidia-smi / system_profiler / /proc/meminfo chain) → 4-tier model recommendation → NDJSON pull streaming → config write with 0600 permissions

### What Worked
- io.Reader/io.Writer injection in wizard.Run() made unit tests hermetic with httptest — zero flaky tests, clean test structure
- Ordering Phase 29 (Tavily) before Phase 30 (wizard) prevented a hidden dependency: wizard writes config keys that must match Phase 29's struct tags exactly — verified at integration check
- All 4 human UAT items validated in one session — explicit test scripts in VERIFICATION.md made manual validation fast
- Code review fix commits (WR-01 through WR-04) applied cleanly — the review→fix chain continues to be a reliable quality layer

### What Was Inefficient
- REQUIREMENTS.md traceability table not updated during development — all SETUP/SRCH requirements showed Pending at autonomous mode start; updated manually at milestone close
- 30-02-SUMMARY.md was never created despite Plan 02 being fully executed — documentation artifact gap noticed only at audit time

### Patterns Established
- Wizard testability pattern: wire io.Reader/io.Writer through the call stack instead of reading os.Stdin directly — enables httptest injection for network calls and io.Reader for user input
- Multi-GPU nvidia-smi fix: always parse only the first line of nvidia-smi output to handle multi-GPU machines
- mergeHomeConfig: read existing config into map[string]interface{}, merge only specified keys, write back — non-destructive, zero risk of silent key deletion

### Key Lessons
1. Cross-phase config key alignment is a real integration risk — wizard writes keys that the search package reads; checking key name matches at integration review (not runtime) is the right gate
2. Hermetic test injection (io.Reader/io.Writer, http client vars) is worth the upfront setup cost — makes wizard-style CLI code fully unit-testable without mocking the OS
3. goreleaser v2 schema change (version: 2 header required) is a silent failure mode — always check schema version when upgrading goreleaser-action
4. Verify install.sh extraction path against a real goreleaser snapshot before tagging — wrap_in_directory behavior is easy to get wrong and silent to fail

### Cost Observations
- Model mix: sonnet throughout (autonomous mode)
- Sessions: 2 (Phases 28-29 in session 1; Phase 30 + lifecycle in session 2)
- Notable: +6,205 lines across 53 files; first milestone to ship distribution infrastructure

---

## Cross-Milestone Trends

### Process Evolution

| Milestone | Sessions | Phases | Key Change |
|-----------|----------|--------|------------|
| v1.0 | 1 | 1 | Initial baseline |
| v1.1 | 2 | 3 | TDD throughout; worktree isolation per plan |
| v1.2 | 1 | 4 | Two-pass retrieval; code review pipeline; 40 commits in one session |
| v1.3 | 2 | 5 | Structured retrieval pipeline; artifact index; per-command strategies; 66 commits |
| v3.1 | 1 | 3 | SearXNG client + search gate + re-rank; chat+web-search primary identity established |
| v3.2 | 1 | 3 | inspect command + spinners + debt cleanup; fastest milestone yet (single day, 5 plans) |
| v4.0 | 1 | 2 | Dead retrieval pipeline deleted (−5,500 lines); inspect rewritten as web diagnostic |
| v5.0 | 2 | 3 | Distribution infrastructure + Tavily provider + setup wizard; first milestone to ship binaries |

### Cumulative Quality

| Milestone | Tests | Coverage | Notes |
|-----------|-------|----------|-------|
| v1.0 | 3 human UAT | pending | Requires live Ollama instance |
| v1.1 | 18 automated (Go test) | all packages | integration path (live Ollama) still requires manual verification |
| v1.2 | 2,331 LOC tests | all packages | test LOC exceeds source LOC; integration (live Ollama) still manual |
| v1.3 | ~2,900 LOC tests | all packages | 13 retrieval unit tests; Phase 11 VERIFICATION.md gap; integration still manual |
| v3.2 | ~2,900 LOC tests (unchanged) | all packages | no new tests; debt cleanup + wiring; pre-existing planner fixture failure unrelated |
| v4.0 | ~1,400 LOC tests (deleted ~1,500 with dead packages) | cmd, config, history, ollama, search | all 5 packages pass; retrieval/scanner/context/planner tests deleted with packages |
| v5.0 | ~1,400 + 216 wizard + 459 search (new) = ~2,075 LOC tests | all 6 packages | new wizard and search tests; hermetic httptest injection throughout |

### Top Lessons (Verified Across Milestones)

1. Keep system prompts minimal from day one — retrofitting is painful
2. Separate interactive prompts (stderr) from model output (stdout) by convention
3. TDD creates clean interfaces: Wave 2 built directly to Wave 1's test contracts — zero integration bugs
4. Stale ROADMAP.md checkboxes accumulate silently — verify at each phase boundary, not just milestone completion
5. Run milestone audits after all phases complete — mid-milestone audits produce stale gap reports that mislead completion checks
6. VERIFICATION.md files should be created immediately after each phase — deferred verification becomes invisible tech debt
7. Before writing a fix for a suspected bug, grep callers first — sometimes the bug path doesn't exist (CTX-03: no code change needed)
8. A dedicated cleanup phase at milestone end is a strong pattern — explicit bug/removal requirements execute fast with low risk
9. When architecture shifts, delete dead code aggressively — the longer you carry it, the higher the carry cost; v4.0's net −5,466 lines proves the payoff
10. Stub-then-rewrite across phases works cleanly for command rewrites: keep cobra registration in phase N, replace body in phase N+1
11. Cross-phase config key alignment is a real integration risk — when two packages write/read the same JSON keys, verify name matches at integration check (v5.0: wizard → search.Config)
12. io.Reader/io.Writer injection makes wizard-style CLI code fully unit-testable — avoid os.Stdin in library packages