# Phase 29: Tavily Search Provider - Pattern Map

**Mapped:** 2026-05-09
**Files analyzed:** 2 (modified only — no new files required)
**Analogs found:** 2 / 2

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `internal/search/search.go` | service | request-response | `internal/search/search.go` (self — extend existing) | exact |
| `internal/search/search_test.go` | test | request-response | `internal/search/search_test.go` (self — extend existing) | exact |

No changes to `cmd/root.go` or `cmd/search.go` are required. `search.LoadConfig()` is already the only call site for config loading, and `search.Search()` is already the only call site in `buildUserMessage`. Provider dispatch is internal to the package.

---

## Pattern Assignments

### `internal/search/search.go` (service, request-response — extended)

**Analog:** `internal/search/search.go` (self) + `internal/config/config.go` (env-var-over-file pattern)

**Config struct extension** — current struct at lines 17-19, extend to:
```go
// current (lines 17-19):
type Config struct {
    Endpoint string `json:"search_endpoint"`
}

// extended form — add Provider and TavilyKey:
type Config struct {
    Endpoint  string `json:"search_endpoint"`
    Provider  string `json:"search_provider"` // "tavily" | "searxng"
    TavilyKey string `json:"tavily_key"`
}
```

**Internal response types pattern** — existing SearXNG types at lines 29-37, add Tavily types in the same style:
```go
// existing pattern (lines 29-37):
type searxResponse struct {
    Results []searxResult `json:"results"`
}
type searxResult struct {
    Title   string `json:"title"`
    URL     string `json:"url"`
    Content string `json:"content"`
}

// new Tavily types follow identical structure:
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

**LoadConfig env-var-over-file pattern** — existing pattern at lines 44-62 in `search.go`, extended for new fields:
```go
// existing pattern (lines 44-62):
func LoadConfig() Config {
    cfg := Config{Endpoint: DefaultSearchEndpoint}
    if loaded, ok := loadConfigFile(homeConfigPath()); ok {
        if loaded.Endpoint != "" {
            cfg.Endpoint = loaded.Endpoint
        }
    }
    if loaded, ok := loadConfigFile(localConfigPath()); ok {
        if loaded.Endpoint != "" {
            cfg.Endpoint = loaded.Endpoint
        }
    }
    if v := os.Getenv("MYHELPER_SEARCH_ENDPOINT"); v != "" {
        cfg.Endpoint = v
    }
    return cfg
}

// extended form — add Provider, TavilyKey fields and auto-selection block:
func LoadConfig() Config {
    cfg := Config{Endpoint: DefaultSearchEndpoint}
    if loaded, ok := loadConfigFile(homeConfigPath()); ok {
        if loaded.Endpoint != "" { cfg.Endpoint = loaded.Endpoint }
        if loaded.Provider != "" { cfg.Provider = loaded.Provider }
        if loaded.TavilyKey != "" { cfg.TavilyKey = loaded.TavilyKey }
    }
    if loaded, ok := loadConfigFile(localConfigPath()); ok {
        if loaded.Endpoint != "" { cfg.Endpoint = loaded.Endpoint }
        if loaded.Provider != "" { cfg.Provider = loaded.Provider }
        if loaded.TavilyKey != "" { cfg.TavilyKey = loaded.TavilyKey }
    }
    // env vars (highest priority — same layer ordering as config.Load() in internal/config/config.go lines 61-71)
    if v := os.Getenv("MYHELPER_SEARCH_ENDPOINT"); v != "" {
        cfg.Endpoint = v
    }
    if v := os.Getenv("MYHELPER_TAVILY_KEY"); v != "" {
        cfg.TavilyKey = v
    }
    // auto-select provider: only when no explicit search_provider was set in config
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

**Provider-dispatching Search function** — existing `Search` at lines 92-128 becomes `searxngSearch`; new `Search` dispatches:
```go
// existing Search (lines 92-128) — rename to searxngSearch, change signature to unexported:
func searxngSearch(query string, cfg Config) ([]Result, error) {
    // body unchanged from current Search()
}

// new exported dispatcher replaces the old Search:
func Search(query string, cfg Config) ([]Result, error) {
    if cfg.Provider == "tavily" {
        return tavilySearch(query, cfg)
    }
    return searxngSearch(query, cfg)
}
```

**tavilySearch function** — mirrors the error-handling and result-mapping structure of `searxngSearch` (lines 92-128), using POST + Bearer auth instead of GET:
```go
func tavilySearch(query string, cfg Config) ([]Result, error) {
    body, err := json.Marshal(map[string]any{
        "query":       query,
        "max_results": 10,
    })
    if err != nil {
        return nil, fmt.Errorf("marshal tavily request: %w", err)
    }
    req, err := http.NewRequest(http.MethodPost, "https://api.tavily.com/search", bytes.NewReader(body))
    if err != nil {
        return nil, err
    }
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Authorization", "Bearer "+cfg.TavilyKey)

    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return nil, fmt.Errorf("POST tavily: %w", err)
    }
    defer resp.Body.Close()

    // mirrors searxngSearch non-200 pattern (lines 106-109):
    if resp.StatusCode != http.StatusOK {
        b, _ := io.ReadAll(resp.Body)
        return nil, fmt.Errorf("tavily returned %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
    }

    var tr tavilyResponse
    if err := json.NewDecoder(resp.Body).Decode(&tr); err != nil {
        return nil, fmt.Errorf("decode tavily response: %w", err)
    }

    // mirrors searxngSearch result-mapping pattern (lines 116-127):
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

**Imports** — existing imports at lines 3-12 need `"bytes"` added (required by `bytes.NewReader`):
```go
import (
    "bytes"           // NEW — for bytes.NewReader in tavilySearch
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "net/url"
    "os"
    "path/filepath"
    "strings"
)
```

---

### `internal/search/search_test.go` (test, request-response — extended)

**Analog:** `internal/search/search_test.go` (self — extend existing test functions)

**httptest server pattern** — existing pattern at lines 13-31 (`TestSearch/returns_results_on_200`). All new Tavily tests use the same `httptest.NewServer` + `defer srv.Close()` structure. Key difference: Tavily uses POST, so test handlers must verify `r.Method == http.MethodPost` and `r.Header.Get("Authorization")`.

```go
// existing SearXNG httptest pattern (lines 13-31):
srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]any{
        "results": []map[string]string{
            {"title": "Go Tour", "url": "https://tour.golang.org", "content": "Learn Go"},
        },
    })
}))
defer srv.Close()
cfg := search.Config{Endpoint: srv.URL}
results, err := search.Search("golang channels", cfg)

