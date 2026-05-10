# Phase 4: Summarization & Re-condensation - Context

**Gathered:** 2026-04-07
**Status:** Ready for planning

<domain>
## Phase Boundary

When `hist.ExceedsLimit()` fires inside `runConversationLoop`, the model compresses its own conversation history. The loop continues with a compact summary in place of the old turns. Phase 4 modifies only `runConversationLoop` in `cmd/helpers.go` and adds supporting infrastructure (non-streaming Ollama call, summarization prompt lookup per command).

No new user-facing commands. No config changes. No changes to the History struct API beyond what is needed to replace messages after summarization.

</domain>

<decisions>
## Implementation Decisions

### Summarization Trigger & Flow
- **D-01:** `hist.ExceedsLimit()` is the trigger — checked before each model call in `runConversationLoop` (same place Phase 3 set up; Phase 4 inserts the summarization call there).
- **D-02:** Messages to summarize: everything AFTER the system prompt (index [0]) and BEFORE the most recent user+assistant exchange. The system prompt is never touched.
- **D-03:** After summarization, the old messages are replaced with a single message: `{"role": "system", "content": "Summary of previous conversation: [text]"}`.

### Summarization Prompts (Command-Specific)
- **D-04:** First-time summarization uses a command-specific prompt. Base template:
  `"Summarize the key decisions and code structures discussed above into a concise technical summary."`
  Variants per command reflect what each tool prioritizes:
  - `plan`: emphasize subtasks, ordered decisions, blockers
  - `lookup`: emphasize API/library choices and rationale
  - `starter`: emphasize code structure and patterns chosen
  - `pattern`: emphasize the idiomatic pattern identified and its constraints

- **D-05:** Re-condensation uses a different command-specific prompt. Base template:
  `"Given the following summary of past events and these new interactions, create an updated, comprehensive summary that preserves all technical decisions and project state."`
  Variants per command follow the same per-tool logic as D-04.

- **D-06:** Re-condensation is detected by checking whether a prior summary message already exists in the slice being summarized (i.e., a `role: "system"` message with content prefixed `"Summary of previous conversation:"`). If detected → use re-condensation prompt. Same code path, no special loop logic.

### Non-Streaming Ollama Call
- **D-07:** Summarization must NOT stream to stdout — it would be jarring mid-conversation. Add a non-streaming `ollama.Chat(cfg, messages) (string, error)` alongside the existing `StreamChat`. The summarization flow calls this internally.

### User Feedback
- **D-08:** Print `[Condensing history...]` to `os.Stderr` immediately before the summarization Ollama call. This is the only output during summarization — the summary text itself is not shown to the user.

### Error Handling
- **D-09:** If the summarization `ollama.Chat` call fails, return the error and end the session. Consistent with how `StreamChat` errors are handled elsewhere in the loop — no silent failures.

### History Mutation API
- **D-10:** Claude's discretion on how to replace messages in the History struct after summarization (e.g., add a `Replace(messages []Message)` method, or `Reset()` + rebuild). Must preserve the system message at index [0].

### Claude's Discretion
- Exact per-command summarization prompt text (within the framework of D-04 and D-05)
- Implementation of the `Replace` / `Reset` method on `History`
- Whether the summarization call assembles its own message slice or delegates to a helper
- Internal structure of the `ollama.Chat` function (reuse HTTP/JSON logic from `StreamChat`, just write to a `strings.Builder` instead of stdout)

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Code to extend
- `cmd/helpers.go` — `runConversationLoop` is the sole modification point for Phase 4; `initiateConversation` shows the existing StreamChat call pattern
- `internal/history/history.go` — `ExceedsLimit()`, `Messages()`, `Add()`; Phase 4 will likely add a `Replace()` or `Reset()` method here
- `internal/ollama/client.go` — `StreamChat` is the reference implementation; Phase 4 adds a non-streaming `Chat` alongside it
- `cmd/plan.go`, `cmd/lookup.go`, `cmd/starter.go`, `cmd/pattern.go` — each command's `RunE` passes a `streamFn` to `runConversationLoop`; the summarization prompt must be keyed to the command

### Project context
- `.planning/REQUIREMENTS.md` — SUMM-01, SUMM-02, SUMM-03 are the requirements for this phase
- `.planning/ROADMAP.md` — Phase 4 success criteria define what "done" means
- `.planning/phases/02-history-token-infrastructure/02-CONTEXT.md` — D-03 defines the summary message structure; D-12 defines History API
- `.planning/phases/03-conversation-loop/03-CONTEXT.md` — D-05 defines where summarization goes in the loop

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `internal/ollama/client.go:StreamChat` — HTTP/JSON structure, NDJSON scanner, error handling pattern. `ollama.Chat` (non-streaming) reuses all of this, just writes to `strings.Builder` instead of `os.Stdout` and skips line-by-line streaming.
- `internal/history/history.go:ExceedsLimit()` — already implemented and tested; Phase 4 just calls it at the top of each loop iteration.
- `cmd/helpers.go:runConversationLoop` — the loop body is the sole insertion point; the `streamFn func(config.Config, []history.Message) (string, error)` parameter pattern already established.

### Established Patterns
- Stderr for all non-model output (`[Condensing history...]` follows this)
- Errors returned from `streamFn` cause the loop to exit — same behavior for summarization errors
- Command-specific system prompts are already defined per command file (`plan.go`, etc.) — per-command summarization prompts follow the same locality

### Integration Points
- `runConversationLoop` receives `hist *history.History` — after summarization, the history's internal slice is mutated in-place (via a new `Replace` method); the loop continues with the updated slice without any parameter changes
- The `streamFn` passed to `runConversationLoop` is `ollama.StreamChat` — the new `ollama.Chat` for summarization is called directly inside the loop, not through `streamFn`

</code_context>

<specifics>
## Specific Ideas

**User's mental model of the 8K budget:**
- 4,100 tokens for history → triggers compression
- Leaves ~4K for current task + model response within the 8K window
- System prompt + context.md at index [0] never removed — always preserved

**Re-condensation example (user's description):**
After first summarization, message slice: `[system][summary][user][asst][user][asst]...`
When threshold fires again: take `[summary][user][asst]...(all but most recent exchange)` → send to model with re-condensation prompt → replace with new `[system: "Summary of previous conversation: ..."]` message.

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---

*Phase: 04-summarization-re-condensation*
*Context gathered: 2026-04-07*
