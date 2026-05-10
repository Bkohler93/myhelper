# Phase 19: Search Gate & Injection - Research

**Researched:** 2026-04-11
**Domain:** Go CLI integration — LLM gate, result re-ranking, token-budget-aware injection
**Confidence:** HIGH

---

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
All implementation choices are at Claude's discretion — discuss phase was skipped per user setting. Use ROADMAP phase goal, success criteria, and codebase conventions to guide decisions.

Key constraints from success criteria:
- Gate: LLM yes/no "does this query need current web information?" — mirrors existing relevance gate in `internal/retrieval/retrieval.go`
- `--search` flag forces gate=true; `--no-search` flag forces gate=false
- Injected block: clearly delimited (`[WEB RESULTS]`), contains title + URL + snippet per result, fits token budget
- Re-rank failure: fall back to all results (not zero results)
- Zero relevant results after re-rank: skip injection entirely, LLM answers from own knowledge

### Claude's Discretion
All implementation details not covered by the success criteria above.

### Deferred Ideas (OUT OF SCOPE)
None — discuss phase skipped.
</user_constraints>

---

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| GATE-01 | Before responding, a yes/no LLM call determines whether the query needs current/real-time information — returns `true` or `false` | Mirror `needsContext()` in `internal/retrieval/retrieval.go` lines 155–169; use `ollama.Chat()` with a minimal single-message slice |
| GATE-02 | Gate fails open — if the gate call errors, search is SKIPPED (not triggered) | Opposite polarity from `needsContext()` which fails open to include context; gate must return false on error per GATE-02 |
| GATE-03 | `--search` flag forces search on, bypassing the gate | Cobra persistent flag on `rootCmd`; boolean `searchForce` package var; checked before gate call |
| GATE-04 | `--no-search` flag suppresses search entirely, bypassing the gate | Cobra persistent flag on `rootCmd`; boolean `searchSuppress` package var; checked before gate call |
| RANK-01 | Second LLM call filters fetched results — model returns indices/IDs of relevant results | New function `reRankResults()` in new `internal/searchgate` package; pass numbered result list in prompt |
| RANK-02 | Re-rank fails gracefully — errors or no valid indices → all fetched results injected | Same fallback pattern as `llmReRank()` in retrieval.go lines 289–295 |
| RANK-03 | Zero relevant results after re-rank → skip injection, model answers from own knowledge | Distinct from RANK-02: when re-rank returns an empty confirmed list (not an error), return nil |
| INJ-01 | Surviving snippets injected as `[WEB RESULTS]` delimited block in user message | Prepend block to user message content before `hist.Add()` / `initiateConversation` |
| INJ-02 | Injected results are token-budget-aware — snippets truncated or dropped to fit token limit | Use `history.New(threshold, msgs).TokenCount()` pattern from retrieval.go line 557; iterate results and drop when budget exceeded |
| INJ-03 | Injected block includes title and URL alongside each snippet | Format: `[1] Title\nURL\nSnippet\n\n` per result in `[WEB RESULTS]` block |
</phase_requirements>

---

## Summary

Phase 19 wires `internal/search` (built in Phase 18) into the chat path inside `cmd/root.go` and `cmd/helpers.go`. The work is almost entirely integration — no new packages needed unless code organization demands it. Three sub-problems map cleanly to existing codebase patterns:

**Search gate** mirrors `needsContext()` in `internal/retrieval/retrieval.go`. The key polarity difference: `needsContext` fails open (returns true on error because omitting code context is bad); the search gate must fail closed per GATE-02 (return false on error because triggering an unwanted network call is worse than skipping). The gate uses `ollama.Chat()` with a single-message prompt.

**Re-ranking** mirrors `llmReRank()` from retrieval.go. The fallback behavior matches RANK-02 (all results on error). RANK-03 adds a zero-result case that the retrieval re-ranker does not have: if re-rank succeeds but returns empty, skip injection entirely rather than falling back to all results.

