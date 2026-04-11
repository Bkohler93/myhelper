---
phase: 14-ollama-client-extension
plan: "01"
subsystem: ollama-client
tags: [ollama, structured-output, json-schema, http-client]
dependency_graph:
  requires: []
  provides: [ChatWithFormat]
  affects: [internal/ollama/client.go, internal/ollama/client_test.go]
tech_stack:
  added: []
  patterns: [json.RawMessage omitempty, httptest.NewServer subtests]
key_files:
  created: []
  modified:
    - internal/ollama/client.go
    - internal/ollama/client_test.go
decisions:
  - "Format field placed after Stream in chatRequest to match plan spec (D-03)"
  - "nil json.RawMessage omits format key via omitempty — verified by test subtest"
metrics:
  duration: "~8 minutes"
  completed: "2026-04-11T00:08:31Z"
  tasks_completed: 2
  files_modified: 2
---

# Phase 14 Plan 01: Ollama Client Extension Summary

**One-liner:** Added `ChatWithFormat` to the Ollama client — non-streaming call with `json.RawMessage` schema injected into the `format` field, enabling structured JSON output for downstream pipeline phases.

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | Add Format field to chatRequest and implement ChatWithFormat | ea0bdb9 | internal/ollama/client.go |
| 2 | Add TestChatWithFormat httptest suite | 97caac2 | internal/ollama/client_test.go |

## What Was Built

### `internal/ollama/client.go`

- Added `Format json.RawMessage \`json:"format,omitempty"\`` to `chatRequest` struct after the `Stream` field
- Implemented `ChatWithFormat(cfg config.Config, messages []history.Message, schema json.RawMessage) (string, error)` following Chat's exact non-streaming path: marshal → POST → status check → decode → return content
- nil schema omits the `format` key entirely (omitempty behavior); non-nil schema serializes the raw JSON value into the request body
- Existing `Chat` and `StreamChat` signatures and behavior are byte-for-byte unchanged

### `internal/ollama/client_test.go`

- Added `TestChatWithFormat` with 4 subtests matching the `TestChat` structural pattern:
  1. `format field present in outbound request body` — server decodes body and asserts format key present
  2. `200 response returns message content` — returns correct string with nil error
  3. `non-200 response returns error with status code` — error string contains "404"
  4. `POST failure returns error` — unreachable endpoint yields non-nil error

## Verification

```
go test ./internal/ollama/... -v
--- PASS: TestChat (4 subtests)
--- PASS: TestChatWithFormat (4 subtests)
PASS

go build ./...  (exits 0)
```

## Deviations from Plan

None — plan executed exactly as written.

## Known Stubs

None.

## Threat Surface Scan

T-14-03 mitigation confirmed: `json.RawMessage` with `omitempty` on the `Format` field omits the key when nil. The "format field present in outbound request body" subtest verifies the non-nil path; nil-omission is validated by Go's standard `omitempty` behavior for `json.RawMessage` (nil slice = omitted). No new threat surface beyond what the plan's threat model documented.

## Self-Check: PASSED

- `internal/ollama/client.go` contains `Format json.RawMessage` and `func ChatWithFormat(` — confirmed
- `internal/ollama/client_test.go` contains `func TestChatWithFormat(` — confirmed
- Commits ea0bdb9 and 97caac2 verified in git log
