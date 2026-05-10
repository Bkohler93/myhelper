---
phase: 19
slug: search-gate-and-injection
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-04-11
---

# Phase 19 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go testing (stdlib) |
| **Config file** | none — `go test` discovers `*_test.go` files |
| **Quick run command** | `go test ./cmd/... -run "TestSearch\|TestBuildWebBlock\|TestReRank\|TestBuildUserMessage"` |
| **Full suite command** | `go test ./...` |
| **Estimated runtime** | ~5 seconds |

---

## Sampling Rate

- **After every task commit:** Run quick run command above
- **After every plan wave:** Run `go test ./...`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 10 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 19-01-01 | 01 | 0 | GATE-01,GATE-02,GATE-03,GATE-04,RANK-01,RANK-02,RANK-03,INJ-01,INJ-02,INJ-03 | — | N/A | unit | `go build ./cmd/...` | ❌ W0 | ⬜ pending |
| 19-01-02 | 01 | 1 | GATE-01,GATE-02 | — | fail-closed gate | unit | `go test ./cmd/... -run TestSearchGate` | ✅ | ⬜ pending |
| 19-01-03 | 01 | 1 | RANK-01,RANK-02,RANK-03 | — | N/A | unit | `go test ./cmd/... -run TestReRankResults` | ✅ | ⬜ pending |
| 19-01-04 | 01 | 1 | INJ-01,INJ-02,INJ-03 | — | N/A | unit | `go test ./cmd/... -run TestBuildWebBlock` | ✅ | ⬜ pending |
| 19-01-05 | 01 | 1 | GATE-03,GATE-04 | — | N/A | unit | `go test ./cmd/... -run TestBuildUserMessage` | ✅ | ⬜ pending |
| 19-01-06 | 01 | 1 | GATE-01,GATE-02,GATE-03,GATE-04,RANK-01,RANK-02,RANK-03,INJ-01,INJ-02,INJ-03 | — | N/A | integration | `go test ./cmd/... && go build ./...` | ✅ | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `cmd/search_gate_test.go` — stubs for all 10 requirements (GATE-01 through INJ-03); must compile before production code is written

*Wave 0 creates the test file so the quick-run command is available from the first production task commit.*

---

## Manual-Only Verifications

*All phase behaviors have automated verification via httptest mocks for search and ollama stubs.*

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 10s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