**Injection** prepends a `[WEB RESULTS]` block to the user message content using the same `history.New(...).TokenCount()` token-budget pattern already established. Budget enforcement drops trailing results to fit.

**Primary recommendation:** Implement all logic in `cmd/root.go` / a new `cmd/search.go` file (not a new internal package). Keep the search pipeline functions close to the call site; they are not reused by any other package and the codebase convention is thin cmd-layer orchestration calling internal packages directly.

---

## Standard Stack

### Core (all already in go.mod — no new dependencies)
[VERIFIED: codebase grep]

| Library | Purpose | Why Standard |
|---------|---------|--------------|
| `internal/search` | SearXNG fetch, `Search(query, cfg)` | Built in Phase 18; direct import |
| `internal/ollama` | `ollama.Chat()` for gate and re-rank LLM calls | Already used for summarization in `summarize()` |
| `internal/history` | `history.New(threshold, msgs).TokenCount()` for budget | Established pattern in retrieval.go |
| `github.com/spf13/cobra` | `rootCmd.PersistentFlags().BoolVar()` for `--search`/`--no-search` | Already used in root.go |

### No New Dependencies
This phase adds zero new third-party dependencies. [VERIFIED: all required functionality is in existing packages]

---

## Architecture Patterns

### Where the Logic Lives

The search pipeline is invoked from `cmd/root.go`'s `RunE` function, specifically in two places:

1. **One-shot mode** (line 23–26 of root.go): after `hist.Add("user", args[0])`, before `initiateConversation`
2. **REPL mode** (line 137 of helpers.go): after `hist.Add("user", input)`, before `streamFn` call

The REPL path in `runConversationLoop` needs access to the flag state. The cleanest approach: add a `searchFn` parameter (or thread flag values through) rather than reaching for package-level globals. However, given the existing pattern where `runConversationLoop` already takes `streamFn` as a parameter, the natural extension is a `searchFn func(query string) ([]search.Result, error)` that encapsulates gate + fetch + re-rank, returning nil on skip.

**Alternative simpler approach:** Keep `runConversationLoop` signature unchanged. Add a new `buildUserMessage(query string, cfg config.Config, searchCfg search.Config, forceSearch, noSearch bool) string` helper that returns the (potentially augmented) user message string. Call it from both one-shot and REPL paths. This avoids changing function signatures and keeps the existing tests intact.

The "build augmented user message" approach is preferred because:
- `runConversationLoop` has 5 existing tests that would need updating if its signature changes
- The injection is purely a message-content transformation — it belongs as a pre-processing step before `hist.Add`
- Both code paths (one-shot and REPL) call the same helper

### Recommended Structure

```
cmd/
├── root.go        — add --search/--no-search flags; call buildUserMessage before hist.Add
├── helpers.go     — add searchGate(), reRankResults(), buildWebBlock(), buildUserMessage()
├── helpers_test.go — existing tests unchanged
└── search_gate_test.go  — new tests for gate, re-rank, injection, budget
```

No new internal package needed. [ASSUMED: could also go in a new `cmd/search.go` file for organization — same package either way]

### Pattern 1: Search Gate

Mirror of `needsContext()` with inverted fail-open/fail-closed behavior.

```go
// Source: mirrors internal/retrieval/retrieval.go:155-169

const searchGatePrompt = `Answer only "yes" or "no". Does answering the following query require current or real-time information from the internet? Query: `

// searchGate returns true if the query needs web search.
// Fails CLOSED (returns false on error) per GATE-02.
func searchGate(query string, cfg config.Config) bool {
    messages := []history.Message{
        {Role: "user", Content: searchGatePrompt + query},
    }
    response, err := ollama.Chat(cfg, messages)
    if err != nil {
        return false // fail closed: skip search on error
    }
    lower := strings.ToLower(strings.TrimSpace(response))
    return strings.HasPrefix(lower, "yes")
}
```

