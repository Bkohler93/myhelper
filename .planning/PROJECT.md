# myhelper

## What This Is

A Go CLI tool for Go developers that offloads common coding micro-tasks to a locally-hosted LLM (Ollama + qwen2.5-coder:7b). Six focused subcommands — `init`, `sync`, `plan`, `lookup`, `starter`, `pattern`, `inspect` — provide project-aware answers by building and querying a structured, token-budgeted representation of the codebase. Instead of injecting large amounts of raw code, the tool enables the model to navigate a hierarchical index (project → package → file → symbol) and pull only the context it needs through a four-stage retrieval pipeline. All four query commands are interactive multi-turn conversations that automatically compress history when the token threshold is hit. No external API dependencies. v1.3 shipped 2026-04-10.

## Core Value

Get a precise, project-aware answer from a local 7B model by enabling it to navigate a structured map of the codebase—without context bloat or external APIs.

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

### Active

- [ ] Symbol summaries — LLM-generated one-line summary per exported symbol stored in `symbols.json` (SUM-01, SUM-02)
- [ ] Consume stored `CallEdges`/`TypeRefs` in retrieval — currently stored but not read by pipeline
- [ ] Eliminate dual context injection — `context.md` + `proj.Summary` both carry project description from same source
- [ ] Phase 11 VERIFICATION.md — close process gap for RET-01–06

### Out of Scope

- Nested sub-indexes for extremely large projects (revisit if hierarchical index exceeds budget)
- Global/fallback context.md — per-directory only, avoids cross-project bleed
- Web search or external API calls — local inference only
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
- **Codebase state (v1.3)**: ~3,200 LOC Go (source, estimated), ~2,900 LOC tests; 37 source files
- **Tech stack**: Go, cobra, Bubble Tea, bufio scanner (NDJSON streaming), go-tiktoken, go/ast, JSON-based index, `internal/retrieval` pipeline
- **Known tech debt (v1.3)**:
  - `Symbol.CallEdges`/`TypeRefs` stored but not consumed by retrieval pipeline
  - `PackageEntry.Responsibility` written to `packages.json` but unused in `llmReRank`
  - Dual context injection (`context.md` + `proj.Summary`) — same source, redundant tokens
  - Phase 11 VERIFICATION.md missing — RET-01–06 confirmed by downstream but not formally verified
  - `inspect` ignores `--no-context` flag
  - `BuildContext`/`BuildInspectContext` silently discard `llmReRank` error return

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

## Evolution

This document evolves at milestone boundaries.

**After each milestone** (via `/gsd-complete-milestone`):
1. Full review of all sections
2. Core Value validation
3. Out of Scope audit
4. Context + architecture update

---
*Last updated: 2026-04-10 after v1.3 Structured Code Intelligence milestone*
