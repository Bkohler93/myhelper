# Phase 2: History & Token Infrastructure - Context

**Gathered:** 2026-04-07
**Status:** Ready for planning

<domain>
## Phase Boundary

Deliver the infrastructure that underpins multi-turn conversation:
- In-memory conversation history (Message struct + History type)
- Token counting via go-tiktoken after each turn
- Configurable token threshold (default 4,100)

No user-visible loop behavior in this phase — that comes in Phase 3. Phase 2 is the foundation Phase 3 and Phase 4 will build on.

</domain>

<decisions>
## Implementation Decisions

### Ollama API Strategy
- **D-01:** Replace the current `/api/generate` client with a `/api/chat` client. `StreamPrompt` is removed and replaced with a new chat-based streaming function that accepts a `[]Message` and streams the response while returning the full accumulated response text.
- **D-02:** Messages array is structured as follows:
  - Index [0]: `{"role": "system", "content": "<system instructions> + context.md"}` — permanent, never removed or summarized
  - Subsequent elements: `{"role": "user", "content": "..."}` and `{"role": "assistant", "content": "..."}` pairs
- **D-03:** Summarization (Phase 4) compresses messages AFTER the system prompt and BEFORE the most recent exchange. The most recent user+assistant turn is preserved. Old messages are replaced with a single `{"role": "system", "content": "Summary of previous conversation: [text]"}` message.
- **D-04:** This is a full client replacement — existing command files (`plan.go`, `lookup.go`, etc.) are updated in Phase 2 to use the new function signature.

### Token Threshold Configuration
- **D-05:** Add `TokenThreshold int` to the `Config` struct in `internal/config/config.go`. Default value: `4100`.
- **D-06:** Per-project config file path changes from `.myhelper.json` to `.myhelper/config.json` (the `.myhelper/` directory is the canonical location for all myhelper config and context files).
- **D-07:** Global config path stays at `~/.config/myhelper/config.json` (unchanged from v1.0).
- **D-08:** Config precedence: env > `.myhelper/config.json` (CWD) > `~/.config/myhelper/config.json` > defaults.
- **D-09:** Environment variable: `MYHELPER_TOKEN_LIMIT`. CLI persistent flag: `--token-limit`. Both accept an integer.

### History Package Design
- **D-10:** Create `internal/history` package. This package owns the `Message` type, `History` struct, token counting, and threshold check.
- **D-11:** `Message` struct: `{ Role string; Content string }` — maps directly to Ollama's `/api/chat` message format.
- **D-12:** `History` struct exposes:
  - `Add(role, content string)` — appends a `Message`
  - `TokenCount() int` — returns current total token count of all messages (via go-tiktoken)
  - `ExceedsLimit() bool` — returns `true` when `TokenCount() > threshold`
  - History is constructed with the threshold value so `ExceedsLimit()` can answer without external state.
- **D-13:** Token counting library: `github.com/pkoukk/tiktoken-go` (noted in STATE.md as the candidate).

### Claude's Discretion
- Exact encoder/model string passed to tiktoken (e.g., `"cl100k_base"`) — Claude picks reasonable default for a general-purpose model
- Internal field layout of the `History` struct (slice + threshold field)
- Whether the new Ollama chat function returns `(string, error)` or uses a callback pattern — Claude picks the option most consistent with existing `StreamPrompt` style

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Existing code to extend / replace
- `internal/config/config.go` — Config struct and Load() function; TokenThreshold and new config file path extend this
- `internal/ollama/client.go` — StreamPrompt is replaced; new /api/chat client goes here
- `cmd/helpers.go` — buildPrompt() will be superseded by messages-based approach; review before modifying
- `cmd/plan.go` — representative command file; all 4 query commands follow this pattern

### Dependency
- `go.mod` — add `github.com/pkoukk/tiktoken-go` dependency here

### Project context
- `.planning/REQUIREMENTS.md` — HIST-01, HIST-02, HIST-03, CONF-01 are the requirements for this phase
- `.planning/ROADMAP.md` — Phase 2 success criteria define what "done" means

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `cmd/helpers.go:buildPrompt()` — currently stitches projectContext + systemPrompt + userInput into a string. In the new /api/chat world, this becomes the system message at index [0] of the messages array. The logic can inform how the system message is constructed.
- `internal/config/config.go:Load()` — precedence logic (env > local file > global file > defaults) is the pattern to follow for `TokenThreshold`. Add the new field, update `localConfigPath()` to return `.myhelper/config.json`.

### Established Patterns
- Config values live in `internal/config` and are passed as a `Config` struct to downstream functions
- Functions return `error` only where no data needs to come back; the new chat function will need to return `(string, error)` to capture the model response for history
- Interactive prompts go to `os.Stderr`; streamed model output goes to `os.Stdout`

### Integration Points
- Every command's `RunE` function calls `config.Load()` then calls the Ollama client — Phase 2 updates the client call in all 4 query commands
- Phase 3 will import `internal/history` and drive the `History` struct; Phase 2 creates the package and writes tests for it

</code_context>

<specifics>
## Specific Ideas

**Summarization message structure (user's design, verbatim):**
```json
{"role": "system", "content": "Summary of previous conversation: [summary text here]"}
```
This replaces the older messages (after system prompt, before most recent exchange) when the token threshold is hit.

**8K budget reasoning (user's mental model):**
- 4,100 tokens for history → triggers compression
- Leaves ~4K for current task + model response within the 8K window
- System prompt + context.md at index [0] never removed — always preserved

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---

*Phase: 02-history-token-infrastructure*
*Context gathered: 2026-04-07*
