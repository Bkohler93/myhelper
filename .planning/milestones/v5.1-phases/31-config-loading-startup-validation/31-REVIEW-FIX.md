---
phase: 31-config-loading-startup-validation
fixed_at: 2026-05-10T00:00:00Z
review_path: .planning/phases/31-config-loading-startup-validation/31-REVIEW.md
iteration: 1
findings_in_scope: 4
fixed: 4
skipped: 0
status: all_fixed
---

# Phase 31: Code Review Fix Report

**Fixed at:** 2026-05-10T00:00:00Z
**Source review:** .planning/phases/31-config-loading-startup-validation/31-REVIEW.md
**Iteration:** 1

**Summary:**
- Findings in scope: 4 (1 Critical + 3 Warning)
- Fixed: 4
- Skipped: 0

## Fixed Issues

### CR-01: `ApplyFlagOverrides` silently accepts negative `--token-limit` values

**Files modified:** `cmd/root.go`, `cmd/root_test.go`
**Commit:** 2c6a22c
**Applied fix:** Changed the guard in `ApplyFlagOverrides` from `tokenLimitFlag != 0` to `tokenLimitFlag > 0`. Created `cmd/root_test.go` with `TestApplyFlagOverrides` covering three subtests: positive override, zero (unchanged), and negative (ignored).

### WR-01: `config.Load()` silently swallows malformed `MYHELPER_TOKEN_LIMIT`

**Files modified:** `internal/config/config.go`
**Commit:** 7e1e394
**Applied fix:** Added an `else` branch after the `strconv.Atoi` failure that prints `fmt.Fprintf(os.Stderr, "warning: MYHELPER_TOKEN_LIMIT %q is not a valid integer; using default\n", v)`. Also added `"fmt"` to the import block, which was not previously imported in this file.

### WR-02: `loadFile` silently swallows JSON parse errors

**Files modified:** `internal/config/config.go`
**Commit:** 6ef268d
**Applied fix:** In `loadFile`, added a `!os.IsNotExist(err)` guard so permission errors and other read failures emit a stderr warning. Added a stderr warning when `json.Unmarshal` fails. Both paths continue to return `Config{}, false` so load-order behavior is unchanged.

### WR-03: CLAUDE.md documents stale hardcoded defaults

**Files modified:** `CLAUDE.md`
**Commit:** b6c670e
**Applied fix:** Replaced line 3 of the Config resolution order from `"Hardcoded defaults: endpoint 192.168.0.9:11434, model qwen2.5-coder:7b, threshold 4100"` to `"Hardcoded defaults: threshold 4100 only (endpoint and model have no default — run myhelper setup)"`.

---

_Fixed: 2026-05-10T00:00:00Z_
_Fixer: Claude (gsd-code-fixer)_
_Iteration: 1_
