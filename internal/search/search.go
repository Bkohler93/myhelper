package search

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const DefaultSearchEndpoint = "http://192.168.0.9:8083"
const DefaultTavilyEndpoint = "https://api.tavily.com/search"

var httpClient = &http.Client{Timeout: 15 * time.Second}

// Config holds the resolved search configuration.
type Config struct {
	Endpoint       string `json:"search_endpoint"`
	Provider       string `json:"search_provider"` // "tavily" | "searxng"
	// TavilyKey holds the Tavily Bearer token. Do not log or serialize to user-facing output.
	TavilyKey      string `json:"tavily_key"`
	TavilyEndpoint string `json:"tavily_endpoint"` // overridable for tests; defaults to DefaultTavilyEndpoint
}

// MarshalJSON redacts TavilyKey to prevent accidental key exposure in logs or output.
func (c Config) MarshalJSON() ([]byte, error) {
	type Alias Config
	return json.Marshal(&struct {
		Alias
		TavilyKey string `json:"tavily_key"`
	}{
		Alias:     Alias(c),
		TavilyKey: "[REDACTED]",
	})
}

// Result represents a single search result.
type Result struct {
	Title   string
	URL     string
	Snippet string
}

// Internal SearXNG JSON response types.
type searxResponse struct {
	Results []searxResult `json:"results"`
}

type searxResult struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Content string `json:"content"`
}

// Internal Tavily JSON response types.
type tavilyResponse struct {
	Results []tavilyResult `json:"results"`
}

type tavilyResult struct {
	Title   string  `json:"title"`
	URL     string  `json:"url"`
	Content string  `json:"content"`
	Score   float64 `json:"score"`
}

// LoadConfig resolves the search configuration with the following precedence (highest to lowest):
//  1. MYHELPER_SEARCH_ENDPOINT and MYHELPER_TAVILY_KEY environment variables
//  2. local .myhelper/config.json
//  3. ~/.config/myhelper/config.json
//  4. DefaultSearchEndpoint / no Tavily key
//
// Provider auto-selection: if search_provider is not explicitly set in config,
// Provider is set to "tavily" when TavilyKey is non-empty, otherwise "searxng".
// An explicit search_provider in config always overrides auto-selection.
func LoadConfig() Config {
	cfg := Config{Endpoint: DefaultSearchEndpoint}

	if loaded, ok := loadConfigFile(homeConfigPath()); ok {
		if loaded.Endpoint != "" {
			cfg.Endpoint = loaded.Endpoint
		}
		if loaded.Provider != "" {
			cfg.Provider = loaded.Provider
		}
		if loaded.TavilyKey != "" {
			cfg.TavilyKey = loaded.TavilyKey
		}
		if loaded.TavilyEndpoint != "" {
			cfg.TavilyEndpoint = loaded.TavilyEndpoint
		}
	}
	if loaded, ok := loadConfigFile(localConfigPath()); ok {
		if loaded.Endpoint != "" {
			cfg.Endpoint = loaded.Endpoint
		}
		if loaded.Provider != "" {
			cfg.Provider = loaded.Provider
		}
		if loaded.TavilyKey != "" {
			cfg.TavilyKey = loaded.TavilyKey
		}
		if loaded.TavilyEndpoint != "" {
			cfg.TavilyEndpoint = loaded.TavilyEndpoint
		}
	}

	if v := os.Getenv("MYHELPER_SEARCH_ENDPOINT"); v != "" {
		cfg.Endpoint = v
	}
	if v := os.Getenv("MYHELPER_TAVILY_KEY"); v != "" {
		cfg.TavilyKey = v
	}
	if v := os.Getenv("MYHELPER_SEARCH_PROVIDER"); v != "" {
		cfg.Provider = v
	}

	// Auto-select provider only when not explicitly set in config.
	// This block runs after env vars so that a key supplied via env still triggers auto-selection.
	if cfg.Provider == "" {
		if cfg.TavilyKey != "" {
			cfg.Provider = "tavily"
		} else {
			cfg.Provider = "searxng"
		}
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

// searxngSearch queries the SearXNG instance at cfg.Endpoint and returns parsed results.
// Results with an empty Title or URL are dropped from the returned slice.
// Returns nil, err on network failure or non-200 response.
func searxngSearch(query string, cfg Config) ([]Result, error) {
	endpoint := cfg.Endpoint
	if !strings.HasPrefix(endpoint, "http://") && !strings.HasPrefix(endpoint, "https://") {
		endpoint = "http://" + endpoint
	}

	reqURL := strings.TrimRight(endpoint, "/") + "/search?q=" + url.QueryEscape(query) + "&format=json&pageno=1&num_results=10"

	resp, err := httpClient.Get(reqURL)
	if err != nil {
		return nil, fmt.Errorf("GET %s: %w", reqURL, err)
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

	out := make([]Result, 0, len(sr.Results))
	for _, r := range sr.Results {
		if r.Title == "" || r.URL == "" {
			continue
		}
		out = append(out, Result{
			Title:   r.Title,
			URL:     r.URL,
			Snippet: r.Content,
		})
	}
	return out, nil
}

// tavilySearch queries the Tavily API and returns parsed results.
// Uses POST with Bearer token authentication.
// TavilyKey is the Bearer token; never interpolated into URL or body — kept in header only.
// Results with an empty Title or URL are dropped from the returned slice.
// Returns nil, err on network failure or non-200 response.
func tavilySearch(query string, cfg Config) ([]Result, error) {
	if cfg.TavilyKey == "" {
		return nil, fmt.Errorf("tavily provider selected but TavilyKey is not configured")
	}

	endpoint := cfg.TavilyEndpoint
	if endpoint == "" {
		endpoint = DefaultTavilyEndpoint
	}

	body, err := json.Marshal(map[string]any{
		"query":       query,
		"max_results": 10,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal tavily request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	// TavilyKey is the Bearer token; never interpolated into URL or body — kept in header only.
	req.Header.Set("Authorization", "Bearer "+cfg.TavilyKey)

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("POST tavily: %w", err)
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

// Search routes the query to the configured provider.
// When cfg.Provider is "tavily", Tavily is used. All other values route to SearXNG.
func Search(query string, cfg Config) ([]Result, error) {
	if cfg.Provider == "tavily" {
		return tavilySearch(query, cfg)
	}
	return searxngSearch(query, cfg)
}