**Key difference from `needsContext`:** `needsContext` uses `!strings.HasPrefix(lower, "no")` (fail open). Search gate uses `strings.HasPrefix(lower, "yes")` (fail closed). This correctly implements GATE-02.

### Pattern 2: Re-ranking

```go
// Source: mirrors internal/retrieval/retrieval.go:266-296

const reRankPrompt = `You are a search result filter. Given the user's query and a numbered list of search results, output ONLY the numbers (1-based) of results that are genuinely relevant, one per line. Output nothing else.`

// reRankResults returns the subset of results that are relevant to the query.
// Returns (allResults, nil) on LLM error (RANK-02 fallback).
// Returns (nil, nil) when re-rank succeeds but finds zero relevant results (RANK-03).
func reRankResults(query string, results []search.Result, cfg config.Config) ([]search.Result, error) {
    if len(results) == 0 {
        return nil, nil
    }

    var sb strings.Builder
    for i, r := range results {
        fmt.Fprintf(&sb, "[%d] %s\n%s\n\n", i+1, r.Title, r.Snippet)
    }

    messages := []history.Message{
        {Role: "system", Content: reRankPrompt},
        {Role: "user", Content: "Query: " + query + "\n\nResults:\n" + sb.String()},
    }
    response, err := ollama.Chat(cfg, messages)
    if err != nil {
        return results, nil // RANK-02: error → all results
    }

    selected := filterByIndices(results, response)
    if len(selected) == 0 {
        return nil, nil // RANK-03: re-rank succeeded but found nothing → skip injection
    }
    return selected, nil
}
```

**Critical RANK-02 vs RANK-03 distinction:**
- `err != nil` → return all results (RANK-02: graceful degradation)
- `len(selected) == 0` after successful LLM call → return nil (RANK-03: skip injection entirely)

### Pattern 3: Index Parsing for Re-rank

```go
// filterByIndices parses "1\n3\n5" or "[1] [3]" style responses and returns
// matching results by 1-based index.
func filterByIndices(results []search.Result, response string) []search.Result {
    var selected []search.Result
    // Extract all integers from the response
    for _, field := range strings.Fields(response) {
        // Strip brackets/punctuation
        cleaned := strings.Trim(field, "[]().,")
        if idx, err := strconv.Atoi(cleaned); err == nil {
            if idx >= 1 && idx <= len(results) {
                selected = append(selected, results[idx-1])
            }
        }
    }
    return selected
}
```

**Why integer indices instead of URLs/titles (like stableIDs in retrieval):** Results don't have stable IDs. Integer indices are unambiguous and easier for small models to produce reliably.

### Pattern 4: Token-Budget-Aware Injection

```go
// Source: mirrors internal/retrieval/retrieval.go:556-558 (tokenCount helper)

// buildWebBlock builds a [WEB RESULTS] block from results, dropping trailing
// results to stay within budgetTokens. Returns "" if nothing fits.
func buildWebBlock(results []search.Result, budgetTokens int, cfg config.Config) string {
    if len(results) == 0 || budgetTokens <= 0 {
        return ""
    }

    header := "[WEB RESULTS]\n"
    footer := "[/WEB RESULTS]\n\n"
    overhead := countTokens(header+footer, cfg)
    if overhead >= budgetTokens {
        return ""
    }

    var sb strings.Builder
    used := overhead
    for i, r := range results {
        entry := fmt.Sprintf("[%d] %s\n%s\n%s\n\n", i+1, r.Title, r.URL, r.Snippet)
        cost := countTokens(entry, cfg)
        if used+cost > budgetTokens {
            break
        }
        sb.WriteString(entry)
        used += cost
    }

    if sb.Len() == 0 {
        return "" // nothing fit even after header overhead
    }
    return header + sb.String() + footer
}

