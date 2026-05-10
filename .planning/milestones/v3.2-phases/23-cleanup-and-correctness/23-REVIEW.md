---
phase: 23-cleanup-and-correctness
reviewed: 2026-04-24T20:55:34Z
depth: standard
files_reviewed: 4
files_reviewed_list:
  - internal/search/search.go
  - internal/retrieval/retrieval.go
  - cmd/search.go
  - internal/scanner/artifacts.go
findings:
  critical: 0
  warning: 1
  info: 2
  total: 3
status: issues_found
---

# Phase 23: Code Review Report

**Reviewed:** 2026-04-24T20:55:34Z
**Depth:** standard
**Files Reviewed:** 4
**Status:** issues_found

## Summary

Four files were reviewed covering the Phase 23 cleanup and correctness changes: BUG-01 (URL trim), BUG-02 (named error var), CLN-01 (removed `countTokens`), CLN-02 (removed `pkgs` param from `llmReRank`), CLN-03 (reserved comments on `CallEdges`/`TypeRefs`), and PERF-01 (`microPassFile` refactor).

BUG-01 and PERF-01 are correct. The URL trim in `search.go` is applied before concatenation as intended. The `microPassFile` symbol filtering logic properly handles both the stored-symbols path and the AST-fallback path.

One warning was found: the PERF-01 symbol map output does not guard against zero-value `Start`/`End` fields, which would produce misleading `lines 0-0` entries in the prompt sent to the LLM. Two info items are noted: `llmReRank` always returns a `nil` error (BUG-02's named variable is never non-nil), and there is a stale "Phase 12" comment in `retrieval.go` that refers to a completed planning phase.

## Warnings

### WR-01: `microPassFile` emits `lines 0-0` in symbol map when `Start`/`End` are zero

**File:** `internal/retrieval/retrieval.go:699-701`
**Issue:** The stored-symbols code path in `microPassFile` builds a symbol map line by writing `sym.Name: lines sym.Start-sym.End` for each relevant symbol. `Symbol.Start` and `Symbol.End` are set by `ExtractSymbolsFull`; if a symbol was persisted with zero values (e.g., from an older index or a parse edge case), the prompt will contain `lines 0-0`, which is an out-of-bounds description and may cause the LLM to return an invalid or unparseable line range. The fallback chain will recover (truncation at D-08), but the LLM call is wasted.

**Fix:**
```go
for _, sym := range relevantSyms {
    if sym.Start > 0 && sym.End >= sym.Start {
        fmt.Fprintf(&mapSB, "%s: lines %d-%d\n", sym.Name, sym.Start, sym.End)
    }
}
// If no valid entries were written, fall through to ExtractSymbolMap fallback.
if mapSB.Len() == 0 {
    return "", false
}
```
Alternatively, fall back to `ExtractSymbolMap` when any symbol has `Start == 0`, mirroring the existing `len(relevantSyms) == 0` guard.

## Info

### IN-01: `llmReRank` always returns `nil` error — named error var is always `nil`

**File:** `internal/retrieval/retrieval.go:269-298`
**Issue:** BUG-02 changed the call sites to capture a named `reRankErr` instead of discarding with `_`. However, `llmReRank` returns `nil` for the error in every code path (lines 276, 290, 295, 297). The fallback logic is embedded inside the function (returning `candidates` directly on LLM failure), so the error return value is structurally unused. The `if reRankErr != nil` branches at lines 125-127 and 896-898 are dead code.

**Fix:** Either remove the error return from `llmReRank` to make the API honest:
```go
func llmReRank(
    query string,
    candidates []scanner.Symbol,
    cfg config.Config,
    chatFn scanner.ChatFn,
) []scanner.Symbol {
    ...
}
```
Or, if a future caller genuinely needs to distinguish "LLM failed" from "LLM returned nothing useful," propagate a real error in the LLM failure case. The current hybrid (named return, always nil) is misleading.

### IN-02: Stale planning-phase comment in `retrieval.go`

**File:** `internal/retrieval/retrieval.go:29`
**Issue:** The `Strategy` struct carries the comment `// Phase 12 will wire per-command strategies; Phase 11 uses a single default.` Phase 12 is long complete. The comment is now misleading documentation noise rather than a useful guide.

**Fix:** Replace with a plain description:
```go
// Strategy configures which pipeline stages run and at what depth.
// Per-command strategies are defined as package-level vars below.
type Strategy struct {
```

---

_Reviewed: 2026-04-24T20:55:34Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