// Tavily test handler — same structure, verify POST + auth header:
srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "expected POST", http.StatusMethodNotAllowed)
        return
    }
    if !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") {
        http.Error(w, "missing Bearer token", http.StatusUnauthorized)
        return
    }
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]any{
        "results": []map[string]string{
            {"title": "T", "url": "U", "content": "S"},
        },
    })
}))
defer srv.Close()
// Override the Tavily endpoint for the test server (requires tavilySearch to accept configurable URL):
cfg := search.Config{Provider: "tavily", TavilyKey: "test-key"}
```

**Note on testability:** Because `tavilySearch` hardcodes `https://api.tavily.com/search`, integration with `httptest` requires either (a) accepting a URL override via a package-level variable, (b) a functional option, or (c) using `http.NewRequest` with a configurable base URL injected via `Config`. The cleanest approach consistent with the existing `Config.Endpoint` pattern: add `TavilyEndpoint string` to `Config` that defaults to `"https://api.tavily.com/search"` when empty, allowing tests to override it with `srv.URL`. This is the same pattern as `Config.Endpoint` for SearXNG.

**TestLoadConfig env-var pattern** — existing pattern at lines 223-240. New sub-tests follow identical `t.Setenv` structure:
```go
// existing env var override test (lines 233-239):
t.Run("env_var_overrides_default", func(t *testing.T) {
    t.Setenv("MYHELPER_SEARCH_ENDPOINT", "http://custom:9999")
    cfg := search.LoadConfig()
    if cfg.Endpoint != "http://custom:9999" {
        t.Errorf("expected endpoint 'http://custom:9999', got %q", cfg.Endpoint)
    }
})

// new Tavily env var test follows same pattern:
t.Run("tavily_key_env_var_overrides_default", func(t *testing.T) {
    t.Setenv("MYHELPER_TAVILY_KEY", "tvly-testkey")
    t.Setenv("MYHELPER_SEARCH_ENDPOINT", "") // isolate
    cfg := search.LoadConfig()
    if cfg.TavilyKey != "tvly-testkey" {
        t.Errorf("expected TavilyKey 'tvly-testkey', got %q", cfg.TavilyKey)
    }
    if cfg.Provider != "tavily" {
        t.Errorf("expected Provider 'tavily' (auto-selected), got %q", cfg.Provider)
    }
})
```

**Imports for test file** — existing imports at lines 3-10. New test cases for Tavily POST need `"strings"`:
```go
import (
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "strings"   // NEW — for HasPrefix check on Authorization header
    "testing"

    "github.com/bkohler93/myhelper/internal/search"
)
```

---

## Shared Patterns

