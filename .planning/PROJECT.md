# myhelper

## What This Is

A Go CLI tool for Go developers that offloads common coding micro-tasks to a locally-hosted LLM (Ollama + qwen2.5-coder:7b). Five focused subcommands — `init`, `plan`, `lookup`, `starter`, `pattern` — give context-aware answers scoped to the current project via `context.md`, with streaming output and no external API dependencies. v1.0 shipped 2026-04-07.

## Core Value

Get a useful, context-aware answer from the local model in one command, without context-bloat or round-trips to an external API.

## Current Milestone: v1.1 Conversational Mode

**Goal:** Turn one-shot commands into rolling conversations with automatic history summarization to stay within model context limits.

**Target features:**
- All 4 query commands (plan, lookup, starter, pattern) enter a conversation loop after the first response
- History accumulates as Q&A pairs; token count tracked via go-tiktoken
- Configurable token threshold (default 4,100); when hit, model summarizes history focused on the original goal
- Re-condensation: subsequent summaries are compressed together with new turns when the threshold is hit again
- Session ends on user typing "quit" or Ctrl+C
- `init` command stays one-shot

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

### Active

- Conversation loop for plan, lookup, starter, pattern commands — v1.1
- Rolling history with go-tiktoken token counting — v1.1 *(infrastructure complete — Phase 2)*
- History summarization when threshold reached (default 4,100 tokens) — v1.1
- Re-condensation of prior summaries with new history — v1.1
- Configurable token threshold (env var / flag) — v1.1 *(CONF-01 validated — Phase 2)*
- Session exit on "quit" input or Ctrl+C — v1.1

### Validated in Phase 2

- ✓ `internal/history` package — `Message`, `History`, `TokenCount()`, `ExceedsLimit()` with tiktoken cl100k_base
- ✓ `Config.TokenThreshold` — default 4100, overridable via `MYHELPER_TOKEN_LIMIT` env var or `--token-limit` flag
- ✓ Local project config path changed to `.myhelper/config.json`
- ✓ Ollama client migrated from `/api/generate` (StreamPrompt) to `/api/chat` (StreamChat) — all 4 query commands updated

### Out of Scope

- Auto-detection of project metadata on init — pure template, user fills it — keeps init fast and predictable
- Global/fallback context.md — per-directory only, avoids cross-project bleed
- Conversation history / multi-turn sessions — single-shot prompts only, respects the 8k context limit
- Web search or external API calls — local inference only
- Non-Go output — tool is optimized for Go projects; other languages are incidental

## Context

- **Inference server**: Ollama at `192.168.0.9:11434`, model `qwen2.5-coder:7b`
- **Model constraint**: ~8k context window — prompts must be short and factual; system prompt + context.md + user query must fit comfortably
- **User workflow**: Developer runs the tool from within a Go project directory; `context.md` at the project root scopes the model's responses to that project's stack and patterns
- **Primary use case**: Solo developer productivity tool; no multi-user, auth, or network exposure concerns
- **Codebase state**: ~488 LOC Go, cobra CLI framework, single binary output
- **Tech stack**: Go, cobra, bufio scanner for NDJSON streaming, config from file/env/defaults

## Constraints

- **Language**: Go — the tool itself is written in Go
- **Model context**: 8k limit — system prompts and context.md must be kept minimal and factual
- **Inference**: Local Ollama only — no external API dependencies
- **Output**: Streaming to stdout — no buffering full responses before display
- **Config**: Endpoint/model overridable via config file or env var — hardcoded defaults are fine for v1

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| context.md is a pure template (no auto-detect) | Keeps init simple and fast; user knows their stack better than heuristics do | ✓ Good — init is instant and predictable |
| Per-directory context.md only | Avoids cross-project bleed; forces explicit initialization per project | ✓ Good — clear mental model |
| Stream output as tokens arrive (bufio.Scanner over NDJSON) | Model is slow enough that streaming feels faster; matches developer expectation | ✓ Good — token-by-token via os.Stdout |
| Single-shot prompts (no history) | Preserves context window budget; each command is self-contained | ✓ Good — well within 8k budget |
| cobra for CLI framework | Standard Go CLI library; enables subcommands cleanly | ✓ Good |
| Config precedence: env > .myhelper.json (CWD) > ~/.config/myhelper/config.json > defaults | Flexible override without requiring config files | ✓ Good |
| Interactive prompts written to stderr | Keeps stdout clean for streamed model output | ✓ Good |
| System prompts kept under 230 chars each | Well within 8k context budget | ✓ Good — leaves room for context.md |
| Binary artifact added to .gitignore | Build artifact not committed | ✓ Good |
| Package alias `stdctx` for stdlib context | `context` package name shadows stdlib; alias avoids confusion | ✓ Good — documented for future contributors |

## Evolution

This document evolves at phase transitions and milestone boundaries.

**After each phase transition** (via `/gsd:transition`):
1. Requirements invalidated? → Move to Out of Scope with reason
2. Requirements validated? → Move to Validated with phase reference
3. New requirements emerged? → Add to Active
4. Decisions to log? → Add to Key Decisions
5. "What This Is" still accurate? → Update if drifted

**After each milestone** (via `/gsd:complete-milestone`):
1. Full review of all sections
2. Core Value check — still the right priority?
3. Audit Out of Scope — reasons still valid?
4. Update Context with current state

---
*Last updated: 2026-04-07 — v1.1 Conversational Mode milestone started*