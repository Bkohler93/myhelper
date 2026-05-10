---
phase: 28-distribution
plan: "03"
subsystem: distribution
tags: [install, bash, curl-pipe, sha256, goreleaser]

dependency_graph:
  requires:
    - phase: 28-distribution
      plan: "01"
      provides: [goreleaser-config, archive-name-template]
  provides:
    - install.sh curl-pipe installer for myhelper (DIST-01)
  affects: [docs, README]

tech-stack:
  added: []
  patterns:
    - "curl-pipe install script: uname-based OS/arch detection, mktemp temp dir, trap cleanup, sha256sum+shasum fallback"

key-files:
  created:
    - install.sh
  modified: []

key-decisions:
  - "Default INSTALL_DIR is ~/.local/bin — no sudo required, follows XDG convention"
  - "VERSION strips 'v' prefix from GitHub tag_name to match goreleaser archive naming"
  - "sha256sum --ignore-missing handles multi-file checksums.txt without false failures"
  - "shasum -a 256 fallback for macOS (no sha256sum shipped on darwin)"
  - "curl -sfL: -f flag makes curl fail on HTTP errors (prevents installing 404 garbage)"
  - "set -euo pipefail aborts on any error, unset variable, or pipe failure"
  - "trap 'rm -rf $TMP' EXIT guarantees temp dir cleanup on early exit"

patterns-established:
  - "Archive naming: ${BINARY}_${VERSION}_${OS}_${ARCH}.tar.gz — must match goreleaser name_template exactly"
  - "OS detection: uname -s | tr upper lower; arch detection: uname -m with case mapping"

requirements-completed: [DIST-01]

duration: 1min
completed: "2026-05-10"
---

# Phase 28 Plan 03: Install Script Summary

**Single-command curl-pipe installer for myhelper — auto-detects platform, downloads correct binary from GitHub Releases, verifies sha256 checksum, installs to ~/.local/bin (no sudo), and prints a PATH hint if needed.**

## Performance

- **Duration:** 41 seconds
- **Started:** 2026-05-10T03:12:20Z
- **Completed:** 2026-05-10T03:13:01Z
- **Tasks:** 1
- **Files modified:** 1

## Accomplishments

- Created `install.sh` at project root implementing DIST-01: single curl-pipe install command for WSL/Linux and macOS users without Go toolchain
- OS/arch detection via `uname_os()` and `uname_arch()` maps platform output to goreleaser's archive naming convention (`linux`/`darwin`, `amd64`/`arm64`)
- Archive name construction `${BINARY}_${VERSION}_${OS}_${ARCH}.tar.gz` matches goreleaser `name_template` exactly, verified against `.goreleaser.yaml`
- sha256 checksum verification with `sha256sum --ignore-missing` on Linux and `shasum -a 256` fallback for macOS
- PATH hint printed when `~/.local/bin` is not in user's `$PATH`

## Task Commits

1. **Task 1: Create install.sh with OS/arch detection, checksum verification, and PATH install** - `062527a` (feat)

## Files Created/Modified

- `install.sh` — curl-pipe installer: OS/arch detection, GitHub API release fetch, sha256 checksum verification, tar extraction to temp dir, binary copy to INSTALL_DIR

## Decisions Made

- `INSTALL_DIR` defaults to `~/.local/bin` — avoids sudo requirement; user can override with `INSTALL_DIR=/usr/local/bin bash`
- `VERSION="${TAG#v}"` strips the leading 'v' from GitHub API `tag_name` to match goreleaser archive names (goreleaser uses `{{.Version}}` which is the tag without 'v')
- `curl -sfL` flags: `-s` silent output, `-f` fail on HTTP errors (critical — without this, a 404 returns 0 and installs garbage), `-L` follows redirects
- `sha256sum --ignore-missing` handles multi-archive checksums.txt (one file covers all four platform archives) without failing on non-present entries
- Binary extracted to `$TMP` then explicitly copied to `$INSTALL_DIR/$BINARY` — prevents tar path traversal (T-28-10 mitigated)

## Deviations from Plan

None — plan executed exactly as written.

## Issues Encountered

None.

## Threat Surface Scan

All security-relevant surfaces are covered by the plan's threat model:

| Threat | Disposition | Mitigation |
|--------|-------------|-----------|
| T-28-08: Downloaded binary tampering | mitigate | sha256sum verification against checksums.txt (both TLS-fetched from GitHub Releases) |
| T-28-09: curl-pipe execution | accept | Script public on GitHub; user can inspect with `| less` before running |
| T-28-10: tar path traversal | mitigate | Extract to `$TMP`, then explicit `cp $TMP/$BINARY $INSTALL_DIR/$BINARY` — no wildcard paths |
| T-28-11: GitHub API rate limiting | accept | 60 req/hr per IP; error message directs to releases page |
| T-28-12: INSTALL_DIR injection | mitigate | Used only in `mkdir -p` and `cp` — no eval, no command injection possible |
| T-28-13: OS/arch disclosure in output | accept | Not sensitive; aids debugging |

## Known Stubs

None.

## Next Phase Readiness

- `install.sh` ready for users once a GitHub release tag exists (requires 28-02 GitHub Actions workflow to create a release)
- End-to-end test (`env -i HOME="$HOME" PATH="..." bash install.sh`) requires a real GitHub release with attached archives — can be validated after first `v1.0.0` tag push

## Self-Check: PASSED

- `install.sh` exists at worktree root: FOUND
- `install.sh` is executable: FOUND
- `bash -n install.sh` exits 0: PASSED
- `grep 'REPO="bkohler93/myhelper"' install.sh`: FOUND
- `grep "INSTALL_DIR.*local/bin"`: FOUND
- `grep "sha256sum"`: FOUND
- `grep "trap.*TMP"`: FOUND
- Commit `062527a` exists: FOUND

---
*Phase: 28-distribution*
*Completed: 2026-05-10*
