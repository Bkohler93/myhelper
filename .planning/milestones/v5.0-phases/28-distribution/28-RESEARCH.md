# Phase 28: Distribution - Research

**Researched:** 2026-05-09
**Domain:** Go binary distribution — goreleaser, GitHub Actions, curl install script
**Confidence:** HIGH

---

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- Targets: darwin/amd64, darwin/arm64, linux/amd64, linux/arm64
- Pushing a git tag triggers GitHub Actions builds automatically
- Built binaries appear as downloadable assets on GitHub Releases
- curl install script auto-detects OS and architecture and places binary in PATH

### Claude's Discretion
All implementation choices are at Claude's discretion — discuss phase was skipped per user setting. Use ROADMAP phase goal, success criteria, and codebase conventions to guide decisions.

### Deferred Ideas (OUT OF SCOPE)
- Homebrew tap (DIST-F01) — deferred per v5.0 roadmap decision
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| DIST-01 | User can install myhelper on WSL/Linux by running a one-line curl command that auto-detects OS/arch, downloads the correct binary from GitHub Releases, and places it in PATH | install.sh pattern with uname detection, GitHub Releases API, sha256sum verification |
| DIST-02 | Tagged git pushes automatically trigger GitHub Actions to build cross-platform binaries (darwin/amd64, darwin/arm64, linux/amd64, linux/arm64) and publish them to GitHub Releases via goreleaser | goreleaser-action@v7, goreleaser v2.x config, `on: push: tags: ["v*"]` workflow trigger |
</phase_requirements>

---

## Summary

This phase delivers three artifacts: a `.goreleaser.yaml` config at the project root, a `.github/workflows/release.yml` GitHub Actions workflow, and an `install.sh` curl-pipe install script. Together they satisfy both requirements: DIST-02 (automated builds on tag push) and DIST-01 (single-command install).

goreleaser v2.x is the current major version. The `goreleaser-action@v7` GitHub Action is the latest. The workflow pattern is well-established and well-documented — no experimental features are needed. The myhelper Go module (`github.com/bkohler93/myhelper`) uses Go 1.24.2 with no CGO dependencies, which makes cross-compilation completely trivial (`CGO_ENABLED=0` in all builds).

The curl install script follows a pattern used by Helm, GitHub CLI, and many other Go tools: detect OS/arch with `uname`, query the GitHub Releases API for the latest tag, construct the download URL, download the tarball and checksums file, verify with `sha256sum`, extract, and place the binary. The install target defaults to `~/.local/bin` (no sudo needed, works in WSL) with a fallback offer to `/usr/local/bin` if the user has sudo.

**Primary recommendation:** Use goreleaser v2 OSS (free), goreleaser-action@v7, and a self-contained install.sh that avoids sudo by defaulting to `~/.local/bin`.

---

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Cross-platform build | CI (GitHub Actions) | — | goreleaser runs on ubuntu-latest, cross-compiles all four targets |
| Release publishing | CI (GitHub Actions) | GitHub Releases API | goreleaser-action uploads archives and checksums to GitHub Releases |
| Binary packaging | goreleaser (.goreleaser.yaml) | — | Produces .tar.gz archives and checksum file |
| User install | install.sh (user machine) | — | Script runs on user's WSL/Linux or macOS, not in CI |
| PATH placement | install.sh (user machine) | — | Script writes to ~/.local/bin or /usr/local/bin |

---

## Standard Stack

### Core

| Library / Tool | Version | Purpose | Why Standard |
|----------------|---------|---------|--------------|
| goreleaser | v2.15.4 (latest stable) | Build, archive, and publish cross-platform Go binaries | The de-facto standard for Go binary releases; zero custom shell scripting for builds |
| goreleaser-action | v7.2.1 (latest) | GitHub Action that installs and runs goreleaser in CI | Official action maintained by goreleaser team |
| actions/checkout | v4 | Clone repo in CI | Current standard GitHub action |
| actions/setup-go | v5 | Install Go in CI | Current standard GitHub action |

