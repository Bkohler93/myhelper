# Phase 30: Setup Wizard - Context

**Gathered:** 2026-05-10
**Status:** Ready for planning
**Mode:** Auto-generated (discuss skipped via workflow.skip_discuss)

<domain>
## Phase Boundary

Implement `myhelper setup` — an interactive first-run wizard that takes a new user from zero to a working chat session. The wizard: (1) checks whether Ollama is installed and reachable, (2) shows platform-specific install instructions if not, (3) detects GPU VRAM or RAM and recommends a model, (4) optionally pulls the recommended model, (5) prompts for a Tavily API key and writes it to `~/.config/myhelper/config.json`, (6) optionally prompts for a SearXNG endpoint and writes it to config.

Depends on Phase 29 (Tavily provider must exist — `tavily_key` and `search_provider` JSON fields are defined there and the wizard writes them).

</domain>

<decisions>
## Implementation Decisions

### Claude's Discretion
All implementation choices are at Claude's discretion — discuss phase was skipped per user setting. Use ROADMAP phase goal, success criteria, and codebase conventions to guide decisions.

**Success criteria to satisfy:**
1. Running `myhelper setup` on a machine without Ollama shows platform-specific install instructions (brew on macOS, curl on Linux/WSL)
2. Running `myhelper setup` on a machine with Ollama shows a model recommendation based on detected GPU VRAM or RAM
3. User can confirm in-wizard to pull the recommended model without leaving the terminal
4. User is prompted for a Tavily API key and the key is written to `~/.config/myhelper/config.json`
5. User can optionally enter a SearXNG endpoint and it is written to config

**UI hint:** yes — this is an interactive terminal wizard. Terminal interaction pattern should be simple stdin prompts (bufio.Scanner or fmt.Scan), not a TUI library, consistent with the project's minimal-dependency philosophy.

**Config write format:** Must write `tavily_key` and `search_provider` JSON fields matching the exact tags defined in Phase 29 `search.Config` struct. Must also write `search_endpoint` for SearXNG (matching `search.Config.Endpoint` tag).

**Requirements:** SETUP-01, SETUP-02, SETUP-03, SETUP-04, SETUP-05, SETUP-06

</decisions>

<code_context>
## Existing Code Insights

Codebase context will be gathered during plan-phase research.

</code_context>

<specifics>
## Specific Ideas

No specific requirements — open to standard approaches. The wizard should be non-destructive: if config already exists, it should merge (update fields) rather than overwrite the whole file.

</specifics>

<deferred>
## Deferred Ideas

None — discuss phase skipped.

</deferred>
