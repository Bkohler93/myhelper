# myhelper

## What This Is

A Go CLI tool for Go developers that offloads common coding micro-tasks to a locally-hosted LLM (Ollama + qwen2.5-coder:7b). Five focused subcommands — `init`, `sync`, `plan`, `lookup`, `starter`, `pattern` — give project-aware answers by auto-scanning the codebase into a token-budgeted index and injecting only relevant file content into each query via two-pass retrieval. All four query commands are interactive: after the first response, the user can ask follow-ups in a rolling conversation that automatically compresses history when the token threshold is hit. No external API dependencies. v1.2 shipped 2026-04-08.

## Core Value

Get a useful, project-aware answer from the local model in one command, without context-bloat or round-trips to an external API.

## Requirements

### Validated

- ✓ `init` command — writes a blank `context.md` template to the current directory — v1.0
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
- ✓ Ollama client migrated from `/api/generate` (StreamPrompt) to `/api/chat` (StreamChat) — v1.1
- ✓ Conversation loop for plan, lookup, starter, pattern — multi-turn with stdin injection for testing — v1.1
- ✓ Session exit on "quit" input or Ctrl+C (SIGINT handler scoped to loop lifetime) — v1.1
- ✓ History summarization when token threshold reached — command-specific prompts — v1.1
- ✓ Re-condensation: prior summary + new turns condensed together on subsequent threshold hits — v1.1
- ✓ `init` auto-scans project and generates `.myhelper/context.md`, `.myhelper/index.json`, and per-package summaries — token-budgeted to fit `Config.TokenThreshold` — v1.2
- ✓ `sync` command refreshes index/summaries when `.myhelper/` already exists (mtime-based delta rescan) — v1.2
- ✓ Two-pass context injection in all 4 query commands: index → model selects files → inject content — v1.2
- ✓ Large file handling: micro-pass using `go/ast` symbol map to request line range; truncate as safety net — v1.2

### Active

(None — planning next milestone)

### Out of Scope

- Nested sub-indexes for large projects when flat `index.json` exceeds context budget — auto-triggered when flat index overflows (revisit v1.3 if needed)
- Global/fallback context.md — per-directory only, avoids cross-project bleed
- Web search or external API calls — local inference only
- Non-Go output — tool is optimized for Go projects; other languages are incidental
- Conversation history persistence across sessions — single-session only; stateless between invocations
- Vector/embedding search — the two-pass LLM-as-retriever design makes this unnecessary and avoids infrastructure overhead
- Gitignore library dependency — hardcoded skip list covers Go projects; avoids an unmaintained dependency

## Context

- **Inference server**: Ollama at `192.168.0.9:11434`, model `qwen2.5-coder:7b`
- **Model constraint**: ~8k context window — token threshold at 4,100 triggers summarization before the window fills; two-pass injection budget is 80% of threshold
- **User workflow**: Developer runs the tool from within a Go project directory; `init` scans the project into `.myhelper/`; each query command does a two-pass retrieval (index → file selection → content injection); multi-turn follow-ups work within the same invocation
- **Primary use case**: Solo developer productivity tool; no multi-user, auth, or network exposure concerns
- **Codebase state**: ~2,147 LOC Go (source), ~2,331 LOC tests; Bubble Tea TUI for spinner feedback; go/ast for symbol extraction
- **Tech stack**: Go, cobra, Bubble Tea, bufio scanner for NDJSON streaming, go-tiktoken (cl100k_base), go/ast, config from file/env/defaults
- **Known tech debt**: `ApplyFlagOverrides` not called in query commands (--token-limit flag silently no-ops); sync guard checks `meta.json` instead of `index.json`; `generateContextMD` fails fast on empty summaries dir

## Constraints

