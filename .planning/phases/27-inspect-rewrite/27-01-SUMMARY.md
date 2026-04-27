---
phase: 27-inspect-rewrite
plan: 01
status: complete
completed: 2026-04-26
commit: 466c40b
---

# Phase 27-01 Summary: Inspect Rewrite

## What Was Built

Rewrote `cmd/inspect.go` from a stub to a full web search diagnostic dry-run. The command runs the search pipeline (gate → fetch → re-rank → build block) in diagnostic mode, printing each stage's output without sending any message to the chat model.

## Implementation

- **Single file changed:** `cmd/inspect.go` (109 insertions, 3 deletions)
- **No new packages:** reuses `reRankResults`, `buildWebBlock`, `searchGatePrompt`, `searchForce`, `searchSuppress` directly from the same `cmd` package
- **Gate call:** `ollama.Chat` called directly (not `searchGate`) to capture and print the raw LLM response
- **Token counting:** `history.New(...).TokenCount()` on the injected block string

## Output Sections

```
---
[Gate Decision]
Gate: YES/NO/BYPASSED (--search flag)
<raw LLM response>

---
[Fetched Results]
[1] Title
URL
Snippet

---
[Re-rank]
Survivors (N):
Dropped (N):

---
[Injected Block]
[WEB RESULTS]
...
[/WEB RESULTS]

Token cost: N tokens
```

## Flag Behavior

| Flag | Behavior |
|------|----------|
| `--no-search` | Prints "search suppressed", exits immediately (INSP-07) |
| `--search` | Prints "Gate: BYPASSED", skips gate LLM, runs full pipeline (INSP-06) |
| (default) | Calls gate LLM; NO → "search not needed" exit; YES → full pipeline |

## Requirements Satisfied

INSP-01, INSP-02, INSP-03, INSP-04, INSP-05, INSP-06, INSP-07

## Smoke Test Results

All three live tests passed:
- `--no-search` → "search suppressed" (single line, immediate exit)
- `--search` → full pipeline with all four section headers and token cost line
- Normal gate → gate runs, full pipeline on YES
