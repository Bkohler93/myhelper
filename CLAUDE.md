# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# Build
go build ./...

# Install binary to $GOPATH/bin
go install .

# Run all tests
go test ./...

# Run a single test
go test ./internal/retrieval/... -run TestBuildContext

# Run tests in a specific package
go test ./internal/scanner/...

# Run tests with verbose output
go test -v ./cmd/...
```

## Configuration

Config resolution order (highest to lowest priority):
1. Env vars: `MYHELPER_ENDPOINT`, `MYHELPER_MODEL`, `MYHELPER_TOKEN_LIMIT`
2. `.myhelper/config.json` in CWD, then `~/.config/myhelper/config.json`
3. Hardcoded defaults: threshold `4100` only (endpoint and model have no default — run `myhelper setup`)

## Architecture

### How a command works end-to-end

Every command (starter, plan, lookup, pattern) follows the same flow:

1. **Resolve input** — positional arg or interactive prompt via `cmd/helpers.go:resolveInput`
2. **Load project context** — `internal/context/context.go:LoadContext` reads `.myhelper/context.md`; returns empty string if absent (graceful degradation)
3. **Build retrieval context** — `internal/retrieval/retrieval.go:BuildContext` runs the 4-stage pipeline (see below), returns assembled `[]history.Message`
4. **Initiate conversation** — `cmd/helpers.go:initiateConversation` sends the initial messages to Ollama and streams the response
5. **Conversation loop** — `cmd/helpers.go:runConversationLoop` handles multi-turn follow-ups, SIGINT, and history summarization

### The retrieval pipeline (`internal/retrieval/retrieval.go`)

`BuildContext` runs four stages to select project symbols and files to inject as context:

1. **Relevance gate** — LLM answers yes/no: "does this query need source code?" Fails open (skips only on explicit "no").
2. **Pre-filter** — Deterministic keyword scoring of all symbols. Small corpus (≤40 files): all symbols pass. Large corpus: only scoring-positive symbols pass.
3. **LLM re-ranking** — LLM confirms which pre-filtered candidates are relevant by returning their `stableID`s. Falls back to all candidates on failure.
4. **Dependency expansion** — Adds depth-1 import neighbors of selected files, bounded by 60% of remaining token budget.

Each command uses a `Strategy` (`StarterStrategy`, `PlanStrategy`, etc.) that controls which stages run and the token budget ratio.

### Token budget management (`internal/history/history.go`)

Uses `tiktoken` (`cl100k_base` encoding) to count tokens. `History.ExceedsLimit()` triggers summarization in the conversation loop. Summarization preserves `messages[0]` (system prompt) and the last user+assistant pair; everything in between is condensed via a non-streaming `ollama.Chat` call.

### Artifact files (`.myhelper/`)

`myhelper init` produces four JSON files consumed by the retrieval pipeline:
- `project.json` — module path, go version, file/symbol counts, LLM-generated project summary
- `packages.json` — per-package import paths, file lists, and LLM-generated responsibility strings
- `files.json` — per-file exported names and imports
- `symbols.json` — all exported symbols with full rich profiles (name, kind, signature, stableID, filePath)

`myhelper sync` does delta updates: only re-indexes and re-summarizes packages containing changed files (mtime-based), then rebuilds all four artifact files.

### Ollama client (`internal/ollama/client.go`)

Two functions:
- `StreamChat` — streams tokens to stdout, returns accumulated string. Used for user-facing responses.
- `Chat` — non-streaming, returns full response. Used internally for relevance gate, re-ranking, and summarization.

### Command strategies

| Command   | UseSymbols | UseFiles | MaxTokenRatio |
|-----------|-----------|----------|---------------|
| `starter` | true      | true     | 0.80          |
| `plan`    | false     | false    | 0.50          |
| `lookup`  | true      | false    | 0.30          |
| `pattern` | false     | false    | 0.10          |
| `inspect` | n/a (dry run) | n/a | n/a           |

`inspect` is a dry-run that prints per-stage diagnostics (gate result, tokens per stage, selected symbols/files) without assembling or sending messages.

### Large file handling

When a file's full content would exceed the token budget, `retrieval.microPassFile` asks the model which line range is needed (using a symbol map as a hint), extracts that range, and falls back to truncation if the range still doesn't fit.

### `--no-context` flag

Bypasses the entire retrieval pipeline; sends only the bare user query. Available on all commands via `rootCmd`.
