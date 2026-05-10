# Requirements: myhelper v5.1

**Defined:** 2026-05-10
**Core Value:** Fast, local AI chat with optional web search — inference runs locally via Ollama, search is pluggable (Tavily or self-hosted SearXNG), no cloud AI required.

## v5.1 Requirements

### Configuration

- [ ] **CFG-01**: Config loading returns empty string for model when not set in config or env (no hardcoded default)
- [ ] **CFG-02**: Config loading returns empty string for endpoint when not set in config or env (no hardcoded default)

### Validation

- [ ] **VAL-01**: User sees a clear error with "run myhelper setup" hint when running `chat` with no model configured
- [ ] **VAL-02**: User sees a clear error with "run myhelper setup" hint when running `chat` with no endpoint configured
- [ ] **VAL-03**: `inspect` validates model and endpoint before executing — same error format as chat
- [ ] **VAL-04**: `search` validates model and endpoint before executing — same error format as chat
- [ ] **VAL-05**: Setting `MYHELPER_MODEL` and `MYHELPER_ENDPOINT` env vars satisfies validation (no error)

### Setup Wizard

- [ ] **WIZ-01**: `myhelper setup` always writes a model to config before exiting
- [ ] **WIZ-02**: When user skips the recommended model pull, wizard prompts for an existing local model name before saving
- [ ] **WIZ-03**: Wizard validates that endpoint is non-empty before writing config

## Future Requirements

- OpenAI-compatible endpoint support (any `/v1/chat/completions` server)
- Homebrew tap formula (`brew install brettkohler/tap/myhelper`)

## Out of Scope

| Feature | Reason |
|---------|--------|
| `myhelper config set` subcommand | Setup is the only user-facing config path for v5.1 |
| Auto-redirect to setup on missing config | Hard fail + clear message is simpler and more predictable |
| Default token_threshold removal | Internal tuning param — not user-visible, retain 4100 default |
| Listing available Ollama models in wizard | Adds complexity; user can run `ollama list` themselves |

## Traceability

| Requirement | Phase | Status |
|-------------|-------|--------|
| CFG-01 | Phase 31 | Pending |
| CFG-02 | Phase 31 | Pending |
| VAL-01 | Phase 31 | Pending |
| VAL-02 | Phase 31 | Pending |
| VAL-03 | Phase 31 | Pending |
| VAL-04 | Phase 31 | Pending |
| VAL-05 | Phase 31 | Pending |
| WIZ-01 | Phase 32 | Pending |
| WIZ-02 | Phase 32 | Pending |
| WIZ-03 | Phase 32 | Pending |

**Coverage:**
- v5.1 requirements: 10 total
- Mapped to phases: 10
- Unmapped: 0 ✓

---
*Requirements defined: 2026-05-10*
*Last updated: 2026-05-10 after initial definition*
