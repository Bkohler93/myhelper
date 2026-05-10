---
phase: 32-setup-wizard-hardening
verified: 2026-05-10T17:00:00Z
status: passed
score: 6/6 must-haves verified
overrides_applied: 0
---

# Phase 32: Setup Wizard Hardening Verification Report

**Phase Goal:** The setup wizard is guaranteed to write a usable model and endpoint to config before it exits — no more silent exit leaving the user with an incomplete config
**Verified:** 2026-05-10
**Status:** passed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | After any path through `myhelper setup`, the written config always contains a non-empty model field | VERIFIED | `mergeHomeConfig(map[string]interface{}{"model": model})` at line 116; fallback path at line 135; `pullSucceeded` flag gates the fallback — model is written on pull success (line 116) or fallback success (line 135); error returned if both fallback attempts are empty (line 133) |
| 2 | After any path through `myhelper setup`, the written config always contains a non-empty endpoint field | VERIFIED | `mergeHomeConfig(map[string]interface{}{"endpoint": endpointValue})` at line 87; loop at lines 70–84 enforces a valid URL before the write; endpoint is written before `checkOllama()` fires |
| 3 | When the user declines or fails to pull the recommended model, the wizard asks for an existing local model name | VERIFIED | `if !pullSucceeded` branch at line 123; prompt `"Enter the name of a local model (run 'ollama list' to see available): "` at line 124; tested by `TestRun_SkipModel_FallbackWritesModel` and `TestRun_PullFail_FallbackWritesModel` |
| 4 | The wizard loops and re-prompts (once) when the user enters an empty model name at the fallback prompt | VERIFIED | Lines 127–130: `if modelName == "" { prompt again; re-read }` — exactly one re-prompt on empty; tested by `TestRun_SkipModel_EmptyThenProvided` |
| 5 | The wizard returns an error (does not write config) if the user enters an empty model name twice | VERIFIED | Line 132–133: `if modelName == "" { return fmt.Errorf("no model name provided — setup incomplete") }` — returns error before any `mergeHomeConfig` call on model; tested by `TestRun_SkipModel_EmptyTwice` |
| 6 | The wizard loops and re-prompts when the user enters an invalid endpoint (empty or no http/https prefix) | VERIFIED | Lines 70–84: loop with `url.Parse` validation requiring non-empty Host and http/https scheme; empty input accepted as default (line 74–76); tested by `TestRun_EndpointPrompt_InvalidThenValid` |

**Score:** 6/6 truths verified

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `internal/wizard/wizard.go` | Stage 1.5 endpoint prompt + Stage 3 skip-model fallback | VERIFIED | 372 lines; Stage 1 endpoint loop (lines 67–90); Stage 3 fallback block (lines 122–140); `pullSucceeded` flag (line 108); `mergeHomeConfig` called with `"endpoint"` (line 87) and `"model"` (lines 116, 135) |
| `internal/wizard/wizard_test.go` | Tests for WIZ-01, WIZ-02, WIZ-03 including `TestRun_EndpointPrompt*` | VERIFIED | 412 lines; all 7 new tests present and passing; `TestRun_SkipAll` updated to new input sequence |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| `wizard.go` Stage 1 endpoint loop | `mergeHomeConfig` | `mergeHomeConfig(map[string]interface{}{"endpoint": endpointValue})` | WIRED | Line 87 — called unconditionally after loop exits with valid URL |
| `wizard.go` Stage 3 fallback | `mergeHomeConfig` | `mergeHomeConfig(map[string]interface{}{"model": modelName})` | WIRED | Line 135 — called when fallback model name is non-empty |

### Data-Flow Trace (Level 4)

Not applicable — wizard.go is an interactive CLI, not a data-rendering component. Config writes are direct `os.WriteFile` calls via `mergeHomeConfig`.

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| All 15 wizard tests pass | `go test ./internal/wizard/... -count=1` | 15/15 PASS in 11.2s | PASS |
| Full test suite passes (no regressions) | `go test ./...` | all packages ok | PASS |
| Build succeeds | `go build ./...` | exit 0 | PASS |
| Stage 1 endpoint prompt present | `grep -c "Ollama endpoint" wizard.go` | 2 (comment + prompt) | PASS |
| Fallback prompt present | `grep -c "Enter the name of a local model" wizard.go` | 1 | PASS |
| mergeHomeConfig called with "endpoint" | `grep -c '"endpoint"' wizard.go` | 1 | PASS |
| mergeHomeConfig called with "model" in fallback | `grep -c '"model"' wizard.go` | 2 (pull success + fallback) | PASS |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|------------|-------------|--------|----------|
| WIZ-01 | 32-01-PLAN.md | `myhelper setup` always writes a model to config before exiting | SATISFIED | `pullSucceeded` flag + fallback path + error return; `TestRun_SkipModel_FallbackWritesModel`, `TestRun_SkipModel_EmptyTwice` |
| WIZ-02 | 32-01-PLAN.md | When user skips recommended model pull, wizard prompts for an existing local model name before saving | SATISFIED | `if !pullSucceeded` block at line 123 with prompt text at line 124; `TestRun_SkipModel_FallbackWritesModel`, `TestRun_PullFail_FallbackWritesModel` |
| WIZ-03 | 32-01-PLAN.md | Wizard validates that endpoint is non-empty before writing config | SATISFIED | Loop at lines 70–84 using `url.Parse` — rejects empty, bare schemes, and non-http(s) prefixes; `TestRun_EndpointPrompt_InvalidThenValid`, `TestRun_EndpointPrompt_AcceptDefault` |

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| `wizard_test.go` | 264 | `TestRun_EndpointPrompt_CustomValue` supplies `http://192.168.0.9:11434` as custom endpoint; wizard updates `ollamaBaseURL` to that value and then calls `checkOllama()`, hitting a 5s HTTP timeout | Warning | Test takes 5s instead of ~0.1s; same issue in `TestRun_EndpointPrompt_InvalidThenValid`; tests pass correctly but are slow |

No blocker anti-patterns. No TODO/FIXME/placeholder comments in wizard.go. No stub return values. No empty handlers.

### Deviation from PLAN Specification

The PLAN specified placing the endpoint prompt as "Stage 1.5" (after the reachability check). The code review (32-REVIEW.md CR-02) correctly identified that this ordering was wrong — endpoint must come before the reachability check. The implementation places it as Stage 1, before `checkOllama()`. This is the correct fix and was already present in the committed code. The goal contract ("wizard always writes endpoint") is fully satisfied regardless of stage numbering.

### Human Verification Required

None. All behaviors are verifiable programmatically via the test suite. The wizard's interactive I/O is fully exercised via `strings.NewReader` injection in the tests.

### Gaps Summary

No gaps. All 6 observable truths are VERIFIED. All 3 requirements (WIZ-01, WIZ-02, WIZ-03) are SATISFIED. The full test suite passes. The only finding is a test performance warning (two tests take 5s each due to HTTP timeout against an unreachable IP), which does not affect correctness.

---

_Verified: 2026-05-10T17:00:00Z_
_Verifier: Claude (gsd-verifier)_
