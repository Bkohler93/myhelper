---
phase: 21-inspect-command
plan: "01"
subsystem: retrieval, cmd
tags: [inspect, pre-filter, gate, helpers, flags]
dependency_graph:
  requires: []
  provides:
    - PreFilterCandidate type in internal/retrieval
    - InspectResult.GateAnswer field
    - InspectResult.PreFilterCandidates field
    - cmd.resolveInput helper
    - cmd.readInteractive helper
    - cmd.noContextFlag package var
    - cmd.tokenLimitFlag package var
    - cmd.ApplyFlagOverrides function
  affects:
    - internal/retrieval/retrieval.go (BuildInspectContext, InspectResult)
    - cmd/helpers.go
    - cmd/root.go
tech_stack:
  added: []
  patterns:
    - Inline gate call in BuildInspectContext to capture raw LLM answer before bool-parsing
    - PreFilterCandidate wrapper pairs scanner.Symbol with keyword score
key_files:
  created: []
  modified:
    - internal/retrieval/retrieval.go
    - cmd/helpers.go
    - cmd/root.go
decisions:
  - Inline gate logic in BuildInspectContext rather than adding a return-value variant of needsContext — avoids changing a stable function used by BuildContext
  - resolveInput/readInteractive placed before summarize in helpers.go to keep helper functions grouped together
metrics:
  duration: "~2 minutes"
  completed: "2026-04-24"
  tasks_completed: 2
  files_modified: 3
---

# Phase 21 Plan 01: Inspect Command Infrastructure Summary

Extended `InspectResult` with `PreFilterCandidates []PreFilterCandidate` (symbol + keyword score) and `GateAnswer string` (raw LLM string), then restored `resolveInput`/`readInteractive` cmd helpers and `noContextFlag`/`ApplyFlagOverrides` in root.go — providing all the building blocks Plan 02 needs to wire and render the `inspect` command output.

## Tasks Completed

| Task | Name | Commit |
|------|------|--------|
| 1 | Add PreFilterCandidate type, extend InspectResult, update BuildInspectContext | d2329ab |
| 2 | Restore resolveInput/readInteractive in cmd/helpers.go and noContextFlag/ApplyFlagOverrides in cmd/root.go | 1e9ed14 |

## What Was Built

### Task 1 — retrieval.go changes

- **`PreFilterCandidate` struct** added immediately before `InspectResult`: `{ Symbol scanner.Symbol; Score int }`. Used by the inspect command to display per-symbol keyword scores (INSP-03).
- **`InspectResult.GateAnswer string`** — captures the raw LLM string from the relevance gate before bool-parsing, so inspect output can show the actual model response alongside PASS/FAIL (INSP-02).
- **`InspectResult.PreFilterCandidates []PreFilterCandidate`** — appended in `BuildInspectContext` Stage 2 loop using `scoreSymbol(c, queryTerms)` after `applyTokenCap`.
- **Inlined gate call** in `BuildInspectContext` Stage 1 — duplicates the logic from `needsContext` inside a scoped `{}` block so the raw response string is captured into `result.GateAnswer`. `needsContext` itself is unchanged (still used by `BuildContext`).

### Task 2 — cmd/helpers.go and cmd/root.go changes

- **`resolveInput`** — returns first positional arg if non-empty, else calls `readInteractive`.
- **`readInteractive`** — writes prompt to stderr, reads one line from `stdinReader` (test-injectable), returns trimmed input or error.
- **`noContextFlag bool`** and **`tokenLimitFlag int`** — package-level vars registered as `--no-context` and `--token-limit` persistent flags in `rootCmd.init()`.
- **`ApplyFlagOverrides(cfg *config.Config)`** — applies `tokenLimitFlag` to `cfg.TokenThreshold` when non-zero. Called by subcommands after `config.Load()`.
- Existing `--search` and `--no-search` flags are unaffected.

## Deviations from Plan

None — plan executed exactly as written.

## Known Stubs

None. All fields and functions are fully implemented; no placeholder values.

## Threat Flags

None. No new network endpoints, auth paths, file access patterns, or schema changes introduced. `tokenLimitFlag` is a local performance knob; `readInteractive` reads only from `os.Stdin` or a test pipe (no network exposure). Both accepted per plan threat model (T-21-01, T-21-02).

## Pre-existing Issues (Out of Scope)

`internal/planner.TestParsePlan` fails because it references `.planning/phases/14-ollama-client-extension/14-01-PLAN.md` which no longer exists on disk. This failure predates this plan and is unrelated to any changes made here.

## Self-Check: PASSED

- `d2329ab` exists: confirmed via `git log`
- `1e9ed14` exists: confirmed via `git log`
- `internal/retrieval/retrieval.go` modified: confirmed
- `cmd/helpers.go` modified: confirmed
- `cmd/root.go` modified: confirmed
- `grep -c "type PreFilterCandidate struct" internal/retrieval/retrieval.go` → 1
- `grep -c "GateAnswer" internal/retrieval/retrieval.go` → 3
- `grep -c "PreFilterCandidates" internal/retrieval/retrieval.go` → 2
- `grep -c "noContextFlag" cmd/root.go` → 2
- `grep -c "func resolveInput" cmd/helpers.go` → 1
- `go build ./cmd/...` → 0
- `go build ./internal/retrieval/...` → 0