- **Language**: Go — the tool itself is written in Go
- **Model context**: 8k limit — system prompts and context.md must be kept minimal and factual; summarization budget: 4,100 tokens for history before compression
- **Inference**: Local Ollama only — no external API dependencies
- **Output**: Streaming to stdout — no buffering full responses before display
- **Config**: Endpoint/model overridable via config file or env var — hardcoded defaults are fine

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| context.md is a pure template (no auto-detect) | Keeps init simple and fast; user knows their stack better than heuristics do | ✓ Good — init is instant and predictable (v1.0) |
| Per-directory context.md only | Avoids cross-project bleed; forces explicit initialization per project | ✓ Good — clear mental model |
| Stream output as tokens arrive (bufio.Scanner over NDJSON) | Model is slow enough that streaming feels faster; matches developer expectation | ✓ Good — token-by-token via os.Stdout |
| cobra for CLI framework | Standard Go CLI library; enables subcommands cleanly | ✓ Good |
| Config precedence: env > .myhelper/config.json (CWD) > defaults | Flexible override without requiring config files | ✓ Good |
| Interactive prompts written to stderr | Keeps stdout clean for streamed model output | ✓ Good |
| System prompts kept under 230 chars each | Well within 8k context budget | ✓ Good — leaves room for context.md |
| Binary artifact added to .gitignore | Build artifact not committed | ✓ Good |
| Package alias `stdctx` for stdlib context | `context` package name shadows stdlib; alias avoids confusion | ✓ Good |
| cl100k_base tiktoken encoding for token counting | Consistent counting; tiktoken is the de-facto standard | ✓ Good — ExceedsLimit() accurate |
| history.New() panics on encoder load failure | Programmer error, not runtime error | ✓ Good — fast fail |
| history.Messages() returns a copy | Prevents callers from mutating internal history state | ✓ Good |
| stdinReader package-level var for test injection | Avoids cobra dependency in cmd package; tests inject stdin without refactoring | ✓ Good |
| streamFn injected as parameter to runConversationLoop | Tests never call real ollama.StreamChat; pure unit tests | ✓ Good |
| SIGINT handler inside runConversationLoop | Signal lifetime matches loop lifetime; no global state | ✓ Good |
| ollama.Chat uses Stream: false | Non-streaming for summarization; single JSON response, no stdout writes | ✓ Good |
| history.Replace uses make+copy pattern | Copy-safe; matches Messages() convention | ✓ Good |
| summarize() calls ollama.Chat directly (not streamFn) | Non-streaming internal op; injection not needed here | ✓ Good |
| len(msgs) < 5 guard in summarize() | No content to compress when only system+user+assistant exist | ✓ Good |
| Re-condensation detected via "Summary of previous conversation:" prefix | Simple, no extra state; same code path as first summarization | ✓ Good |
| Two-pass LLM retrieval over vector/embedding search | No infrastructure overhead; model already understands the index format | ✓ Good — no extra deps (v1.2) |
| go/ast for symbol extraction (stdlib) | Zero external deps; accurate for Go projects | ✓ Good (v1.2) |
| Bubble Tea spinner for init/sync TUI | Smooth feedback during long scans; composable with Cobra | ✓ Good (v1.2) |
| mtime-based delta rescan in sync | Simpler than hash tracking; accurate for single-machine dev workflow | ✓ Good (v1.2) |
| Token budget at 80% of threshold per file | Leaves headroom for Pass-1 overhead and conversation history | ✓ Good (v1.2) |
| microPassFile asks LLM for line range before truncation | Preserves relevant context even for oversized files | ✓ Good (v1.2) |
| Sync guard checks meta.json not index.json | Overly strict on interrupted init — should check index.json | ⚠️ Revisit (tech debt, v1.2) |
| generateContextMD fails fast on empty summaries | Aborts init/sync if no packages have exported symbols | ⚠️ Revisit (tech debt, v1.2) |

## Evolution

This document evolves at phase transitions and milestone boundaries.

**After each milestone** (via `/gsd:complete-milestone`):
1. Full review of all sections
2. Core Value check — still the right priority?
3. Audit Out of Scope — reasons still valid?
4. Update Context with current state

---
*Last updated: 2026-04-08 after v1.2 Smart Context milestone*
