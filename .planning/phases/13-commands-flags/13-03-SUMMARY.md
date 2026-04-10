---
phase: 13-commands-flags
plan: "03"
subsystem: retrieval, cmd
tags: [inspect, dry-run, retrieval-pipeline, cmd-flags, CMD-02]
dependency_graph:
  requires: [13-01]
  provides: [retrieval.BuildInspectContext, retrieval.InspectResult, retrieval.SelectionSource, cmd.inspectCmd]
  affects: [internal/retrieval/retrieval.go, cmd/inspect.go]
tech_stack:
  added: []
  patterns: [dry-run-pipeline, per-stage-diagnostics, one-shot-command]
key_files:
  created: [cmd/inspect.go]
  modified: [internal/retrieval/retrieval.go]
decisions:
  - "BuildInspectContext mirrors BuildContext pipeline stages but never calls assembleMessages — diagnostic only"
  - "SourcePreFilter not used in output (only SourceReRank assigned to symbols) — pre-filter is a candidate gate not a final selection stage"
  - "inspect uses DefaultStrategy — per-command strategy override is out of scope for Phase 13"
  - "printInspectResult writes to stdout via fmt.Printf/Println — not stderr — inspect output IS the intended output"
metrics:
  duration: "~10 minutes"
  completed: "2026-04-10"
  tasks_completed: 2
  files_modified: 2
---

# Phase 13 Plan 03: InspectResult + inspect command Summary

Implemented `SelectionSource`, `SymbolResult`, `FileResult`, `StageMetrics`, `InspectResult` types and `BuildInspectContext` function in the retrieval package, then wired the new `inspect` cobra command to it for one-shot dry-run diagnostic output.

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | Add InspectResult types and BuildInspectContext to retrieval.go | 7572c26 | internal/retrieval/retrieval.go |
| 2 | Create cmd/inspect.go wired to BuildInspectContext | 9095776 | cmd/inspect.go |

## What Was Built

### Task 1 — retrieval.go additions

Appended to `internal/retrieval/retrieval.go` (after `microPassFile`):

- `SelectionSource` int type with `SourcePreFilter`, `SourceReRank`, `SourceExpansion` constants and `String()` method
- `SymbolResult` struct pairing `scanner.Symbol` with `SelectionSource`
- `FileResult` struct pairing a file path with `SelectionSource`
- `StageMetrics` struct recording stage name and token cost
- `InspectResult` struct holding all per-stage diagnostic output (no messages field — never calls `assembleMessages`)
- `BuildInspectContext` function running the four-stage pipeline in dry-run mode:
  - Returns `InspectResult{GatePassed: false}` with nil error when artifacts missing
  - Returns `InspectResult{GatePassed: false}` with nil error when relevance gate blocks
  - Populates StageMetrics for pre-filter, re-rank, and expansion stages
  - Assigns `SourceReRank` to symbols selected by `llmReRank`
  - Assigns `SourceReRank` to files from `uniqueFilePaths(selected)` and `SourceExpansion` to additional files from `expandDeps`

### Task 2 — cmd/inspect.go (new file)

Created `cmd/inspect.go` in `package cmd`:

- `inspectCmd` cobra command (`inspect <query>`, MaximumNArgs(1))
- `runInspect` calls `resolveInput`, `config.Load()`, `ApplyFlagOverrides`, `os.Getwd()`, then `retrieval.BuildInspectContext`
- `printInspectResult` formats gate status, per-stage metrics, selected symbols with source tags, selected files with source tags, and final token count to stdout
- Does NOT call `initiateConversation` or `runConversationLoop` — one-shot exit after printing

## Verification Results

```
go test ./internal/retrieval/... -run "TestSelectionSource_String|TestBuildInspectContext" -count=1
PASS

go test ./internal/retrieval/... -count=1
ok  github.com/bkohler93/myhelper/internal/retrieval

go build ./...
exit 0
```

Pre-existing RED tests `TestApplyFlagOverrides_QueryCommands` and `TestNoContextFlag_Registered` remain failing — they guard CMD-01 and CMD-03 deliverables implemented by plan 13-02 (parallel wave-2 agent). Not in scope for this plan.

## Deviations from Plan

None — plan executed exactly as written.

## Known Stubs

None — `BuildInspectContext` runs the full pipeline and returns real diagnostics. `printInspectResult` prints actual retrieved data.

## Threat Flags

None — inspect outputs only local `.myhelper/` artifact metadata to stdout (same surface as existing BuildContext, accepted per T-13-04 and T-13-05 in the plan threat model).

## Self-Check

- [x] `internal/retrieval/retrieval.go` contains `func BuildInspectContext` — FOUND
- [x] `internal/retrieval/retrieval.go` contains `SelectionSource` — FOUND
- [x] `internal/retrieval/retrieval.go` contains `SourcePreFilter` — FOUND
- [x] `internal/retrieval/retrieval.go` contains `InspectResult` — FOUND
- [x] `cmd/inspect.go` exists — FOUND
- [x] `cmd/inspect.go` contains `inspectCmd` — FOUND
- [x] `cmd/inspect.go` contains `BuildInspectContext` call — FOUND
- [x] `cmd/inspect.go` contains `printInspectResult` — FOUND
- [x] `cmd/inspect.go` does NOT contain `initiateConversation` or `runConversationLoop` — CONFIRMED
- [x] Commit `7572c26` exists — FOUND
- [x] Commit `9095776` exists — FOUND

## Self-Check: PASSED
