# Requirements: myhelper

**Defined:** 2026-05-09
**Core Value:** Fast, local AI chat with optional web search — inference runs locally via Ollama, search is pluggable (Tavily or self-hosted SearXNG), no cloud AI required.

## v5.0 Requirements

### Distribution

- [ ] **DIST-01**: User can install myhelper on WSL/Linux by running a one-line curl command that auto-detects OS/arch, downloads the correct binary from GitHub Releases, and places it in PATH
- [ ] **DIST-02**: Tagged git pushes automatically trigger GitHub Actions to build cross-platform binaries (darwin/amd64, darwin/arm64, linux/amd64, linux/arm64) and publish them to GitHub Releases via goreleaser

### Setup Wizard

- [ ] **SETUP-01**: User can run `myhelper setup` to check whether Ollama is installed and reachable on port 11434
- [ ] **SETUP-02**: User sees platform-specific Ollama install instructions when Ollama is not detected (macOS: `brew install ollama`, Linux/WSL: curl install script)
- [ ] **SETUP-03**: User sees a recommended model size based on detected GPU VRAM (nvidia-smi on Linux/WSL, system_profiler on macOS) or RAM when no discrete GPU is found
- [ ] **SETUP-04**: User can confirm to have the wizard run `ollama pull <recommended-model>` directly without leaving the terminal
- [ ] **SETUP-05**: User is prompted to enter their Tavily API key during setup; key is written to `~/.config/myhelper/config.json`
- [ ] **SETUP-06**: User can optionally enter a SearXNG endpoint during setup; endpoint is written to config

### Search Providers

- [ ] **SRCH-01**: User can use Tavily as the search provider by configuring a Tavily API key in config or via env var; Tavily is the default provider when a key is present
- [ ] **SRCH-02**: User can switch search provider between Tavily and SearXNG via `search_provider` field in config.json
- [ ] **SRCH-03**: User can provide their Tavily API key via `MYHELPER_TAVILY_KEY` environment variable, which takes precedence over config

## Future Requirements

### Distribution

- **DIST-F01**: Homebrew tap formula for `brew install brettkohler/tap/myhelper` — deferred until distribution pattern established
- **DIST-F02**: Windows native binary (non-WSL) — WSL/Linux is the target for Windows machines in v5.0

### Inference

- **INFER-F01**: OpenAI-compatible `/v1/chat/completions` endpoint support — Ollama-only for now

## Out of Scope

| Feature | Reason |
|---------|--------|
| Homebrew tap | Deferred — install.sh covers the primary WSL use case; tap adds setup overhead |
| OpenAI-compatible endpoint | Explicitly deferred by user — Ollama-only for v5.0 |
| Windows native binary (non-WSL) | WSL is the Windows target; native Windows paths add cross-compilation complexity |
| LM Studio / LocalAI support | Follows from OpenAI-compat endpoint work, deferred with it |
| Conversation history persistence | Single-session only — explicitly out of scope from prior milestones |

## Traceability

| Requirement | Phase | Status |
|-------------|-------|--------|
| DIST-01 | Phase 28 | Pending |
| DIST-02 | Phase 28 | Pending |
| SRCH-01 | Phase 29 | Pending |
| SRCH-02 | Phase 29 | Pending |
| SRCH-03 | Phase 29 | Pending |
| SETUP-01 | Phase 30 | Pending |
| SETUP-02 | Phase 30 | Pending |
| SETUP-03 | Phase 30 | Pending |
| SETUP-04 | Phase 30 | Pending |
| SETUP-05 | Phase 30 | Pending |
| SETUP-06 | Phase 30 | Pending |

**Coverage:**
- v5.0 requirements: 11 total
- Mapped to phases: 11
- Unmapped: 0 ✓

---
*Requirements defined: 2026-05-09*
*Last updated: 2026-05-09 — traceability updated after roadmap creation (SRCH→Phase 29, SETUP→Phase 30)*