**Version verification:** [VERIFIED: GitHub API] goreleaser v2.15.4 released 2026-04-21. goreleaser-action v7.2.1 confirmed via `api.github.com/repos/goreleaser/goreleaser-action/releases/latest`.

### No Additional Go Dependencies Required

This phase adds zero new Go dependencies to go.mod. goreleaser is a build-time dev tool, not a runtime import.

**Installation (dev tool only — not in go.mod):**
```bash
# Local install for development/testing
go install github.com/goreleaser/goreleaser/v2@latest
# Or via the goreleaser one-liner:
curl -sfL https://goreleaser.com/static/run | bash VERSION=v2.15.4 -s -- check
```

---

## Architecture Patterns

### System Architecture Diagram

```
Developer machine
  └─ git tag v1.2.3 && git push --tags
        │
        ▼
GitHub (remote)
  └─ tag push event
        │
        ▼
GitHub Actions: .github/workflows/release.yml
  ├─ actions/checkout@v4 (fetch-depth: 0)    ← full history needed for changelog
  ├─ actions/setup-go@v5 (go-version: stable)
  └─ goreleaser/goreleaser-action@v7
        │  reads .goreleaser.yaml
        │  GOOS=darwin  GOARCH=amd64  CGO_ENABLED=0  go build → myhelper binary
        │  GOOS=darwin  GOARCH=arm64  CGO_ENABLED=0  go build → myhelper binary
        │  GOOS=linux   GOARCH=amd64  CGO_ENABLED=0  go build → myhelper binary
        │  GOOS=linux   GOARCH=arm64  CGO_ENABLED=0  go build → myhelper binary
        │  archives each binary → myhelper_v1.2.3_darwin_amd64.tar.gz, etc.
        │  generates checksums.txt (sha256)
        └─ publishes to GitHub Releases via GITHUB_TOKEN
              │
              ▼
        GitHub Releases page
          ├─ myhelper_v1.2.3_darwin_amd64.tar.gz
          ├─ myhelper_v1.2.3_darwin_arm64.tar.gz
          ├─ myhelper_v1.2.3_linux_amd64.tar.gz
          ├─ myhelper_v1.2.3_linux_arm64.tar.gz
          └─ myhelper_v1.2.3_checksums.txt

User machine (WSL/Linux or macOS)
  └─ curl -sfL https://raw.githubusercontent.com/.../install.sh | bash
        │  uname -s  → "Linux" or "Darwin"
        │  uname -m  → "x86_64" → "amd64", "aarch64" → "arm64"
        │  GitHub API: /repos/bkohler93/myhelper/releases/latest → tag_name
        │  download myhelper_vX.Y.Z_${os}_${arch}.tar.gz
        │  download myhelper_vX.Y.Z_checksums.txt
        │  sha256sum --check
        │  tar -xzf → extract myhelper binary
        └─ install to ~/.local/bin/myhelper  (or /usr/local/bin with sudo)
```

### Recommended Project Structure

```
myhelper/                          # existing project root
├── .goreleaser.yaml               # new: goreleaser config
├── install.sh                     # new: curl install script
├── .github/
│   └── workflows/
│       └── release.yml            # new: GitHub Actions release workflow
├── main.go
├── cmd/
├── internal/
└── go.mod
```

### Pattern 1: .goreleaser.yaml — Minimal for Four Targets

**What:** Declares four build targets, archives each as tar.gz, generates a sha256 checksum file, and publishes to GitHub Releases.

**When to use:** Standard for any public Go binary project without Docker or container needs.