### Env-var-over-file Precedence
**Source:** `internal/config/config.go` lines 30-74 and `internal/search/search.go` lines 44-62
**Apply to:** `LoadConfig()` extension in `internal/search/search.go`

The precedence stack is: hardcoded default → home config file → local config file → env var. The local config file takes precedence over home (applied second, overrides home). Env var always wins (applied last). This exact ordering must be preserved when adding `MYHELPER_TAVILY_KEY`.

```go
// Pattern from internal/config/config.go lines 38-71:
// Layer 2: config files (CWD takes precedence over home dir)
if loaded, ok := loadFile(localConfigPath()); ok {
    // apply non-zero fields
} else if loaded, ok := loadFile(homeConfigPath()); ok {
    // apply non-zero fields
}
// Layer 1: environment variables (highest priority)
if v := os.Getenv("MYHELPER_ENDPOINT"); v != "" {
    cfg.Endpoint = v
}
```

Note: `search.LoadConfig` currently loads home-then-local (both layers always applied, not else-if), whereas `config.Load` uses else-if. The search package's home-then-local approach allows both files to contribute non-overlapping fields, and the local file wins on any field it sets. Keep this search-package convention; do not switch to else-if.

### Error Handling (network failures, non-200)
**Source:** `internal/search/search.go` lines 100-109
**Apply to:** `tavilySearch` in `internal/search/search.go`

```go
// existing pattern (lines 100-109):
resp, err := http.Get(reqURL)
if err != nil {
    return nil, fmt.Errorf("GET %s: %w", reqURL, err)
}
defer resp.Body.Close()
if resp.StatusCode != http.StatusOK {
    body, _ := io.ReadAll(resp.Body)
    return nil, fmt.Errorf("searxng returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
}
```

`tavilySearch` copies this structure exactly, swapping `http.Get` for `http.NewRequest`+`Do` and the error message prefix.

### Result Filtering (drop empty title/url)
**Source:** `internal/search/search.go` lines 116-127
**Apply to:** `tavilySearch` result mapping

```go
// existing pattern (lines 116-127):
out := make([]Result, 0, len(sr.Results))
for _, r := range sr.Results {
    if r.Title == "" || r.URL == "" {
        continue
    }
    out = append(out, Result{Title: r.Title, URL: r.URL, Snippet: r.Content})
}
return out, nil
```

`tavilySearch` uses identical guard and mapping — `r.Content` maps to `Snippet` in both cases.

### Graceful Degradation at Call Site
**Source:** `cmd/search.go` lines 151-154
**Apply to:** No change needed — already handles `err != nil` and `len(results) == 0`

```go
// existing pattern (lines 151-154):
results, err := search.Search(query, searchCfg)
sp2.done()
if err != nil || len(results) == 0 {
    return query // network/empty: degrade gracefully
}
```

Tavily errors (401, 429, 432, 500) propagate as `error` from `tavilySearch` and are caught here unchanged — no call-site modification needed.

---

## No Analog Found

None — all patterns are directly derived from existing code in `internal/search/search.go`, `internal/search/search_test.go`, and `internal/config/config.go`.

---

## Key Implementation Notes for Planner

1. **Testability requires `TavilyEndpoint` field in Config.** The hardcoded `https://api.tavily.com/search` must be overridable per the same pattern as `Config.Endpoint` for SearXNG. Default it to the constant when empty in `tavilySearch`. Add a `DefaultTavilyEndpoint = "https://api.tavily.com/search"` constant alongside `DefaultSearchEndpoint`.

2. **`bytes` import must be added to `search.go`.** The existing import block (lines 3-12) has no `"bytes"`. `bytes.NewReader(body)` in `tavilySearch` requires it.

3. **No changes to `cmd/root.go` or `cmd/search.go`** — `search.LoadConfig()` and `search.Search()` signatures are unchanged. Provider selection is entirely internal to the search package.

4. **JSON field names are load-bearing.** The `Config` struct JSON tags must be `json:"search_provider"` and `json:"tavily_key"` exactly — these are the keys Phase 30's Setup Wizard will write to `~/.config/myhelper/config.json`.

5. **Auto-selection runs after env vars** — the `if cfg.Provider == ""` block must execute after `MYHELPER_TAVILY_KEY` is applied, so an env-var-supplied key triggers Tavily auto-selection. An explicit `search_provider` in config always wins (it sets `Provider` to a non-empty string before auto-selection runs).

---

## Metadata

**Analog search scope:** `internal/search/`, `internal/config/`, `cmd/`
**Files scanned:** 5 (`search.go`, `search_test.go`, `root.go`, `search.go` (cmd), `config.go`)
**Pattern extraction date:** 2026-05-09
