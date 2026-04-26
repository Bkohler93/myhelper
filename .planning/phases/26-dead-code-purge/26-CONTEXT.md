# Phase 26: Dead Code Purge - Context

**Gathered:** 2026-04-26
**Status:** Ready for planning
**Mode:** Auto-generated (infrastructure phase — discuss skipped)

<domain>
## Phase Boundary

Delete the four dead internal packages (`internal/context`, `internal/planner`, `internal/retrieval`, `internal/scanner`) and remove the `--no-context` flag from root. The codebase should build and test cleanly with no references to deleted code.

</domain>

<decisions>
## Implementation Decisions

### Claude's Discretion
All implementation choices are at Claude's discretion — pure infrastructure phase. Use ROADMAP phase goal, success criteria, and codebase conventions to guide decisions.

</decisions>

<code_context>
## Existing Code Insights

### Files to delete
- `internal/context/context.go` — entire directory
- `internal/planner/` — entire directory (planner.go, discover.go, discover_test.go)
- `internal/retrieval/` — entire directory (retrieval.go, retrieval_test.go)
- `internal/scanner/` — entire directory (artifacts.go, artifacts_test.go, ast.go, ast_test.go, index.go)

### Live files that reference dead packages
- `cmd/inspect.go:9` — imports `github.com/bkohler93/myhelper/internal/retrieval`
- `cmd/inspect.go:38` — uses `noContextFlag` (references the `--no-context` path)
- `cmd/root.go:17` — declares `noContextFlag bool`
- `cmd/root.go:24` — registers `--no-context` PersistentFlag

### cmd/inspect.go notes
`cmd/inspect.go` currently uses `internal/retrieval` and `noContextFlag`. The entire inspect command will be rewritten in Phase 27. For Phase 26, stub it out minimally (remove the retrieval import and noContextFlag usage, keep the command registered but do something benign like printing "inspect rewrite coming in Phase 27") so the build stays clean without the full rewrite.

### Integration Points
- `cmd/root.go` is the persistent flag root — noContextFlag removal must happen there
- After deletions: run `go build ./...`, `go test ./...`, `go mod tidy`

</code_context>

<specifics>
## Specific Ideas

- STATE.md notes: "Dead packages (context/planner/retrieval/scanner) have no imports from any live cmd/ file except `inspect.go` importing `retrieval` — safe to delete after inspect rewrite"
- Since inspect.go will be fully rewritten in Phase 27, stub it in Phase 26 to unblock the build

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>