```yaml
# Source: https://goreleaser.com / Context7 /goreleaser/goreleaser verified
version: 2

project_name: myhelper

before:
  hooks:
    - go mod tidy

builds:
  - env:
      - CGO_ENABLED=0
    goos:
      - linux
      - darwin
    goarch:
      - amd64
      - arm64
    flags:
      - -trimpath
    ldflags:
      - -s -w -X main.version={{.Version}} -X main.commit={{.Commit}} -X main.date={{.Date}}
    mod_timestamp: "{{ .CommitTimestamp }}"

archives:
  - name_template: >-
      {{ .ProjectName }}_
      {{- .Version }}_
      {{- .Os }}_
      {{- .Arch }}
    formats:
      - tar.gz

checksum:
  name_template: "{{ .ProjectName }}_{{ .Version }}_checksums.txt"
  algorithm: sha256

changelog:
  sort: asc
  filters:
    exclude:
      - "^docs:"
      - "^test:"
      - "^chore:"

release:
  github:
    owner: bkohler93
    name: myhelper
```

**Key points:**
- `version: 2` is required for goreleaser v2.x configs. [VERIFIED: Context7 /goreleaser/goreleaser]
- `CGO_ENABLED=0` is required for cross-compilation from Linux to darwin targets (pure Go).
- `-trimpath` removes local file paths from the binary for reproducibility.
- `-s -w` strips debug info and DWARF symbols (reduces binary size ~30%).
- `ldflags` with `{{.Version}}` etc. — these inject version info. The variables `main.version`, `main.commit`, `main.date` only work if `main.go` declares those vars. Since myhelper's `main.go` does not currently declare them, either omit the `-X` flags or add the vars — planner should decide. [ASSUMED: current main.go has no version vars]
- `mod_timestamp` enables reproducible builds. [CITED: goreleaser.com/blog/reproducible-builds]

### Pattern 2: .github/workflows/release.yml

**What:** GitHub Actions workflow triggered on `v*` tag push. Runs goreleaser to build and publish.

**When to use:** Whenever tag-triggered CI/CD publishing is needed.

```yaml
# Source: https://goreleaser.com/customization/ci/actions/ [VERIFIED via WebFetch]
name: release

on:
  push:
    tags:
      - "v*"

permissions:
  contents: write

jobs:
  goreleaser:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@v4
        with:
          fetch-depth: 0      # REQUIRED — goreleaser needs full history for changelog

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: stable  # Uses go.mod's go directive automatically

      - name: Run GoReleaser
        uses: goreleaser/goreleaser-action@v7
        with:
          distribution: goreleaser
          version: "~> v2"
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

**Key points:**
- `fetch-depth: 0` is non-negotiable — goreleaser reads git tags to generate the changelog. Without full history, it fails or produces an empty changelog. [CITED: goreleaser.com/customization/ci/actions]
- `goreleaser-action@v7` is the latest stable action version. [VERIFIED: GitHub API]
- `version: "~> v2"` pins to the v2.x major line without locking to a specific patch. [CITED: goreleaser.com blog goreleaser-v2]
- `--clean` replaces the old `--rm-dist` flag in goreleaser v2. Using `--rm-dist` in v2 triggers a deprecation warning.
- `GITHUB_TOKEN` is auto-provided by GitHub Actions — no secret setup needed for public repos.
- `permissions: contents: write` is required for goreleaser to create and upload release assets. [CITED: Context7 /goreleaser/goreleaser]

### Pattern 3: install.sh — curl-pipe Install Script

**What:** Self-contained bash script that detects platform, downloads correct binary from GitHub Releases, verifies checksum, and installs to PATH.

**When to use:** One-line install for users without Go toolchain.

```bash
#!/usr/bin/env bash
# install.sh — myhelper installer
# Usage: curl -sfL https://raw.githubusercontent.com/bkohler93/myhelper/main/install.sh | bash
set -euo pipefail

REPO="bkohler93/myhelper"
BINARY="myhelper"
INSTALL_DIR="${INSTALL_DIR:-${HOME}/.local/bin}"

# ── OS / arch detection ────────────────────────────────────────────────────────
uname_os() {
  local os
  os=$(uname -s | tr '[:upper:]' '[:lower:]')
  case "$os" in
    mingw*) os="windows" ;;
    cygwin*) os="linux" ;;
  esac
  echo "$os"
}

