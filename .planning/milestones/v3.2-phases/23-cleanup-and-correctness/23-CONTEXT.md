# Phase 23: Cleanup & Correctness - Context

**Gathered:** 2026-04-24
**Status:** Ready for planning
**Mode:** Auto-generated (discuss skipped via workflow.skip_discuss)

<domain>
## Phase Boundary

All known v3.1 tech debt eliminated — bugs fixed, duplicate code removed, dormant fields either wired or documented. PROJECT.md Core Value updated to reflect the current chat+web-search primary use case.

</domain>

<decisions>
## Implementation Decisions

### BUG-01: SearXNG Double-Slash URL
**File:** `internal/search/search.go:98`
**Fix:** Strip trailing slash from endpoint before concatenation:
```go
// Change:
reqURL := endpoint + "/search?q=..."
// To:
reqURL := strings.TrimRight(endpoint, "/") + "/search?q=..."
```
Add `"strings"` import if not present.

### BUG-02: llmReRank error silently discarded
**Files:** `internal/retrieval/retrieval.go` lines ~124 and ~843
Two call sites use `selected, _ := llmReRank(...)`. Change to surface the error. On error, fall back to all candidates (preserving existing graceful-degradation behavior), but log or return the error to the caller:
```go
// In BuildContext (line ~124):
selected, err := llmReRank(query, candidates, pkgs.Packages, cfg, chatFn)
if err != nil {
    selected = candidates // fallback: use all candidates
}

// In BuildInspectContext (line ~843):
selected, reRankErr := llmReRank(query, candidates, pkgs.Packages, cfg, chatFn)
if reRankErr != nil {
    selected = candidates // fallback
}
```
Note: BuildContext returns `[]history.Message, error` — the re-rank error can be returned or swallowed (preserving fail-open). BuildInspectContext already returns `(InspectResult, error)`. Use judgment on whether to propagate or swallow; the key is that it's no longer silently discarded with `_`.

### CLN-01: Remove countTokens duplicate from cmd/search.go
**File:** `cmd/search.go`
Delete the `countTokens` function (lines ~131-133). Update its two callers in `buildWebBlock` to inline the call:
```go
// Change countTokens(header+footer, cfg) to:
history.New(cfg.TokenThreshold, []history.Message{{Role: "user", Content: header + footer}}).TokenCount()
```
Or extract a one-liner helper using `history.New`. The `history` import is already present in `cmd/search.go` via the `reRankResults` function.

### CLN-02: Remove unused pkgs parameter from llmReRank
**File:** `internal/retrieval/retrieval.go`
`llmReRank` receives `pkgs []scanner.PackageEntry` (line ~269) but never uses it in the function body. Remove it:
1. Remove `pkgs []scanner.PackageEntry` from the function signature
2. Update all call sites (BuildContext line ~124, BuildInspectContext line ~843) to drop the `pkgs.Packages` argument

### CLN-03: Document CallEdges/TypeRefs as reserved
**File:** `internal/scanner/artifacts.go`
Add a comment to the `CallEdges` and `TypeRefs` fields in the `Symbol` struct:
```go
CallEdges []string `json:"call_edges"` // reserved for future call-graph-aware ranking
TypeRefs  []string `json:"type_refs"`  // reserved for future type-aware ranking
```
No functional change — schema is preserved to avoid breaking existing .myhelper/ directories.

### CTX-03: Already resolved — document as closed
`LoadContext` is defined in `internal/context/context.go` but is never called anywhere. The dual injection concern (context.md + proj.Summary) does not exist in the current codebase. No code change needed. Note in SUMMARY.md that this was verified as already resolved.

### PERF-01: microPassFile uses stored Symbol.Start/End
**File:** `internal/retrieval/retrieval.go`
Add a `symbols []scanner.Symbol` parameter to `microPassFile`. Filter to symbols whose `FilePath` matches `path`. Use those symbols' `Start`/`End` values directly instead of calling `scanner.ExtractSymbolMap(absPath)`:

```go
// New signature:
func microPassFile(root, path, query string, cfg config.Config, chatFn scanner.ChatFn, budget int, symbols []scanner.Symbol) (string, bool)

// Inside, replace:
symbols, err := scanner.ExtractSymbolMap(absPath)
// With:
var relevantSyms []scanner.Symbol
for _, s := range symbols {
    if s.FilePath == path {
        relevantSyms = append(relevantSyms, s)
    }
}
// Use relevantSyms in the symbol map builder
```

Update `assembleMessages` call site (line ~531) to pass `symbols` from its parameter.

If `relevantSyms` is empty (symbols not available for this file), fall back to the existing `ExtractSymbolMap` call to preserve correctness.

### PROJECT.md Core Value Update
Update `.planning/PROJECT.md`:
- **Core Value** section: change from "project-aware coding assistant" framing to "fast, local chat with optional web search for current information"
- **What This Is** section: already correct ("fast, local-model-powered chat with optional web search augmentation via SearXNG") — verify and adjust if needed
- Update the Last Updated footer

### Claude's Discretion
All other implementation choices at Claude's discretion.

</decisions>

<code_context>
## Existing Code Insights

### Key Files
- `internal/search/search.go:98` — URL construction: `reqURL := endpoint + "/search?q=..."`
- `internal/retrieval/retrieval.go:124` — `selected, _ := llmReRank(...)` in BuildContext
- `internal/retrieval/retrieval.go:843` — `selected, _ := llmReRank(...)` in BuildInspectContext
- `internal/retrieval/retrieval.go:266-295` — `llmReRank` signature includes unused `pkgs []scanner.PackageEntry`
- `internal/retrieval/retrieval.go:531` — `microPassFile(root, fp, query, cfg, chatFn, remaining)` call site
- `internal/retrieval/retrieval.go:624-680` — `microPassFile` calls `scanner.ExtractSymbolMap`
- `cmd/search.go:131-133` — duplicate `countTokens` function
- `cmd/search.go:110,118` — two callers of `countTokens`
- `internal/scanner/artifacts.go:~148` — `Symbol` struct with `CallEdges`, `TypeRefs` fields
- `.planning/PROJECT.md` — Core Value needs updating

### Established Patterns
- Error handling: fail-open (return all candidates on LLM error) — preserve this
- `strings.TrimRight` for URL cleanup
- `history.New(...).TokenCount()` for token counting

</code_context>

<specifics>
## Specific Ideas

- For BUG-02, consider whether to propagate or swallow the re-rank error. Given the tool is local and fail-open is intentional, swallowing is fine as long as we don't use `_` (use named variable `reRankErr` and explicitly assign fallback).
- For PERF-01 fallback: if `len(relevantSyms) == 0`, still call `ExtractSymbolMap` as a safety net so existing behavior is preserved for edge cases.
- Group the fixes into logical plans: bugs + dead code removal in one wave, PERF-01 + PROJECT.md in same or separate.

</specifics>

<deferred>
## Deferred Ideas

None — all items from v3.2 requirements scope.

</deferred>
