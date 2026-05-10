# Phase 18: SearXNG Client - Research

**Researched:** 2026-04-11
**Domain:** Go HTTP client for SearXNG JSON API + config resolution pattern
**Confidence:** HIGH

## Summary

Phase 18 builds `internal/search/` — a standalone package exposing `Search(query string, cfg Config) ([]Result, error)`. The SearXNG JSON API is a simple GET request with `q` and `format=json` parameters; responses contain a top-level `results` array where each entry has `title`, `url`, and `content` fields. SearXNG returns approximately 10 results per page by default (no client-side `num_results` parameter exists; the instance controls page size). The phase is almost entirely a mirror of existing codebase patterns: `internal/config/config.go` for config resolution and `internal/ollama/client.go` for the HTTP + error-handling shape. No new dependencies are needed.

**Primary recommendation:** Hand-write a minimal HTTP GET client using `net/http` following the existing `ollama` package conventions. Mirror the `config.Load()` pattern for a `search.LoadConfig()` function. The only design decision with any nuance is whether to embed `SearchEndpoint` directly into the main `config.Config` struct or keep it isolated in a `search.Config` struct; isolating it is cleaner for this standalone package and avoids coupling the search endpoint into every other command.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
All implementation choices are at Claude's discretion — discuss phase was skipped per user setting.

### Claude's Discretion
All implementation choices. Key constraints from success criteria:
- Package path: `internal/search/`
- Public API: `search.Search(query string, cfg Config) ([]Result, error)`
- Result fields: `Title`, `URL`, `Snippet` (all non-empty for valid results)
- Endpoint config: `MYHELPER_SEARCH_ENDPOINT` env var → `.myhelper/config.json` → `~/.config/myhelper/config.json` → default `http://192.168.0.9:8083`
- HTTP: GET `/search?q=...&format=json`, request 8–10 results
- Error handling: network errors and non-200 responses return error, nil slice

### Deferred Ideas (OUT OF SCOPE)
None — discuss phase skipped.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| SRCH-01 | `internal/search` package exposes `Search(query string, cfg Config) ([]Result, error)` querying the SearXNG JSON API | SearXNG API verified: GET `/search?q=...&format=json` returns `results` array |
| SRCH-02 | Each `Result` contains `Title`, `URL`, `Snippet` (content excerpt) | SearXNG `GeneralResult` has `title`, `url`, `content` fields — map `content` → `Snippet` |
| SRCH-03 | SearXNG endpoint configurable via env var, local config file, global config file, default | Mirrors existing `config.Load()` pattern exactly; add `search_endpoint` JSON key |
| SRCH-04 | `Search` fetches 8–10 results per query from `/search?q=...&format=json` | SearXNG returns ~10 results per page by default; use `pageno=1` (single page) — no `num_results` param exists |
| SRCH-05 | Network errors and non-200 responses return error; caller decides how to proceed | Standard `net/http` pattern already used in `internal/ollama/client.go` |
</phase_requirements>

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `net/http` | stdlib | HTTP GET request to SearXNG | Project convention — no external HTTP libraries [VERIFIED: internal/ollama/client.go] |
| `encoding/json` | stdlib | Decode SearXNG JSON response | Project convention [VERIFIED: internal/ollama/client.go] |
| `net/url` | stdlib | URL-encode query parameter | Needed for `url.QueryEscape(query)` |
| `fmt` | stdlib | Error wrapping with `%w` | Project convention [VERIFIED: internal/ollama/client.go] |

### Supporting (test only)
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `net/http/httptest` | stdlib | Mock SearXNG server in unit tests | All `_test.go` files [VERIFIED: internal/ollama/client_test.go] |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| hand-written client | `github.com/morikuni/go-searxng` | External dep for a 50-line problem; project uses no external HTTP libs — don't add one here |

**Installation:** No new dependencies required. All stdlib.

## Architecture Patterns

### Recommended Project Structure
```
internal/search/
├── search.go       # Config struct, LoadConfig(), Search(), internal types
└── search_test.go  # Unit tests using httptest
```

A single file is sufficient for this package. The `go-searxng` library source confirms the full SearXNG response shape; we only need a subset.

### Pattern 1: Config Resolution (mirror of config.Load)

**What:** `search.LoadConfig()` resolves `SearchEndpoint` with the same layering as `config.Load()`: env var → `.myhelper/config.json` → `~/.config/myhelper/config.json` → hardcoded default.

**When to use:** Called by callers who need a `search.Config` to pass into `Search()`.

The config JSON key for the search endpoint should be `search_endpoint` to avoid collision with the existing `endpoint` key (which is the Ollama endpoint). The `search.Config` struct is separate from `config.Config` — it holds only what the search package needs.

