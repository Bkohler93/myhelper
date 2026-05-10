# myhelper

## What This Is

A Go CLI that provides fast, local-model-powered chat (`myhelper chat`) with optional web search augmentation. For queries needing current information, the tool automatically gates on a yes/no LLM call, fetches results from Tavily or a self-hosted SearXNG instance, re-ranks them, and injects surviving snippets into context before the model responds. `myhelper inspect <query>` is a diagnostic dry-run that prints the full web search decision path — gate, fetch, re-rank, injected block, and token cost — without calling the chat model. All inference runs locally via Ollama — no cloud AI required.

## Core Value

Fast, local AI chat with optional web search — inference runs locally via Ollama, search is pluggable (Tavily or self-hosted SearXNG), no cloud AI required.

## Current Milestone: v5.1 Configuration Validation & Setup Hardening

**Goal:** Remove all hardcoded model/endpoint defaults and fail fast with clear errors when required config is missing — myhelper should never silently use a model the user didn't choose.

**Target features:**
- Remove hardcoded defaults for model (`qwen2.5-coder:7b`) and endpoint (`192.168.0.9:11434`) from config loading
- Config validation on startup: chat, inspect, and search hard-fail with clear error + "run myhelper setup" hint when model or endpoint is unset (env vars count as "set")
- Setup wizard fix: always write a model — if user skips the recommended pull, prompt for an existing local model name before exiting

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
- ✓ `--token-limit` flag correctly caps retrieval budget via `ApplyFlagOverrides` — v1.3
- ✓ Large file handling — micro-pass using `go/ast` symbol map to request line range; truncate as safety net — v1.2
- ✓ SearXNG client (`internal/search/`) — fetches 8–10 results, parses title/url/snippet, configurable endpoint — v3.1
- ✓ Auto-detect search gate — yes/no LLM call; triggers when query needs current/real-time information; fails open — v3.1
- ✓ LLM re-rank pass — filters fetched results to relevant subset; graceful fallback on error or zero results — v3.1
- ✓ Result injection — `[WEB RESULTS]` block prepended to query; token-budget-aware truncation — v3.1
- ✓ `--search` / `--no-search` flags — force or suppress search regardless of gate decision — v3.1
- ✓ Goroutine-based spinners at all 3 search pipeline waits (gate, fetch, re-rank); stdlib only — v3.2
- ✓ SearXNG trailing-slash URL bug fixed; `llmReRank` error surfaced with named variable — v3.2
- ✓ Readline-style input with line editing (arrow keys, backspace, history) — v3.3
- ✓ Multi-line input: `\`-continuation with bare Enter to submit — v3.3
- ✓ Markdown rendering of model responses after stream completes (glamour, erase-and-replace) — v3.3
- ✓ Dead retrieval pipeline deleted — `internal/context`, `internal/planner`, `internal/retrieval`, `internal/scanner` removed — v4.0
- ✓ `--no-context` flag removed — no longer meaningful without retrieval pipeline — v4.0
- ✓ `myhelper inspect <query>` — web search diagnostic dry-run: gate decision, fetched results, re-rank survivors/dropped, injected block preview with token cost — v4.0

### Validated (v5.0)

- ✓ goreleaser build pipeline + GitHub Actions release workflow (darwin/amd64, darwin/arm64, linux/amd64, linux/arm64) — v5.0
- ✓ curl-pipe install script with SHA256 verification and ~/.local/bin install — v5.0
- ✓ Tavily as default search provider with API key in config or MYHELPER_TAVILY_KEY env var — v5.0
- ✓ `myhelper setup` interactive wizard: Ollama check, hardware detection, model pull, Tavily key, SearXNG endpoint — v5.0

### Active

- [ ] Remove hardcoded model/endpoint defaults; require explicit config — v5.1
- [ ] Config validation: hard-fail with clear error when model or endpoint unset — v5.1
- [ ] Setup wizard: always write a model (prompt for existing if pull skipped) — v5.1

### Deferred (post v5.1)

- OpenAI-compatible endpoint support (any `/v1/chat/completions` server, not Ollama-specific)
- Homebrew tap formula for `brew install brettkohler/tap/myhelper`

### Out of Scope

- Nested sub-indexes for extremely large projects (revisit if hierarchical index exceeds budget)
- Global/fallback context.md — per-directory only, avoids cross-project bleed
- Non-Go optimization — tool remains Go-first; other languages incidental
- Conversation history persistence across sessions — single-session only
- Vector/embedding search as a primary retrieval mechanism — structured retrieval is preferred
- Gitignore library dependency — hardcoded skip list covers Go projects
- Iterative retrieval as default-on — adaptive builder handles most cases; optional flag remains possible
- Dynamic call resolution — no type-checker; interface satisfiers and dynamic dispatch out of scope
- Re-implementing .myhelper/ retrieval for chat — deleted in v4.0; chat is web-search-first

## Context

- **Inference server**: Ollama at `192.168.0.9:11434`, model `qwen2.5-coder:7b`
- **Model constraint**: ~8k context window — token threshold at 4,100 triggers summarization before overflow
- **Primary use case**: Solo developer productivity tool; local-only execution
- **Target platforms**: macOS (dev machine, Apple Silicon) + WSL/Linux (Windows work laptop)
- **Codebase state (v5.0)**: `cmd/`: chat, inspect, search, helpers, root, setup; `internal/`: config, history, ollama, search, wizard — lean and live
- **Tech stack**: Go, cobra, chzyer/readline, glamour (markdown rendering), go-tiktoken, Tavily API, SearXNG JSON API
- **Distribution**: goreleaser v2, GitHub Actions, curl-pipe install.sh — multi-platform binaries on GitHub Releases
- **Search providers**: Tavily (API key, free 1k/month) as default when key present; SearXNG (self-hosted) as configurable alternative
- **Known tech debt (v5.0)**:
  - CLAUDE.md Architecture section describes deleted packages — documentation drift, does not affect build
  - `llmReRank` always returns nil error by design — named `reRankErr` branches are technically dead code
  - install.sh extraction path assumes binary at archive root — verify against real goreleaser archive on first tag push

## Constraints

- **Language**: Go
- **Model context**: 8k limit — strict budgeting required
- **Inference**: Local Ollama (primary); any OpenAI-compatible endpoint supported
- **Platform**: macOS (darwin/amd64, darwin/arm64) + WSL/Linux (linux/amd64, linux/arm64)
- **Output**: Streaming to stdout
- **Config**: Endpoint/model/search-provider configurable via env or config file

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Structured index over flat index | Enables hierarchical retrieval and navigation | ✓ Shipped v1.3 |
| Symbol-level indexing via go/ast | Precise retrieval without embeddings | ✓ Strong foundation |
| Hybrid retrieval (deterministic + LLM) | Reduces hallucinated context selection | ✓ Shipped v1.3; deleted v4.0 |
| Adaptive context expansion | Avoids over-injection in small context window | ✓ Shipped v1.3; deleted v4.0 |
| Task-aware retrieval strategies | Different commands need different context | ✓ Shipped v1.3; deleted v4.0 |
| No vector DB by default | Avoids unnecessary complexity; structure is sufficient | ✓ Maintained |
| Fail-open relevance gate | Context omission is worse than extra tokens for 7B | ✓ Applied to both old retrieval and new search gate |
| Streaming output via NDJSON | Improves perceived latency | ✓ Stable |
| SearXNG as search backend | Aggregates multiple engines; self-hostable; no API key | ✓ v3.1 — clean integration |
| Search gate fails open (skips search on LLM error) | Unwanted network call worse than missing context | ✓ v3.1 — GATE-02 |
| LLM re-rank before injection | Reduces irrelevant noise in context window | ✓ v3.1 — RANK-01/02/03 |
| 25% token budget for web context | Leaves 75% for codebase context and history | ✓ v3.1 — INJ-02 |
| num_results=10 hardcoded | CLI surface stays small; default sufficient | ✓ v3.1 — SRCH-04 |
| Spinner in cmd/search.go (not a new file) | Search-layer helpers co-located; stdlib goroutine, no Bubble Tea | ✓ v3.2 — UX-01/02/03 |
| Delete dead retrieval pipeline | v4.0: retrieval packages unused; chat is web-search-first | ✓ v4.0 — 5,500 lines removed, build cleaner |
| inspect rewritten as web search diagnostic | Old inspect showed retrieval decisions (now moot); new inspect shows gate/fetch/re-rank/block | ✓ v4.0 — real LLM+SearXNG calls, diagnostic dry-run |
| --search in inspect always runs re-rank | Diagnostic parity: inspect --search should show full pipeline same as gate=YES path | ✓ v4.0 — INSP-06 |
| goreleaser v2 schema (version: 2) | v2 schema required for goreleaser-action@v7; version: 2 header mandatory | ✓ v5.0 — DIST-02 |
| Wizard uses io.Reader/io.Writer injection | Enables hermetic unit tests with httptest; avoids os.Stdin hardcoding | ✓ v5.0 — SETUP |
| mergeHomeConfig map-based merge | Preserves unrelated config keys (endpoint, model, token_threshold) when writing tavily_key or search_endpoint | ✓ v5.0 — SETUP-05/06 |
| Tavily ordered before SearXNG in Phase 29 | Wizard must write tavily_key config field; provider must exist before wizard can configure it | ✓ v5.0 — ordering |
| Homebrew tap deferred | curl installer covers WSL primary use case; tap adds CI complexity | — v5.0; DIST-F01 |
| OpenAI-compatible endpoint deferred | Ollama-only for v5.0; OpenAI-compat is a future nice-to-have | — v5.0; INFER-F01 |

## Evolution

This document evolves at milestone boundaries.

**After each milestone** (via `/gsd-complete-milestone`):
1. Full review of all sections
2. Core Value validation
3. Out of Scope audit
4. Context + architecture update

---
*Last updated: 2026-05-10 — v5.1 Configuration Validation & Setup Hardening milestone started*
