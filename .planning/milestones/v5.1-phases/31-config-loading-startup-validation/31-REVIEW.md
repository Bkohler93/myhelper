---
phase: 31-config-loading-startup-validation
reviewed: 2026-05-10T00:00:00Z
depth: standard
files_reviewed: 6
files_reviewed_list:
  - internal/config/config.go
  - internal/config/config_test.go
  - cmd/helpers.go
  - cmd/helpers_test.go
  - cmd/root.go
  - cmd/inspect.go
findings:
  critical: 0
  warning: 0
  info: 2
  total: 2
status: clean
---

# Phase 31: Code Review Report

**Reviewed:** 2026-05-10T00:00:00Z
**Depth:** standard
**Files Reviewed:** 6
**Status:** issues_found

## Summary

This phase removes hardcoded model/endpoint defaults from `config.Load()`, adds a `validateConfig` function in `cmd/helpers.go` that gates Ollama calls when endpoint or model are unconfigured, and expands `TestLoad` with CFG-01 and CFG-02 subtests. The core design is sound. However, there is one behavioral bug in `ApplyFlagOverrides` that silently accepts a zero value as a valid override (preventing users from setting `--token-limit 0` intentionally, but more dangerously allowing a negative value to slip through), three warnings around silent error swallowing and a documentation/contract mismatch with CLAUDE.md, and two informational items.

---

## Critical Issues

### CR-01: `ApplyFlagOverrides` silently accepts negative `--token-limit` values

**File:** `cmd/root.go:79-80`
**Issue:** `ApplyFlagOverrides` only checks `tokenLimitFlag != 0` before writing the value into `cfg.TokenThreshold`. A caller who passes `--token-limit -1` (or any negative integer) bypasses the guard and stores a negative threshold. Downstream consumers (`history.New`, `buildWebBlock`, `budget := cfg.TokenThreshold / 4`) then operate with a negative or zero token budget. `buildWebBlock` will immediately return `""` for every query (silently degrading search injection), and `history.ExceedsLimit()` may behave incorrectly depending on its implementation. This is a behavioral correctness bug introduced by this phase (the guard was added here, not inherited).

**Fix:**
```go
// ApplyFlagOverrides applies CLI flag values to cfg, overriding config-file values.
func ApplyFlagOverrides(cfg *config.Config) {
    if tokenLimitFlag > 0 {
        cfg.TokenThreshold = tokenLimitFlag
    }
}
```

A test should also be added:
```go
t.Run("negative token-limit is ignored", func(t *testing.T) {
    tokenLimitFlag = -1
    defer func() { tokenLimitFlag = 0 }()
    cfg := config.Config{TokenThreshold: 4100}
    ApplyFlagOverrides(&cfg)
    if cfg.TokenThreshold != 4100 {
        t.Errorf("negative flag must not override threshold, got %d", cfg.TokenThreshold)
    }
})
```

---

## Warnings

### WR-01: `config.Load()` silently swallows malformed `MYHELPER_TOKEN_LIMIT`

**File:** `internal/config/config.go:63-67`
**Issue:** When `MYHELPER_TOKEN_LIMIT` contains a non-numeric string (e.g., `"4k"`, `"abc"`), `strconv.Atoi` fails and the assignment is silently skipped. The user believes they have configured a token limit; the config silently falls back to the file or hardcoded default with no warning. This is a latent misconfiguration trap — especially confusing when a file-based threshold is active, because the env var that should have overridden it is quietly dropped.

**Fix:**
```go
if v := os.Getenv("MYHELPER_TOKEN_LIMIT"); v != "" {
    if n, err := strconv.Atoi(v); err == nil {
        cfg.TokenThreshold = n
    } else {
        fmt.Fprintf(os.Stderr, "warning: MYHELPER_TOKEN_LIMIT %q is not a valid integer; using default\n", v)
    }
}
```

Alternatively, `Load()` could return `(Config, error)` so callers can handle it, but a stderr warning is the least-invasive fix given the current signature.

### WR-02: `loadFile` silently swallows JSON parse errors

