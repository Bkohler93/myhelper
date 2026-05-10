---
phase: 29
slug: tavily-search-provider
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-05-09
---

# Phase 29 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test |
| **Config file** | none |
| **Quick run command** | `go test ./internal/search/...` |
| **Full suite command** | `go test ./...` |
| **Estimated runtime** | ~5 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/search/...`
- **After every plan wave:** Run `go test ./...`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 10 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| extend-search-config | 01 | 1 | SRCH-01, SRCH-02, SRCH-03 | — | TavilyKey not logged | unit | `go test ./internal/search/... -run TestLoadConfig` | internal/search/search.go | pending |
| tavily-client | 01 | 1 | SRCH-01 | — | Key in header not body | unit | `go test ./internal/search/... -run TestTavily` | internal/search/search.go | pending |
| provider-dispatch | 01 | 1 | SRCH-01, SRCH-02 | — | No provider defaults to SearXNG | unit | `go test ./internal/search/... -run TestSearch` | internal/search/search.go | pending |
| env-var-override | 01 | 1 | SRCH-03 | — | Env var takes precedence | unit | `go test ./internal/search/... -run TestLoadConfig` | internal/search/search.go | pending |
| backwards-compat | 01 | 1 | SRCH-01 | — | No key → SearXNG unchanged | unit | `go test ./...` | cmd/root.go | pending |
