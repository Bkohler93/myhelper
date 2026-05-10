---
phase: 30-setup-wizard
reviewed: 2026-05-10T00:00:00Z
depth: standard
files_reviewed: 3
files_reviewed_list:
  - cmd/setup.go
  - internal/wizard/wizard.go
  - internal/wizard/wizard_test.go
findings:
  critical: 0
  warning: 0
  info: 2
  total: 2
status: clean
---

# Phase 30: Code Review Report (Re-review)

**Reviewed:** 2026-05-10
**Depth:** standard
**Files Reviewed:** 3
**Status:** clean

## Summary

Re-review performed after fixes for CR-01, CR-02, WR-01, and WR-02. All four Critical and Warning findings from the original review are resolved. Two Info-level items remain (one carried over from the original review, one newly identified in tests). No blockers or warnings remain.

---

## CR-01 — Resolved

`pullModel` now checks `resp.StatusCode != http.StatusOK` at line 242 before scanning the NDJSON body. A non-200 response drains the body and returns a descriptive error.

## CR-02 — Resolved

`pullModel` now tracks a `succeeded bool` (line 248). If the scanner loop exits at EOF without having seen a `"success"` status line, the function returns `fmt.Errorf("model pull ended without confirmation — download may be incomplete")` (line 274). False-success on truncated downloads is no longer possible.

## WR-01 — Resolved

Two package-level HTTP clients with explicit timeouts replace the default no-timeout client:
- `ollamaHTTPClient` (`Timeout: 5s`) used in `checkOllama` (line 130)
- `pullHTTPClient` (`Timeout: 5m`) used in `pullModel` (line 235)

## WR-02 — Resolved

`detectMemoryMiB` now splits `nvidia-smi` output on newlines and uses only `lines[0]` (lines 159-162). Multi-GPU systems no longer fall back to RAM.

---

## Info

### IN-01: `json.Marshal` error silently ignored in `pullModel`

**File:** `internal/wizard/wizard.go:233`
**Issue:** `body, _ := json.Marshal(pullRequest{Name: name, Stream: true})` discards the marshal error. `json.Marshal` cannot realistically fail on a struct with only `string` and `bool` fields, but the blank-identifier discard suppresses any future failure if `pullRequest` grows a field type that can fail serialization, and it is inconsistent with the project's error-handling style.
**Fix:**
```go
body, err := json.Marshal(pullRequest{Name: name, Stream: true})
if err != nil {
    return fmt.Errorf("marshal pull request: %w", err)
}
```

---

### IN-02: Package-level test-injection vars are not safe for parallel test execution

**File:** `internal/wizard/wizard_test.go` — all test functions
**Issue:** `ollamaBaseURL` and `configPathOverride` are package-level variables mutated directly by every test function. No test calls `t.Parallel()`, so they are currently safe. However, there is no comment warning future contributors against adding `t.Parallel()`. If parallelism is ever introduced, the `-race` detector will flag these writes and reads.
**Fix:** Add a comment at the top of the test file:
```go
// NOTE: tests in this package mutate package-level globals (ollamaBaseURL,
// configPathOverride). Do NOT add t.Parallel() to any test here without first
// protecting those vars with a sync.Mutex or refactoring to a deps-injection approach.
```

---

_Reviewed: 2026-05-10_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