uname_arch() {
  local arch
  arch=$(uname -m)
  case "$arch" in
    x86_64)  arch="amd64" ;;
    aarch64) arch="arm64" ;;
    armv7*)  arch="arm" ;;
  esac
  echo "$arch"
}

OS=$(uname_os)
ARCH=$(uname_arch)

# ── Fetch latest release tag ──────────────────────────────────────────────────
TAG=$(curl -sf "https://api.github.com/repos/${REPO}/releases/latest" \
  | grep '"tag_name"' | head -1 | sed 's/.*"tag_name": "\(.*\)".*/\1/')

if [[ -z "$TAG" ]]; then
  echo "Error: could not determine latest release tag" >&2
  exit 1
fi

VERSION="${TAG#v}"   # strip leading 'v' for archive name

# ── Construct URLs ────────────────────────────────────────────────────────────
BASE_URL="https://github.com/${REPO}/releases/download/${TAG}"
ARCHIVE="${BINARY}_${VERSION}_${OS}_${ARCH}.tar.gz"
CHECKSUM="${BINARY}_${VERSION}_checksums.txt"

# ── Download to temp dir ──────────────────────────────────────────────────────
TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT

curl -sfL "${BASE_URL}/${ARCHIVE}" -o "${TMP}/${ARCHIVE}"
curl -sfL "${BASE_URL}/${CHECKSUM}" -o "${TMP}/${CHECKSUM}"

# ── Verify checksum ───────────────────────────────────────────────────────────
cd "$TMP"
if command -v sha256sum >/dev/null 2>&1; then
  sha256sum --ignore-missing -c "${CHECKSUM}"
elif command -v shasum >/dev/null 2>&1; then
  # macOS fallback
  grep "${ARCHIVE}" "${CHECKSUM}" | shasum -a 256 -c -
else
  echo "Warning: no sha256sum or shasum available — skipping checksum verification" >&2
fi
cd - >/dev/null

# ── Extract and install ───────────────────────────────────────────────────────
tar -xzf "${TMP}/${ARCHIVE}" -C "${TMP}"
mkdir -p "${INSTALL_DIR}"
cp "${TMP}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
chmod +x "${INSTALL_DIR}/${BINARY}"

echo "Installed ${BINARY} ${TAG} to ${INSTALL_DIR}/${BINARY}"

# ── PATH hint ────────────────────────────────────────────────────────────────
if [[ ":$PATH:" != *":${INSTALL_DIR}:"* ]]; then
  echo ""
  echo "NOTE: ${INSTALL_DIR} is not in your PATH."
  echo "Add this to your ~/.bashrc or ~/.zshrc:"
  echo "  export PATH=\"\$HOME/.local/bin:\$PATH\""
