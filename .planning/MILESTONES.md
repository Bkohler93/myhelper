# Milestones

## v5.0 Distribution & First-Run Setup (Shipped: 2026-05-10)

**Phases completed:** 3 phases (28–30), 6 plans
**Files changed:** 53 files, +6,205 / -637 lines
**Timeline:** 2026-05-09 → 2026-05-10 (2 days)

**Key accomplishments:**

- goreleaser v2 config with 4 platform targets (darwin/amd64, darwin/arm64, linux/amd64, linux/arm64) + GitHub Actions release workflow triggered on `v*` tag pushes using goreleaser-action@v7 (Phase 28)
- curl-pipe install script with uname OS/arch detection, SHA256 checksum verification, and `~/.local/bin` install with PATH hint (Phase 28)
- Tavily HTTP client with POST + Bearer auth; provider-dispatching `Search()` function; `LoadConfig` auto-selects Tavily when key is present; `MYHELPER_TAVILY_KEY` and `MYHELPER_SEARCH_PROVIDER` env var support (Phase 29)
- `myhelper setup` interactive wizard: `checkOllama` (HTTP probe), platform-specific install instructions, `detectMemoryMiB` (nvidia-smi / system_profiler / /proc/meminfo chain), 4-tier `recommendModel` table, `pullModel` NDJSON streaming, `mergeHomeConfig` with 0600 permissions — all human-validated (Phase 30)

---

## v4.0 Search-First Simplification (Shipped: 2026-04-26)

**Phases completed:** 2 phases (26–27), 2 plans, ~5 tasks
**Files changed:** 24 Go files, +86 / -5,552 lines (net −5,466 — massive dead code removal)
**Timeline:** 2026-04-25 → 2026-04-26 (2 days)

**Key accomplishments:**

- Deleted 22 files across 4 dead internal packages (`internal/context`, `internal/planner`, `internal/retrieval`, `internal/scanner`) — ~5,500 lines of unused retrieval infrastructure gone (Phase 26)
- Removed `--no-context` flag and `noContextFlag` var from `cmd/root.go` — flag was only meaningful alongside the deleted retrieval pipeline (Phase 26)
- Rewrote `cmd/inspect.go` as a web search diagnostic dry-run: gate decision (YES/NO/BYPASSED), all fetched SearXNG results, re-rank Survivors/Dropped groups, full `[WEB RESULTS]` block with token cost — all without sending to the chat model (Phase 27)
- `go build ./...`, `go test ./...`, and `go mod tidy` all pass clean after all deletions (Phase 26)

---

## v3.2 Observability & Polish (Shipped: 2026-04-24)

**Phases completed:** 3 phases (21–23), 5 plans
**Files changed:** 19 files, +1,811 / -149 lines
**Timeline:** 2026-04-24 (single day)

**Key accomplishments:**

- `myhelper inspect` dry-run command wired to `BuildInspectContext` — per-stage gate/pre-filter/re-rank/metrics output; `--no-context` bypass; missing-artifacts detection (Phase 21)
- Goroutine-based terminal spinners at all 3 search pipeline waits (gate, fetch, re-rank) using stdlib only — zero new dependencies (Phase 22)
- SearXNG double-slash URL bug fixed; `llmReRank` error now named with explicit fallback to all candidates (Phase 23)
- Dead code eliminated: `countTokens` duplicate, `pkgs` param from `llmReRank`, `CallEdges`/`TypeRefs` documented as reserved-for-future (Phase 23)
- `microPassFile` uses stored `Symbol.Start/End` from artifacts — eliminates per-call AST re-parse for large files (Phase 23)

Known deferred items at close: 2 (see STATE.md Deferred Items)

---

## v3.1 Web Search (Shipped: 2026-04-11)

**Phases completed:** 3 phases, 4 plans, 3 tasks

**Key accomplishments:**

- Standalone SearXNG HTTP client package (internal/search) with url.QueryEscape injection, result filtering, and env/file config resolution — all five SRCH requirements satisfied via TDD
- One-liner:
- One-liner:
- One-liner:

---

## v1.3 Structured Code Intelligence (Shipped: 2026-04-10)

**Phases completed:** 5 phases, 11 plans
**Files changed:** 77 files, +15,250 / -108 lines
**Timeline:** 2026-04-08 → 2026-04-10 (2 days)

**Key accomplishments:**

