# Phase 31: Config Loading & Startup Validation - Research

**Researched:** 2026-05-10
**Domain:** Go CLI config loading, startup validation, cobra RunE error handling
**Confidence:** HIGH

## Summary

Phase 31 is a focused, low-risk refactor with a small blast radius: two files need substantive changes (`internal/config/config.go` and `cmd/helpers.go`) and three command entry points need one validation call each (`cmd/root.go`, `cmd/inspect.go`, `cmd/search.go`). The current `Load()` function pre-seeds `cfg.Endpoint` and `cfg.Model` with hardcoded defaults before applying file and env overrides — removing those two pre-seeds is the complete CFG-01/CFG-02 change. The validation logic (VAL-01 through VAL-05) is a pure addition: a `validateConfig` helper in `cmd/helpers.go` that checks for empty strings and returns a formatted error if either field is unset.

No new dependencies are required. The existing error-surfacing path (cobra `RunE` returns an error, which `Execute()` prints to stderr and exits 1) is the correct place for validation errors to land. The validation message itself needs a consistent format including a "run myhelper setup" hint, and must appear before any Ollama call is made in all three commands.

The primary risk is the existing test suite: `config_test.go` currently does NOT assert that `cfg.Endpoint` and `cfg.Model` are empty when no config or env is present — those tests only cover `TokenThreshold`. After removing the hardcoded defaults, the existing tests will continue to pass but new tests are needed to cover CFG-01 and CFG-02. Tests in `cmd/` that pass `config.Config{}` (zero value) to helpers will continue to work because they bypass the real `Load()` call entirely.

**Primary recommendation:** Remove the two default assignments in `Load()`, add a `validateConfig(cfg config.Config) error` helper in `cmd/helpers.go`, call it at the top of each command's RunE before any other work, and add unit tests for both the config package and the validation helper.

---

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
All implementation choices are at Claude's discretion — discuss phase was skipped per user setting.

Key constraints from STATE.md and REQUIREMENTS.md:
- Hard fail (not auto-redirect) on missing config — simpler, more predictable than silently launching setup
- Env vars (`MYHELPER_MODEL`, `MYHELPER_ENDPOINT`) count as "set" for validation purposes
- `myhelper config set` subcommand is out of scope
- Error format must be consistent across chat, inspect, and search commands
- `default_token_threshold` (4100) is an internal tuning param — retain its default, do not remove

### Claude's Discretion
All implementation choices delegated to Claude. Use ROADMAP phase goal, success criteria, and codebase conventions to guide decisions.

### Deferred Ideas (OUT OF SCOPE)
None — discuss phase skipped.
</user_constraints>

---

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| CFG-01 | Config loading returns empty string for model when not set in config or env (no hardcoded default) | Remove `Model: DefaultModel` from `Load()` initial struct literal; `DefaultModel` const can remain for other uses (none currently) or be deleted |
| CFG-02 | Config loading returns empty string for endpoint when not set in config or env (no hardcoded default) | Remove `Endpoint: DefaultEndpoint` from `Load()` initial struct literal; same approach as CFG-01 |
| VAL-01 | User sees a clear error with "run myhelper setup" hint when running `chat` with no model configured | Add `validateConfig(cfg)` call at top of `rootCmd.RunE` before history/Ollama work |
| VAL-02 | User sees a clear error with "run myhelper setup" hint when running `chat` with no endpoint configured | Same call — `validateConfig` checks both fields and returns on first failure (or combines them) |
| VAL-03 | `inspect` validates model and endpoint before executing — same error format as chat | Add `validateConfig(cfg)` call at top of `runInspect` before any Ollama calls |
| VAL-04 | `search` validates model and endpoint before executing — same error format as chat | `buildUserMessage` in `search.go` calls Ollama indirectly via `searchGate`; validate in the root command RunE before calling `buildUserMessage`, not inside `buildUserMessage` itself |
| VAL-05 | Setting `MYHELPER_MODEL` and `MYHELPER_ENDPOINT` env vars satisfies validation (no error) | Env var resolution already happens in `Load()` before `validateConfig` is called — no extra work needed |
</phase_requirements>