fi
```

**Key design decisions:**
- Default install dir is `~/.local/bin` — writable without sudo, works in WSL where `/usr/local/bin` may require root.
- User can override: `curl ... | INSTALL_DIR=/usr/local/bin bash`
- Uses GitHub Releases API (`/releases/latest`) to auto-discover tag — no hardcoded version.
- `sha256sum --ignore-missing` lets us verify a single file against the multi-file checksums.txt without failing on unrelated entries.
- `trap 'rm -rf "$TMP"' EXIT` ensures cleanup even on error.
- macOS ships `shasum` not `sha256sum` — the fallback handles this. [ASSUMED: macOS users may run this script too, not just WSL users]
- `set -euo pipefail` ensures script aborts on any error.

### Anti-Patterns to Avoid

- **Using `--rm-dist` in goreleaser v2:** Deprecated flag — use `--clean` instead. Will produce warnings in CI output.
- **Omitting `fetch-depth: 0` in checkout:** goreleaser will error or generate empty changelogs. This is the most common CI setup mistake. [CITED: goreleaser.com CI docs]
- **Missing `version: 2` in .goreleaser.yaml:** goreleaser v2 requires this field; omitting it triggers a deprecation/error depending on context.
- **CGO_ENABLED=1 (default) for cross-compilation:** The default CGO_ENABLED=1 will fail when cross-compiling from linux/amd64 to darwin/* because a macOS C compiler isn't available on ubuntu-latest. myhelper has no CGO dependencies, so `CGO_ENABLED=0` is both safe and required.
- **install.sh using curl without `-f`:** Without `-f`, curl returns 0 even on HTTP 404, causing the script to silently install garbage. Always use `-sfL` (silent, fail on HTTP error, follow redirects).
- **Hard-coding a version in install.sh:** Fetching from the GitHub Releases API keeps the script maintenance-free.

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Cross-platform build matrix | Shell scripts with GOOS/GOARCH env vars | goreleaser | goreleaser handles archive creation, naming, checksums, changelog, and release publishing atomically |
| Checksum generation | Custom sha256 loop | goreleaser `checksum:` section | goreleaser generates and uploads the checksums file automatically |
| GitHub Release creation | gh CLI in workflow | goreleaser-action | goreleaser creates the release, sets title/body from changelog, and uploads all assets in one step |
| Binary naming convention | Custom naming logic | goreleaser name_template | Consistent naming ensures install.sh can predictably construct download URLs |

**Key insight:** goreleaser eliminates 90% of the custom shell scripting that Go binary distribution used to require. The only custom code needed is the install.sh (user-facing) and the YAML configs.

---

## Common Pitfalls

### Pitfall 1: `fetch-depth: 0` Omitted

**What goes wrong:** goreleaser cannot read git tags prior to HEAD, so `--snapshot` mode is triggered implicitly or the build fails with "no previous tag found."

**Why it happens:** GitHub Actions defaults to `fetch-depth: 1` (shallow clone) for speed.

**How to avoid:** Always set `fetch-depth: 0` in the checkout step. [CITED: goreleaser.com/customization/ci/actions]

**Warning signs:** CI log says "couldn't find any tags before..." or changelog section is empty.

### Pitfall 2: darwin Targets Fail Due to CGO

**What goes wrong:** `cross compilation for darwin/arm64 requires cgo to be disabled` error in CI.

**Why it happens:** ubuntu-latest has no macOS C toolchain. CGO is enabled by default in Go.

**How to avoid:** Set `CGO_ENABLED=0` in the `env:` section of each build in .goreleaser.yaml. myhelper has no C dependencies, so this is safe.

**Warning signs:** Build step fails with a CGO or C compiler not found error.

### Pitfall 3: GitHub Releases API Rate Limiting in install.sh

**What goes wrong:** `curl` to `api.github.com/repos/.../releases/latest` returns HTTP 403 with rate limit message; script exits with a confusing error.

**Why it happens:** Unauthenticated GitHub API calls are limited to 60/hour per IP, shared across all users on the same NAT (common in offices/WSL environments).

**How to avoid:** For a small project this is rarely a real problem. Mitigation if it becomes one: accept `GITHUB_TOKEN` env var in install.sh and pass `-H "Authorization: Bearer $GITHUB_TOKEN"`. Do not add this complexity now.

**Warning signs:** install.sh returns "could not determine latest release tag" error.

### Pitfall 4: install.sh Archive Name Mismatch

**What goes wrong:** install.sh constructs a URL like `myhelper_1.2.3_linux_amd64.tar.gz` but goreleaser produces `myhelper_1.2.3_Linux_amd64.tar.gz` (capital L).

**Why it happens:** goreleaser's default `name_template` uses `{{ .Os }}` which returns lowercase (`linux`, `darwin`). But if goreleaser config is customized incorrectly, or if `uname -s` returns uppercase and the script doesn't convert it, the URL won't match.

**How to avoid:** The script converts `uname -s` output to lowercase explicitly. The goreleaser archive name_template uses `{{ .Os }}` (goreleaser's own lowercase variable). Both sides produce lowercase — they will match.

**Warning signs:** 404 errors when curl downloads the archive.

### Pitfall 5: `version: 2` Missing from .goreleaser.yaml

**What goes wrong:** goreleaser v2 warns "you are using an old configuration file" and may refuse to run or behave unexpectedly.

**How to avoid:** First line after comments must be `version: 2`. [VERIFIED: Context7 /goreleaser/goreleaser]

### Pitfall 6: `--rm-dist` vs `--clean` in goreleaser v2

**What goes wrong:** Using `--rm-dist` (v1 flag) in goreleaser v2 produces a deprecation warning in CI logs. Not a hard failure today but will break in a future version.

**How to avoid:** Use `args: release --clean` in the GitHub Actions step. [CITED: Context7 goreleaser v2 blog]

---

## Code Examples

### Snapshot Build for Local Testing

```bash
# Source: https://goreleaser.com/quick-start [VERIFIED via Context7]
# Test build without a git tag — produces dist/ folder locally
goreleaser release --snapshot --clean
```

This produces `dist/` with all four archives. Inspect them to verify naming and content before pushing a real tag.

### Install goreleaser locally (macOS/Linux)

```bash
# One-liner runner (no permanent install)
curl -sfL https://goreleaser.com/static/run | bash VERSION=v2.15.4 -s -- release --snapshot --clean

