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

## Cross-Milestone Trends

### Process Evolution

| Milestone | Sessions | Phases | Key Change |
|-----------|----------|--------|------------|
| v1.0 | 1 | 1 | Initial baseline |
| v1.1 | 2 | 3 | TDD throughout; worktree isolation per plan |
| v1.2 | 1 | 4 | Two-pass retrieval; code review pipeline; 40 commits in one session |

### Cumulative Quality

| Milestone | Tests | Coverage | Notes |
|-----------|-------|----------|-------|
| v1.0 | 3 human UAT | pending | Requires live Ollama instance |
| v1.1 | 18 automated (Go test) | all packages | integration path (live Ollama) still requires manual verification |
| v1.2 | 2,331 LOC tests | all packages | test LOC exceeds source LOC; integration (live Ollama) still manual |

### Top Lessons (Verified Across Milestones)

1. Keep system prompts minimal from day one — retrofitting is painful
2. Separate interactive prompts (stderr) from model output (stdout) by convention
3. TDD creates clean interfaces: Wave 2 built directly to Wave 1's test contracts — zero integration bugs
4. Stale ROADMAP.md checkboxes accumulate silently — verify at each phase boundary, not just milestone completion
5. Run milestone audits after all phases complete — mid-milestone audits produce stale gap reports that mislead completion checks
6. VERIFICATION.md files should be created immediately after each phase — deferred verification becomes invisible tech debt