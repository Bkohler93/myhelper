---
phase: 28
slug: distribution
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-05-09
---

# Phase 28 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | goreleaser check (local snapshot builds), bash (install.sh), GitHub Actions |
| **Config file** | `.goreleaser.yaml` |
| **Quick run command** | `goreleaser check` |
| **Full suite command** | `goreleaser build --snapshot --clean` |
| **Estimated runtime** | ~60 seconds |

---

## Sampling Rate

- **After every task commit:** Run `goreleaser check` (validates config syntax)
- **After every plan wave:** Run `goreleaser build --snapshot --clean` (builds all targets)
- **Before `/gsd-verify-work`:** Snapshot build must succeed for all 4 targets
- **Max feedback latency:** 60 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 28-01-01 | 01 | 1 | DIST-02 | — | N/A | config-check | `goreleaser check` | ✅ | ⬜ pending |
| 28-02-01 | 02 | 2 | DIST-02 | — | N/A | file-exists | `test -f .github/workflows/release.yml` | ✅ | ⬜ pending |
| 28-03-01 | 03 | 3 | DIST-01 | — | N/A | file-exists + manual | `test -f install.sh && bash install.sh --dry-run 2>/dev/null \|\| true` | ✅ | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- Existing Go build infrastructure covers all phase requirements.

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Binaries appear on GitHub Releases | DIST-02 | Requires actual tag push to GitHub | Push `v0.0.1-test` tag, verify Releases page, delete tag after |
| curl install script places binary in PATH | DIST-01 | Requires a real binary download from Releases | Run `bash install.sh` in a clean WSL/Linux env after Releases are live |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 60s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
