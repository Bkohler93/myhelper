# Phase 4: Summarization & Re-condensation - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-04-07
**Phase:** 04-summarization-re-condensation
**Areas discussed:** Summarization prompt, User feedback, Error handling, Re-condensation flow

---

## Summarization Prompt

| Option | Description | Selected |
|--------|-------------|----------|
| Goal-aware | Include user's original question, focus on goal and key decisions | |
| Minimal imperative | "Summarize the conversation above concisely" | |
| Command-specific (user-described) | Different prompt per command, base: "Summarize the key decisions and code structures discussed above into a concise technical summary." | ✓ |

**User's choice:** Command-specific prompts tailored per tool (plan/lookup/starter/pattern).

**Notes:** User described the full compression strategy: system prompt at index [0] is never touched; messages after system prompt but before most recent exchange are extracted and sent to Ollama with a command-specific summarization prompt; result replaces those messages as a single system message `{"role": "system", "content": "Summary of previous conversation: [text]"}`.

---

## User Feedback During Summarization

| Option | Description | Selected |
|--------|-------------|----------|
| Silent | No output during summarization | |
| Stderr indicator | Print `[Condensing history...]` to stderr before the call | ✓ |
| You decide | Claude picks | |

**User's choice:** Stderr indicator — `[Condensing history...]`

**Notes:** None beyond the selection.

---

## Summarization Error Handling

| Option | Description | Selected |
|--------|-------------|----------|
| Return error, end session | Treat like any StreamChat failure — exit loop | ✓ |
| Continue with full history | Log to stderr, carry on despite possible overflow | |
| You decide | Claude picks consistent option | |

**User's choice:** Return error, end session.

**Notes:** None beyond the selection.

---

## Re-condensation Flow

| Option | Description | Selected |
|--------|-------------|----------|
| Same code path | ExceedsLimit() fires again, prior summary is just another message to compress | |
| Different prompt for re-condensation | Use distinct command-specific prompt that acknowledges prior summary | ✓ |

**User's choice:** Different prompt for re-condensation, but same code path.

**Notes:** Re-condensation prompt base: `"Given the following summary of past events and these new interactions, create an updated, comprehensive summary that preserves all technical decisions and project state."` — also varies per command. Detection: check if a prior summary message exists in the slice to choose which prompt to use.

---

## Claude's Discretion

- Exact per-command prompt text (within the framework decided above)
- `History.Replace()` / `Reset()` method implementation
- Internal structure of non-streaming `ollama.Chat`
- Whether summarization assembles its own message slice or delegates to a helper

## Deferred Ideas

None.
