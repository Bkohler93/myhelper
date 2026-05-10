# Phase 28: Distribution - Pattern Map

**Mapped:** 2026-05-09
**Files analyzed:** 3 new files + 2 modifications
**Analogs found:** 0 / 3 (no existing CI/CD or distribution files in codebase)

---

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `.goreleaser.yaml` | config | batch (build pipeline) | none | no analog |
| `.github/workflows/release.yml` | config | event-driven (tag push) | none | no analog |
| `install.sh` | utility | request-response (HTTP download) | none | no analog |
| `main.go` | entry point | — | `main.go` (existing) | exact (modification) |
| `.gitignore` | config | — | `.gitignore` (existing) | exact (modification) |

---

## Existing File Findings

### `main.go` — Current State (lines 1-7)

```go
package main

import "github.com/bkohler93/myhelper/cmd"

func main() {
	cmd.Execute()
}
```

**Key finding:** No `version`, `commit`, or `date` package-level variables declared. The goreleaser ldflags `-X main.version={{.Version}}` will compile without error but inject nothing visible. Two options for the planner:

- Option A (recommended): Add `var version = "dev"` to main.go and wire a `version` subcommand or `--version` flag through cobra so users can run `myhelper version`.
- Option B: Omit the `-X` ldflags from .goreleaser.yaml entirely — binary builds fine, no version info surfaced.

**Cobra version flag pattern** (from `cmd/root.go`, lines 26-31): rootCmd already uses `cobra.Command`. The standard cobra approach for version is:
```go
var version = "dev"  // set by ldflags at build time

// in rootCmd definition:
Version: version,  // cobra exposes --version flag automatically when Version is set
```

### `.gitignore` — Current Contents

```
.planning/
myhelper
.myhelper/
tmp/
.claude/
```

**Key finding:** goreleaser writes its output to `dist/` by default. This directory is not currently in `.gitignore`. The planner must add `dist/` to avoid accidentally committing build artifacts.

### `go.mod` — Module Path and Go Version (lines 1-3)

```
module github.com/bkohler93/myhelper

go 1.24.2
```

**Key findings:**
- Module path is `github.com/bkohler93/myhelper` — owner is `bkohler93`, repo is `myhelper`.
- Go version is `1.24.2` — `go-version: stable` in the GitHub Actions workflow reads this automatically.
- Zero CGO dependencies — `CGO_ENABLED=0` is safe for all four targets.

### `dev.sh` — Existing Build Script (line 1)

```bash
go build -o tmp/myhelper . && ./tmp/myhelper $1 $2 $3
```

**Key finding:** No makefile, no existing CI/CD. dev.sh is a single-line local dev script — not a pattern to replicate. goreleaser replaces this for release builds.

### Git Remote

No git remote configured (`git remote get-url origin` returned empty). The planner must note that the user needs to push the repo to `github.com/bkohler93/myhelper` before the GitHub Actions workflow can trigger.

---

## Pattern Assignments

### `.goreleaser.yaml` (config, batch)

**Analog:** None in codebase. Use RESEARCH.md Pattern 1 directly.

**Critical fields derived from this codebase:**

- `project_name: myhelper` — matches binary name used in `dev.sh` and `.gitignore`
- `release.github.owner: bkohler93` — from `go.mod` module path
- `release.github.name: myhelper` — from `go.mod` module path
- `go-version` in builds: not needed (goreleaser uses current Go, workflow pins via `setup-go`)

**ldflags decision point:** The planner must choose Option A or B from the main.go section above before writing the ldflags line. If Option A, the goreleaser config should include:

```yaml
ldflags:
  - -s -w -X main.version={{.Version}} -X main.commit={{.Commit}} -X main.date={{.Date}}
```

If Option B, omit the `-X` flags:

```yaml
ldflags:
  - -s -w
```

**Full pattern from RESEARCH.md (Pattern 1, lines 154-204):**