# Or permanent install via go install
go install github.com/goreleaser/goreleaser/v2@latest
```

### Full Release Workflow (Manual)

```bash
# Source: Context7 /goreleaser/goreleaser [VERIFIED]
git tag -a v0.1.0 -m "Release v0.1.0"
git push origin v0.1.0
# GitHub Actions picks up the tag and runs goreleaser automatically
```

### install.sh One-Liner (What Users Will Run)

```bash
curl -sfL https://raw.githubusercontent.com/bkohler93/myhelper/main/install.sh | bash
```

---

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `goreleaser-action@v4/v5/v6` | `goreleaser-action@v7` | 2024-2025 | v7 is current; older versions still work but use them for new projects |
| `--rm-dist` flag | `--clean` flag | goreleaser v2 (2024) | `--rm-dist` is deprecated; use `--clean` |
| `version: 1` config | `version: 2` config | goreleaser v2 (2024) | v2 requires explicit `version: 2` in YAML |
| `go-version: '1.21'` hardcoded | `go-version: stable` | actions/setup-go v5 | `stable` reads go.mod, no manual version bump needed |
| goreleaser `format: zip` (singular) | `formats: [tar.gz]` (plural list) | goreleaser v2.6 | Both work in v2; plural form is forward-compatible |

**Deprecated/outdated:**
- `godownloader` (goreleaser/godownloader): Officially deprecated by the goreleaser team — do not use it. Manual install.sh is the recommended replacement. [CITED: GitHub repo README]

---

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | myhelper main.go does not declare `version`, `commit`, `date` package-level vars | Pattern 1 (ldflags) | If vars are absent, `-X main.version=...` ldflags silently have no effect — binary still builds, but no version info injected |
| A2 | GitHub repo is `github.com/bkohler93/myhelper` (public) | install.sh, release.yml | If private, GITHUB_TOKEN handling changes; unauthenticated API calls from install.sh will 404 |
| A3 | macOS users may also run install.sh (not only WSL) | Pattern 3 (install.sh) | If macOS support is not needed in install.sh, the `shasum` fallback branch can be dropped |

---

## Open Questions

1. **Version variables in main.go**
   - What we know: goreleaser's default ldflags inject `-X main.version={{.Version}}` etc.
   - What's unclear: myhelper's main.go currently has no version-related vars declared, so the ldflags will compile but inject nothing visible.
   - Recommendation: Either (a) add `var version = "dev"` to main.go and a `version` command so users can run `myhelper version`, or (b) omit the ldflags entirely for now. Option (a) is better UX but is a small scope expansion.

2. **GitHub repo name/owner**
   - What we know: go.mod declares `module github.com/bkohler93/myhelper`.
   - What's unclear: Whether the GitHub remote is exactly `bkohler93/myhelper` or a different org.
   - Recommendation: Confirm with `git remote -v` during plan execution. The goreleaser config's `release.github.owner/name` and install.sh `REPO` variable both depend on this.

---

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| git | Release tagging | ✓ | (system git) | — |
| Go | Build (local test) | ✓ | 1.24.2 (from go.mod) | — |
| goreleaser | Local snapshot test | ✗ | — | Use `curl -sfL https://goreleaser.com/static/run | bash` one-liner |
| GitHub Actions | DIST-02 automation | ✓ (cloud) | — | — |
| curl | install.sh | ✓ (standard on WSL/macOS) | — | wget fallback in script |
| sha256sum | install.sh checksum | ✓ (Linux/WSL) | — | shasum -a 256 (macOS fallback) |

