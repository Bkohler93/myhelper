# myhelper

## What This Is

A Go CLI that provides fast, local-model-powered chat (`myhelper chat`) with optional web search augmentation via SearXNG. For queries needing current information, the tool automatically gates on a yes/no LLM call, fetches and re-ranks results, then injects surviving snippets into the context before the model responds. Ad-hoc query commands (`starter`, `lookup`, `pattern`, `plan`) remain for project-aware coding assistance using a hierarchical codebase index. No external API dependencies — all inference is local Ollama.

## Core Value

Fast, local chat with optional web search for current information — powered by a local Ollama model, no external API dependencies required.

## Current Milestone: v4.0 Search-First Simplification

**Goal:** Remove the dead .myhelper/ retrieval pipeline and rewrite `inspect` to show the full web search decision path.

**Target features:**
- Purge dead packages: `internal/context`, `internal/planner`, `internal/retrieval`, `internal/scanner`
- Remove `--no-context` flag (no longer meaningful without retrieval pipeline)
- Rewrite `inspect` as a web search diagnostic dry-run: gate decision, fetched results, re-rank survivors vs dropped, injected block preview

## Requirements

### Validated

- ✓ `init` command — scans project and generates `.myhelper/context.md`, artifact files, and per-package summaries — v1.2/v1.3
- ✓ `sync` command — refreshes index/summaries when `.myhelper/` already exists (mtime-based delta rescan) — v1.2
- ✓ `plan` command — breaks a feature/task description into ordered subtasks — v1.0
- ✓ `lookup` command — recommends the right API or library for a given step — v1.0
- ✓ `starter` command — prints minimal working Go code for a given task — v1.0
- ✓ `pattern` command — describes the idiomatic Go way to write or structure something — v1.0
- ✓ All commands accept input as a CLI argument; prompt interactively if argument is omitted — v1.0
- ✓ All commands stream model output to stdout as tokens arrive — v1.0
- ✓ `context.md` in the current directory is read and prepended to every prompt — v1.0
- ✓ Ollama endpoint and model are configurable (default: `192.168.0.9:11434`, `qwen2.5-coder:7b`) — v1.0
- ✓ `internal/history` package — `Message`, `History`, `TokenCount()`, `ExceedsLimit()` with tiktoken cl100k_base — v1.1
- ✓ `Config.TokenThreshold` — default 4100, overridable via `MYHELPER_TOKEN_LIMIT` env var or `--token-limit` flag — v1.1
- ✓ Local project config path `.myhelper/config.json` — v1.1
- ✓ Ollama client uses `/api/chat` (StreamChat) for streaming responses — v1.1
- ✓ Conversation loop for plan, lookup, starter, pattern — multi-turn with stdin injection for testing — v1.1
- ✓ Session exit on "quit" input or Ctrl+C (SIGINT handler scoped to loop lifetime) — v1.1
- ✓ History summarization when token threshold reached — command-specific prompts — v1.1
- ✓ Re-condensation: prior summary + new turns condensed together on subsequent threshold hits — v1.1
- ✓ Hierarchical index structure — `project.json`, `packages.json`, `files.json`, `symbols.json` replacing flat `index.json` — v1.3
- ✓ Symbol-level indexing — kind, signature, line range, call edges, type refs, stable IDs via `go/ast` — v1.3
- ✓ Hybrid retrieval pipeline — relevance gate → deterministic pre-filter → LLM re-ranking → dep expansion — v1.3
- ✓ Dependency-aware expansion — depth-1 import graph neighbors, bounded at ≤ 60% remaining budget — v1.3
- ✓ Adaptive context builder — staged assembly (project summary → symbols → file list → file content) — v1.3
- ✓ Task-aware retrieval strategies per command — v1.3
- ✓ Context relevance gate — skip injection when repo context not needed — v1.3
- ✓ `inspect` command — expose retrieval decisions, selected context, and token usage per stage — v1.3
- ✓ `--no-context` flag on all query commands — bypass all project-based context pulling — v1.3
- ✓ `--token-limit` flag correctly caps retrieval budget via `ApplyFlagOverrides` — v1.3
- ✓ Large file handling — micro-pass using `go/ast` symbol map to request line range; truncate as safety net — v1.2

- ✓ SearXNG client (`internal/search/`) — fetches 8–10 results, parses title/url/snippet, configurable endpoint — v3.1
- ✓ Auto-detect search gate — yes/no LLM call; triggers when query needs current/real-time information; fails open — v3.1
- ✓ LLM re-rank pass — filters fetched results to relevant subset; graceful fallback on error or zero results — v3.1
- ✓ Result injection — `[WEB RESULTS]` block prepended to query; token-budget-aware truncation — v3.1
- ✓ `--search` / `--no-search` flags — force or suppress search regardless of gate decision — v3.1

- ✓ `inspect` dry-run command — per-stage gate/pre-filter/re-rank/metrics diagnostics; `--no-context` bypass — v3.2
- ✓ Goroutine-based spinners at all 3 search pipeline waits (gate, fetch, re-rank); stdlib only — v3.2
- ✓ SearXNG trailing-slash URL bug fixed; `llmReRank` error surfaced with named variable — v3.2
- ✓ `countTokens` duplicate removed; `pkgs` param removed from `llmReRank`; `CallEdges`/`TypeRefs` reserved — v3.2
- ✓ `microPassFile` uses stored `Symbol.Start/End` — eliminates per-call AST re-parse — v3.2