// countTokens mirrors the retrieval.tokenCount helper.
func countTokens(s string, cfg config.Config) int {
    return history.New(cfg.TokenThreshold, []history.Message{{Role: "user", Content: s}}).TokenCount()
}
```

### Pattern 5: Orchestration in buildUserMessage

```go
// buildUserMessage augments the user query with web search results if appropriate.
// Returns the original query unchanged when search is skipped.
func buildUserMessage(query string, cfg config.Config, searchCfg search.Config, forceSearch, noSearch bool) string {
    // GATE-04: --no-search suppresses search
    if noSearch {
        return query
    }

    // Determine whether to search
    doSearch := forceSearch // GATE-03: --search forces true
    if !doSearch {
        doSearch = searchGate(query, cfg) // GATE-01/GATE-02
    }
    if !doSearch {
        return query
    }

    // Fetch results
    results, err := search.Search(query, searchCfg)
    if err != nil || len(results) == 0 {
        return query // network error or empty: degrade gracefully
    }

    // Re-rank (RANK-01, RANK-02, RANK-03)
    ranked, _ := reRankResults(query, results, cfg)
    if ranked == nil {
        return query // RANK-03: zero relevant → skip
    }

    // Budget: use a fixed fraction of TokenThreshold for web context
    // Leave at least 50% of threshold for conversation history
    budget := cfg.TokenThreshold / 4 // 25% for web context is conservative
    block := buildWebBlock(ranked, budget, cfg)
    if block == "" {
        return query
    }
    return block + query
}
```

**Budget allocation decision (Claude's discretion):** The token threshold is shared between search injection and conversation history. Using 25% of `TokenThreshold` for web context is a reasonable default — it provides ~1000 tokens at the default 4100 threshold, enough for 3–5 search snippets. This is a discretionary choice and can be adjusted.

### Pattern 6: Flag Registration in root.go

```go
// Source: mirrors existing --no-context flag pattern

var (
    searchForce    bool
    searchSuppress bool
)

