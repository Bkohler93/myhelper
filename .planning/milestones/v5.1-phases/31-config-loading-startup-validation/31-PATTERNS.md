# Phase 31: Config Loading & Startup Validation - Pattern Map

**Mapped:** 2026-05-10
**Files analyzed:** 6
**Analogs found:** 6 / 6

---

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|---|---|---|---|---|
| `internal/config/config.go` | config | request-response | `internal/config/config.go` (self) | self — surgical deletion |
| `internal/config/config_test.go` | test | — | `internal/config/config_test.go` (self) | self — extend existing table |
| `cmd/helpers.go` | utility | request-response | `cmd/helpers.go` (self) | self — pure addition |
| `cmd/root.go` | controller | request-response | `cmd/inspect.go` | exact — same Load()+guard pattern |
| `cmd/inspect.go` | controller | request-response | `cmd/root.go` | exact — same Load()+guard pattern |
| `cmd/helpers_test.go` | test | — | `cmd/helpers_test.go` (self) | self — extend existing table |

---

## Pattern Assignments

### `internal/config/config.go` — remove hardcoded defaults (CFG-01, CFG-02)

**Change type:** Surgical deletion — two lines removed from the `Load()` struct literal.

**Current struct literal** (`config.go` lines 31–35):
```go
cfg := Config{
    Endpoint:       DefaultEndpoint,
    Model:          DefaultModel,
    TokenThreshold: DefaultTokenThreshold,
}
```

**After change** — remove the two pre-seeds, delete the two orphaned constants:
```go
cfg := Config{
    // Endpoint and Model intentionally omitted: empty string until set via
    // config file or env var. Commands validate non-empty before use.
    TokenThreshold: DefaultTokenThreshold,
}
```

**Constants to delete** (`config.go` lines 11–12):
```go
DefaultEndpoint = "192.168.1.146:11434"
DefaultModel    = "qwen2.5-coder:7b"
```
`DefaultTokenThreshold` on line 13 must be retained.

**No callers of the deleted constants exist outside `Load()`** — confirmed by reading all files in `cmd/` and `internal/config/`.

---

### `internal/config/config_test.go` — add CFG-01 / CFG-02 test cases

**Analog:** The file itself (lines 10–32) — follow the exact pattern of the `"default TokenThreshold is 4100"` subtest.

**Existing pattern to copy** (`config_test.go` lines 11–32):
```go
t.Run("default TokenThreshold is 4100", func(t *testing.T) {
    t.Setenv("MYHELPER_TOKEN_LIMIT", "")
    t.Setenv("MYHELPER_ENDPOINT", "")
    t.Setenv("MYHELPER_MODEL", "")

    dir := t.TempDir()
    orig, err := os.Getwd()
    if err != nil {
        t.Fatal(err)
    }
    if err := os.Chdir(dir); err != nil {
        t.Fatal(err)
    }
    defer func() { _ = os.Chdir(orig) }()

    cfg := Load()
    if cfg.TokenThreshold != 4100 {
        t.Errorf("expected TokenThreshold 4100, got %d", cfg.TokenThreshold)
    }
})
```

**New test case to add** (append inside `TestLoad`, after line 122):
```go
t.Run("model and endpoint are empty when no config or env set", func(t *testing.T) {
    t.Setenv("MYHELPER_MODEL", "")
    t.Setenv("MYHELPER_ENDPOINT", "")
    t.Setenv("MYHELPER_TOKEN_LIMIT", "")

    dir := t.TempDir()
    orig, err := os.Getwd()
    if err != nil {
        t.Fatal(err)
    }
    if err := os.Chdir(dir); err != nil {
        t.Fatal(err)
    }
    defer func() { _ = os.Chdir(orig) }()

    cfg := Load()
    if cfg.Model != "" {
        t.Errorf("expected empty Model, got %q (CFG-01)", cfg.Model)
    }
    if cfg.Endpoint != "" {
        t.Errorf("expected empty Endpoint, got %q (CFG-02)", cfg.Endpoint)
    }
})
```

**Key conventions to match:**
- `t.Setenv` for all three env vars (clears interference)
- `t.TempDir()` + `os.Chdir` + deferred restore (isolates from real config files)
- `t.Fatal` for setup errors, `t.Errorf` for assertion failures
- Requirement ID in the error string (e.g., `"(CFG-01)"`)

