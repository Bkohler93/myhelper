# Phase 22: Search Pipeline Spinners - Context

**Gathered:** 2026-04-24
**Status:** Ready for planning
**Mode:** Auto-generated (discuss skipped via workflow.skip_discuss)

<domain>
## Phase Boundary

Users see a loading spinner during each async wait in the search pipeline so the tool feels responsive instead of silently blocking. Three wait points: SearXNG fetch, LLM gate call, LLM re-rank call.

</domain>

<decisions>
## Implementation Decisions

### Claude's Discretion
All implementation choices are at Claude's discretion — discuss phase was skipped per user setting.

Key facts to respect:
- **No Bubble Tea in the codebase** — go.mod has only tiktoken-go, cobra, and their transitive deps. Do NOT add charmbracelet/bubbletea or charmbracelet/bubbles as dependencies.
- **Use a lightweight goroutine-based spinner** — channel-based goroutine with a 100ms ticker, rotating frames, writes to stderr. Zero new dependencies.
- **Spinner prints to stderr** — stdout is reserved for the model's streaming response.
- **Three wait points in cmd/search.go:**
  1. `searchGate(query, cfg)` — LLM gate call (line ~117 in buildUserMessage)
  2. `search.Search(query, searchCfg)` — SearXNG HTTP fetch (line ~123 in buildUserMessage)
  3. `reRankResults(query, results, cfg)` — LLM re-rank call (line ~128 in buildUserMessage)
- **Spinner labels:**
  - Gate: "Checking if web search is needed..."
  - Fetch: "Fetching web results..."
  - Re-rank: "Filtering results..."
- **Spinner clears itself** on stop (writes spaces to overwrite, then CR) so it doesn't pollute stdout or leave artifacts on the line.
- **Place spinner logic in cmd/search.go** as a package-private helper. Do NOT modify internal/search/ — spinners are a cmd-layer concern.
- **The spinner is only active during the search pipeline** — model streaming output is unaffected.

</decisions>

<code_context>
## Existing Code Insights

### Reusable Assets
- `cmd/search.go:buildUserMessage` — the three wait points all live here
- `cmd/search.go:searchGate` — wraps `ollama.Chat` for gate
- `cmd/search.go:reRankResults` — wraps `ollama.Chat` for re-rank
- `internal/search/client.go:Search` — SearXNG HTTP fetch

### Established Patterns
- stderr for progress output (consistent with `readInteractive` using `fmt.Fprint(os.Stderr, prompt)`)
- Goroutines + channels used in conversation loop (cmd/helpers.go runConversationLoop)

### Integration Points
- Only `cmd/search.go` needs modification
- Spinner helper lives in `cmd/search.go` or a new `cmd/spinner.go`

</code_context>

<specifics>
## Specific Ideas

Spinner struct pattern:
```go
type spinner struct{ stop chan struct{} }

func startSpinner(label string) *spinner {
    s := &spinner{stop: make(chan struct{})}
    go func() {
        frames := []rune{'|', '/', '-', '\\'}
        i := 0
        t := time.NewTicker(100 * time.Millisecond)
        defer t.Stop()
        for {
            fmt.Fprintf(os.Stderr, "\r%c %s", frames[i], label)
            i = (i + 1) % len(frames)
            select {
            case <-s.stop:
                fmt.Fprintf(os.Stderr, "\r%s\r", strings.Repeat(" ", len(label)+3))
                return
            case <-t.C:
            }
        }
    }()
    return s
}

func (s *spinner) done() { close(s.stop) }
```

Usage in buildUserMessage:
```go
sp := startSpinner("Checking if web search is needed...")
doSearch = searchGate(query, cfg)
sp.done()
```

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>
