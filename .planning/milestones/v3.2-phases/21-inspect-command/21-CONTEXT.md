# Phase 21: inspect Command - Context

**Gathered:** 2026-04-24
**Status:** Ready for planning
**Mode:** Auto-generated (discuss skipped via workflow.skip_discuss)

<domain>
## Phase Boundary

Users can run `myhelper inspect <query>` to see exactly which symbols and files the retrieval pipeline would select, and why, without triggering a model response.

Wire `cmd/inspect.go` as a new cobra subcommand that calls the already-implemented `BuildInspectContext` function at `internal/retrieval/retrieval.go:776`. Extend `InspectResult` with pre-filter candidate storage to satisfy INSP-03. Format and print per-stage diagnostics to stdout.

</domain>

<decisions>
## Implementation Decisions

### Claude's Discretion
All implementation choices are at Claude's discretion — discuss phase was skipped per user setting. Use ROADMAP phase goal, success criteria, and codebase conventions to guide decisions.

Key facts to respect:
- `BuildInspectContext` already exists at `internal/retrieval/retrieval.go:776` — do NOT re-implement it
- `InspectResult` struct needs a `PreFilterCandidates` field (or equivalent) added to expose pre-filter results (INSP-03)
- The current `InspectResult` already tracks: `Symbols []SymbolResult`, `Files []FileResult`, `StageMetrics []StageMetrics`, `FinalTokens int`, `GatePassed bool`
- Pre-filter candidates are computed via `preFilter()` in `BuildInspectContext` but not stored — add storage there
- `--no-context` flag should skip all retrieval stages and print a bypass message (INSP-05)
- Output format: plain text to stdout; follow the style of existing `inspect`-adjacent code in the repo
- The `inspect` command should use the `StarterStrategy` or a dedicated `InspectStrategy` — check existing strategies

</decisions>

<code_context>
## Existing Code Insights

### Reusable Assets
- `BuildInspectContext` in `internal/retrieval/retrieval.go:776` — the core inspect engine
- `InspectResult` struct at `internal/retrieval/retrieval.go:760` — needs `PreFilterCandidates` added
- `preFilter()` function in `internal/retrieval/retrieval.go` — returns `[]scanner.Symbol`; its output needs to be captured in `InspectResult`
- `resolveInput` in `cmd/helpers.go` — shared positional-arg-or-prompt helper
- `rootCmd` and flag registration pattern in `cmd/root.go`
- Existing command files (`cmd/search.go`, `cmd/helpers.go`) for structural reference

### Established Patterns
- Commands register via `init()` in their `cmd/*.go` file and call `rootCmd.AddCommand()`
- All query commands use `resolveInput` for input handling
- `--no-context` flag is a root-level persistent flag (check `cmd/root.go`)

### Integration Points
- New file: `cmd/inspect.go`
- Modified: `internal/retrieval/retrieval.go` — add `PreFilterCandidates []scanner.Symbol` to `InspectResult`; capture pre-filter output in `BuildInspectContext`
- Wire into `rootCmd` via `init()` in `cmd/inspect.go`

</code_context>

<specifics>
## Specific Ideas

- Output should show: gate decision (pass/fail + raw answer) → pre-filter count + list → re-rank survivors vs dropped
- `--no-context` behavior: print "Context bypassed (--no-context)" and exit; do not call BuildInspectContext
- No streaming — `inspect` is diagnostic output only

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>
