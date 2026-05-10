---
phase: 22-search-pipeline-spinners
verified: 2026-04-24T20:30:00Z
status: human_needed
score: 4/5 must-haves verified
overrides_applied: 0
human_verification:
  - test: "Run `myhelper search 'what happened today'` against a live Ollama instance and SearXNG"
    expected: "Three spinners appear and clear sequentially — gate ('Checking if web search is needed...'), fetch ('Fetching web results...'), re-rank ('Filtering results...') — with no frame/label artifacts visible after each clears"
    why_human: "Goroutine timing, terminal escape-sequence rendering, and the race between the goroutine's first write and done() cannot be confirmed programmatically; only a live TTY run shows whether artifacts remain after the \r<spaces>\r clear"
---

# Phase 22: Search Pipeline Spinners Verification Report

**Phase Goal:** Users see a Bubble Tea loading spinner during each async wait in the search pipeline so the tool feels responsive instead of silently blocking.
**Verified:** 2026-04-24T20:30:00Z
**Status:** human_needed
**Re-verification:** No — initial verification

---

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | Spinner with label "Checking if web search is needed..." appears on stderr during searchGate LLM call and clears when it returns | VERIFIED | `cmd/search.go:146-148` — sp1 started before searchGate call, done() on next line after return |
| 2 | Spinner with label "Fetching web results..." appears on stderr during search.Search HTTP call and clears when it returns | VERIFIED | `cmd/search.go:154-156` — sp2 started before search.Search call, done() on next line after return |
| 3 | Spinner with label "Filtering results..." appears on stderr during reRankResults LLM call and clears when it returns | VERIFIED | `cmd/search.go:161-163` — sp3 started before reRankResults call, done() on next line after return |
| 4 | The terminal line is fully cleared after each spinner stops (no frame/label artifacts remain) | HUMAN NEEDED | `cmd/search.go:30` emits `\r<spaces>\r` (len(label)+3 spaces) — logic is correct but TTY rendering must be confirmed on a real terminal |
| 5 | go.mod is unchanged — no new dependencies introduced | VERIFIED | `git diff go.mod go.sum` returned empty; spinner uses only `fmt`, `os`, `strings`, `time` (all stdlib, confirmed in import block lines 4-8) |

**Score:** 4/5 truths auto-verified (Truth 4 requires human confirmation)

---

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `cmd/search.go` | spinner type + startSpinner helper + done method + 3 wired call sites in buildUserMessage | VERIFIED | `type spinner struct` at line 16; `func startSpinner` at line 18; `func (s *spinner) done()` at line 39; sp1/sp2/sp3 at lines 146, 154, 161 |

**Artifact depth checks:**
- **Level 1 (exists):** File present at `cmd/search.go`
- **Level 2 (substantive):** 175 lines; spinner struct is a real goroutine with a 100ms ticker loop, 4-frame charset `{'|', '/', '-', '\\'}`, channel-based stop signal
- **Level 3 (wired):** `startSpinner` called 3 times in `buildUserMessage`; `done()` called 3 times immediately after each blocking call, before any conditional check on the result
- **Level 4 (data flow):** Not applicable — spinner is a pure stderr UX artifact, not a data-rendering component

---

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| `buildUserMessage` (cmd/search.go) | `startSpinner` / `sp.done()` | inline wrap: sp := startSpinner(...); call; sp.done() | WIRED | grep confirms 3 startSpinner calls and 3 sp.done() calls; each done() is on the line immediately following the blocking call |

---

### Data-Flow Trace (Level 4)

Not applicable. The spinner is a UX signaling mechanism writing hardcoded label strings to stderr; it carries no user data and renders no dynamic content.

---

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| spinner type defined | `grep -c "type spinner struct" cmd/search.go` | 1 | PASS |
| startSpinner defined | `grep -c "func startSpinner" cmd/search.go` | 1 | PASS |
| done method defined | `grep -c "func (s \*spinner) done" cmd/search.go` | 1 | PASS |
| exactly 3 startSpinner call sites | `grep -c "startSpinner" cmd/search.go` (excluding definition) | 3 call sites (lines 146, 154, 161) | PASS |
| exactly 3 done() calls | `grep -c "sp.*\.done()" cmd/search.go` | 3 (lines 148, 156, 163) | PASS |
| project builds cleanly | `go build ./...` | exit 0, no output | PASS |
| go.mod unchanged | `git diff go.mod go.sum` | empty diff | PASS |
| terminal line-clear logic present | `grep "Repeat.*label" cmd/search.go` | line 30: `\r%s\r` with `strings.Repeat(" ", len(label)+3)` | PASS |
| live spinner appearance + clear | requires TTY | not runnable without server | SKIP — routed to human verification |

---

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|-------------|-------------|--------|----------|
| UX-01 | 22-01-PLAN.md | Spinner shown while SearXNG fetches; disappears when fetch completes | SATISFIED | sp2 wraps `search.Search` at lines 154-156; label "Fetching web results..." |
| UX-02 | 22-01-PLAN.md | Spinner shown while LLM search-gate call runs; disappears when decision returned | SATISFIED | sp1 wraps `searchGate` at lines 146-148; label "Checking if web search is needed..." |
| UX-03 | 22-01-PLAN.md | Spinner shown while LLM re-rank call runs; disappears when re-ranking completes | SATISFIED | sp3 wraps `reRankResults` at lines 161-163; label "Filtering results..." |

---

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| `internal/planner/planner_test.go` | 18, 43 | `TestParsePlan` references missing `14-01-PLAN.md` fixture | Info | Pre-existing failure documented in SUMMARY; not introduced by phase 22; confirmed by `git show 6ba67bb --stat` and `git show fe2a9eb --stat` showing only `cmd/search.go` was modified |

No stubs, placeholder returns, or empty handlers found in the phase 22 changes.

---

### Human Verification Required

#### 1. Spinner visual appearance and clean clearing

**Test:** Run `myhelper search "what is the latest Go release"` against a live Ollama instance (with SearXNG configured). Observe the terminal output.

**Expected:**
- The label "Checking if web search is needed..." appears on stderr with a rotating frame character (`|`, `/`, `-`, `\`) cycling every 100ms
- When searchGate returns, that line is wiped clean (no residual characters)
- The label "Fetching web results..." then appears, cycles, and clears identically
- The label "Filtering results..." then appears, cycles, and clears identically
- The final model response follows with no spinner artifacts on any preceding line

**Why human:** The goroutine writes `\r%c %s` to advance frames and clears with `\r<N spaces>\r` on stop. Whether this leaves artifacts depends on terminal emulator width, buffering, and race timing between the goroutine's last write and `done()` being called. Code inspection confirms the clear logic is correct; only a live TTY run confirms the user-visible outcome.

---

### Gaps Summary

No gaps blocking goal achievement. All three spinner call sites are implemented correctly with proper labels, correct ordering (start → call → done), and no-defer placement. The go.mod is unchanged confirming the zero-new-dependency constraint. The single open item (Truth 4 — terminal clearing confirmed on a live TTY) is a human-observable UX quality check, not a code deficiency.

---

_Verified: 2026-04-24T20:30:00Z_
_Verifier: Claude (gsd-verifier)_
