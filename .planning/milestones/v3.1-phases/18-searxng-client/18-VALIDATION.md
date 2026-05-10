---
phase: 18
slug: searxng-client
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-04-11
---

# Phase 18 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test (stdlib) |
| **Config file** | none — `go test` discovers `*_test.go` files |
| **Quick run command** | `go test ./internal/search/...` |
| **Full suite command** | `go test ./...` |
| **Estimated runtime** | ~3 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/search/...`
- **After every plan wave:** Run `go test ./...`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 5 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 18-01-01 | 01 | 0 | SRCH-01,SRCH-02,SRCH-03,SRCH-04,SRCH-05 | — | N/A | unit | `go test ./internal/search/...` | ❌ W0 | ⬜ pending |
| 18-01-02 | 01 | 1 | SRCH-01,SRCH-02 | — | N/A | unit | `go test ./internal/search/... -run TestSearch` | ✅ | ⬜ pending |
| 18-01-03 | 01 | 1 | SRCH-03 | — | N/A | unit | `go test ./internal/search/... -run TestLoadConfig` | ✅ | ⬜ pending |
| 18-01-04 | 01 | 1 | SRCH-04 | — | N/A | unit | `go test ./internal/search/... -run TestSearch_RequestParams` | ✅ | ⬜ pending |
| 18-01-05 | 01 | 1 | SRCH-05 | — | N/A | unit | `go test ./internal/search/... -run TestSearch_Errors` | ✅ | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/search/search_test.go` — test stubs covering SRCH-01 through SRCH-05 (must compile before production code is written)

*Wave 0 creates the test file so the quick-run command is available from the first task commit.*

---

## Manual-Only Verifications

*All phase behaviors have automated verification.*

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 5s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
