---
phase: 28-distribution
reviewed: 2026-05-09T00:00:00Z
fixed: 2026-05-09T00:00:00Z
depth: standard
files_reviewed: 5
files_reviewed_list:
  - .goreleaser.yaml
  - cmd/root.go
  - .gitignore
  - .github/workflows/release.yml
  - install.sh
findings:
  critical: 0
  warning: 0
  info: 3
  total: 3
status: clean
---

# Phase 28: Code Review Report

**Reviewed:** 2026-05-09
**Depth:** standard
**Files Reviewed:** 5
**Status:** issues_found

## Summary

Reviewed the distribution infrastructure for the myhelper v5.0 milestone: the installer script, GoReleaser config, GitHub Actions release workflow, root command version wiring, and .gitignore. The build pipeline is structurally sound — CGO_ENABLED=0 is set, fetch-depth: 0 is present for goreleaser changelog generation, ldflags paths match the module and package, the archive name template folds correctly to no-space filenames that match install.sh expectations, and dist/ is excluded from git. No critical bugs or security vulnerabilities were found.

Four warnings need attention before this ships: the PATH hint in install.sh is hardcoded to a different path than INSTALL_DIR when overridden; the checksum-skip warning does not abort the install; GitHub Actions use mutable tag pins rather than SHA pins; and the Commit/Date build variables are set by ldflags but never consumed anywhere in the binary.

---

## Warnings

### WR-01: install.sh PATH hint hardcodes default dir instead of using `$INSTALL_DIR`

**File:** `install.sh:93`
**Issue:** When `INSTALL_DIR` is overridden (e.g., `INSTALL_DIR=/usr/local/bin bash`), the check on line 89 correctly detects that the custom directory is not in `$PATH`, but the suggested export on line 93 is hardcoded to `$HOME/.local/bin` instead of the actual install directory. A user who overrides `INSTALL_DIR` receives instructions to add the wrong path.

**Fix:**
```bash
# Replace line 93:
  echo "  export PATH=\"${INSTALL_DIR}:\$PATH\""
```

---

### WR-02: Missing checksum tools causes silent install without verification

**File:** `install.sh:76-77`
**Issue:** When neither `sha256sum` nor `shasum` is found, the script prints a warning and continues, installing the binary without any integrity check. Under `set -euo pipefail` the warning is printed but the script does not exit. An attacker who can interfere with the download (or a corrupted CDN response) gets a free pass.

**Fix:**
```bash
else
  echo "Error: sha256sum and shasum not found — cannot verify checksum. Aborting." >&2
  exit 1
fi
```
If soft-fail is intentional (for minimal environments), the warning is acceptable, but a comment explaining the conscious trade-off should be added.

---

### WR-03: GitHub Actions pinned to mutable tags, not commit SHAs

**File:** `.github/workflows/release.yml:16,21,26`
**Issue:** All three actions use mutable floating tags (`@v4`, `@v5`, `@v7`). A tag can be force-pushed to point to a different (potentially malicious) commit. This is a supply-chain risk for a release workflow that has `contents: write` permission and uploads binaries to GitHub Releases.

```yaml
uses: actions/checkout@v4          # mutable
uses: actions/setup-go@v5          # mutable
uses: goreleaser/goreleaser-action@v7  # mutable
```

**Fix:** Pin each action to its full commit SHA:
```yaml
uses: actions/checkout@11bd71901bbe5b1630ceea73d27597364c9af683  # v4.2.2
uses: actions/setup-go@d60b41a563a30bb0b0dc4a7a450de86e8d1dda3e  # v5.5.0
uses: goreleaser/goreleaser-action@9c156ee5f32f69f1409b26aafcf9aadf7a40f37  # v7
```
Exact SHAs should be retrieved from the action repos to confirm they correspond to the tagged releases.

---

### WR-04: `Commit` and `Date` variables are set by ldflags but never consumed

**File:** `cmd/root.go:27-28`
**Issue:** The `Commit` and `Date` package-level variables are declared, documented in comments, and wired into the goreleaser ldflags — but they are referenced nowhere in the binary's output. `rootCmd.Version` only receives `Version`; the cobra `--version` flag prints `myhelper version 1.2.3` with no commit or date. The ldflags injection for these two variables is dead weight in every release build.

This is not a build failure (the linker accepts unused variables), but it means build provenance information is silently dropped and the commenting intent is misleading.

**Fix:** Either wire both into a version template string, or remove the ldflags entries and variable declarations if they are not planned:
```go
// In root.go, replace the Version field:
Version: fmt.Sprintf("%s (commit %s, built %s)", Version, Commit, Date),
```
Or if unused, remove `Commit` and `Date` from both `root.go` and the ldflags in `.goreleaser.yaml`.

---

## Info

### IN-01: Workflow permissions declared at workflow level instead of job level

**File:** `.github/workflows/release.yml:8-9`
**Issue:** The `permissions: contents: write` block is declared at the workflow level (before `jobs:`). This applies to all jobs in the workflow. Since there is currently only one job, the practical effect is the same as job-level. However, workflow-level permissions become a problem if a second job is added that should not have `contents: write`.

**Fix:** Move permissions inside the `goreleaser` job block for explicit least-privilege scoping:
```yaml
jobs:
  goreleaser:
    runs-on: ubuntu-latest
    permissions:
      contents: write
    steps:
```

---

### IN-02: `cygwin` mapped to `"linux"` but goreleaser builds no Windows/Cygwin binary

**File:** `install.sh:23`
**Issue:** The `uname_os` function maps `cygwin*` to `"linux"`. Cygwin is a Windows environment; the linux binary will not run there. Since goreleaser does not build a Windows target, no correct binary exists to download regardless. The mapping gives a false impression that Cygwin is supported — it will fail at `curl` (404) rather than at detection. A clear error at detection time is preferable.

**Fix:**
```bash
cygwin*) echo "Error: Cygwin is not a supported platform." >&2; exit 1 ;;
```

---

### IN-03: TAG extraction uses fragile `sed` instead of `jq`

**File:** `install.sh:43-44`
**Issue:** The GitHub API response is parsed with `grep '"tag_name"' | head -1 | sed 's/.*"tag_name": "\(.*\)".*/\1/'`. This works for current GitHub API output, but the greedy `\(.*\)` regex is sensitive to formatting changes and would misparse a tag name containing a double-quote character (unlikely but not impossible). `jq` is more robust and widely available on modern systems.

**Fix:**
```bash
TAG=$(curl -sf "https://api.github.com/repos/${REPO}/releases/latest" | jq -r '.tag_name')
```
If `jq` is not guaranteed to be present, add a detection step or fall back to the current pattern with a `jq`-first approach.

---

_Reviewed: 2026-05-09_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
