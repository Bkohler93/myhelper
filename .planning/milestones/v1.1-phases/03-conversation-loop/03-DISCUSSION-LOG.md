# Phase 3: Conversation Loop - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.

**Date:** 2026-04-07
**Phase:** 03-conversation-loop

---

## Area: Follow-up prompt text

**Q:** What should the follow-up prompt say between turns?

| Option | Selected |
|--------|----------|
| `> ` (minimal shell-style) | ✓ |
| `Follow-up: ` | |
| `Ask: ` | |
| You decide | |

**Decision:** `> ` written to stderr between each turn.

---

## Area: Empty input handling

**Q:** What should happen when the user presses Enter with no input?

| Option | Selected |
|--------|----------|
| Reprompt silently | ✓ |
| Treat as quit | |
| Pass to model | |

**Decision:** Silently reprint `> ` and wait again.

---

## Area: Ctrl+C exit behavior

**Q:** How should Ctrl+C exit?

| Option | Selected |
|--------|----------|
| Silent exit, code 0 (signal handler) | ✓ |
| Default Go behavior (code 130) | |

**Decision:** Install `os/signal` SIGINT handler → exit 0, no output.

---

## Area: Loop architecture

**Q:** Where should the conversation loop live in the code?

| Option | Selected |
|--------|----------|
| Shared helper in helpers.go | ✓ |
| Inline per command | |

**Decision:** `runConversationLoop` in `cmd/helpers.go`. All 4 commands call it. Phase 4 modifies one place.

---

*Log generated: 2026-04-07*