---

### `cmd/helpers.go` — add `validateConfig` helper (VAL-01 through VAL-05)

**Change type:** Pure addition — append to end of file.

**Import already present** (`helpers.go` line 14): `"github.com/bkohler93/myhelper/internal/config"` — no new imports needed.

**`fmt` import already present** (`helpers.go` line 6) — no new imports needed.

**Error pattern to follow** — existing error returns in `helpers.go` use `fmt.Errorf` with lowercase first letter and no trailing period (`helpers.go` line 125: `return fmt.Errorf("readline init: %w", err)`).

**New function to add** (append after line 321):
```go
// validateConfig returns a descriptive error when required config fields are missing.
// Callers should return this error from cobra RunE; cobra will print it to stderr.
// Satisfies VAL-01 through VAL-05.
func validateConfig(cfg config.Config) error {
    if cfg.Endpoint == "" && cfg.Model == "" {
        return fmt.Errorf("endpoint and model are not configured\nRun 'myhelper setup' to configure myhelper")
    }
    if cfg.Endpoint == "" {
        return fmt.Errorf("endpoint is not configured\nRun 'myhelper setup' to configure myhelper")
    }
    if cfg.Model == "" {
        return fmt.Errorf("model is not configured\nRun 'myhelper setup' to configure myhelper")
    }
    return nil
}
```

---

### `cmd/root.go` — add `validateConfig` call in `RunE` (VAL-01, VAL-02, VAL-04)

**Analog:** `cmd/inspect.go` `runInspect` — same `config.Load()` + `ApplyFlagOverrides` setup, same position for the guard.

**Current `RunE` open** (`root.go` lines 44–47):
```go
RunE: func(cmd *cobra.Command, args []string) error {
    cfg := config.Load()
    ApplyFlagOverrides(&cfg)
    searchCfg := search.LoadConfig()
```

**After change** — insert `validateConfig` immediately after `ApplyFlagOverrides`, before any Ollama or search work:
```go
RunE: func(cmd *cobra.Command, args []string) error {
    cfg := config.Load()
    ApplyFlagOverrides(&cfg)
    if err := validateConfig(cfg); err != nil {
        return err
    }
    searchCfg := search.LoadConfig()
```

**Also add to `init()` block** (or at `rootCmd` var declaration) — set silence flags to prevent double-printing (Pitfall 1 from RESEARCH.md):
```go
func init() {
    rootCmd.SilenceErrors = true
    rootCmd.SilenceUsage = true
    // ... existing flag registrations ...
}
```

**Error surfacing path already exists** (`root.go` lines 65–69):
```go
func Execute() {
    if err := rootCmd.Execute(); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}
```
With `SilenceErrors = true`, cobra suppresses its own "Error: ..." print; `Execute()` produces the sole output line.

---

### `cmd/inspect.go` — add `validateConfig` call in `runInspect` (VAL-03)

**Analog:** `cmd/root.go` `RunE` — identical setup sequence: `Load()` → `ApplyFlagOverrides` → guard → proceed.

**Current `runInspect` open** (`inspect.go` lines 25–30):
```go
func runInspect(cmd *cobra.Command, args []string) error {
    query := args[0]
    cfg := config.Load()
    ApplyFlagOverrides(&cfg)
    searchCfg := search.LoadConfig()
```

**After change** — insert `validateConfig` immediately after `ApplyFlagOverrides`, before the `searchSuppress` check and any Ollama calls:
```go
func runInspect(cmd *cobra.Command, args []string) error {
    query := args[0]
    cfg := config.Load()
    ApplyFlagOverrides(&cfg)
    if err := validateConfig(cfg); err != nil {
        return err
    }
    searchCfg := search.LoadConfig()
```

**No import changes needed** — `validateConfig` is in the same `cmd` package.

---

### `cmd/helpers_test.go` — add `TestValidateConfig` (VAL-01 through VAL-05)

**Analog:** `helpers_test.go` lines 50–65 — subtest structure with `config.Config{}` literal (zero-value) passed directly, no `config.Load()` call, no chdir needed.

