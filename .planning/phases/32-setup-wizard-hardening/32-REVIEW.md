---
phase: 32-setup-wizard-hardening
reviewed: 2026-05-10T00:00:00Z
depth: standard
files_reviewed: 2
files_reviewed_list:
  - internal/wizard/wizard.go
  - internal/wizard/wizard_test.go
findings:
  critical: 2
  warning: 2
  info: 1
  total: 5
status: issues_found
---

# Phase 32: Code Review Report

**Reviewed:** 2026-05-10
**Depth:** standard
**Files Reviewed:** 2
**Status:** issues_found

## Summary

The wizard implementation is well-structured overall — it uses injectable I/O, bounded HTTP clients, validates HTTP status before consuming NDJSON, and confirms pull success via a sentinel line. However, there are two correctness defects that will cause silent failures in the most important user-facing scenario (remote Ollama): the model pull uses the hardcoded default URL rather than the endpoint the user just entered, and the reachability check fires against localhost before the user can provide a custom endpoint — making the wizard completely unusable for any non-localhost Ollama setup.

---

## Critical Issues

### CR-01: `pullModel` ignores the user-supplied endpoint — always posts to `ollamaBaseURL`

**File:** `internal/wizard/wizard.go:105` and `:282`

**Issue:** Stage 1.5 prompts the user for their Ollama endpoint, stores it in `endpointValue` (line 85), and saves it to config (line 88), but `pullModel` is called at line 105 without passing the endpoint. Inside `pullModel`, the pull POST goes to `ollamaBaseURL+"/api/pull"` (line 282), which is still the package-level default (`http://localhost:11434`) unless overridden by a test. Any user whose Ollama runs on a remote host or non-default port will see a connection error or pull against the wrong server, even though their endpoint was just validated and saved correctly.

**Fix:** Pass `endpointValue` to `pullModel` and use it for the POST:

```go
// wizard.go — update signature
func pullModel(name string, endpoint string, w io.Writer) error {
    body, _ := json.Marshal(pullRequest{Name: name, Stream: true})
    resp, err := pullHTTPClient.Post(endpoint+"/api/pull", "application/json", bytes.NewReader(body))
    // ...
}

// Run() — pass endpointValue at the call site
if err := pullModel(model, endpointValue, w); err != nil {
```

The test `TestPullModel` already injects `ollamaBaseURL = srv.URL` before calling `pullModel` directly, so it does not catch this regression — it needs to be updated to pass an explicit endpoint argument.

---

### CR-02: Reachability check (`checkOllama`) fires against `ollamaBaseURL` (localhost) before the user can provide a custom endpoint

**File:** `internal/wizard/wizard.go:66` and `:177`

**Issue:** `checkOllama()` is called at Stage 1 (line 66) using the package-level `ollamaBaseURL`, which defaults to `http://localhost:11434`. The endpoint prompt (Stage 1.5) is not reached until after the check succeeds. Any user whose Ollama is bound to a non-localhost address (e.g., `http://192.168.0.9:11434`) will see "Ollama is not running" and the wizard will exit — they have no opportunity to specify the correct endpoint. This makes the wizard entirely non-functional for the documented remote-Ollama configuration.

**Fix:** Move the endpoint prompt before the reachability check (swap Stage 1 and Stage 1.5), so `checkOllama` can be called with the user's endpoint:

```go
// 1. Prompt for endpoint first (or read from existing config as default).
// 2. Set/update ollamaBaseURL to the resolved endpoint.
// 3. Call checkOllama() using the now-correct URL.
```

Alternatively, pass the endpoint explicitly rather than relying on a package-level var:

```go
func checkOllama(endpoint string) bool {
    resp, err := ollamaHTTPClient.Get(endpoint + "/")
    // ...
}
```

---

## Warnings

### WR-01: Endpoint validation accepts bare schemes (`http://`, `https://`) as valid

**File:** `internal/wizard/wizard.go:81-83` and `:156-158`

**Issue:** Both the Ollama endpoint loop (line 81) and the SearXNG endpoint guard (line 156) accept any string with an `http://` or `https://` prefix, including `http://` alone (no host). A bare scheme passes `strings.HasPrefix` and will be saved to config, producing an unparseable URL that causes a silent network failure on every subsequent use.

**Fix:** Use `net/url.Parse` to require a non-empty host:

```go
import "net/url"

u, err := url.Parse(line)
if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
    fmt.Fprintf(w, "Endpoint must be a valid http:// or https:// URL with a host. Try again.\n")
    continue
}
```

Apply the same check at line 156 for the SearXNG endpoint (currently it silently skips rather than re-prompting, so an invalid but prefix-passing value like `http://` would be saved).

---

### WR-02: `TestCheckOllama` leaves `ollamaBaseURL` in a dirty state mid-test with no per-mutation cleanup

**File:** `internal/wizard/wizard_test.go:30`

**Issue:** `TestCheckOllama` registers a single `t.Cleanup` (line 23) that restores `ollamaBaseURL` to `http://localhost:11434`. It then mutates `ollamaBaseURL` a second time at line 30 to `http://127.0.0.1:19999`. The registered cleanup fires at end-of-test and does restore the value, so within a single sequential test run this is benign. However, if `-race` or `go test -parallel` is ever introduced at the file level, the unprotected write to a package-level var between two sub-assertions inside a single test function is a data race. More concretely: if the failing-case path is extracted into a subtest with `t.Run`, the cleanup would fire too early for the second mutation.

**Fix:** Register a second cleanup immediately after the second mutation, or restructure as two `t.Run` subtests each with their own setup/cleanup:

```go
ollamaBaseURL = "http://127.0.0.1:19999"
t.Cleanup(func() { ollamaBaseURL = "http://localhost:11434" })
if checkOllama() {
    t.Error("expected checkOllama() false on connection refused")
}
```

---

## Info

### IN-01: `TestRecommendModel` does not test exact boundary values

**File:** `internal/wizard/wizard_test.go:50-63`

**Issue:** The test cases use values well inside each tier (`30*1024`, `14*1024`, `7*1024`, `2*1024`) but never test the exact boundary values (`24*1024`, `12*1024`, `6*1024`). If the `>=` comparisons were accidentally changed to `>`, the tests would not catch it.

**Fix:** Add cases for the exact boundaries:

```go
{24 * 1024, "qwen2.5-coder:14b"},
{12 * 1024, "qwen2.5-coder:7b"},
{6 * 1024,  "llama3.2:3b"},
{0,         "llama3.2:1b"},
```

---

_Reviewed: 2026-05-10_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