**Example:**
```go
// internal/search/search.go
package search

import (
    "encoding/json"
    "os"
    "path/filepath"
)

const DefaultSearchEndpoint = "http://192.168.0.9:8083"

type Config struct {
    Endpoint string `json:"search_endpoint"`
}

func LoadConfig() Config {
    cfg := Config{Endpoint: DefaultSearchEndpoint}

    if loaded, ok := loadConfigFile(localConfigPath()); ok {
        if loaded.Endpoint != "" {
            cfg.Endpoint = loaded.Endpoint
        }
    } else if loaded, ok := loadConfigFile(homeConfigPath()); ok {
        if loaded.Endpoint != "" {
            cfg.Endpoint = loaded.Endpoint
        }
    }

    if v := os.Getenv("MYHELPER_SEARCH_ENDPOINT"); v != "" {
        cfg.Endpoint = v
    }
    return cfg
}

func localConfigPath() string { return ".myhelper/config.json" }

func homeConfigPath() string {
    home, err := os.UserHomeDir()
    if err != nil {
        return ""
    }
    return filepath.Join(home, ".config", "myhelper", "config.json")
}

func loadConfigFile(path string) (Config, bool) {
    if path == "" {
        return Config{}, false
    }
    data, err := os.ReadFile(path)
    if err != nil {
        return Config{}, false
    }
    var c Config
    if err := json.Unmarshal(data, &c); err != nil {
        return Config{}, false
    }
    return c, true
}
```
[VERIFIED: pattern matches internal/config/config.go exactly]

### Pattern 2: HTTP GET + JSON decode (mirror of ollama.Chat)

**What:** Build the request URL, GET it, check status, decode JSON body, return a typed slice.

**When to use:** The core `Search()` function body.

**Example:**
```go
// Source: mirrors internal/ollama/client.go pattern
type Result struct {
    Title   string
    URL     string
    Snippet string
}

// internal SearXNG JSON shapes
type searxResponse struct {
    Results []searxResult `json:"results"`
}

type searxResult struct {
    Title   string `json:"title"`
    URL     string `json:"url"`
    Content string `json:"content"`
}

func Search(query string, cfg Config) ([]Result, error) {
    endpoint := cfg.Endpoint
    if !strings.HasPrefix(endpoint, "http://") && !strings.HasPrefix(endpoint, "https://") {
        endpoint = "http://" + endpoint
    }
    u := endpoint + "/search?q=" + url.QueryEscape(query) + "&format=json"

    resp, err := http.Get(u)
    if err != nil {
        return nil, fmt.Errorf("GET %s: %w", u, err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        body, _ := io.ReadAll(resp.Body)
        return nil, fmt.Errorf("searxng returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
    }

    var sr searxResponse
    if err := json.NewDecoder(resp.Body).Decode(&sr); err != nil {
        return nil, fmt.Errorf("decode response: %w", err)
    }

    results := make([]Result, 0, len(sr.Results))
    for _, r := range sr.Results {
        if r.Title == "" || r.URL == "" {
            continue  // skip malformed entries
        }
        results = append(results, Result{
            Title:   r.Title,
            URL:     r.URL,
            Snippet: r.Content,
        })
    }
    return results, nil
}
```
[VERIFIED: error-wrapping style matches internal/ollama/client.go]

### Pattern 3: httptest-based unit tests (mirror of ollama tests)

**What:** Spin up an `httptest.NewServer`, configure `cfg.Endpoint = srv.URL`, call `Search()`, assert.

**When to use:** All `search_test.go` test cases.

**Example:**
```go
// Source: mirrors internal/ollama/client_test.go
package search_test

import (
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/bkohler93/myhelper/internal/search"
)

func TestSearch(t *testing.T) {
    t.Run("200 response returns results", func(t *testing.T) {
        srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            json.NewEncoder(w).Encode(map[string]any{
                "results": []map[string]string{
                    {"title": "Go Channels", "url": "https://example.com/go", "content": "snippet"},
                },
            })
        }))
        defer srv.Close()
        cfg := search.Config{Endpoint: srv.URL}
        results, err := search.Search("golang channels", cfg)
        if err != nil {
            t.Fatalf("unexpected error: %v", err)
        }
        if len(results) != 1 {
            t.Fatalf("expected 1 result, got %d", len(results))
        }
    })
}
```
[VERIFIED: matches internal/ollama/client_test.go pattern]