```yaml
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

**Non-negotiable flags (from RESEARCH.md pitfalls):**
- `version: 2` — required; omitting causes goreleaser v2 to error
- `CGO_ENABLED=0` — required for cross-compilation to darwin on ubuntu-latest
- `formats: [tar.gz]` — plural form (v2.6+ forward-compatible)

---

### `.github/workflows/release.yml` (config, event-driven)

**Analog:** None in codebase. Use RESEARCH.md Pattern 2 directly.

**Critical fields derived from this codebase:**
- `go-version: stable` — reads `go 1.24.2` from `go.mod` automatically, no hardcoding needed

**Full pattern from RESEARCH.md (Pattern 2, lines 221-254):**

```yaml
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
          fetch-depth: 0

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: stable

      - name: Run GoReleaser
        uses: goreleaser/goreleaser-action@v7
        with:
          distribution: goreleaser
          version: "~> v2"
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

**Non-negotiable fields (from RESEARCH.md pitfalls):**
- `fetch-depth: 0` — required; omitting causes goreleaser to fail or produce empty changelog
- `permissions: contents: write` — required for goreleaser to create release assets
- `args: release --clean` — use `--clean` not `--rm-dist` (deprecated in v2)
- `version: "~> v2"` — pins to v2.x major without locking to a patch version

---

### `install.sh` (utility, request-response)

**Analog:** None in codebase. Use RESEARCH.md Pattern 3 directly.

**Critical values derived from this codebase:**
- `REPO="bkohler93/myhelper"` — from `go.mod` module path
- `BINARY="myhelper"` — from `dev.sh` build output name

**Archive name construction** must match goreleaser's `name_template`:
- goreleaser produces: `myhelper_1.2.3_linux_amd64.tar.gz` (lowercase OS from `{{ .Os }}`)
- install.sh OS detection must also produce lowercase: `uname -s | tr '[:upper:]' '[:lower:]'`
- This is already correct in the RESEARCH.md pattern — no adjustment needed

**Full pattern from RESEARCH.md (Pattern 3, lines 270-355):**

```bash
#!/usr/bin/env bash
# install.sh — myhelper installer
# Usage: curl -sfL https://raw.githubusercontent.com/bkohler93/myhelper/main/install.sh | bash
set -euo pipefail

REPO="bkohler93/myhelper"
BINARY="myhelper"
INSTALL_DIR="${INSTALL_DIR:-${HOME}/.local/bin}"

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

TAG=$(curl -sf "https://api.github.com/repos/${REPO}/releases/latest" \
  | grep '"tag_name"' | head -1 | sed 's/.*"tag_name": "\(.*\)".*/\1/')

if [[ -z "$TAG" ]]; then
  echo "Error: could not determine latest release tag" >&2
  exit 1
fi

VERSION="${TAG#v}"

BASE_URL="https://github.com/${REPO}/releases/download/${TAG}"
ARCHIVE="${BINARY}_${VERSION}_${OS}_${ARCH}.tar.gz"
CHECKSUM="${BINARY}_${VERSION}_checksums.txt"

TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT

curl -sfL "${BASE_URL}/${ARCHIVE}" -o "${TMP}/${ARCHIVE}"
curl -sfL "${BASE_URL}/${CHECKSUM}" -o "${TMP}/${CHECKSUM}"

cd "$TMP"
if command -v sha256sum >/dev/null 2>&1; then
  sha256sum --ignore-missing -c "${CHECKSUM}"
elif command -v shasum >/dev/null 2>&1; then
  grep "${ARCHIVE}" "${CHECKSUM}" | shasum -a 256 -c -
else
  echo "Warning: no sha256sum or shasum available — skipping checksum verification" >&2
fi
cd - >/dev/null

tar -xzf "${TMP}/${ARCHIVE}" -C "${TMP}"
mkdir -p "${INSTALL_DIR}"
cp "${TMP}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
chmod +x "${INSTALL_DIR}/${BINARY}"

echo "Installed ${BINARY} ${TAG} to ${INSTALL_DIR}/${BINARY}"

if [[ ":$PATH:" != *":${INSTALL_DIR}:"* ]]; then
  echo ""
  echo "NOTE: ${INSTALL_DIR} is not in your PATH."
  echo "Add this to your ~/.bashrc or ~/.zshrc:"
  echo "  export PATH=\"\$HOME/.local/bin:\$PATH\""
fi
```