func init() {
    rootCmd.PersistentFlags().BoolVar(&searchForce, "search", false, "Force web search regardless of gate")
    rootCmd.PersistentFlags().BoolVar(&searchSuppress, "no-search", false, "Suppress web search entirely")
}
```

**Note on flag naming:** Cobra supports `--no-search` as a BoolVar flag name (hyphen in flag name is valid). [VERIFIED: `--no-context` already exists in this codebase pattern — need to verify it uses the same approach]

### Pattern 7: Wiring into One-Shot and REPL

**One-shot (root.go RunE):**
```go
// Load search config alongside main config
searchCfg := search.LoadConfig()
// Augment user message before adding to history
augmented := buildUserMessage(args[0], cfg, searchCfg, searchForce, searchSuppress)
hist.Add("user", augmented)
return initiateConversation(cfg, hist, ollama.StreamChat)
```

**REPL (helpers.go runConversationLoop):** The REPL loop has `hist.Add("user", input)` at line 137. The search augmentation needs to happen here. Two options:

Option A: Pass `searchFn func(string) string` into `runConversationLoop` — changes the signature and requires updating 5 existing tests.

Option B: Pass flag values + configs into `runConversationLoop` as additional parameters.

Option C: Add a `messagePreprocessor func(string) string` parameter — a general-purpose hook that is nil-safe (identity function when nil). **Recommended:** this is the cleanest extension point and doesn't hardcode search-specific params into the loop signature.

```go
func runConversationLoop(
    cfg config.Config,
    hist *history.History,
    streamFn func(config.Config, []history.Message) (string, error),
    summarizePrompt string,
    recondensePrompt string,
    preprocessor func(string) string, // NEW: nil = identity
) error {
    // ...
    // At line 137, replace hist.Add("user", input) with:
    msg := input
    if preprocessor != nil {
        msg = preprocessor(input)
    }
    hist.Add("user", msg)
    // ...
}
```

Existing callers pass `nil` preprocessor. One-shot mode doesn't use `runConversationLoop` at all. This is the lowest-impact change. [ASSUMED: Option C is the cleanest; other approaches are valid]

**Existing test update required:** All 5 `runConversationLoop` calls in `helpers_test.go` and `chat_test.go` must add a trailing `nil` argument.

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Token counting | Custom byte/word counter | `history.New(threshold, msgs).TokenCount()` | Established pattern using tiktoken cl100k_base; already in codebase |
| HTTP fetch | Raw `http.Get` with manual parsing | `search.Search(query, cfg)` | Phase 18 built this with scheme normalization, error handling, result filtering |
| Non-streaming LLM call | Direct HTTP to Ollama | `ollama.Chat(cfg, messages)` | Already handles marshaling, non-200 errors, response decoding |
| Flag registration | Manual `os.Args` parsing | `rootCmd.PersistentFlags().BoolVar()` | Cobra already wired; mirrors existing `--no-context` pattern |

---

## Common Pitfalls

### Pitfall 1: Gate Polarity Inversion
**What goes wrong:** Copying `needsContext()` verbatim inverts the intended behavior — `!strings.HasPrefix(lower, "no")` fails open (triggers search on ambiguous responses) instead of failing closed per GATE-02.
**Why it happens:** `needsContext` omitting context is bad (fail open). Search gate triggering an unwanted network call is bad (fail closed).
**How to avoid:** Use `strings.HasPrefix(lower, "yes")` for the search gate — only trigger on a clear affirmative.
**Warning signs:** Test where LLM returns "I don't know" causes search to be triggered.

### Pitfall 2: RANK-02 vs RANK-03 Confusion
**What goes wrong:** Treating zero-result re-rank the same as re-rank error — either always falls back to all results, or always skips.
**Why it happens:** The distinction is subtle: error = uncertain, use all; zero results = LLM said nothing is relevant, skip.
**How to avoid:** Use a two-value return: `([]search.Result, error)`. `err != nil` → return all; empty slice + nil → return nil to signal skip.
**Warning signs:** `myhelper "what is the latest Go release?"` injects irrelevant web results after re-rank claimed none were relevant.

### Pitfall 3: Injection Position in Message
**What goes wrong:** Appending the `[WEB RESULTS]` block after the query instead of prepending it — model may not see the context before generating.
**Why it happens:** Natural string concatenation appends.
**How to avoid:** `block + query` not `query + block`. Context before the question matches the retrieval pattern (`assembleMessages` appends `## Query` last).
**Warning signs:** Model response doesn't cite web results even when injection succeeded.

### Pitfall 4: Budget Not Accounting for Existing History
**What goes wrong:** Allocating `TokenThreshold / 4` for web context without considering that the conversation history already consumes tokens — injection pushes total over the summarization threshold on the first message.
**Why it happens:** Token budget is computed in isolation from existing history size.
**How to avoid:** For one-shot mode this is fine (history has one message). For REPL mode, compute remaining budget as `cfg.TokenThreshold - hist.TokenCount()` and take a fraction of that. Or simply keep the web block allocation modest (25%) and accept that summarization handles overflow.
**Warning signs:** `[Condensing history...]` prints immediately after the first REPL response with web results.

### Pitfall 5: runConversationLoop Signature Change Breaks Existing Tests
**What goes wrong:** Adding parameters to `runConversationLoop` without updating all call sites — compile error.
**Why it happens:** The function is called from both `root.go` and indirectly tested via `helpers_test.go` and `chat_test.go`.
**How to avoid:** After adding the `preprocessor` parameter, grep for all call sites and add `nil` to each. There are at least 7 calls in tests plus 1 in root.go.
**Warning signs:** `go test ./cmd/...` fails with "too few arguments in call to runConversationLoop".

### Pitfall 6: search.LoadConfig() Called with No Config File
**What goes wrong:** Assuming search config is always loaded at startup — REPL turns after config has been loaded still need the same `searchCfg` value.
**Why it happens:** `search.LoadConfig()` reads env/files at call time. If called in a closure it works; if called once in `RunE`, must be passed to `runConversationLoop` (or captured in a closure).
**How to avoid:** Call `search.LoadConfig()` once in `RunE`, capture in the preprocessor closure. Do not call it per-turn.

