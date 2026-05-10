# Phase 29: Tavily Search Provider - Research

**Researched:** 2026-05-09
**Domain:** Go HTTP client, search provider abstraction, config extension
**Confidence:** HIGH

## Summary

Phase 29 adds Tavily as a second search provider alongside the existing SearXNG client in `internal/search/search.go`. The task is self-contained: extend `internal/search` with a Tavily HTTP client, widen the config struct to carry provider selection and the Tavily API key, and update `cmd/root.go` (and `cmd/search.go`) to pass the unified config through the existing `buildUserMessage` call site.

The Tavily REST API is well-documented and straightforward: one `POST https://api.tavily.com/search` endpoint, Bearer token auth, JSON body with `query` and optional `max_results`, JSON response with a `results` array whose items have `title`, `url`, and `content` fields. The shape maps directly onto the existing `search.Result` struct — no adapter gymnastics required.

The existing codebase uses two separate config structs (`config.Config` for Ollama and `search.Config` for SearXNG). The cleanest approach is to extend `search.Config` (adding `Provider`, `TavilyKey`) and introduce a dispatch function `Search(query, cfg)` that routes to `searxngSearch` or `tavilySearch` based on `cfg.Provider`. This keeps the `cmd/search.go` call site (`search.Search(query, searchCfg)`) unchanged.

**Primary recommendation:** Extend `search.Config` with provider/key fields, introduce a provider-dispatching `Search` that delegates to `searxngSearch` (existing logic) or a new `tavilySearch`, and thread Tavily key resolution into `search.LoadConfig` following the same env-var-over-file precedence pattern used throughout the project.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Tavily HTTP call | `internal/search` | — | All provider I/O lives in the search package, mirroring SearXNG |
| Provider selection logic | `internal/search` (LoadConfig) | — | Config resolution is package-private; cmd layer only reads the result |
| Config field storage | `~/.config/myhelper/config.json` | `.myhelper/config.json` | Follows existing file-priority pattern |
| Env var override | `internal/search` (LoadConfig) | — | Same layer where `MYHELPER_SEARCH_ENDPOINT` is applied |
| Call-site wiring | `cmd/root.go` / `cmd/search.go` | — | `buildUserMessage` already receives `search.Config`; no structural change needed |

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
All implementation choices are at Claude's discretion — discuss phase was skipped per user setting.

### Claude's Discretion
Use ROADMAP phase goal, success criteria, and codebase conventions to guide all decisions.

**Success criteria to satisfy:**
1. User with a Tavily API key in config gets Tavily search results instead of SearXNG by default
2. User can set `MYHELPER_TAVILY_KEY` env var to provide their Tavily key, overriding config
3. User can switch between Tavily and SearXNG by changing `search_provider` in config.json
4. User with no Tavily key and a SearXNG endpoint continues to get SearXNG results unchanged

### Deferred Ideas (OUT OF SCOPE)
None — discuss phase skipped.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| SRCH-01 | User can use Tavily as the search provider by configuring a Tavily API key in config or via env var; Tavily is the default provider when a key is present | `tavilySearch` function + `LoadConfig` key resolution; when `TavilyKey != ""`, set `Provider = "tavily"` automatically |
| SRCH-02 | User can switch search provider between Tavily and SearXNG via `search_provider` field in config.json | `search.Config.Provider` JSON field; dispatch in `Search()` checks this field |
| SRCH-03 | User can provide their Tavily API key via `MYHELPER_TAVILY_KEY` environment variable, which takes precedence over config | `LoadConfig` applies `os.Getenv("MYHELPER_TAVILY_KEY")` after file layer, identical pattern to `MYHELPER_SEARCH_ENDPOINT` |
</phase_requirements>

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `net/http` (stdlib) | go 1.24.2 | HTTP POST to Tavily API | Already used for SearXNG — no new dependency |
| `encoding/json` (stdlib) | go 1.24.2 | Encode request body, decode response | Already used throughout `internal/search` |
| `os` (stdlib) | go 1.24.2 | Read `MYHELPER_TAVILY_KEY` env var | Existing pattern in `LoadConfig` |

[VERIFIED: codebase — go.mod shows go 1.24.2; existing search.go already imports these three]

### Supporting

None required. No external Go SDK for Tavily is needed; the REST API is simple enough to call with `net/http` directly. Third-party Tavily Go clients exist (`github.com/lwileczek/tavily`, `github.com/jzf21/tavily-client`) but adding a dependency for a single POST request would be inconsistent with the project's minimal-dependency philosophy.

