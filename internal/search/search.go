package search

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

const DefaultSearchEndpoint = "http://192.168.0.9:8083"

// Config holds the resolved SearXNG endpoint.
type Config struct {
	Endpoint string `json:"search_endpoint"`
}

// Result represents a single search result returned by SearXNG.
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

// LoadConfig resolves the SearXNG endpoint with the following precedence (highest to lowest):
//  1. MYHELPER_SEARCH_ENDPOINT environment variable
//  2. local .myhelper/config.json (search_endpoint key)
//  3. ~/.config/myhelper/config.json (search_endpoint key)
//  4. DefaultSearchEndpoint
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

// Search queries the SearXNG instance at cfg.Endpoint and returns parsed results.
// Results with an empty Title or URL are dropped from the returned slice.
// Returns nil, err on network failure or non-200 response.
func Search(query string, cfg Config) ([]Result, error) {
	endpoint := cfg.Endpoint
	if !strings.HasPrefix(endpoint, "http://") && !strings.HasPrefix(endpoint, "https://") {
		endpoint = "http://" + endpoint
	}

	reqURL := endpoint + "/search?q=" + url.QueryEscape(query) + "&format=json&pageno=1&num_results=10"

	resp, err := http.Get(reqURL)
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
