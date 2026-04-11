---
phase: 19-search-gate-and-injection
plan: "02"
subsystem: cmd
tags: [search-gate, injection, flags, wiring, tdd]
dependency_graph:
  requires:
    - cmd/search.go:searchGate (Plan 01)
    - cmd/search.go:reRankResults (Plan 01)
    - cmd/search.go:buildWebBlock (Plan 01)
    - internal/search.Search / LoadConfig
    - cmd/helpers.go:runConversationLoop
  provides:
    - cmd/search.go:buildUserMessage
    - cmd/root.go:--search flag (searchForce)
    - cmd/root.go:--no-search flag (searchSuppress)
    - cmd/helpers.go:runConversationLoop preprocessor parameter
  affects:
    - cmd/root.go (one-shot and REPL paths now augment user messages)
    - cmd/helpers_test.go (all call sites updated)
    - cmd/chat_test.go (call site updated)
tech_stack:
  added: []
  patterns:
    - preprocessor func(string) string nil-safe hook on runConversationLoop
    - search.LoadConfig() called once per RunE invocation (not per REPL turn)
    - buildUserMessage returns bare query on any search skip/error path
key_files:
  created: []
  modified:
    - cmd/search.go
    - cmd/helpers.go
    - cmd/root.go
    - cmd/helpers_test.go
    - cmd/chat_test.go
    - cmd/search_gate_test.go
decisions:
  - "preprocessor func(string) string as 6th param to runConversationLoop — cleanest extension point, nil-safe, no search-specific params in loop signature"
  - "search.LoadConfig() called once in RunE, captured in preprocessor closure — avoids per-REPL-turn config re-read (Pitfall 6)"
  - "buildUserMessage block + query ordering — context before question per INJ-01 and retrieval.assembleMessages pattern"
  - "noSearch takes priority over forceSearch — defensive ordering matches GATE-04 spec"
metrics:
  duration: "~15 minutes"
  completed: "2026-04-11"
  tasks_completed: 2
  files_created: 0
  files_modified: 6
---

# Phase 19 Plan 02: Wire Search into Conversation Path Summary

**One-liner:** buildUserMessage orchestrator wired into one-shot and REPL paths via preprocessor closure; --search/--no-search flags registered on rootCmd; runConversationLoop extended with nil-safe preprocessor param.

## What Was Built

**`buildUserMessage` in `cmd/search.go`** — Orchestrates the full search pipeline for a single user query: GATE-04 (noSearch) check first, then gate bypass (GATE-03) or LLM gate call (GATE-01/GATE-02), then `search.Search`, re-rank (RANK-01–03), token-budget-aware block assembly (INJ-02), and prepend (INJ-01/INJ-03). Returns bare query on any skip/error path.

**`runConversationLoop` signature change in `cmd/helpers.go`** — Added `preprocessor func(string) string` as 6th parameter. Nil-safe: identity behavior when nil. The `hist.Add("user", input)` call replaced with a preprocessor-guarded version that transforms input before storage. All 6 call sites in `cmd/helpers_test.go` and 1 in `cmd/chat_test.go` updated with trailing `nil`.

**Flag registration and wiring in `cmd/root.go`** — Package-level `searchForce`/`searchSuppress` bools, `init()` registers `--search` and `--no-search` persistent flags on rootCmd. `search.LoadConfig()` called once in RunE body. One-shot path calls `buildUserMessage` before `hist.Add`. REPL path passes a preprocessor closure that captures `cfg`, `searchCfg`, `searchForce`, and `searchSuppress`.

**New tests in `cmd/search_gate_test.go`** — `TestBuildUserMessage_NoSearch` (GATE-04: noSearch=true returns query unchanged) and `TestBuildUserMessage_ForceSearch` (GATE-03: forceSearch=true with unreachable server degrades gracefully).

## Test Coverage

| Test | Requirement | Result |
|------|-------------|--------|
| TestBuildUserMessage_NoSearch | GATE-04 | PASS |
| TestBuildUserMessage_ForceSearch | GATE-03 + graceful degrade | PASS |
| All pre-existing cmd tests | CHAT-01–05, D-01–08 | PASS |
| go test ./... (except pre-existing planner) | full suite | PASS |

## Verification Checks

- `grep "preprocessor func(string) string" cmd/helpers.go` — MATCH
- `grep "preprocessor != nil" cmd/helpers.go` — MATCH
- `grep "searchForce" cmd/root.go` — MATCH
- `grep "searchSuppress" cmd/root.go` — MATCH
- `grep "buildUserMessage" cmd/root.go` — MATCH
- `grep "search.LoadConfig" cmd/root.go` — MATCH
- `grep -c "runConversationLoop.*nil" cmd/helpers_test.go` — 6
- `go build ./...` — clean
- `go test ./cmd/...` — all PASS

## Deviations from Plan

None — plan executed exactly as written.

## Known Stubs

None. All paths are fully implemented.

## Threat Flags

None. No new network endpoints, auth paths, or schema changes introduced beyond what the plan's threat model covers (T-19-05 through T-19-08).

## Self-Check: PASSED

- cmd/search.go contains `func buildUserMessage(`: FOUND
- cmd/helpers.go has `preprocessor func(string) string`: FOUND
- cmd/root.go has `searchForce`: FOUND
- Commit 11124d7 (Task 1): FOUND
- Commit ff1ffc5 (Task 2): FOUND
- `go test ./cmd/...`: all PASS
- `go build ./...`: clean