[ASSUMED] — the project has no explicit "no-new-deps" policy, but the existing pattern of using only stdlib for HTTP calls makes this the right default.

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| stdlib `net/http` POST | `github.com/lwileczek/tavily` SDK | SDK adds a dep for code that is ~30 lines; overkill |
| Extending `search.Config` | New `TavilyConfig` struct | Separate struct complicates `LoadConfig` call site; unified struct keeps `buildUserMessage` signature unchanged |
| `LoadConfig` returning provider | `cmd` layer deciding provider | Business logic belongs in the package that owns config; consistent with existing design |

**Installation:** No new packages required.

**Version verification:** No npm/go modules to add — stdlib only.

## Architecture Patterns

### System Architecture Diagram

```
cmd/root.go
  └── search.LoadConfig()  ──► reads ~/.config/myhelper/config.json
  │                             reads .myhelper/config.json
  │                             reads MYHELPER_SEARCH_ENDPOINT
  │                             reads MYHELPER_TAVILY_KEY          ← NEW
  │                             auto-selects provider              ← NEW
  │                             returns search.Config{Provider, TavilyKey, Endpoint}
  │
  └── buildUserMessage(query, cfg, searchCfg, ...)
        └── search.Search(query, searchCfg)  ← dispatch point (NEW)
              ├── Provider=="tavily"  → tavilySearch(query, searchCfg)
              │     POST https://api.tavily.com/search
              │     Authorization: Bearer <TavilyKey>
              │     Body: {"query":"...", "max_results":10}
              │     Returns []Result{Title, URL, Snippet}
              │
              └── Provider=="searxng" (or default)
                    → searxngSearch(query, searchCfg)   (existing logic, renamed)
                    GET <Endpoint>/search?q=...&format=json
                    Returns []Result{Title, URL, Snippet}
```

### Recommended Project Structure

No new files or directories required. All changes land in:

```
internal/search/
├── search.go          # extend Config, add tavilySearch, rename/wrap existing Search
└── search_test.go     # add Tavily tests alongside existing SearXNG tests
```

### Pattern 1: Provider-Dispatching Search Function

**What:** The exported `Search` function becomes a dispatcher; SearXNG logic moves to an unexported `searxngSearch`; Tavily logic is a new unexported `tavilySearch`.

**When to use:** Two implementations of the same interface without defining a formal interface type — consistent with Go's "keep it simple" idiom and the existing codebase style.