---

## Code Examples

### Verified Token Count Pattern
```go
// Source: internal/retrieval/retrieval.go:556-558
func tokenCount(cfg config.Config, content string) int {
    return history.New(cfg.TokenThreshold, []history.Message{{Role: "user", Content: content}}).TokenCount()
}
```

### Verified ollama.Chat Usage (for gate/re-rank)
```go
// Source: cmd/helpers.go:177
summaryText, err := ollama.Chat(cfg, summarizeMessages)
if err != nil {
    return fmt.Errorf("summarize: %w", err)
}
```

### Verified Cobra BoolVar Flag Pattern
```go
// Source: existing pattern — need to confirm --no-context flag location
// rootCmd.PersistentFlags().BoolVar(&flagVar, "flag-name", false, "description")
```

### Verified search.Search Signature
```go
// Source: internal/search/search.go:92
func Search(query string, cfg Config) ([]Result, error)
// Result fields: Title string, URL string, Snippet string
```

### Verified search.LoadConfig Signature
```go
// Source: internal/search/search.go:44
func LoadConfig() Config
// Config field: Endpoint string (json:"search_endpoint")
```

---

## State of the Art

| Old Approach | Current Approach | Impact |
|--------------|------------------|--------|
| No web search | SearXNG via `internal/search` (Phase 18) | Phase 19 wires it into chat path |
| `needsContext()` fails open | Search gate fails closed | Different requirement: unwanted network call vs. missing code context |

---

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `buildUserMessage` helper in cmd package (not a new internal package) is the right home for gate+fetch+rerank+inject logic | Architecture Patterns | Low — same package either way; rename and move if desired |
| A2 | `messagePreprocessor func(string) string` parameter is the cleanest way to thread search into `runConversationLoop` | Pattern 7 | Low — alternative: pass cfg/flags directly; more parameters but more explicit |
| A3 | 25% of `TokenThreshold` (≈1025 tokens at default 4100) is a reasonable web block budget | Pattern 5 (buildUserMessage) | Low — can be adjusted without changing architecture |
| A4 | `filterByIndices` parsing integers from LLM response is more reliable than asking for URLs/titles from small models | Pattern 3 | Medium — if small model cannot reliably produce integers, re-rank fallback (all results) handles it |

---

## Open Questions (RESOLVED)

1. **Where does `--no-context` flag currently live?**
   - **RESOLVED:** `--no-context` is absent from the current codebase — removed during Phase 16/17 CLI cleanup. `root.go` is complete as read and contains no `--no-context` registration. Register `--search`/`--no-search` in a new `init()` block in `root.go` per Plan 02 Task 2 action.

2. **Does `runConversationLoop` preprocessor approach vs. alternative flag threading matter for testability?**
   - **RESOLVED:** Preprocessor closure approach chosen. All existing test call sites pass `nil` as the trailing argument — no test refactoring beyond appending `nil`. Seven call sites in `cmd/helpers_test.go` and one in `cmd/chat_test.go` are enumerated in Plan 02 Task 1.

---

## Environment Availability

| Dependency | Required By | Available | Notes |
|------------|-------------|-----------|-------|
| SearXNG instance at `http://192.168.0.9:8083` | `search.Search()` at runtime | Unknown (runtime) | Unit tests use `httptest.NewServer`; integration success criteria requires live instance |
| Ollama at `192.168.0.9:11434` | Gate and re-rank LLM calls | Unknown (runtime) | Same as existing chat path — if chat works, gate/re-rank work |

**Unit tests do not require live services** — both SearXNG and Ollama calls are injected via function parameters or httptest stubs.