---

### `main.go` modification (entry point, optional)

**Analog:** `/Users/brettkohler/dev/apps/myhelper/main.go` (lines 1-7) — exact match, modification only.

**If Option A is chosen** (add version variable + cobra Version field):

Current state:
```go
package main

import "github.com/bkohler93/myhelper/cmd"

func main() {
	cmd.Execute()
}
```

Required addition — add `version` var before `main()`:
```go
package main

import "github.com/bkohler93/myhelper/cmd"

var version = "dev" // overridden by goreleaser ldflags at build time

func main() {
	cmd.Execute()
}
```

Then in `cmd/root.go`, add `Version: version` to `rootCmd` — but `version` is in `main` package, not `cmd`. Standard cobra pattern for cross-package version:

Option A1: Pass version into Execute():
```go
// main.go
cmd.Execute(version)

// cmd/root.go — Execute signature
func Execute(version string) {
    rootCmd.Version = version
    ...
}
```

Option A2: Declare `version` in `cmd` package directly (simpler):
```go
// cmd/version.go (new file) or top of root.go
var Version = "dev" // set by ldflags: -X github.com/bkohler93/myhelper/cmd.Version=...
```
Then goreleaser ldflags: `-X github.com/bkohler93/myhelper/cmd.Version={{.Version}}`

**Recommendation for planner:** Option A2 is simpler — declare `Version` in `cmd` package, set `rootCmd.Version = Version`. Goreleaser ldflags reference the full package path `github.com/bkohler93/myhelper/cmd.Version`.

---

### `.gitignore` modification (config)

**Analog:** `/Users/brettkohler/dev/apps/myhelper/.gitignore` (existing file) — append only.

Current contents:
```
.planning/
myhelper
.myhelper/
tmp/
.claude/
```

Add:
```
dist/
```

goreleaser writes all build output to `dist/` by default. Without this entry, `dist/` would appear as untracked files after a local snapshot build.

---

## Shared Patterns

### No Shared Cross-Cutting Patterns

This phase creates infrastructure/config files only — no Go source patterns (auth, error handling, validation, logging) apply. All three new files are self-contained with no shared runtime dependencies.

### Naming Consistency (applies to all three new files)

The binary name `myhelper` appears in four places and must be consistent:

| Location | Value | Source of Truth |
|----------|-------|-----------------|
| `.goreleaser.yaml` `project_name:` | `myhelper` | dev.sh output `-o tmp/myhelper` |
| `.goreleaser.yaml` `release.github.name:` | `myhelper` | go.mod module path |
| `install.sh` `BINARY=` | `myhelper` | same |
| `install.sh` `REPO=` | `bkohler93/myhelper` | go.mod module path |

### GitHub Repo Coordinates

The go.mod module path `github.com/bkohler93/myhelper` implies:
- Owner: `bkohler93`
- Repo: `myhelper`
- Full REPO string for install.sh: `bkohler93/myhelper`

Note: No git remote is configured locally (`git remote get-url origin` returns empty). The planner should add a note that the user must create the GitHub repo and push before the release workflow can function.

---

## No Analog Found

All three new files have no codebase analog — there is no existing CI/CD infrastructure in this project.

| File | Role | Data Flow | Reason |
|------|------|-----------|--------|
| `.goreleaser.yaml` | config | batch | No build pipeline config exists |
| `.github/workflows/release.yml` | config | event-driven | No GitHub Actions workflows exist |
| `install.sh` | utility | request-response | No install or distribution scripts exist |

For all three, the planner should use the RESEARCH.md patterns verbatim (they are verified against official goreleaser docs and real-world scripts). The only codebase-specific substitutions are the `REPO`, `BINARY`, `project_name`, and `release.github.owner/name` values derived above.

---

## Metadata

**Analog search scope:** project root, cmd/, internal/, .github/ (does not exist)
**Files scanned:** main.go, go.mod, cmd/root.go, dev.sh, .gitignore
**Pattern extraction date:** 2026-05-09
