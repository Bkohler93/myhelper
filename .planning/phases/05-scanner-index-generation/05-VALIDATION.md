---
phase: 05
slug: scanner-index-generation
status: compliant
nyquist_compliant: true
wave_0_complete: true
created: 2026-04-08
audited: 2026-04-08
---

# Phase 5 — Validation Strategy

> Per-phase validation contract reconstructed from SUMMARY and VERIFICATION artifacts.
> Status: **NYQUIST-COMPLIANT** — all requirements have automated verification.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test (stdlib) |
| **Config file** | none — stdlib test runner |
| **Quick run command** | `go test ./internal/scanner/ -count=1` |
| **Full suite command** | `go test ./internal/scanner/ -v -count=1` |
| **Estimated runtime** | ~2 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/scanner/ -count=1`
- **After every plan wave:** Run `go test ./internal/scanner/ -v -count=1`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** ~2 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 05-01-T1 | 01 | 1 | INIT-08 | — | Walk() excludes .git, vendor, testdata, .myhelper, generated files | unit | `go test ./internal/scanner/ -run TestWalk` | ✅ | ✅ green |
| 05-01-T2 | 01 | 1 | INIT-02 | — | ExtractSymbols() returns exported funcs/types/interfaces; excludes consts/vars | unit | `go test ./internal/scanner/ -run TestExtractSymbols` | ✅ | ✅ green |
| 05-02-T1 | 02 | 1 | INIT-03, INIT-04, INIT-05 | — | ReadMeta() parses go.mod, README, config files; graceful absence | unit | `go test ./internal/scanner/ -run TestReadMeta` | ✅ | ✅ green |
| 05-03-T1 | 03 | 1 | INIT-06 | — | BuildIndex() applies 80% token budget; drops test files first | unit | `go test ./internal/scanner/ -run TestBuildIndex` | ✅ | ✅ green |
| 05-04-T1 | 04 | 1 | INIT-07 | — | GenerateSummaries() calls ChatFn once per package; writes {pkg}.md | unit | `go test ./internal/scanner/ -run TestGenerateSummaries` | ✅ | ✅ green |
| 05-05-T1 | 05 | 1 | INIT-01 | — | Scan() coordinates BuildIndex + GenerateSummaries; produces index.json + summaries/ | integration | `go test ./internal/scanner/ -run TestScan` | ✅ | ✅ green |
| 05-06-T1 | 06 | 1 | INIT-03, INIT-04, INIT-05 | — | BuildIndex() serializes {meta: {...}, files: [...]}; ReadMeta wired into pipeline | unit | `go test ./internal/scanner/ -run TestBuildIndex_Meta` | ✅ | ✅ green |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

Existing infrastructure covers all phase requirements.

All test files were created during plan execution (TDD red/green per plan):
- `internal/scanner/walker_test.go` — INIT-08 (Walk exclusion rules)
- `internal/scanner/ast_test.go` — INIT-02 (ExtractSymbols)
- `internal/scanner/meta_test.go` — INIT-03/04/05 (ReadMeta)
- `internal/scanner/index_test.go` — INIT-06 (BuildIndex budget), INIT-03/04/05 (meta in index)
- `internal/scanner/summary_test.go` — INIT-07 (GenerateSummaries)
- `internal/scanner/scan_test.go` — INIT-01 (Scan integration)

---

## Manual-Only Verifications

All phase behaviors have automated verification.

Note: INIT-01 is PARTIAL at the scanner API level — full init CLI wiring verified in Phase 6.

---

## Validation Sign-Off

- [x] All tasks have automated verification commands
- [x] Sampling continuity: every task has a test command; no 3-consecutive-task gap
- [x] Wave 0: no missing references — all tests exist
- [x] No watch-mode flags
- [x] Feedback latency ~2s (well under any practical limit)
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** approved 2026-04-08 (reconstructed post-execution — all 55 tests passing at time of audit)
