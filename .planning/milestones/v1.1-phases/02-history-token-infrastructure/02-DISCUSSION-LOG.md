# Phase 2: History & Token Infrastructure - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-04-07
**Phase:** 02-history-token-infrastructure
**Areas discussed:** Ollama API strategy, Token threshold config, History type design

---

## Ollama API Strategy

| Option | Description | Selected |
|--------|-------------|----------|
| Stay with /api/generate — stitch history as string | Build full conversation into one prompt string. Simpler, no API migration. | |
| Switch to /api/chat — native messages array | Ollama's chat endpoint, role/content pairs like OpenAI. | ✓ |

**User's choice:** Switch to /api/chat — native messages array

| Option | Description | Selected |
|--------|-------------|----------|
| Replace StreamPrompt with a new chat-based function | Single code path, Phase 2 migrates client. | ✓ |
| Add new chat function, keep StreamPrompt | Additive, lower risk per phase. | |

**User's choice:** Replace StreamPrompt with a new chat-based function

**Notes:** User provided detailed architectural design for the messages array and summarization strategy:
- System prompt + context.md pinned at index [0], never removed
- 4,100 token threshold leaves ~4K for current task + model response in 8K window
- Compression: take messages after system prompt, before most recent exchange → summarize → replace with `{"role": "system", "content": "Summary of previous conversation: [text]"}`
- Most recent user+assistant exchange preserved through compression

| Option | Description | Selected |
|--------|-------------|----------|
| system role for summary message | `{"role": "system", ...}` — treated as background context | ✓ |
| assistant role | `{"role": "assistant", ...}` | |

**User's choice:** system role

---

## Token Threshold Config

| Option | Description | Selected |
|--------|-------------|----------|
| Add TokenThreshold to Config struct | Extend internal/config/config.go, env var + file config. | ✓ |
| Persistent root flag only | cobra PersistentFlags, no Config struct change. | |

**User's choice:** Add TokenThreshold to Config struct

**Notes:** User also specified that the per-project config file path should change from `.myhelper.json` to `.myhelper/config.json`. The `.myhelper/` directory is the canonical location for all myhelper config and context files going forward.

| Option | Description | Selected |
|--------|-------------|----------|
| ~/.config/myhelper/config.json (keep as-is) | XDG-compliant, unchanged from v1.0 | ✓ |
| ~/.myhelper/config.json | Mirrors per-project .myhelper/ pattern | |

**User's choice:** ~/.config/myhelper/config.json (keep as-is)

| Option | Description | Selected |
|--------|-------------|----------|
| --token-limit / MYHELPER_TOKEN_LIMIT | Reads as a limit | ✓ |
| --token-threshold / MYHELPER_TOKEN_THRESHOLD | Matches internal terminology | |

**User's choice:** --token-limit / MYHELPER_TOKEN_LIMIT

---

## History Type Design

| Option | Description | Selected |
|--------|-------------|----------|
| New internal/history package | Owns Message type, History struct, token counting, threshold check | ✓ |
| Inside internal/ollama | Simpler, couples history to transport layer | |

**User's choice:** New internal/history package

| Option | Description | Selected |
|--------|-------------|----------|
| History struct with Add, TokenCount, ExceedsLimit methods | Clean API for Phase 3 | ✓ |
| Just the Message slice and free function for counting | Simpler, Phase 3 owns more state management | |

**User's choice:** History struct with Add, TokenCount, ExceedsLimit methods
