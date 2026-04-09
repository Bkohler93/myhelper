# Milestones

## v1.2 Smart Context (Shipped: 2026-04-09)

**Phases completed:** 4 phases, 13 plans, 13 tasks

**Key accomplishments:**

- FileEntry/ChatFn type contracts, exclusion-aware Walk(), and go/ast-based ExtractSymbols() — scanner core primitives, 16 TDD-verified tests (Phase 5)
- Token-budgeted index assembler serializes `{"meta": {...}, "files": [...]}` to `.myhelper/index.json` via ReadMeta + BuildIndex, dropping test files first when 80% budget is exceeded (Phase 5)
- Per-package LLM summary generation and `scanner.Scan()` entry point completing the two-step scanner pipeline (Phase 5)
- `init` and `sync` commands rewritten with Bubble Tea `RunWithSpinner`, `generateContextMD` LLM-based context.md generation, and mtime-based delta rescan (Phase 6)
- `buildInjectedMessages` two-pass context injection helper with full token budget logic wired into all 4 query commands (plan, lookup, starter, pattern) (Phase 7)
- `microPassFile` AST symbol-map + LLM line-range micro-pass for large files, replacing symbol-block fallback in `buildInjectedMessages` (Phase 8)

---

## v1.1 Conversational Mode (Shipped: 2026-04-08)

**Phases completed:** 3 phases, 7 plans, 10 tasks

**Key accomplishments:**

- Config.TokenThreshold field (default 4100) with MYHELPER_TOKEN_LIMIT env var, .myhelper/config.json local path, and --token-limit persistent cobra flag
- One-liner:
- Replaced /api/generate client (StreamPrompt) with /api/chat client (StreamChat); all 4 query commands updated to use two-element messages slice (system + user) with history.Message type
- One-liner:
- Multi-turn conversation wired into all 4 query commands via initiateConversation + runConversationLoop pattern
- history.Replace and ollama.Chat(non-streaming) added — the two primitives that runConversationLoop needs to summarize and replace conversation history
- Command-specific summarization prompts and summarize() helper integrated into runConversationLoop — all 4 query commands now automatically condense history when token threshold is exceeded

---

## v1.0 Initial Release (Shipped: 2026-04-07)

**Phases completed:** 1 phases, 4 plans, 7 tasks

**Key accomplishments:**

- Go module scaffolded with cobra CLI, config loading from env/file/defaults, and streaming Ollama client posting to /api/generate via NDJSON line-scan
- context.md loader (LoadContext) and init subcommand that writes a Go-focused commented template without overwriting existing files
- Four cobra subcommands (plan, lookup, starter, pattern) wired to rootCmd with shared helpers for positional-arg-or-interactive-prompt input and prompt assembly
- One-liner:

---