---

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Remove hardcoded defaults | Config package (`internal/config`) | — | Config package owns all loading logic; defaults are set there |
| Validation logic | CMD layer (`cmd/helpers.go`) | — | Commands decide what constitutes "runnable state"; config package just loads |
| Error message formatting | CMD layer | — | Message is user-facing CLI output, not a concern of the config package |
| Env var resolution | Config package (`internal/config`) | — | Already implemented correctly; no changes needed |

---

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `encoding/json` | stdlib | Config file parsing | Already used in `loadFile` |
| `os` | stdlib | Env var reads, file reads | Already used throughout |
| `fmt` | stdlib | Error message formatting | Project convention for error strings |
| `github.com/spf13/cobra` | already in go.mod | CLI command wiring, RunE error return | Project uses cobra throughout |

No new dependencies required for this phase.

**Installation:** None needed.

---

## Architecture Patterns

### System Architecture Diagram

```
myhelper <command> invoked
        |
        v
cobra dispatches to RunE
        |
        v
config.Load()          <-- reads env vars, then config file, then returns (no defaults for model/endpoint)
        |
        v
validateConfig(cfg)    <-- new: returns error if model or endpoint is ""
        |
   [error?]
   YES: cobra prints error to stderr, exits 1
   NO: continue to Ollama calls
```

### Recommended Project Structure

No structural changes. Changes are:

```
internal/config/config.go   # remove DefaultEndpoint and DefaultModel from Load() struct literal
cmd/helpers.go              # add validateConfig helper
cmd/root.go                 # call validateConfig after Load()
cmd/inspect.go              # call validateConfig after Load()
internal/config/config_test.go   # add tests for CFG-01 and CFG-02
cmd/helpers_test.go (or new cmd/validate_test.go)  # add tests for validateConfig
```

### Pattern 1: Removing Hardcoded Defaults (CFG-01, CFG-02)

**What:** Strip the `Endpoint` and `Model` fields from the initial `Config{}` literal in `Load()`. `TokenThreshold` retains its default of `DefaultTokenThreshold`.

**When to use:** This is the only change needed in `internal/config/config.go`.

**Example (before):**
```go
// Source: internal/config/config.go (current)
cfg := Config{
    Endpoint:       DefaultEndpoint,
    Model:          DefaultModel,
    TokenThreshold: DefaultTokenThreshold,
}
```

**Example (after):**
```go
// Source: internal/config/config.go (proposed)
cfg := Config{
    TokenThreshold: DefaultTokenThreshold,
}
```

The `DefaultEndpoint` and `DefaultModel` constants can be deleted (no callers outside `Load()`) — or retained as documentation. Deleting them removes dead code; the choice is Claude's discretion.

### Pattern 2: validateConfig Helper (VAL-01 through VAL-05)

**What:** A single exported (or unexported) function in `cmd/helpers.go` that returns a formatted `error` when model or endpoint is empty. Commands call it immediately after `config.Load()`.

**When to use:** Called in `rootCmd.RunE`, `runInspect`, and anywhere in the search path before Ollama is contacted.

