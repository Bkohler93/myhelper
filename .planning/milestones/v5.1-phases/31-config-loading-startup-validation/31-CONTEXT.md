# Phase 31: Config Loading & Startup Validation - Context

**Gathered:** 2026-05-10
**Status:** Ready for planning
**Mode:** Auto-generated (discuss skipped via workflow.skip_discuss)

<domain>
## Phase Boundary

myhelper refuses to run without explicit model and endpoint configuration, and tells the user exactly how to fix it. Remove hardcoded defaults for model (`qwen2.5-coder:7b`) and endpoint (`192.168.0.9:11434`) from config loading. `chat`, `inspect`, and `search` hard-fail with a clear error message and "run myhelper setup" hint when model or endpoint is unset. Env vars (`MYHELPER_MODEL`, `MYHELPER_ENDPOINT`) count as "set" for validation purposes.

Requirements in scope: CFG-01, CFG-02, VAL-01, VAL-02, VAL-03, VAL-04, VAL-05

</domain>

<decisions>
## Implementation Decisions

### Claude's Discretion
All implementation choices are at Claude's discretion — discuss phase was skipped per user setting. Use ROADMAP phase goal, success criteria, and codebase conventions to guide decisions.

Key constraints from STATE.md and REQUIREMENTS.md:
- Hard fail (not auto-redirect) on missing config — simpler, more predictable than silently launching setup
- Env vars (`MYHELPER_MODEL`, `MYHELPER_ENDPOINT`) count as "set" for validation purposes
- `myhelper config set` subcommand is out of scope
- Error format must be consistent across chat, inspect, and search commands
- `default_token_threshold` (4100) is an internal tuning param — retain its default, do not remove

</decisions>

<code_context>
## Existing Code Insights

### Reusable Assets
- `internal/config/` — config loading package; currently sets hardcoded defaults for model and endpoint
- `cmd/chat.go`, `cmd/inspect.go`, `cmd/search.go` — entry points that receive loaded config; validation goes here or in a shared helper
- `cmd/helpers.go` — shared command utilities; good place for a `validateConfig` helper

### Established Patterns
- Config is loaded via `internal/config` package, returned as a struct to commands
- Commands call `initiateConversation` / `runConversationLoop` from helpers.go
- Errors are printed with `fmt.Fprintf(os.Stderr, ...)` and exit via `os.Exit(1)` or cobra's `RunE` return

### Integration Points
- `internal/config/config.go` — remove hardcoded defaults for model and endpoint fields
- `cmd/chat.go`, `cmd/inspect.go`, `cmd/search.go` — add validation before any Ollama calls
- Env var resolution already happens in config loading (`MYHELPER_MODEL`, `MYHELPER_ENDPOINT` already mapped)

</code_context>

<specifics>
## Specific Ideas

- Success criteria require the error to include a "run myhelper setup" hint
- Consistent error format across all three commands (chat, inspect, search)
- Config loading must return empty string (not default) for model and endpoint when unset in both config file and env

</specifics>

<deferred>
## Deferred Ideas

None — discuss phase skipped.

</deferred>