**Example:**
```go
// Source: [ASSUMED] — pattern derived from existing search.go style

func Search(query string, cfg Config) ([]Result, error) {
    if cfg.Provider == "tavily" {
        return tavilySearch(query, cfg)
    }
    return searxngSearch(query, cfg)
}

func searxngSearch(query string, cfg Config) ([]Result, error) {
    // existing Search() body, unchanged
}

func tavilySearch(query string, cfg Config) ([]Result, error) {
    body, _ := json.Marshal(map[string]any{
        "query":       query,
        "max_results": 10,
    })
    req, err := http.NewRequest(http.MethodPost, "https://api.tavily.com/search", bytes.NewReader(body))
    if err != nil {
        return nil, err
    }
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Authorization", "Bearer "+cfg.TavilyKey)

    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return nil, fmt.Errorf("tavily POST: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        b, _ := io.ReadAll(resp.Body)
        return nil, fmt.Errorf("tavily returned %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
    }

    var tr tavilyResponse
    if err := json.NewDecoder(resp.Body).Decode(&tr); err != nil {
        return nil, fmt.Errorf("decode tavily response: %w", err)
    }
    out := make([]Result, 0, len(tr.Results))
    for _, r := range tr.Results {
        if r.Title == "" || r.URL == "" {
            continue
        }
        out = append(out, Result{Title: r.Title, URL: r.URL, Snippet: r.Content})
    }
    return out, nil
}
```
[CITED: https://docs.tavily.com/documentation/api-reference/endpoint/search]

### Pattern 2: Config Extension with Auto-Provider Selection

**What:** `search.Config` gains `Provider string` and `TavilyKey string`. `LoadConfig` auto-sets `Provider = "tavily"` when a key is present and no explicit `search_provider` overrides it.

**When to use:** Whenever a new optional credential determines which provider is active — matches success criterion 1 ("key in config → Tavily by default").

**Example:**
```go
// Source: [ASSUMED] — pattern derived from existing LoadConfig in search.go

type Config struct {
    Endpoint  string `json:"search_endpoint"`
    Provider  string `json:"search_provider"`   // "tavily" | "searxng"
    TavilyKey string `json:"tavily_key"`
}

func LoadConfig() Config {
    cfg := Config{Endpoint: DefaultSearchEndpoint}

    // Layer: home config file
    if loaded, ok := loadConfigFile(homeConfigPath()); ok {
        applyFileConfig(&cfg, loaded)
    }
    // Layer: local config file (overrides home)
    if loaded, ok := loadConfigFile(localConfigPath()); ok {
        applyFileConfig(&cfg, loaded)
    }

    // Layer: env vars (highest priority)
    if v := os.Getenv("MYHELPER_SEARCH_ENDPOINT"); v != "" {
        cfg.Endpoint = v
    }
    if v := os.Getenv("MYHELPER_TAVILY_KEY"); v != "" {
        cfg.TavilyKey = v
    }

    // Auto-select provider: Tavily when key present and no explicit override
    if cfg.Provider == "" {
        if cfg.TavilyKey != "" {
            cfg.Provider = "tavily"
        } else {
            cfg.Provider = "searxng"
        }
    }

    return cfg
}
```

### Anti-Patterns to Avoid

- **Storing TavilyKey in `config.Config` (the Ollama struct):** Key belongs in `search.Config`; mixing concerns breaks the existing clean separation between chat and search configuration.
- **Adding a Tavily-specific exported function (`TavilySearch`):** Callers (`cmd/search.go`) use `search.Search`; adding a separate export requires changes to all call sites. Provider dispatch belongs inside the package.
- **Returning an error when no provider is configured:** Graceful degradation is the existing pattern. If neither key nor endpoint is usable, `Search` returns an empty slice — `buildUserMessage` already handles `len(results) == 0` gracefully.
- **Using `http.Get` for Tavily:** Tavily requires a POST with a JSON body and an Authorization header; `http.Get` cannot set these. Use `http.NewRequest` + `http.DefaultClient.Do`.
- **Setting `Provider` default to `""` at call time:** Defaulting to empty string means every call site would need a nil/empty check. Set the default in `LoadConfig` so `Provider` is always either `"tavily"` or `"searxng"`.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| HTTP client with auth header | Custom transport | `http.NewRequest` + `req.Header.Set("Authorization", ...)` | stdlib handles this natively in 3 lines |
| JSON request body | String concatenation | `json.Marshal(map[string]any{...})` | Type-safe, handles escaping automatically |
| Provider abstraction | Interface + registry pattern | Simple `if cfg.Provider == "tavily"` dispatch | Two providers don't warrant an interface; add interface only when a third provider arrives |

**Key insight:** The Tavily API is a single POST endpoint with a Bearer token — equivalent complexity to the existing SearXNG GET. No abstraction layer is needed beyond a private function.

## Common Pitfalls

### Pitfall 1: Config File Reads Both `search_endpoint` and `tavily_key`

**What goes wrong:** The `Config` struct is unmarshalled from JSON in `loadConfigFile`. If `TavilyKey` is stored in the same file but under the key `tavily_key`, the field must be tagged `json:"tavily_key"` for unmarshaling to work. Missing or wrong JSON tag means the key is silently ignored.

**Why it happens:** Go json tags must exactly match the JSON key name.

**How to avoid:** Use `json:"tavily_key"` and `json:"search_provider"` on the new fields. Test with a temp config file in `TestLoadConfig`.

**Warning signs:** `LoadConfig` returns `TavilyKey == ""` even when the config file contains the key.

### Pitfall 2: `http.Get` vs `http.NewRequest` for Tavily

**What goes wrong:** Using `http.Get` fails because Tavily is a POST endpoint requiring `Content-Type: application/json` and `Authorization: Bearer <key>` headers. `http.Get` supports none of these.

**Why it happens:** SearXNG uses a GET request; copy-paste error carries it to Tavily.

**How to avoid:** Use `http.NewRequest(http.MethodPost, ...)` + `http.DefaultClient.Do(req)`. Set both `Content-Type` and `Authorization` headers before calling `Do`.

**Warning signs:** `401 Unauthorized` or `400 Bad Request` from Tavily in tests/integration.

### Pitfall 3: Provider Auto-Selection Order

**What goes wrong:** If `Provider` is set in the config file to `"searxng"` but `MYHELPER_TAVILY_KEY` is set in the environment, which wins? Per success criterion 2, the env var provides the key but does NOT force the provider — the user can still explicitly set `search_provider: searxng` to keep SearXNG even with a key present.

**Why it happens:** The auto-selection logic (`if cfg.Provider == "" && cfg.TavilyKey != ""`) must run after env var application. If `Provider` was explicitly set in the config file, auto-selection is skipped.

**How to avoid:** Apply env vars first, then auto-select only when `cfg.Provider == ""`. Explicit `search_provider` in config always wins over auto-detection.

**Warning signs:** User sets `search_provider: searxng` in config but env var key causes Tavily to be used anyway.

### Pitfall 4: Empty Result Slice vs Error on 401

**What goes wrong:** A 401 (bad API key) from Tavily returns an error, which `buildUserMessage` silently swallows (degrades to no-search). This is the correct behavior for network failures but may hide key misconfiguration from the user.

**Why it happens:** `buildUserMessage` treats `err != nil` as "degrade gracefully" (existing pattern). The error is not surfaced to the user.

**How to avoid:** This is acceptable behavior for now — consistent with how SearXNG failures are handled. The user will notice no search results appear. Document this in code comments. Phase 30 (Setup Wizard) can add key validation.

**Warning signs:** User configured Tavily key with a typo; gets no error, just no search injection.

### Pitfall 5: `bytes` Import for Request Body

**What goes wrong:** `tavilySearch` needs `bytes.NewReader(body)` to wrap the marshalled JSON for `http.NewRequest`. Forgetting to import `bytes` causes a compile error.

**Why it happens:** The existing `search.go` does not import `bytes` (SearXNG uses a GET with URL params only). The import must be added.

**How to avoid:** Add `"bytes"` to the import block in `search.go` when writing `tavilySearch`.

## Code Examples

Verified patterns from official sources:

### Tavily Search POST Request
```go
// Source: https://docs.tavily.com/documentation/api-reference/endpoint/search
// POST https://api.tavily.com/search
// Content-Type: application/json
// Authorization: Bearer tvly-YOUR_API_KEY

// Request body:
// { "query": "...", "max_results": 10 }

// Response structure (relevant fields only):
type tavilyResponse struct {
    Results []tavilyResult `json:"results"`
}

type tavilyResult struct {
    Title   string  `json:"title"`
    URL     string  `json:"url"`
    Content string  `json:"content"`
    Score   float64 `json:"score"`
}
```
[CITED: https://docs.tavily.com/documentation/api-reference/endpoint/search]

### Tavily Error Codes to Handle
```
401 Unauthorized   → bad or missing API key
429 Too Many Requests → rate limited
432 Key/Plan Limit  → credit limit exceeded
500 Server Error   → treat as transient failure
```
[CITED: https://docs.tavily.com/documentation/api-reference/endpoint/search]

### Existing SearXNG Config Loading Pattern (reference for Tavily extension)
```go
// Source: internal/search/search.go (VERIFIED: codebase)
func LoadConfig() Config {
    cfg := Config{Endpoint: DefaultSearchEndpoint}
    // home file → local file → env var (ascending priority)
    if loaded, ok := loadConfigFile(homeConfigPath()); ok { ... }
    if loaded, ok := loadConfigFile(localConfigPath()); ok { ... }
    if v := os.Getenv("MYHELPER_SEARCH_ENDPOINT"); v != "" {
        cfg.Endpoint = v
    }
    return cfg
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| SearXNG only | SearXNG + Tavily (provider-selectable) | Phase 29 | Users without self-hosted SearXNG can now use Tavily instead |

**Deprecated/outdated:**
- Nothing — SearXNG support is retained; this is additive.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | No explicit "no new dependencies" policy — stdlib-only approach chosen for consistency | Standard Stack | Low: if a third-party Tavily SDK is desired, plan tasks change to `go get` + use SDK |
| A2 | `max_results: 10` matches the SearXNG `num_results=10` default — appropriate Tavily equivalent | Code Examples | Low: number is adjustable; worst case is slightly more/fewer results than SearXNG |
| A3 | Tavily key in config file stored as `tavily_key` JSON field | Architecture Patterns | Medium: field name must match what Phase 30 Setup Wizard writes; verify coordination with SETUP-05 |

## Open Questions

1. **JSON field name for Tavily key in config file**
   - What we know: REQUIREMENTS.md says Phase 30 wizard writes the key to `~/.config/myhelper/config.json`
   - What's unclear: The exact JSON field name — we're choosing `tavily_key` here; Phase 30 must use the same name
   - Recommendation: Document the field name in code comments; Phase 30 planner should reference Phase 29's `Config` struct

2. **Tavily rate limits**
   - What we know: 401/429/432 error codes exist; credit-based billing
   - What's unclear: Exact requests-per-minute limit on free tier
   - Recommendation: Handle all non-200 responses with the same graceful-degrade pattern as SearXNG; no retry logic needed for v5.0

## Environment Availability

Step 2.6: SKIPPED (no external tools or services required beyond stdlib `net/http`; Tavily is called at runtime via HTTP — no local service dependency)

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Framework | Go standard `testing` package |
| Config file | none (no go test config file in project) |
| Quick run command | `go test ./internal/search/...` |
| Full suite command | `go test ./...` |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| SRCH-01 | Tavily search returns results when key is configured | unit (httptest) | `go test ./internal/search/... -run TestTavilySearch` | ❌ Wave 0 |
| SRCH-02 | `search_provider: searxng` in config → SearXNG used; `search_provider: tavily` → Tavily used | unit | `go test ./internal/search/... -run TestSearch_ProviderDispatch` | ❌ Wave 0 |
| SRCH-03 | `MYHELPER_TAVILY_KEY` env var overrides config file key | unit | `go test ./internal/search/... -run TestLoadConfig_TavilyKeyEnvVar` | ❌ Wave 0 |
| SRCH-01 (fallback) | No key + SearXNG endpoint → SearXNG results unchanged | unit | `go test ./internal/search/... -run TestSearch_SearXNGFallback` | ❌ Wave 0 |

### Sampling Rate

- **Per task commit:** `go test ./internal/search/...`
- **Per wave merge:** `go test ./...`
- **Phase gate:** Full suite green before `/gsd-verify-work`

### Wave 0 Gaps

- [ ] `internal/search/search_test.go` — add Tavily test cases (file exists, needs new test functions)
- [ ] No new test files needed; extend existing `search_test.go`

*(Existing `TestLoadConfig` in `search_test.go` covers the `LoadConfig` base; new sub-tests extend it for Tavily fields)*

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | n/a — Tavily key is outbound-only credential |
| V3 Session Management | no | CLI tool, no sessions |
| V4 Access Control | no | single-user local CLI |
| V5 Input Validation | yes | query string passed to Tavily — already URL-safe via json.Marshal (not string concat) |
| V6 Cryptography | no | API key transmitted over HTTPS to api.tavily.com (TLS handled by stdlib) |

### Known Threat Patterns for this stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| API key leak via debug output | Information Disclosure | Never print `cfg.TavilyKey` to stdout/stderr; existing code never logs config values |
| API key in `.myhelper/config.json` (project-local) | Information Disclosure | `.myhelper/` should be in `.gitignore`; already the case per init artifacts pattern |
| Query injection into Tavily body | Tampering | `json.Marshal` escapes query correctly; no string interpolation used |

## Sources

### Primary (HIGH confidence)

- Tavily REST API — https://docs.tavily.com/documentation/api-reference/endpoint/search — endpoint URL, request fields, response schema, error codes
- `internal/search/search.go` [VERIFIED: codebase] — existing Config struct, LoadConfig pattern, Result struct, SearXNG client implementation
- `internal/config/config.go` [VERIFIED: codebase] — env-var-over-file precedence pattern
- `cmd/search.go` [VERIFIED: codebase] — buildUserMessage call site, search.Config usage
- `go.mod` [VERIFIED: codebase] — go 1.24.2, no existing Tavily dependency

### Secondary (MEDIUM confidence)

- WebSearch results confirming Tavily POST + Bearer auth pattern — cross-verified with official docs above

### Tertiary (LOW confidence)

- Third-party Tavily Go SDKs (`lwileczek/tavily`, `jzf21/tavily-client`) — found via WebSearch, not used (stdlib preferred)

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — stdlib only, verified against go.mod and existing code
- Architecture: HIGH — dispatch pattern derived directly from existing search.go; Tavily API verified from official docs
- Pitfalls: HIGH — derived from code reading + API doc review; not speculation

**Research date:** 2026-05-09
**Valid until:** 2026-06-09 (Tavily API; stable REST API unlikely to change in 30 days)