---

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing stdlib |
| Config file | none (standard `go test`) |
| Quick run command | `go test ./cmd/... -run TestSearch` |
| Full suite command | `go test ./...` |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| GATE-01 | Gate calls LLM with gate prompt, returns bool | unit | `go test ./cmd/... -run TestSearchGate` | No — Wave 0 |
| GATE-02 | Gate returns false on LLM error | unit | `go test ./cmd/... -run TestSearchGate_FailClosed` | No — Wave 0 |
| GATE-03 | `--search` flag bypasses gate (forceSearch=true path) | unit | `go test ./cmd/... -run TestBuildUserMessage_ForceSearch` | No — Wave 0 |
| GATE-04 | `--no-search` flag returns query unchanged | unit | `go test ./cmd/... -run TestBuildUserMessage_NoSearch` | No — Wave 0 |
| RANK-01 | Re-rank LLM call returns filtered subset | unit | `go test ./cmd/... -run TestReRankResults` | No — Wave 0 |
| RANK-02 | Re-rank error returns all results | unit | `go test ./cmd/... -run TestReRankResults_ErrorFallback` | No — Wave 0 |
| RANK-03 | Re-rank returns zero → buildUserMessage returns bare query | unit | `go test ./cmd/... -run TestReRankResults_ZeroRelevant` | No — Wave 0 |
| INJ-01 | Injected block starts with `[WEB RESULTS]` | unit | `go test ./cmd/... -run TestBuildWebBlock` | No — Wave 0 |
| INJ-02 | Results dropped when budget exceeded | unit | `go test ./cmd/... -run TestBuildWebBlock_BudgetTrim` | No — Wave 0 |
| INJ-03 | Each entry contains title, URL, snippet | unit | `go test ./cmd/... -run TestBuildWebBlock_Fields` | No — Wave 0 |

### Sampling Rate
- **Per task commit:** `go test ./cmd/... -run TestSearch -run TestBuildWebBlock -run TestReRank -run TestBuildUserMessage`
- **Per wave merge:** `go test ./...`
- **Phase gate:** Full suite green before `/gsd-verify-work`

### Wave 0 Gaps
- [ ] `cmd/search_gate_test.go` — covers GATE-01 through INJ-03 (all 10 requirements)
- [ ] No framework install needed — Go stdlib testing already in use

---

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | — |
| V3 Session Management | no | — |
| V4 Access Control | no | — |
| V5 Input Validation | yes (low risk) | Query passed to `url.QueryEscape` in `search.Search()` — already handled in Phase 18 |
| V6 Cryptography | no | — |

### Known Threat Patterns

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Query injection into SearXNG URL | Tampering | `url.QueryEscape` in `search.Search()` — already implemented Phase 18 |
| LLM prompt injection via web snippets | Spoofing/Tampering | Snippets are injected as data (delimited block), not as instructions — low risk for local model use |

---

## Sources

### Primary (HIGH confidence — VERIFIED: codebase read)
- `internal/retrieval/retrieval.go` — `needsContext()`, `llmReRank()`, `tokenCount()`, `assembleMessages()` patterns
- `cmd/helpers.go` — `runConversationLoop()`, `initiateConversation()`, `summarize()` integration points
- `cmd/root.go` — flag registration site, `RunE` one-shot and REPL dispatch
- `internal/ollama/client.go` — `Chat()` signature for gate and re-rank
- `internal/history/history.go` — `TokenCount()`, `New()`, `Add()` for budget computation
- `internal/search/search.go` — `Search()`, `LoadConfig()`, `Result`, `Config` (Phase 18 output)
- `internal/config/config.go` — `Config` struct, `Load()` for main config

### Secondary (MEDIUM confidence)
- Phase 18 SUMMARY: confirmed `internal/search` API surface and decisions (url.QueryEscape, results with empty Title/URL dropped, DefaultSearchEndpoint includes scheme)

---

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — all packages verified by codebase read
- Architecture patterns: HIGH — all patterns derived from existing code, not assumed
- Pitfalls: HIGH — derived from reading actual code paths and test structure
- Token budget allocation (25%): MEDIUM — reasonable default, not derived from a hard requirement

**Research date:** 2026-04-11
**Valid until:** 2026-05-11 (stable codebase; no external API churn risk)
