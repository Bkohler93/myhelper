---
phase: 30
slug: setup-wizard
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-05-10
---

# Phase 30 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test |
| **Config file** | none |
| **Quick run command** | `go test ./internal/wizard/...` |
| **Full suite command** | `go test ./...` |
| **Estimated runtime** | ~5 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/wizard/...`
- **After every plan wave:** Run `go test ./...`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 10 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| cobra-cmd | 01 | 1 | SETUP-01 | — | No key logged | unit | `go build ./... && go test ./cmd/...` | cmd/setup.go | pending |
| ollama-check | 01 | 1 | SETUP-01, SETUP-02 | — | HTTP-based detection | unit | `go test ./internal/wizard/... -run TestCheckOllama` | internal/wizard/wizard.go | pending |
| hardware-detect | 01 | 1 | SETUP-03 | — | No crash on unknown platform | unit | `go test ./internal/wizard/... -run TestDetectHardware` | internal/wizard/wizard.go | pending |
| model-pull | 01 | 1 | SETUP-04 | — | No unchecked error | unit | `go test ./internal/wizard/... -run TestPullModel` | internal/wizard/wizard.go | pending |
| config-write | 01 | 1 | SETUP-05, SETUP-06 | — | TavilyKey not echoed | unit | `go test ./internal/wizard/... -run TestWriteConfig` | internal/wizard/wizard.go | pending |
