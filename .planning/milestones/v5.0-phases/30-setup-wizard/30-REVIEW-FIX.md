---
phase: 30
fixed_at: 2026-05-10T00:00:00Z
review_path: .planning/phases/30-setup-wizard/30-REVIEW.md
iteration: 1
findings_in_scope: 4
fixed: 4
skipped: 0
status: all_fixed
---

# Phase 30: Code Review Fix Report

**Fixed at:** 2026-05-10
**Source review:** .planning/phases/30-setup-wizard/30-REVIEW.md
**Iteration:** 1

**Summary:**
- Findings in scope: 4
- Fixed: 4
- Skipped: 0

## Fixed Issues

### CR-01: pullModel doesn't check HTTP status before scanning NDJSON body

**Files modified:** `internal/wizard/wizard.go`
**Commit:** 9845d4c
**Applied fix:** Added a status check immediately after the `http.Post` call in `pullModel`. If `resp.StatusCode != http.StatusOK`, reads the body and returns a formatted error with the status code and body text before the scanner is created.

### CR-02: Stream ending without a "success" status line not detected as error

**Files modified:** `internal/wizard/wizard.go`
**Commit:** 9845d4c
**Applied fix:** Added a `var succeeded bool` flag. On `p.Status == "success"`, sets `succeeded = true` and breaks instead of the old `return nil`. After the scan loop, checks `sc.Err()` first, then returns an error if `!succeeded`. The original `return sc.Err()` which could silently return `nil` on a truncated stream is replaced by this two-stage check.

### WR-01: checkOllama uses http.Get with the default client (no timeout)

**Files modified:** `internal/wizard/wizard.go`
**Commit:** ca9ff48
**Applied fix:** Added two package-level HTTP clients: `ollamaHTTPClient` (`Timeout: 5 * time.Second`) for reachability checks and `pullHTTPClient` (`Timeout: 5 * time.Minute`) for model downloads. Updated `checkOllama` to use `ollamaHTTPClient.Get` and `pullModel` to use `pullHTTPClient.Post`. Added `"time"` to imports.

### WR-02: Multi-GPU nvidia-smi output breaks strconv.ParseInt

**Files modified:** `internal/wizard/wizard.go`
**Commit:** 7b4da8a
**Applied fix:** In `detectMemoryMiB` (Linux branch), after trimming the `nvidia-smi` output, split on `"\n"` and parse `lines[0]` instead of the full trimmed string. This handles multi-GPU systems that emit one memory value per line (e.g. `"16384\n16384"`).

---

_Fixed: 2026-05-10_
_Fixer: Claude (gsd-code-fixer)_
_Iteration: 1_
