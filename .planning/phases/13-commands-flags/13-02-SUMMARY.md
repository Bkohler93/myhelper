---
phase: 13-commands-flags
plan: "02"
subsystem: cmd
tags: [tdd, green-state, cmd-flags, token-limit, no-context]
dependency_graph:
  requires: [13-01]
  provides: [cmd.ApplyFlagOverrides_wired, cmd.noContextFlag_registered]
  affects: [cmd/root.go, cmd/plan.go, cmd/lookup.go, cmd/starter.go, cmd/pattern.go]
tech_stack:
  added: []
  patterns: [TDD-GREEN, persistent-flag, retrieval-bypass]
key_files:
  created: []
  modified:
    - cmd/root.go
    - cmd/plan.go
    - cmd/lookup.go
    - cmd/starter.go
    - cmd/pattern.go
decisions:
  - "noContextFlag registered as PersistentFlag on rootCmd so all subcommands inherit it without per-command registration"
  - "noContextFlag guard placed after appctx.LoadContext() and ApplyFlagOverrides so projectCtx and cfg are bound in both paths"
  - "When --no-context set, rctx.Messages carries a bare user Message; downstream history/conversation code runs unchanged"
metrics:
  duration: "~5 minutes"
  completed: "2026-04-10"
  tasks_completed: 1
  files_modified: 5
---

# Phase 13 Plan 02: Wire ApplyFlagOverrides and --no-context to Query Commands Summary

Fixed the `--token-limit` silent no-op (CMD-03) by calling `ApplyFlagOverrides(&cfg)` in all four query commands, and added the `--no-context` persistent flag (CMD-01) that bypasses `retrieval.BuildContext` and injects a bare user message instead.

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | Add noContextFlag to root.go and ApplyFlagOverrides + --no-context guard to all four query commands | 5f822db | cmd/root.go, cmd/plan.go, cmd/lookup.go, cmd/starter.go, cmd/pattern.go |

## What Was Built

### cmd/root.go

- Added `var noContextFlag bool` package-level variable after `tokenLimitFlag`
- Registered `--no-context` as a persistent flag on `rootCmd` via `PersistentFlags().BoolVar` — available to all subcommands automatically

### cmd/plan.go, cmd/lookup.go, cmd/starter.go, cmd/pattern.go

For each of the four query commands:

1. `ApplyFlagOverrides(&cfg)` added immediately after `cfg := config.Load()` — fixes CMD-03 tech debt; `--token-limit` now flows into `cfg.TokenThreshold` before `retrieval.BuildContext` uses it
2. `noContextFlag` guard replaces single-line `retrieval.BuildContext` call:
   - When `--no-context` is set: `rctx` is populated with a bare `history.Message{Role: "user", Content: input}` — no project artifact reads
   - When flag is not set: `retrieval.BuildContext` runs as before
3. `var rctx retrieval.Context` declared before the guard; downstream `messages` assembly and conversation loop are unchanged in both paths

## Verification Results

```
go build ./cmd/...                                           → exit 0
go test ./cmd/... -run TestApplyFlagOverrides_QueryCommands  → PASS
go test ./cmd/... -run TestNoContextFlag_Registered          → PASS
go test ./cmd/... -count=1                                   → ok (all tests green)
```

RED tests from Plan 01 turned GREEN:
- `TestApplyFlagOverrides_QueryCommands` — source-scan confirms all four files contain `ApplyFlagOverrides(&cfg)`
- `TestNoContextFlag_Registered` — confirms `rootCmd.PersistentFlags().Lookup("no-context") != nil` and default `"false"`

## Deviations from Plan

None — plan executed exactly as written.

## Known Stubs

None — all behavior is fully wired; no placeholder values or TODO paths.

## Threat Flags

None — `--token-limit` and `--no-context` are purely additive CLI flags. No new network endpoints, auth paths, or file access patterns introduced. Threat register items T-13-02 and T-13-03 accepted as documented in plan.

## Self-Check

- [x] `cmd/root.go` contains `noContextFlag bool` — FOUND (line 12)
- [x] `cmd/root.go` contains `no-context` PersistentFlags registration — FOUND (line 23)
- [x] `cmd/plan.go` contains `ApplyFlagOverrides(&cfg)` — FOUND (line 46)
- [x] `cmd/lookup.go` contains `ApplyFlagOverrides(&cfg)` — FOUND (line 46)
- [x] `cmd/starter.go` contains `ApplyFlagOverrides(&cfg)` — FOUND (line 46)
- [x] `cmd/pattern.go` contains `ApplyFlagOverrides(&cfg)` — FOUND (line 46)
- [x] `cmd/plan.go` contains `noContextFlag` guard — FOUND
- [x] `cmd/lookup.go` contains `noContextFlag` guard — FOUND
- [x] `cmd/starter.go` contains `noContextFlag` guard — FOUND
- [x] `cmd/pattern.go` contains `noContextFlag` guard — FOUND
- [x] Commit `5f822db` exists — FOUND

## Self-Check: PASSED