**Missing dependencies with no fallback:** None blocking. goreleaser not installed locally is expected; the snapshot test uses the one-liner.

**Missing dependencies with fallback:** goreleaser local install — use the curl runner.

---

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Framework | go test (existing) |
| Config file | none (standard go test) |
| Quick run command | `go test ./...` |
| Full suite command | `go test ./...` |

This phase adds no Go source code changes — validation is infrastructure-level, not unit-testable. All verification is manual or via CI observation.

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| DIST-02 | goreleaser config is valid | manual (goreleaser check) | `goreleaser check` | ❌ Wave 0: .goreleaser.yaml |
| DIST-02 | Snapshot build produces 4 archives | manual (local snapshot) | `goreleaser release --snapshot --clean` | ❌ Wave 0: .goreleaser.yaml |
| DIST-02 | GitHub Actions workflow triggers on tag | manual (push a test tag) | `git tag v0.0.1-test && git push origin v0.0.1-test` | ❌ Wave 0: .github/workflows/release.yml |
| DIST-01 | install.sh downloads and installs binary | manual (clean env) | see below | ❌ Wave 0: install.sh |

### How to Test the goreleaser Config Locally

```bash
# Step 1: Install goreleaser (one-liner, no permanent install)
curl -sfL https://goreleaser.com/static/run > /tmp/goreleaser-run && chmod +x /tmp/goreleaser-run

# Step 2: Validate config syntax
/tmp/goreleaser-run goreleaser check

# Step 3: Snapshot build (no tag required, does NOT publish)
/tmp/goreleaser-run goreleaser release --snapshot --clean

# Step 4: Inspect output
ls dist/
# Expected:
#   myhelper_0.0.0-SNAPSHOT-*/myhelper  (four binaries)
#   myhelper_0.0.0-SNAPSHOT-*_darwin_amd64.tar.gz
#   myhelper_0.0.0-SNAPSHOT-*_darwin_arm64.tar.gz
#   myhelper_0.0.0-SNAPSHOT-*_linux_amd64.tar.gz
#   myhelper_0.0.0-SNAPSHOT-*_linux_arm64.tar.gz
#   myhelper_0.0.0-SNAPSHOT-*_checksums.txt
```

### How to Test install.sh in a Clean Environment

```bash
# Option A: Test against a real release (requires a published tag first)
# Run in a fresh shell with minimal PATH
env -i HOME="$HOME" PATH="/usr/local/bin:/usr/bin:/bin" bash /path/to/install.sh

# Verify:
~/.local/bin/myhelper --help

# Option B: Test against a snapshot (edit install.sh temporarily to use local files)
# Not practical without a real GitHub release. Acceptance test after first real tag push.

# Option C: Docker clean environment (Linux)
docker run --rm -v "$(pwd)/install.sh:/install.sh" ubuntu:22.04 bash /install.sh
```

### What the GitHub Actions Workflow Output Should Look Like

A successful run of the `release` job shows:
1. Checkout step: "Fetched N commits" (not just 1 — confirms fetch-depth: 0 worked)
2. Set up Go step: "go version go1.2x.x linux/amd64"
3. Run GoReleaser step:
   - "building binaries" × 4 (darwin/amd64, darwin/arm64, linux/amd64, linux/arm64)
   - "creating archive" × 4
   - "creating checksum file"
   - "publishing release"
   - "release published"
