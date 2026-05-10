# Phase 28: Distribution - Context

**Gathered:** 2026-05-09
**Status:** Ready for planning
**Mode:** Auto-generated (discuss skipped via workflow.skip_discuss)

<domain>
## Phase Boundary

myhelper binaries are downloadable and installable without a Go toolchain. This phase delivers: goreleaser build pipeline, GitHub Actions release workflow, and a curl install script. A WSL/Linux user can install myhelper with a single curl command.

</domain>

<decisions>
## Implementation Decisions

### Claude's Discretion
All implementation choices are at Claude's discretion — discuss phase was skipped per user setting. Use ROADMAP phase goal, success criteria, and codebase conventions to guide decisions.

Key constraints from ROADMAP success criteria:
- Targets: darwin/amd64, darwin/arm64, linux/amd64, linux/arm64
- Pushing a git tag triggers GitHub Actions builds automatically
- Built binaries appear as downloadable assets on GitHub Releases
- curl install script auto-detects OS and architecture and places binary in PATH
- Homebrew tap deferred to future (DIST-F01)

</decisions>

<code_context>
## Existing Code Insights

### Reusable Assets
- Go module: existing go.mod defines module path and Go version
- No existing CI/CD configuration

### Established Patterns
- Standard Go project structure with cmd/ and internal/ packages
- No existing release automation

### Integration Points
- .goreleaser.yaml at project root
- .github/workflows/release.yml for GitHub Actions
- install.sh curl script at project root or docs/

</code_context>

<specifics>
## Specific Ideas

No specific requirements — open to standard approaches. Refer to ROADMAP phase description and success criteria.

</specifics>

<deferred>
## Deferred Ideas

- Homebrew tap (DIST-F01) — deferred per v5.0 roadmap decision

</deferred>
