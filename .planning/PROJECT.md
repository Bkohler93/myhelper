# myhelper

## What This Is

A Go CLI tool for Go developers that offloads common coding micro-tasks to a locally-hosted LLM (Ollama + qwen2.5-coder:7b). It answers four focused questions — how to break down a feature, which library/API to use, what minimal working code looks like, and what the idiomatic pattern is — using project-specific context to keep responses relevant and tight.

## Core Value

Get a useful, context-aware answer from the local model in one command, without context-bloat or round-trips to an external API.

## Requirements

### Validated

(None yet — ship to validate)

### Active

- [ ] `init` command — writes a blank `context.md` template to the current directory; user fills it in manually
- [ ] `plan` command — breaks a feature/task description into ordered subtasks
- [ ] `lookup` command — recommends the right API or library for a given step
- [ ] `starter` command — prints minimal working Go code for a given task (user edits after)
- [ ] `pattern` command — describes the idiomatic Go way to write or structure something
- [ ] All commands accept the input as a CLI argument or prompt for it interactively if omitted
- [ ] All commands stream model output to stdout as tokens arrive
- [ ] `context.md` in the current directory is read and prepended to every prompt to scope the model
- [ ] Ollama endpoint and model are configurable (default: `192.168.0.9:11434`, `qwen2.5-coder:7b`)

### Out of Scope

- Auto-detection of project metadata on init — pure template, user fills it — keeps init fast and predictable
- Global/fallback context.md — per-directory only, avoids cross-project bleed
- Conversation history / multi-turn sessions — single-shot prompts only, respects the 8k context limit
- Web search or external API calls — local inference only
- Non-Go output — tool is optimized for Go projects; other languages are incidental

## Context

- **Inference server**: Ollama at `192.168.0.9:11434`, model `qwen2.5-coder:7b`
- **Model constraint**: ~8k context window — prompts must be short and factual; system prompt + context.md + user query must fit comfortably
- **User workflow**: Developer runs the tool from within a Go project directory; `context.md` at the project root scopes the model's responses to that project's stack and patterns
- **Primary use case**: Solo developer productivity tool; no multi-user, auth, or network exposure concerns
- **Existing codebase**: Repo already has some code — skipping codebase map, initializing fresh project scope

## Constraints

- **Language**: Go — the tool itself is written in Go
- **Model context**: 8k limit — system prompts and context.md must be kept minimal and factual
- **Inference**: Local Ollama only — no external API dependencies
- **Output**: Streaming to stdout — no buffering full responses before display
- **Config**: Endpoint/model overridable via config file or env var — hardcoded defaults are fine for v1

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| context.md is a pure template (no auto-detect) | Keeps init simple and fast; user knows their stack better than heuristics do | — Pending |
| Per-directory context.md only | Avoids cross-project bleed; forces explicit initialization per project | — Pending |
| Stream output as tokens arrive | Model is slow enough that streaming feels faster; matches developer expectation from tools like Ollama CLI | — Pending |
| Single-shot prompts (no history) | Preserves context window budget; each command is self-contained | — Pending |

## Evolution

This document evolves at phase transitions and milestone boundaries.

**After each phase transition** (via `/gsd:transition`):
1. Requirements invalidated? → Move to Out of Scope with reason
2. Requirements validated? → Move to Validated with phase reference
3. New requirements emerged? → Add to Active
4. Decisions to log? → Add to Key Decisions
5. "What This Is" still accurate? → Update if drifted

**After each milestone** (via `/gsd:complete-milestone`):
1. Full review of all sections
2. Core Value check — still the right priority?
3. Audit Out of Scope — reasons still valid?
4. Update Context with current state

---
*Last updated: 2026-04-07 after initialization*
