---
phase: 20-fix-srch-04
reviewed: 2026-04-11
status: clean
severity_counts:
  critical: 0
  high: 0
  medium: 0
  low: 1
  info: 2
---

# Phase 20: Code Review Report

**Reviewed:** 2026-04-11
**Depth:** standard
**Files Reviewed:** 3
**Status:** clean (no medium or higher severity findings)

## Summary

Three targeted changes were reviewed: adding `&num_results=10` to the SearXNG request URL, adding a `result_count_present` subtest, and correcting the `searchGate` comment. All three changes are correct and do what they claim. The `num_results` parameter is appended in the right position and the new subtest validates both presence and value. The comment fix accurately reflects the fail-open behavior (error returns `false`, skipping search).

One pre-existing low-severity issue was surfaced during review (URL construction by concatenation). Two info-level nits in the test file.

---

## Low Issues

### LW-01: URL constructed by string concatenation — malformed on non-canonical endpoints

**File:** `internal/search/search.go:98`
**Issue:** The request URL is assembled by raw string concatenation. If `cfg.Endpoint` has a trailing slash (e.g., `http://host:8083/`) the path becomes `//search?...`. If it somehow carries a query string fragment the result is unparseable. The schema-prefix guard on lines 94-96 is the only normalization applied. This is a pre-existing pattern, not introduced in this phase, but it is reachable via the `MYHELPER_SEARCH_ENDPOINT` env var.
**Fix:** Strip a trailing slash before concatenating, or build the URL with `url.URL` and `url.Values`:
```go
base := strings.TrimRight(endpoint, "/")
params := url.Values{
    "q":           {query},
    "format":      {"json"},
    "pageno":      {"1"},
    "num_results": {"10"},
}
reqURL := base + "/search?" + params.Encode()
```

---

## Info Items

### IN-01: Redundant dual-error in result_count_present subtest

**File:** `internal/search/search_test.go:163-167`
**Issue:** The subtest checks `numResults == ""` with `t.Errorf` and then immediately checks `numResults != "10"` with a second `t.Errorf`. When the parameter is absent (empty string), both conditions are true and two error lines fire. This does not affect pass/fail correctness but produces confusing double output on failure.
**Fix:** Collapse into a single check:
```go
if numResults != "10" {
    t.Errorf("expected num_results=10, got %q", numResults)
}
```
An empty string will fail this check with a clear message.

### IN-02: Hand-rolled containsString duplicates strings.Contains

**File:** `internal/search/search_test.go:243-253`
**Issue:** `containsString` reimplements a substring search. The comment says it avoids importing `strings`, but adding one import line is trivial and `strings.Contains` is better-known to readers. The custom implementation is not wrong, just unnecessary complexity in a test file.
**Fix:** Delete `containsString`, add `"strings"` to the import block, and replace the call site:
```go
if !strings.Contains(err.Error(), "500") {
```

---

_Reviewed: 2026-04-11_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
