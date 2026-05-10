# Phase 32: Setup Wizard Hardening - Context

**Gathered:** 2026-05-10
**Status:** Ready for planning

<domain>
## Phase Boundary

The setup wizard (`internal/wizard/wizard.go:Run`) is modified so it always writes a usable model and Ollama endpoint to config before exiting — regardless of which path the user takes through the prompts. Two gaps are closed:

1. When the user declines the recommended model pull (or the pull fails), the wizard now prompts for an existing local model name before saving, ensuring a model field always lands in config.
2. The wizard adds an Ollama endpoint prompt (Stage 1.5) so the `endpoint` field is always written to config. Without this, `validateConfig()` added in Phase 31 will error on every `chat`/`inspect` call even after running `myhelper setup`.

No changes to any command other than the wizard (`cmd/setup.go` entry point unchanged; all logic is in `internal/wizard/wizard.go`).

</domain>

<decisions>
## Implementation Decisions

### Ollama Endpoint Prompt
- Placement: Stage 1.5 — immediately after reachability check passes (before model recommendation)
- Default shown in prompt: `http://localhost:11434` (the URL that just passed the probe)
- User hits Enter: write the pre-fill default (`http://localhost:11434`) to config
- Validation: non-empty AND must begin with `http://` or `https://` (same pattern as existing SearXNG validation)
- If validation fails: print warning, loop and re-prompt

### Skip-Model Fallback UX
- Prompt text: `"Enter the name of a local model (run 'ollama list' to see available): "`
- If user enters empty string at fallback: loop once and re-prompt; if still empty, return an error — wizard cannot exit without a model
- If model pull fails (network/timeout): fall through to the "enter existing model" fallback prompt (same as if user said N)
- Write timing: write model to config immediately when name is confirmed (same as current pull-success path, not batched)

### Claude's Discretion
- Exact loop prompt re-phrasing (e.g., "Please enter a model name:" on second attempt) is at Claude's discretion
- Error message wording on second-empty-entry exit is at Claude's discretion

</decisions>

<code_context>
## Existing Code Insights

### Reusable Assets
- `mergeHomeConfig(map[string]interface{}{...})` — used by Stage 3 (model), Stage 4 (Tavily), Stage 5 (SearXNG); same function handles endpoint write
- `ollamaBaseURL` package var — injectable in tests; default `http://localhost:11434`; already used for Stage 1 probe
- `configPathOverride` package var — injectable in tests; redirects all `mergeHomeConfig` writes to `t.TempDir()`
- `strings.HasPrefix(line, "http://") || strings.HasPrefix(line, "https://")` — existing validation pattern (Stage 5, SearXNG)

### Established Patterns
- All prompts use `fmt.Fprintf(w, "...") + br.ReadString('\n') + strings.TrimSpace(line)`
- Optional fields (Tavily, SearXNG) use `if line != ""` guard before writing
- Required fields must loop until valid or return error (new pattern for this phase)
- Config writes via `mergeHomeConfig(map[string]interface{}{"key": value})` — separate calls per field

### Integration Points
- `internal/wizard/wizard.go:Run()` — all changes here; no changes to `cmd/setup.go`
- `internal/wizard/wizard_test.go` — test harness uses `strings.NewReader` for input injection and `configPathOverride` for config isolation; new tests follow this pattern
- After Phase 31: `validateConfig()` in `cmd/helpers.go` checks both `cfg.Endpoint` and `cfg.Model`; if wizard exits without writing either, the user gets an error on next `myhelper chat`

</code_context>

<specifics>
## Specific Ideas

- Stage 1.5 endpoint prompt: `"Ollama endpoint [http://localhost:11434]: "` — bracket style shows the default without a colon separator ambiguity
- The `ollamaBaseURL` var (used for Stage 1 probe) is the natural source for the pre-fill default in the prompt

</specifics>

<deferred>
## Deferred Ideas

- Probing reachability of a user-changed endpoint (e.g., remote Ollama at a different host) — adds complexity, out of scope for v5.1
- Listing available local models via `ollama list` API during fallback — out of scope per REQUIREMENTS.md

</deferred>