**File:** `internal/config/config.go:84-97`
**Issue:** If `.myhelper/config.json` exists but contains invalid JSON (e.g., a partial write, a merge conflict marker, or a type error like `"token_threshold": "4100"` as a string instead of an integer), `json.Unmarshal` returns an error that is silently discarded. The function returns `Config{}, false` — indistinguishable from "file not found" — and the home-dir fallback is tried instead. A user with a corrupted local config gets no indication of the problem, and unexpected home-dir values silently win.

**Fix:**
```go
func loadFile(path string) (Config, bool) {
    if path == "" {
        return Config{}, false
    }
    data, err := os.ReadFile(path)
    if err != nil {
        if !os.IsNotExist(err) {
            fmt.Fprintf(os.Stderr, "warning: could not read config %s: %v\n", path, err)
        }
        return Config{}, false
    }
    var c Config
    if err := json.Unmarshal(data, &c); err != nil {
        fmt.Fprintf(os.Stderr, "warning: config %s contains invalid JSON: %v\n", path, err)
        return Config{}, false
    }
    return c, true
}
```

### WR-03: CLAUDE.md documents hardcoded defaults that no longer exist in `config.Load()`

**File:** `internal/config/config.go:24-30`
**Issue:** CLAUDE.md states: `"3. Hardcoded defaults: endpoint 192.168.0.9:11434, model qwen2.5-coder:7b, threshold 4100"`. After this phase's changes, `config.Load()` only hardcodes `DefaultTokenThreshold = 4100`; endpoint and model default to empty strings. Any developer reading CLAUDE.md will believe the binary has working defaults for endpoint and model out of the box, leading to confusion when `validateConfig` immediately errors on a fresh install. This is a docs-code mismatch that can cause real confusion for contributors.

**Fix:** Update CLAUDE.md's config section to reflect the new behavior:
```
Config resolution order (highest to lowest priority):
1. Env vars: `MYHELPER_ENDPOINT`, `MYHELPER_MODEL`, `MYHELPER_TOKEN_LIMIT`
2. `.myhelper/config.json` in CWD, then `~/.config/myhelper/config.json`
3. Hardcoded defaults: threshold `4100` only (endpoint and model have no default — run `myhelper setup`)
```

---

## Info

### IN-01: `config_test.go` uses `os.Chdir` in subtests without parallelism guards

**File:** `internal/config/config_test.go:19-26` (and similar in each subtest)
**Issue:** Each subtest in `TestLoad` calls `os.Chdir` and restores it with a `defer`. Go subtests within a single `t.Run` block run sequentially by default unless `t.Parallel()` is called, so this is safe today. However, the tests cannot be made parallel without data races because `os.Chdir` is process-global. A comment noting this restriction would prevent a future `t.Parallel()` from introducing flakiness, which is a pattern that has burned many Go projects.

**Fix:** Add a comment above the first `os.Chdir` call in each subtest:
```go
// NOTE: os.Chdir is process-global; do NOT add t.Parallel() to this subtest.
```

### IN-02: `validateConfig` combines two distinct errors into one branch unnecessarily

**File:** `cmd/helpers.go:282-284`
**Issue:** The first branch of `validateConfig` checks `cfg.Endpoint == "" && cfg.Model == ""` and returns a combined error message. The subsequent two branches then check each field independently. The combined branch is dead code in practice: if both are empty, the first branch fires; but the combined message differs from the two single-field messages only in phrasing. This creates three code paths where two would suffice (remove the combined branch; rely on the `Endpoint == ""` branch firing first), and the combined message string `"endpoint and model are not configured"` is not tested in `helpers_test.go` line 271 — the test only checks for `"myhelper setup"`, so the specific phrasing is untested.

**Fix:** Remove the combined branch and rely on the two individual checks:
```go
func validateConfig(cfg config.Config) error {
    if cfg.Endpoint == "" {
        return fmt.Errorf("endpoint is not configured\nRun 'myhelper setup' to configure myhelper")
    }
    if cfg.Model == "" {
        return fmt.Errorf("model is not configured\nRun 'myhelper setup' to configure myhelper")
    }
    return nil
}
```

This preserves all tested behavior (both fields empty → endpoint error fires first) and eliminates the redundant branch.

---

_Reviewed: 2026-05-10T00:00:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
