---
phase: 06
slug: init-sync-commands
status: compliant
nyquist_compliant: true
wave_0_complete: true
created: 2026-04-08
audited: 2026-04-08
---

# Phase 6 — Validation Strategy

> Per-phase validation contract reconstructed from SUMMARY/PLAN artifacts + generated tests.
> Status: **NYQUIST-COMPLIANT** — all automatable behaviors have automated verification.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test (stdlib) |
| **Config file** | none |
| **Quick run command** | `go test ./cmd/ -run "TestReadLastSync\|TestWriteLastSync\|TestGenerateContextMD\|TestChangedFilesSince\|TestDeltaIndex\|TestDeltaSummaries" -count=1` |
| **Full suite command** | `go test ./cmd/ -count=1` |
| **Estimated runtime** | ~1 second |

---

## Sampling Rate

- **After every task commit:** Run `go test ./cmd/ -count=1`
- **After every plan wave:** Run `go test ./cmd/ -v -count=1`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** ~1 second

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------------|-----------|-------------------|-------------|--------|
| 06-01-T1 | 01 | 1 | SYNC-01, SYNC-02 | RunWithSpinner executes work fn, propagates error | manual | see Manual-Only | — | ⚠️ manual |
| 06-01-T2 | 01 | 1 | SYNC-01, SYNC-02 | generateContextMD reads summaries, calls chatFn, writes context.md | unit | `go test ./cmd/ -run TestGenerateContextMD` | ✅ | ✅ green |
| 06-01-T2 | 01 | 1 | SYNC-01, SYNC-02 | readLastSync returns zero time when meta.json absent | unit | `go test ./cmd/ -run TestReadLastSync` | ✅ | ✅ green |
| 06-01-T2 | 01 | 1 | SYNC-01, SYNC-02 | writeLastSync writes meta.json; round-trip read returns same time | unit | `go test ./cmd/ -run TestWriteLastSync` | ✅ | ✅ green |
| 06-02-T1 | 02 | 2 | SYNC-01 | runInit wires scanner.Scan + generateContextMD + writeLastSync | manual | see Manual-Only | — | ⚠️ manual |
| 06-03-T1 | 03 | 2 | SYNC-01, SYNC-02 | changedFilesSince: zero time returns all files; mtime filter; excluded dirs skipped | unit | `go test ./cmd/ -run TestChangedFilesSince` | ✅ | ✅ green |
| 06-03-T1 | 03 | 2 | SYNC-02 | deltaIndex: adds/updates/removes entries; handles corrupt index | unit | `go test ./cmd/ -run TestDeltaIndex` | ✅ | ✅ green |
| 06-03-T1 | 03 | 2 | SYNC-02 | deltaSummaries: identifies affected packages; calls GenerateSummaries; empty = no-op | unit | `go test ./cmd/ -run TestDeltaSummaries` | ✅ | ✅ green |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ manual*

---

## Wave 0 Requirements

Existing infrastructure covers all automated phase requirements.

Test file generated post-execution:
- `cmd/sync_test.go` — 16 tests covering all 6 testable functions (SYNC-01, SYNC-02)

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| `RunWithSpinner` executes work function and displays spinner on stderr | SYNC-01, SYNC-02 | Bubble Tea spawns a real TTY program; automated test would need a pseudo-terminal harness | Run `myhelper init` in a real terminal; verify spinner appears and "✓ Done." displays |
| `runInit` end-to-end: produces index.json + summaries/ + context.md + meta.json | SYNC-01 | Requires live Ollama endpoint for scanner.Scan and generateContextMD | Run `myhelper init` in a Go project dir; check `.myhelper/` contains all 4 artifacts |
| `runSync` guard: error when meta.json absent | SYNC-02 | Cobra command execution; requires CLI entry point | Run `myhelper sync` in dir without `.myhelper/`; expect error "run 'myhelper init' first" |
| `runSync` end-to-end: delta rescan updates index + summaries + context | SYNC-02 | Requires live Ollama + filesystem with real init state | Run `myhelper init`, modify a .go file, run `myhelper sync`; verify meta.json timestamp updated |

---

## Validation Sign-Off

- [x] All automatable tasks have `go test` commands
- [x] Sampling continuity: all 6 testable functions covered
- [x] Wave 0: test file `cmd/sync_test.go` exists, 16/16 tests green
- [x] No watch-mode flags
- [x] Feedback latency ~1s
- [x] `nyquist_compliant: true` set in frontmatter
- [x] Manual-only items documented with clear test instructions

**Approval:** approved 2026-04-08 (generated post-execution — all 16 new tests + 13 existing cmd tests passing)