**Existing test pattern to copy** (`helpers_test.go` lines 51–65):
```go
t.Run("quit exits immediately with nil, no model call", func(t *testing.T) {
    fs := &fakeStream{}
    hist := history.New(4000, nil)
    restore := replaceStdin(t, "quit\n")
    defer restore()

    err := runConversationLoop(config.Config{}, hist, fs.call, "", "", nil)
    if err != nil {
        t.Fatalf("expected nil error, got %v", err)
    }
    if fs.called != 0 {
        t.Fatalf("expected streamFn not called, called %d times", fs.called)
    }
})
```

**New test function to add** (append after line 236):
```go
func TestValidateConfig(t *testing.T) {
    t.Run("returns nil when both endpoint and model are set", func(t *testing.T) {
        cfg := config.Config{Endpoint: "localhost:11434", Model: "qwen2.5-coder:7b"}
        if err := validateConfig(cfg); err != nil {
            t.Errorf("expected nil, got %v (VAL-05)", err)
        }
    })

    t.Run("returns error when model is empty", func(t *testing.T) {
        cfg := config.Config{Endpoint: "localhost:11434", Model: ""}
        err := validateConfig(cfg)
        if err == nil {
            t.Fatal("expected error for empty model, got nil (VAL-01)")
        }
        if !strings.Contains(err.Error(), "myhelper setup") {
            t.Errorf("error missing 'myhelper setup' hint: %v (VAL-01)", err)
        }
    })

    t.Run("returns error when endpoint is empty", func(t *testing.T) {
        cfg := config.Config{Endpoint: "", Model: "qwen2.5-coder:7b"}
        err := validateConfig(cfg)
        if err == nil {
            t.Fatal("expected error for empty endpoint, got nil (VAL-02)")
        }
        if !strings.Contains(err.Error(), "myhelper setup") {
            t.Errorf("error missing 'myhelper setup' hint: %v (VAL-02)", err)
        }
    })

    t.Run("returns combined error when both endpoint and model are empty", func(t *testing.T) {
        cfg := config.Config{}
        err := validateConfig(cfg)
        if err == nil {
            t.Fatal("expected error for empty endpoint and model, got nil (VAL-01, VAL-02)")
        }
        if !strings.Contains(err.Error(), "myhelper setup") {
            t.Errorf("error missing 'myhelper setup' hint: %v", err)
        }
    })
}
```

**Import addition needed** — `"strings"` must be added to the `helpers_test.go` import block (not currently present; check before adding).

---

## Shared Patterns

### Config load + flag override sequence
**Source:** `cmd/root.go` lines 45–46 and `cmd/inspect.go` lines 27–28
**Apply to:** Both `rootCmd.RunE` and `runInspect`
```go
cfg := config.Load()
ApplyFlagOverrides(&cfg)
```
The new `validateConfig(cfg)` call always goes immediately after `ApplyFlagOverrides`, before any other work.

### Error return from cobra RunE
**Source:** `cmd/inspect.go` lines 48–51 and `cmd/root.go` lines 65–69
**Apply to:** All new validation call sites
```go
if err := validateConfig(cfg); err != nil {
    return err
}
```
Cobra's `RunE` propagates the returned error to `Execute()`, which calls `fmt.Fprintln(os.Stderr, err)` and `os.Exit(1)`. With `SilenceErrors = true` set, cobra's internal "Error: ..." print is suppressed and the user sees only one line.

### Test subtest structure
**Source:** `cmd/helpers_test.go` lines 52–65 and `internal/config/config_test.go` lines 11–32
**Apply to:** All new test cases
- Use `t.Run(description, func(t *testing.T) { ... })` for every case
- Use `t.Fatal` for setup/precondition failures
- Use `t.Errorf` for assertion failures (test continues)
- Include requirement ID in error strings (e.g., `"(VAL-01)"`, `"(CFG-01)"`)
- Pass `config.Config{}` literal directly in `cmd/` tests — do not call `config.Load()`

---

## No Analog Found

None — all six files have clear analogs or are self-analogous (surgical edits to existing files).

---

## Metadata

**Analog search scope:** `internal/config/`, `cmd/`
**Files read:** 7 (`config.go`, `config_test.go`, `helpers.go`, `helpers_test.go`, `root.go`, `inspect.go`, plus CONTEXT.md and RESEARCH.md)
**Pattern extraction date:** 2026-05-10
