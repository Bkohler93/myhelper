---
phase: 31
slug: config-loading-startup-validation
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-05-10
---

# Phase 31 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go standard `testing` package |
| **Config file** | none (uses `go test ./...`) |
| **Quick run command** | `go test ./internal/config/... ./cmd/...` |
| **Full suite command** | `go test ./...` |
| **Estimated runtime** | ~5 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/config/... ./cmd/...`
- **After every plan wave:** Run `go test ./...`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** ~5 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------------|-----------|-------------------|-------------|--------|
| 31-01-01 | 01 | 1 | CFG-01, CFG-02 | Load() returns empty string for model and endpoint when unset | unit | `go test ./internal/config/... -run TestLoad -v` | ✅ (file exists, test cases missing) | ⬜ pending |
| 31-01-02 | 01 | 1 | CFG-01, CFG-02 | New TestLoad subtests verify empty Model and Endpoint | unit | `go test ./internal/config/... -run TestLoad -v` | ✅ (extend existing) | ⬜ pending |
| 31-02-01 | 02 | 2 | VAL-01, VAL-02, VAL-03, VAL-04, VAL-05 | validateConfig returns error with setup hint when model or endpoint empty | unit | `go test ./cmd/... -run TestValidateConfig -v` | ❌ W0 | ⬜ pending |
| 31-02-02 | 02 | 2 | VAL-01, VAL-02, VAL-03, VAL-04 | validateConfig call site in rootCmd.RunE and runInspect; SilenceErrors set | unit | `go build ./... && grep -c "validateConfig" cmd/root.go cmd/inspect.go` | ✅ (call sites added by task) | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `cmd/helpers_test.go` — add `TestValidateConfig` test cases covering VAL-01 through VAL-05 (file exists — extend it)

*Existing test infrastructure covers all other phase requirements — no new test files or framework install needed.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Error appears exactly once on missing config (SilenceErrors suppresses cobra duplicate) | VAL-01, VAL-02 | Requires TTY/real invocation to confirm stderr output count | Run `MYHELPER_MODEL="" MYHELPER_ENDPOINT="" myhelper 2>&1 \| grep -c "myhelper setup"` — expect output `1` |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 10s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