- `ExtractSymbolsFull` captures full symbol profile (kind, signature, line range, imports, call edges, type refs, stable IDs) for all exported Go symbols via go/ast (Phase 9)
- `init` and `sync` produce four layered artifact files (`project.json`, `packages.json`, `files.json`, `symbols.json`) replacing flat `index.json` with schema version field (Phase 10)
- Four-stage LLM retrieval pipeline (relevance gate → keyword pre-filter → LLM re-rank → dep expansion) in `internal/retrieval/` behind `BuildContext` entry point — 13 tests (Phase 11)
- Stage-aware token-bounded context assembly (`assembleMessages`) with per-command strategies — `plan` summaries-only; `starter` symbols+files; `lookup` minimal; `pattern` near-zero (Phase 12)
- `--no-context` flag, `inspect` dry-run command, and `--token-limit` wired via `ApplyFlagOverrides` across all four query commands (Phase 13)

### Known Gaps (accepted as tech debt)

- **RET-01–06** — Phase 11 missing VERIFICATION.md; implementation confirmed by downstream phase verifications and 13 unit tests
- **SYM-05, SYM-06** — `Symbol.CallEdges`/`TypeRefs` stored in `symbols.json` but not consumed by retrieval pipeline
- **IDX-03, RET-03** — `PackageEntry.Responsibility` written to `packages.json` but unused in `llmReRank`
- **CTX-01, CTX-02** — Dual context injection (`context.md` + `proj.Summary`) — same source, redundant tokens

---

## v1.2 Smart Context (Shipped: 2026-04-09)

**Phases completed:** 4 phases, 13 plans, 13 tasks

**Key accomplishments:**

- FileEntry/ChatFn type contracts, exclusion-aware Walk(), and go/ast-based ExtractSymbols() — scanner core primitives, 16 TDD-verified tests (Phase 5)
- Token-budgeted index assembler serializes `{"meta": {...}, "files": [...]}` to `.myhelper/index.json` via ReadMeta + BuildIndex, dropping test files first when 80% budget is exceeded (Phase 5)
- Per-package LLM summary generation and `scanner.Scan()` entry point completing the two-step scanner pipeline (Phase 5)
- `init` and `sync` commands rewritten with Bubble Tea `RunWithSpinner`, `generateContextMD` LLM-based context.md generation, and mtime-based delta rescan (Phase 6)
- `buildInjectedMessages` two-pass context injection helper with full token budget logic wired into all 4 query commands (plan, lookup, starter, pattern) (Phase 7)
- `microPassFile` AST symbol-map + LLM line-range micro-pass for large files, replacing symbol-block fallback in `buildInjectedMessages` (Phase 8)

### Known Gaps (accepted as tech debt)

- **CTX-01, CTX-02, CTX-04** — Phase 07 missing VERIFICATION.md; requirements are functionally wired (integration-verified) but not formally verified
- **SYNC-01, SYNC-02** — Phase 06 missing VERIFICATION.md; requirements are functionally wired (integration-verified) but not formally verified
- **ApplyFlagOverrides** absent from all 4 query commands — `--token-limit` flag silently no-ops on plan/lookup/starter/pattern
- **deltaIndex** re-uses stale `Index.Meta` — go.mod changes not reflected in index.json after sync until next `init`

---

## v1.1 Conversational Mode (Shipped: 2026-04-08)

**Phases completed:** 3 phases, 7 plans, 10 tasks

**Key accomplishments:**

- Config.TokenThreshold field (default 4100) with MYHELPER_TOKEN_LIMIT env var, .myhelper/config.json local path, and --token-limit persistent cobra flag
- One-liner:
- Replaced /api/generate client (StreamPrompt) with /api/chat client (StreamChat); all 4 query commands updated to use two-element messages slice (system + user) with history.Message type
- One-liner:
- Multi-turn conversation wired into all 4 query commands via initiateConversation + runConversationLoop pattern
- history.Replace and ollama.Chat(non-streaming) added — the two primitives that runConversationLoop needs to summarize and replace conversation history
- Command-specific summarization prompts and summarize() helper integrated into runConversationLoop — all 4 query commands now automatically condense history when token threshold is exceeded

---

## v1.0 Initial Release (Shipped: 2026-04-07)

**Phases completed:** 1 phases, 4 plans, 7 tasks

**Key accomplishments:**

- Go module scaffolded with cobra CLI, config loading from env/file/defaults, and streaming Ollama client posting to /api/generate via NDJSON line-scan
- context.md loader (LoadContext) and init subcommand that writes a Go-focused commented template without overwriting existing files
- Four cobra subcommands (plan, lookup, starter, pattern) wired to rootCmd with shared helpers for positional-arg-or-interactive-prompt input and prompt assembly
- One-liner:

---