**Example:**
```go
// Source: cmd/helpers.go (proposed addition)
// validateConfig returns a descriptive error when required config fields are missing.
// Callers should return this error from cobra RunE; cobra will print it to stderr.
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

**Placement in root.go (chat command):**
```go
// Source: cmd/root.go RunE (proposed)
RunE: func(cmd *cobra.Command, args []string) error {
    cfg := config.Load()
    ApplyFlagOverrides(&cfg)
    if err := validateConfig(cfg); err != nil {
        return err
    }
    // ... rest of existing code
```

**Placement in inspect.go:**
```go
// Source: cmd/inspect.go runInspect (proposed)
func runInspect(cmd *cobra.Command, args []string) error {
    query := args[0]
    cfg := config.Load()
    ApplyFlagOverrides(&cfg)
    if err := validateConfig(cfg); err != nil {
        return err
    }
    // ... rest of existing code
```

**Note on search command (VAL-04):** There is no `cmd/search.go` with a `RunE`. The search functionality runs inside `rootCmd.RunE` (the chat command). `buildUserMessage` in `search.go` calls Ollama through `searchGate` and `reRankResults`. The validation in `rootCmd.RunE` covers all search paths because search only executes as part of the chat command flow. `inspect.go` also calls search helpers (`searchGate`, `reRankResults`, `buildWebBlock`) and needs its own `validateConfig` call, which is VAL-03/VAL-04.

### Pattern 3: Error Return via cobra RunE

**What:** Return the error from `validateConfig` directly from `RunE`. Cobra propagates it to `Execute()`, which calls `fmt.Fprintln(os.Stderr, err)` and `os.Exit(1)`.

**Why this is correct:** The project already uses this exact pattern. `cmd/root.go` `Execute()`:
```go
// Source: cmd/root.go (existing)
func Execute() {
    if err := rootCmd.Execute(); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}
```

Cobra also prints the error internally before `Execute()` returns. To suppress the duplicate print (cobra prints "Error: <msg>" and Execute also prints), the standard pattern is to silence cobra's auto-print:

```go
// Option A: suppress cobra's built-in error print (recommended for clean output)
rootCmd.SilenceErrors = true
rootCmd.SilenceUsage = true
```

Or accept cobra's default behavior (two prints). The project currently does not set `SilenceErrors`, so the current behavior on any returned error is cobra prints "Error: <msg>" then Execute prints the bare error. For the validation error, this means the user sees the message twice. The planner should decide: set `SilenceErrors = true` (cleaner) or accept the duplicate. Either is valid. This is flagged as a discretion decision.

### Anti-Patterns to Avoid

- **Validating inside `config.Load()`:** `Load()` is a pure data loader. Returning an error from `Load()` would change its signature (currently returns `Config`, not `(Config, error)`) and break all callers. Validation belongs in the command layer, not the config layer.
- **Auto-redirect to setup:** Explicitly out of scope per CONTEXT.md. Do not call `wizard.Run()` on validation failure.
- **Removing `DefaultTokenThreshold`:** Explicitly out of scope. The 4100 constant must remain and continue to be used.
- **Validating inside `buildUserMessage` or `searchGate`:** These are inner helpers that already fail gracefully (they skip search on Ollama errors). Validation at that level is too late and breaks the "same error format across commands" requirement.

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Error exit | Custom `os.Exit` calls in RunE | Return `error` from `RunE` | Cobra handles exit; existing `Execute()` already prints and exits |
| Multi-field validation accumulation | Loop over fields building error slices | Simple sequential `if` checks | Two fields only; a slice is over-engineering |

---

## Runtime State Inventory

Phase 31 is not a rename/refactor/migration phase. No runtime state audit is required.

None — omitted per section trigger condition (greenfield validation change, not a rename).

---

## Environment Availability

Phase 31 is pure Go code changes. No external tools, databases, or services are required beyond what is already present.

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go toolchain | `go test ./...` | Yes (project already builds) | — | — |

No missing dependencies.

---

## Common Pitfalls

### Pitfall 1: Cobra Double-Prints the Error

**What goes wrong:** When `RunE` returns an error, cobra prints `Error: <message>` to stderr. Then `Execute()` also calls `fmt.Fprintln(os.Stderr, err)`. The user sees the validation message twice.

**Why it happens:** The project's `Execute()` was written to catch errors that cobra does NOT print (e.g., when `SilenceErrors = true`). Without `SilenceErrors`, both cobra and `Execute()` print.

**How to avoid:** Set `rootCmd.SilenceErrors = true` and `rootCmd.SilenceUsage = true` in `init()` or at var declaration time. This suppresses cobra's auto-print and leaves the single `fmt.Fprintln` in `Execute()` as the only output. Alternatively, remove the `fmt.Fprintln` from `Execute()` and rely solely on cobra's print.

**Warning signs:** Running the command with missing config shows the error message twice.

### Pitfall 2: Existing Tests That Assert Default Values

**What goes wrong:** `config_test.go` tests do `t.Setenv("MYHELPER_ENDPOINT", "")` and `t.Setenv("MYHELPER_MODEL", "")` to clear env vars — but they do NOT currently assert that `cfg.Endpoint` and `cfg.Model` are empty. After removing the hardcoded defaults, the existing tests still pass. However, if any test was accidentally relying on `cfg.Endpoint` being non-empty (e.g., in `chat_test.go`), it would break.

**Why it happens:** `chat_test.go` creates `config.Config{}` directly (zero value) — it never calls `config.Load()`. So it is unaffected by removing defaults from `Load()`.

**How to avoid:** Grep for any test or non-test code that calls `config.Load()` and then uses `cfg.Endpoint` or `cfg.Model` as if non-empty. Only `config_test.go` calls `Load()` in tests, and those tests only check `TokenThreshold`.

**Warning signs:** `go test ./...` failures mentioning empty endpoint/model where non-empty was expected.

### Pitfall 3: VAL-04 "search" Command Scope Confusion

**What goes wrong:** CONTEXT.md and REQUIREMENTS.md list `search` as a command that needs validation. There is no `cmd/search.go` with a `RunE` — there is a `search.go` that contains helpers (`buildUserMessage`, `searchGate`, etc.) but no cobra command. Search runs as part of the root chat command.

**Why it happens:** The codebase had a period of evolution where search commands were separate, but the current architecture folds search into the chat REPL. The CONTEXT.md reference to `cmd/search.go` refers to the helpers file, not a command.

**How to avoid:** VAL-04 is satisfied by the `validateConfig` call in `rootCmd.RunE` (chat), since all search calls happen within that flow. VAL-03 is separately satisfied by the `validateConfig` call in `runInspect`. No third command needs a validation call.

### Pitfall 4: Forgetting to Update `DefaultEndpoint`/`DefaultModel` Constants

**What goes wrong:** The constants `DefaultEndpoint` and `DefaultModel` remain in `config.go` after removing them from `Load()`. They are dead code. Not a runtime bug, but confusing — someone might re-add them to `Load()` thinking they are needed.

**Why it happens:** Refactors often leave orphaned constants.

**How to avoid:** Delete both constants in the same commit that removes them from `Load()`. Run `go build ./...` to verify no other code references them. (`DefaultTokenThreshold` must be retained.)

---

## Code Examples

### CFG-01/CFG-02: Updated `Load()` function

```go
// Source: internal/config/config.go (after change)
func Load() Config {
    cfg := Config{
        // Endpoint and Model intentionally omitted — empty string until set via
        // config file or env var. Commands validate non-empty before use.
        TokenThreshold: DefaultTokenThreshold,
    }
    // ... rest unchanged
}
```

### VAL-01 through VAL-05: `validateConfig` in helpers.go

```go
// Source: cmd/helpers.go (new addition)
// validateConfig returns an error when required config fields are missing.
// Returns nil when both endpoint and model are non-empty.
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

### Config test pattern (matching existing test style)

```go
// Source: internal/config/config_test.go (new test, matching existing style)
t.Run("model is empty when no config or env", func(t *testing.T) {
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

---

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Hardcoded defaults baked into Load() | No defaults for model/endpoint; empty string means "not configured" | Phase 31 | Commands must validate before use |

---

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | There is no `cmd/chat.go` — the chat command is `rootCmd` in `cmd/root.go` | Architecture Patterns, VAL-04 note | Low: verified by reading all files in cmd/ |
| A2 | `DefaultEndpoint` and `DefaultModel` have no callers outside `Load()` | Pitfall 4 | Low: verified by reading all cmd/ and config/ files |
| A3 | Cobra's default `SilenceErrors=false` causes double-printing when RunE returns an error | Pitfall 1 | Low: standard cobra behavior, project uses default cobra init |

All three assumptions were verified by direct file inspection in this session and carry LOW risk.

---

## Open Questions (RESOLVED)

1. **Double-print behavior: fix or accept?**
   - RESOLVED: Set `rootCmd.SilenceErrors = true` and `rootCmd.SilenceUsage = true` in `init()`. Standard cobra pattern for clean CLIs. Implemented in Plan 02 Task 2 Change 1.

2. **Combined vs. individual error messages for missing both fields?**
   - RESOLVED: Show a single combined message when both are missing (`"endpoint and model are not configured\nRun 'myhelper setup' to configure myhelper"`). Implemented in Plan 02 Task 1.

3. **Delete or retain `DefaultEndpoint`/`DefaultModel` constants?**
   - RESOLVED: Delete both constants — no callers outside `Load()`, Phase 32 does not need them. Implemented in Plan 01 Task 1.

---

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Framework | Go standard `testing` package |
| Config file | none (uses `go test ./...`) |
| Quick run command | `go test ./internal/config/... ./cmd/...` |
| Full suite command | `go test ./...` |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| CFG-01 | `Load()` returns empty Model when no config or env set | unit | `go test ./internal/config/... -run TestLoad` | Partial (file exists, test case missing) |
| CFG-02 | `Load()` returns empty Endpoint when no config or env set | unit | `go test ./internal/config/... -run TestLoad` | Partial (file exists, test case missing) |
| VAL-01 | `validateConfig` returns error with "myhelper setup" hint when Model is empty | unit | `go test ./cmd/... -run TestValidateConfig` | No — Wave 0 |
| VAL-02 | `validateConfig` returns error with "myhelper setup" hint when Endpoint is empty | unit | `go test ./cmd/... -run TestValidateConfig` | No — Wave 0 |
| VAL-03 | inspect calls `validateConfig` before Ollama — error returned correctly | unit | `go test ./cmd/... -run TestValidateConfig` | No — Wave 0 |
| VAL-04 | chat (root command) calls `validateConfig` before search/Ollama — error returned correctly | unit | `go test ./cmd/... -run TestValidateConfig` | No — Wave 0 |
| VAL-05 | Env vars `MYHELPER_MODEL`/`MYHELPER_ENDPOINT` satisfy validation | unit | `go test ./internal/config/... -run TestLoad` + `go test ./cmd/... -run TestValidateConfig` | No — Wave 0 |

### Sampling Rate

- **Per task commit:** `go test ./internal/config/... ./cmd/...`
- **Per wave merge:** `go test ./...`
- **Phase gate:** `go test ./...` green before `/gsd-verify-work`

### Wave 0 Gaps

- [ ] `cmd/helpers_test.go` (exists) — add `TestValidateConfig` test cases covering VAL-01 through VAL-05
- [ ] `internal/config/config_test.go` (exists) — add test cases for empty Model (CFG-01) and empty Endpoint (CFG-02) within the existing `TestLoad` table

*(No new test files needed — extend existing ones.)*

---

## Security Domain

Phase 31 involves no authentication, sessions, cryptography, or external data handling. Config validation is purely structural (empty string check). No ASVS categories apply.

---

## Sources

### Primary (HIGH confidence)
- Direct file inspection: `internal/config/config.go` — verified current Load() implementation, constant values, struct literal
- Direct file inspection: `cmd/root.go` — verified rootCmd RunE structure, Execute() error handling
- Direct file inspection: `cmd/inspect.go` — verified runInspect structure, placement of config.Load() call
- Direct file inspection: `cmd/search.go` — verified no RunE, helpers only
- Direct file inspection: `cmd/helpers.go` — verified error patterns, fakeStream seam for tests
- Direct file inspection: `internal/config/config_test.go` — verified existing test style, chdir pattern, t.Setenv usage
- Direct file inspection: `cmd/chat_test.go`, `cmd/helpers_test.go`, `cmd/search_gate_test.go` — verified test patterns
- Direct file inspection: `.planning/config.json` — confirmed `nyquist_validation: true`

### Secondary (MEDIUM confidence)
- Cobra documentation (training knowledge): RunE error propagation, SilenceErrors behavior [ASSUMED — standard cobra behavior verified by reading project code that uses it correctly]

---

## Metadata

**Confidence breakdown:**
- Config change (CFG-01/CFG-02): HIGH — change is a two-line deletion in a well-understood function
- Validation helper (VAL-01 through VAL-05): HIGH — pure addition, no existing code modified except call sites
- Test gaps: HIGH — existing test style is consistent and well-understood
- Cobra SilenceErrors: MEDIUM — standard behavior, not explicitly verified against cobra source

**Research date:** 2026-05-10
**Valid until:** 2026-06-10 (stable Go standard library + cobra; no fast-moving dependencies)