- ✓ Readline-style input with line editing (arrow keys, backspace, history) — v3.3
- ✓ Multi-line input: `\`-continuation with bare Enter to submit — v3.3
- ✓ Markdown rendering of model responses after stream completes (glamour, erase-and-replace) — v3.3

### Active

- Delete `internal/context`, `internal/planner`, `internal/retrieval`, `internal/scanner` packages — v4.0
- Remove `--no-context` flag from root.go — v4.0
- `myhelper inspect <query>` prints web search gate decision, fetched results, re-rank results, and injected block preview — v4.0

### Out of Scope

- Nested sub-indexes for extremely large projects (revisit if hierarchical index exceeds budget)
- Global/fallback context.md — per-directory only, avoids cross-project bleed
- Non-Go optimization — tool remains Go-first; other languages incidental
- Conversation history persistence across sessions — single-session only
- Vector/embedding search as a primary retrieval mechanism — structured retrieval is preferred
- Gitignore library dependency — hardcoded skip list covers Go projects
- Iterative retrieval as default-on — adaptive builder handles most cases; optional flag remains possible
- Dynamic call resolution — no type-checker; interface satisfiers and dynamic dispatch out of scope

## Context

- **Inference server**: Ollama at `192.168.0.9:11434`, model `qwen2.5-coder:7b`
- **Model constraint**: ~8k context window — token threshold at 4,100 triggers summarization before overflow
- **Retrieval constraint**: context must fit within ~80% of token threshold after system + history
- **User workflow**: Developer runs tool inside a Go project; `init` builds structured index; query commands retrieve minimal context and expand as needed
- **Primary use case**: Solo developer productivity tool; local-only execution
- **Codebase state (v3.3)**: `cmd/`: chat, inspect, search, helpers, root; `internal/`: config, context, history, ollama, planner, retrieval, scanner, search (context/planner/retrieval/scanner are dead — scheduled for removal in v4.0)
- **Tech stack**: Go, cobra, bufio scanner (NDJSON streaming), go-tiktoken, go/ast, JSON-based index, `internal/retrieval` pipeline, SearXNG JSON API
- **Known tech debt (v3.2)**:
  - `Symbol.CallEdges`/`TypeRefs` stored but not consumed — documented as reserved for future ranking
  - `llmReRank` always returns nil error by design — named `reRankErr` branches are technically dead code
  - `ExtractSymbolsFull` doc comment says "populated by walking function bodies" — contradicts CLN-03 reserved-for-future struct comments (doc only)

## Constraints

- **Language**: Go
- **Model context**: 8k limit — strict budgeting required
- **Inference**: Local Ollama only
- **Output**: Streaming to stdout
- **Config**: Endpoint/model configurable via env or config file

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Structured index over flat index | Enables hierarchical retrieval and navigation | ✓ Shipped v1.3 |
| Symbol-level indexing via go/ast | Precise retrieval without embeddings | ✓ Strong foundation |
| Hybrid retrieval (deterministic + LLM) | Reduces hallucinated context selection | ✓ Replaces pure LLM selection |
| Adaptive context expansion | Avoids over-injection in small context window | ✓ Critical for 7B models |
| Task-aware retrieval strategies | Different commands need different context | ✓ Improves output quality |
| No vector DB by default | Avoids unnecessary complexity; structure is sufficient | ✓ Maintains simplicity |
| Fail-open relevance gate | Context omission is worse than extra tokens for 7B | ✓ Correct default |
| Small corpus (≤40 files) bypass | All symbols pass as additive hints below threshold | ✓ Avoids over-filtering small projects |
| microPassFile in retrieval package | Enforces retrieval-logic-in-retrieval-package rule | ✓ Clean separation |
| context.md remains user-defined | Avoids unreliable auto-detection | ✓ Stable |
| Streaming output via NDJSON | Improves perceived latency | ✓ Stable |
| Iterative retrieval (optional) | Allows model to request missing context | — Pending (not yet implemented) |
| SearXNG as search backend | Aggregates multiple engines; self-hostable; no API key | ✓ v3.1 — clean integration |
| Search gate fails open (skips search on LLM error) | Unwanted network call worse than missing context | ✓ v3.1 — GATE-02 |
| LLM re-rank before injection | Reduces irrelevant noise in context window | ✓ v3.1 — RANK-01/02/03 |
| 25% token budget for web context | Leaves 75% for codebase context and history | ✓ v3.1 — INJ-02 |
| num_results=10 hardcoded | CLI surface stays small; default sufficient per REQUIREMENTS.md | ✓ v3.1 — SRCH-04 |
| Inline gate logic in BuildInspectContext | Captures raw LLM answer without changing stable needsContext used by BuildContext | ✓ v3.2 — INSP-02 |
| Spinner in cmd/search.go (not a new file) | Search-layer helpers co-located; stdlib goroutine, no Bubble Tea, done() at call site | ✓ v3.2 — UX-01/02/03 |
| pkgs param removed from llmReRank (not wired in) | No consumer existed; removal is cleaner than adding unused wiring | ✓ v3.2 — CLN-02 |
| CTX-03 closed without code change | LoadContext defined but never called — no dual injection path exists | ✓ v3.2 — CTX-03 |
| microPassFile falls back to ExtractSymbolMap when relevantSyms empty | Preserves correctness for unindexed files while eliminating AST re-parse for indexed ones | ✓ v3.2 — PERF-01 |

## Evolution

This document evolves at milestone boundaries.

**After each milestone** (via `/gsd-complete-milestone`):
1. Full review of all sections
2. Core Value validation
3. Out of Scope audit
4. Context + architecture update

---
*Last updated: 2026-04-25 — v4.0 milestone started; Search-First Simplification (dead package purge + inspect rewrite)*
