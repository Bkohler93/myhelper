# Phase 3: Conversation Loop - Context

**Gathered:** 2026-04-07
**Status:** Ready for planning

<domain>
## Phase Boundary

Wire a multi-turn conversation loop into all 4 query commands (plan, lookup, starter, pattern). After the first model response, the user is prompted for follow-up input. The loop continues until the user types "quit" or presses Ctrl+C. The `init` command stays one-shot.

No summarization logic in this phase — that is Phase 4. Phase 3 only adds the interactive loop and clean exit handling.

</domain>

<decisions>
## Implementation Decisions

### Follow-up Prompt
- **D-01:** Prompt text is `> ` (minimal shell-style). Written to `os.Stderr`. Printed after each model response to signal the tool is ready for the next input.

### Empty Input Handling
- **D-02:** When the user presses Enter with no input, the loop reprints `> ` and waits again — no error, no model call, no exit. Silent reprompt.

### Ctrl+C Exit Behavior
- **D-03:** Install an `os/signal` handler for `SIGINT`. On Ctrl+C: exit cleanly with code 0, no output. Satisfies CONV-03 strictly (no error output, clean exit).
- **D-04:** "quit" typed by user → detect before sending to model → exit 0 with no output.

### Loop Architecture
- **D-05:** The conversation loop lives in a shared helper function in `cmd/helpers.go` — something like `runConversationLoop(cfg config.Config, messages []history.Message)`. All 4 query commands call this helper after their initial setup. Phase 4 only needs to modify this one function to add summarization.

### Claude's Discretion
- Exact function signature of `runConversationLoop` (return type, parameter names)
- Whether to use `history.History` struct internally or a plain `[]history.Message` slice (History struct preferred since Phase 4 will need `ExceedsLimit()`)
- Whether the signal handler is set up inside `runConversationLoop` or at cobra root level

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Code to extend
- `cmd/helpers.go` — Add `runConversationLoop` here; existing `readInteractive`, `resolveInput`, `buildSystemMessage` are established patterns to follow
- `cmd/plan.go` — Representative command file; all 4 query commands follow this pattern and will each be updated to call `runConversationLoop`
- `cmd/lookup.go` — Must receive the conversation loop
- `cmd/starter.go` — Must receive the conversation loop
- `cmd/pattern.go` — Must receive the conversation loop
- `internal/history/history.go` — `History.New()`, `Add()`, `Messages()` — the loop will use these
- `internal/ollama/client.go` — `StreamChat(cfg, messages) (string, error)` — loop calls this each turn

### Project context
- `.planning/REQUIREMENTS.md` — CONV-01, CONV-02, CONV-03, CONV-04 are the requirements for this phase
- `.planning/ROADMAP.md` — Phase 3 success criteria define what "done" means

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `cmd/helpers.go:readInteractive(prompt)` — reads a line from stdin with a stderr prompt; the loop's `> ` prompt reuses this pattern
- `cmd/helpers.go:resolveInput()` — handles first-turn input (arg or interactive); loop starts AFTER this returns
- `internal/history.New(threshold)` — constructs History with threshold from `cfg.TokenThreshold`; loop should pass messages via `hist.Messages()` into `StreamChat`

### Established Patterns
- Interactive prompts → `os.Stderr`; model output → `os.Stdout`
- `StreamChat` returns `(string, error)` — designed for the loop: capture response, append to history as assistant message
- Each command's `RunE` loads config and context, then calls the Ollama client — Phase 3 inserts the loop call at the end of `RunE`, after the first `StreamChat` call

### Integration Points
- Phase 4 will modify `runConversationLoop` to call a summarization function before each model call when `hist.ExceedsLimit()` returns true — keeping the loop in one place makes that straightforward

</code_context>

<specifics>
## Specific Ideas

No specific implementation references — decisions captured in `<decisions>` above are sufficient to plan.

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---

*Phase: 03-conversation-loop*
*Context gathered: 2026-04-07*