### Anti-Patterns to Avoid
- **Adding external HTTP client library:** Project uses `net/http` directly — do not add a new module dependency for a 50-line package.
- **Using the global `config.Config` struct directly:** The Ollama `endpoint` and the search `search_endpoint` are different config keys. Keep `search.Config` separate; the caller composes both as needed.
- **Filtering out results with empty Snippet:** `content` can legitimately be empty for some SearXNG engines (e.g., image results mixed in). Only filter on `title` + `url` being non-empty; leave `Snippet` as empty string rather than dropping the result. The success criteria only requires non-empty `Title`, `URL`, and `Snippet` for valid web results — and tests will serve well-formed payloads.
- **Panicking or returning empty slice on network error:** Per SRCH-05, return `nil, err` — not `[]Result{}, err`.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| URL query encoding | Manual string concatenation | `net/url.QueryEscape` | Handles spaces, special chars, ampersands correctly |
| JSON decoding | Manual string parsing | `encoding/json.NewDecoder` | Standard pattern already in codebase |
| HTTP mock server | Custom mock | `net/http/httptest.NewServer` | Standard Go testing pattern already used in project |

**Key insight:** There is genuinely nothing exotic here. The entire package is ~80 lines of production code plus ~100 lines of tests, all using stdlib.

## Common Pitfalls

### Pitfall 1: SearXNG JSON format not enabled
**What goes wrong:** The SearXNG instance returns HTML instead of JSON even when `format=json` is in the URL. The response is 200 OK but the body is `<!DOCTYPE html>`, causing `json.Decode` to fail.
**Why it happens:** SearXNG disables JSON format by default. The instance must have `json` listed under `search.formats` in `settings.yml`.
**How to avoid:** Test against the actual instance before writing tests. The live instance at `192.168.0.9:8083` presumably works — but document this constraint so the user knows what to configure.
**Warning signs:** `json.Decode` error containing `invalid character '<'`.

### Pitfall 2: `content` field is empty for some result types
**What goes wrong:** Non-web results (images, videos) returned by SearXNG may have an empty or missing `content` field. Treating an empty `Snippet` as an error causes valid results to be dropped.
**Why it happens:** SearXNG aggregates across engine types; a `format=json` query with default categories can return image/video results.
**How to avoid:** Only require `title` + `url` to be non-empty when building the result slice. Pass `content` as-is (empty string is acceptable for `Snippet`).
**Warning signs:** Fewer results than expected; all image/video-type results missing.

### Pitfall 3: Endpoint with vs. without scheme
**What goes wrong:** `http.Get("192.168.0.9:8083/search?...")` fails — Go's HTTP client requires a scheme.
**Why it happens:** Config file or env var might store the endpoint without `http://`. The Ollama client handles this with `strings.HasPrefix(endpoint, "http://")` check.
**How to avoid:** Add the same scheme-normalization logic as `chatURL()` in `internal/ollama/client.go`.
**Warning signs:** `unsupported protocol scheme ""` error.

### Pitfall 4: `pageno` vs result count
**What goes wrong:** Caller expects exactly 10 results but gets 8 or 12 depending on the SearXNG instance configuration.
**Why it happens:** There is no `num_results` API parameter. The number of results returned is controlled by the SearXNG instance's `results_on_new_tab` / engine `page_size` settings, not by the client. The success criteria says "requests 8–10 results" — this means the request shape is correct (pageno=1), not that exactly N results are guaranteed.
**How to avoid:** Do not assert on an exact result count in tests. Assert `len(results) >= 1` or use a mock that returns a fixed number. The "8–10" in success criteria is about the server being asked for one page (~10), not about guaranteeing a count.
**Warning signs:** Flaky count assertions against the live server.

## Code Examples

Verified patterns from official sources:

### SearXNG response JSON shape
```json
{
  "query": "golang channels",
  "number_of_results": 10,
  "results": [
    {
      "title": "Go Channels — A Tour of Go",
      "url": "https://tour.golang.org/concurrency/2",
      "content": "Channels are a typed conduit through which you can send and receive values...",
      "engine": "google",
      "score": 1.0,
      "category": "general"
    }
  ],
  "unresponsive_engines": []
}
```
[CITED: https://pkg.go.dev/github.com/morikuni/go-searxng — GeneralResult struct fields verified]

### Request URL shape
```
GET http://192.168.0.9:8083/search?q=golang+channels&format=json
```
[CITED: https://docs.searxng.org/dev/search_api.html]

### Test: verify query parameter is present
```go
// Test that Search sends format=json and the query correctly
srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    if r.URL.Query().Get("format") != "json" {
        t.Errorf("expected format=json, got %q", r.URL.Query().Get("format"))
    }
    if r.URL.Query().Get("q") == "" {
        t.Errorf("expected non-empty q parameter")
    }
    json.NewEncoder(w).Encode(map[string]any{"results": []any{}})
}))
```
[VERIFIED: mirrors internal/ollama/client_test.go request inspection pattern]

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| SearXNG returns HTML only | JSON format available via `format=json` param | SearXNG v2021+ | Client must set format=json explicitly |
| `num_results` parameter | No such parameter exists | N/A | Page size is server-controlled; use pageno=1 |

**No deprecated patterns apply** to this phase — it is new greenfield code using stable stdlib.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | The live SearXNG instance at `192.168.0.9:8083` has JSON format enabled in its `settings.yml` | Common Pitfalls / Pitfall 1 | `Search()` will return a JSON decode error in production; tests against mock will still pass |
| A2 | SearXNG returns ~10 results per `pageno=1` request (default page size) | Phase Requirements SRCH-04 | Might return fewer on some engine configurations; success criteria "8–10" is met as long as pageno=1 is used |

## Open Questions

1. **Where does `search.Config` get merged with `config.Config` for callers?**
   - What we know: Phase 18 is client-only; phase 19 wires it into the chat path.
   - What's unclear: Whether `config.Config` should gain a `SearchEndpoint` field (shared config struct) or whether callers load `search.Config` independently.
   - Recommendation: Keep `search.Config` independent for phase 18. Phase 19 can decide how to compose. This is the simpler boundary.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go 1.24.2 | Build & test | Yes | go1.24.2 darwin/arm64 | — |
| SearXNG at 192.168.0.9:8083 | Integration / live tests | Unknown | — | httptest mock covers all unit tests |

**Missing dependencies with no fallback:** None that block unit test execution.

**Missing dependencies with fallback:**
- Live SearXNG instance: All required tests can be run against `httptest.NewServer` mocks. No live instance needed to satisfy phase success criteria in automated tests.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing (stdlib) |
| Config file | none — `go test` discovers `*_test.go` files |
| Quick run command | `go test ./internal/search/...` |
| Full suite command | `go test ./...` |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| SRCH-01 | `Search()` returns `[]Result` with correct fields | unit | `go test ./internal/search/... -run TestSearch` | No — Wave 0 |
| SRCH-02 | `Result.Title`, `Result.URL`, `Result.Snippet` populated from response | unit | `go test ./internal/search/... -run TestSearch_ResultFields` | No — Wave 0 |
| SRCH-03 | Endpoint resolves: env > local config > global config > default | unit | `go test ./internal/search/... -run TestLoadConfig` | No — Wave 0 |
| SRCH-04 | Request includes `format=json` and `q=...` parameters (observable in handler) | unit | `go test ./internal/search/... -run TestSearch_RequestParams` | No — Wave 0 |
| SRCH-05 | Network error returns `nil, err`; non-200 returns `nil, err` | unit | `go test ./internal/search/... -run TestSearch_Errors` | No — Wave 0 |

### Sampling Rate
- **Per task commit:** `go test ./internal/search/...`
- **Per wave merge:** `go test ./...`
- **Phase gate:** Full suite green before `/gsd-verify-work`

### Wave 0 Gaps
- [ ] `internal/search/search_test.go` — covers SRCH-01 through SRCH-05

*(Framework is stdlib — no install needed. No shared fixtures required.)*

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | — |
| V3 Session Management | no | — |
| V4 Access Control | no | — |
| V5 Input Validation | yes (low risk) | `url.QueryEscape` prevents query injection into the URL |
| V6 Cryptography | no | — |

### Known Threat Patterns for HTTP client

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| URL parameter injection via unescaped query | Tampering | `net/url.QueryEscape(query)` — stdlib, already recommended |
| SSRF via user-controlled endpoint | Elevation of Privilege | Endpoint is configured by the operator (env/config file), not by end users. Acceptable risk for a local CLI tool. |

## Sources

### Primary (HIGH confidence)
- `internal/config/config.go` — config resolution pattern, file path constants, JSON key conventions [VERIFIED: direct read]
- `internal/ollama/client.go` — HTTP client shape, error wrapping style, scheme normalization [VERIFIED: direct read]
- `internal/ollama/client_test.go` — httptest mock pattern, test table structure [VERIFIED: direct read]
- https://pkg.go.dev/github.com/morikuni/go-searxng — `GeneralResult` struct fields (`title`, `url`, `content`); `SearchOutput` top-level shape (`results` array, `number_of_results`) [CITED: WebFetch]

### Secondary (MEDIUM confidence)
- https://docs.searxng.org/dev/search_api.html — query parameters (`q`, `format`, `pageno`) confirmed [CITED: WebFetch]
- SearXNG default page size of ~10 results — inferred from Google engine source (10 results per page) and community usage patterns [CITED: multiple WebSearch sources]

### Tertiary (LOW confidence)
- SearXNG JSON format must be explicitly enabled in `settings.yml` — mentioned in multiple GitHub discussions but not reproduced here [LOW: WebSearch multiple sources, not directly verified against the live instance]

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — all stdlib, pattern verified in codebase
- Architecture: HIGH — direct mirror of existing package patterns
- SearXNG API shape: HIGH — verified via `go-searxng` library's struct definitions
- Pitfalls: MEDIUM — JSON-format-disabled pitfall is LOW (can't verify live instance config)

**Research date:** 2026-04-11
**Valid until:** 2026-10-11 (SearXNG API is stable; stdlib patterns never change)