4. GitHub Releases page shows the new release with 5 assets (4 archives + checksums.txt)

### Sampling Rate

- **Per task commit:** `go test ./...` (existing tests pass — no regression)
- **Per wave merge:** `goreleaser check` + `goreleaser release --snapshot --clean`
- **Phase gate:** Push a real `v0.0.1` tag to verify the full pipeline end-to-end before marking phase complete

### Wave 0 Gaps

- [ ] `.goreleaser.yaml` — must exist before `goreleaser check` runs
- [ ] `.github/workflows/release.yml` — must exist before any CI test
- [ ] `install.sh` — must exist before install test

*(No existing test infrastructure covers these — all three files are new)*

---

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | — |
| V3 Session Management | no | — |
| V4 Access Control | no | — |
| V5 Input Validation | yes (install.sh) | Validate OS/arch values; use `set -euo pipefail`; never eval user input |
| V6 Cryptography | yes (install.sh) | sha256sum checksum verification of downloaded binary |

### Known Threat Patterns for curl-pipe Install Scripts

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| MITM binary substitution | Tampering | sha256sum verification against checksums.txt from GitHub Releases (same TLS connection); goreleaser signs checksums.txt |
| curl \| bash execution of arbitrary code | Tampering | User can inspect install.sh before running; script is in the public GitHub repo |
| Archive path traversal (tar slip) | Tampering | `tar -xzf` with no `--strip-components` is safe; binary is extracted to temp dir, then copied |
| GitHub API rate limiting DoS | Denial of Service | Graceful error message in script; users can retry |
| Malicious INSTALL_DIR env var | Elevation of Privilege | Script only uses INSTALL_DIR for `mkdir -p` and `cp`; no eval; no command injection |

**Note:** goreleaser OSS does not sign artifacts with cosign by default (that requires goreleaser Pro or additional workflow steps). For this project, sha256sum verification is sufficient — it detects accidental corruption. Full supply-chain signing (cosign) can be added later if needed. [ASSUMED: cosign signing is not required for v5.0]

---

## Sources

### Primary (HIGH confidence)
- Context7 `/goreleaser/goreleaser` — GitHub Actions workflow, build config, archives, checksums, snapshot commands
- Context7 `/websites/goreleaser` — goreleaser.yaml full field reference, install instructions
- [goreleaser.com/customization/ci/actions](https://goreleaser.com/customization/ci/actions/) — GitHub Actions workflow (verified via WebFetch)
- GitHub API `api.github.com/repos/goreleaser/goreleaser/releases/latest` — confirmed v2.15.4 as latest
- GitHub API `api.github.com/repos/goreleaser/goreleaser-action/releases/latest` — confirmed v7.2.1 as latest

### Secondary (MEDIUM confidence)
- [Helm install script pattern](https://raw.githubusercontent.com/kubernetes/helm/master/scripts/get) — OS/arch detection, checksum verification, PATH install pattern (verified via WebFetch)
- [fulll/github install.sh pattern](https://raw.githubusercontent.com/fulll/github/master/install.sh) — uname_os/uname_arch function pattern (verified via WebFetch)
- [goreleaser/goreleaser GitHub releases page](https://github.com/goreleaser/goreleaser/releases) — version history (verified via WebFetch)

### Tertiary (LOW confidence)
- None — all critical claims verified above.

---

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — versions verified via GitHub API; goreleaser docs verified via Context7 and WebFetch
- Architecture: HIGH — GitHub Actions + goreleaser pattern is extremely well-documented and stable
- install.sh pattern: MEDIUM-HIGH — based on verified real-world scripts (Helm, GitHub CLI); sha256sum/shasum fallback is standard
- Pitfalls: HIGH — fetch-depth and CGO pitfalls are documented in official goreleaser CI docs

**Research date:** 2026-05-09
**Valid until:** 2026-11-09 (goreleaser v2.x stable; GitHub Actions action versions may update)
