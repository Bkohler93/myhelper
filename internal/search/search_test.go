package search_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bkohler93/myhelper/internal/search"
)

func TestSearch(t *testing.T) {
	t.Run("returns_results_on_200", func(t *testing.T) {
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
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		if len(results) < 1 {
			t.Errorf("expected at least 1 result, got %d", len(results))
		}
	})
}

func TestSearch_ResultFields(t *testing.T) {
	t.Run("title_url_snippet_populated", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"results": []map[string]string{
					{"title": "T", "url": "U", "content": "S"},
				},
			})
		}))
		defer srv.Close()
		cfg := search.Config{Endpoint: srv.URL}
		results, err := search.Search("test", cfg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if results[0].Title != "T" {
			t.Errorf("expected Title='T', got %q", results[0].Title)
		}
		if results[0].URL != "U" {
			t.Errorf("expected URL='U', got %q", results[0].URL)
		}
		if results[0].Snippet != "S" {
			t.Errorf("expected Snippet='S', got %q", results[0].Snippet)
		}
	})

	t.Run("skips_entry_missing_title", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"results": []map[string]string{
					{"title": "", "url": "U", "content": "S"},
				},
			})
		}))
		defer srv.Close()
		cfg := search.Config{Endpoint: srv.URL}
		results, err := search.Search("test", cfg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(results) != 0 {
			t.Errorf("expected 0 results (entry with empty title should be skipped), got %d", len(results))
		}
	})

	t.Run("skips_entry_missing_url", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"results": []map[string]string{
					{"title": "T", "url": "", "content": "S"},
				},
			})
		}))
		defer srv.Close()
		cfg := search.Config{Endpoint: srv.URL}
		results, err := search.Search("test", cfg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(results) != 0 {
			t.Errorf("expected 0 results (entry with empty url should be skipped), got %d", len(results))
		}
	})
}

func TestSearch_RequestParams(t *testing.T) {
	t.Run("format_json_present", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			format := r.URL.Query().Get("format")
			if format != "json" {
				http.Error(w, "expected format=json", http.StatusBadRequest)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{"results": []map[string]string{}})
		}))
		defer srv.Close()
		cfg := search.Config{Endpoint: srv.URL}
		_, err := search.Search("test", cfg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("q_param_present", func(t *testing.T) {
		var gotQ string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotQ = r.URL.Query().Get("q")
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{"results": []map[string]string{}})
		}))
		defer srv.Close()
		cfg := search.Config{Endpoint: srv.URL}
		_, err := search.Search("golang channels", cfg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if gotQ != "golang channels" {
			t.Errorf("expected q='golang channels', got %q", gotQ)
		}
	})

	t.Run("pageno_present", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			pageno := r.URL.Query().Get("pageno")
			if pageno != "1" {
				t.Errorf("expected pageno=1, got %q", pageno)
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{"results": []map[string]string{}})
		}))
		defer srv.Close()
		cfg := search.Config{Endpoint: srv.URL}
		_, err := search.Search("test", cfg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("result_count_present", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			numResults := r.URL.Query().Get("num_results")
			if numResults == "" {
				t.Errorf("expected num_results param in request URL, got empty")
			}
			if numResults != "10" {
				t.Errorf("expected num_results=10, got %q", numResults)
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{"results": []map[string]string{}})
		}))
		defer srv.Close()
		cfg := search.Config{Endpoint: srv.URL}
		_, err := search.Search("test", cfg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestSearch_Errors(t *testing.T) {
	t.Run("network_error_returns_nil_slice", func(t *testing.T) {
		cfg := search.Config{Endpoint: "http://127.0.0.1:1"}
		results, err := search.Search("test", cfg)
		if err == nil {
			t.Fatal("expected error on unreachable endpoint, got nil")
		}
		if results != nil {
			t.Errorf("expected nil results on network error, got %v", results)
		}
	})

	t.Run("non200_returns_nil_slice_and_error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "bad", http.StatusInternalServerError)
		}))
		defer srv.Close()
		cfg := search.Config{Endpoint: srv.URL}
		results, err := search.Search("test", cfg)
		if err == nil {
			t.Fatal("expected error on non-200 response")
		}
		if results != nil {
			t.Errorf("expected nil results on non-200, got %v", results)
		}
	})

	t.Run("non200_includes_status_in_error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "internal error", http.StatusInternalServerError)
		}))
		defer srv.Close()
		cfg := search.Config{Endpoint: srv.URL}
		_, err := search.Search("test", cfg)
		if err == nil {
			t.Fatal("expected error on non-200 response")
		}
		if !containsString(err.Error(), "500") {
			t.Errorf("expected error to contain '500', got %q", err.Error())
		}
	})
}

func TestLoadConfig(t *testing.T) {
	t.Run("default_when_no_env_no_file", func(t *testing.T) {
		t.Chdir(t.TempDir())
		// Ensure env var is unset
		t.Setenv("MYHELPER_SEARCH_ENDPOINT", "")
		cfg := search.LoadConfig()
		if cfg.Endpoint != search.DefaultSearchEndpoint {
			t.Errorf("expected default endpoint %q, got %q", search.DefaultSearchEndpoint, cfg.Endpoint)
		}
	})

	t.Run("env_var_overrides_default", func(t *testing.T) {
		t.Chdir(t.TempDir())
		t.Setenv("MYHELPER_SEARCH_ENDPOINT", "http://custom:9999")
		cfg := search.LoadConfig()
		if cfg.Endpoint != "http://custom:9999" {
			t.Errorf("expected endpoint 'http://custom:9999', got %q", cfg.Endpoint)
		}
	})
}

// containsString is a local helper to check substring membership.
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		func() bool {
			for i := 0; i <= len(s)-len(substr); i++ {
				if s[i:i+len(substr)] == substr {
					return true
				}
			}
			return false
		}())
}

func TestTavilySearch(t *testing.T) {
	t.Run("returns_results_on_200", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"results": []map[string]string{
					{"title": "T", "url": "U", "content": "S"},
				},
			})
		}))
		defer srv.Close()
		cfg := search.Config{Provider: "tavily", TavilyKey: "test-key", TavilyEndpoint: srv.URL}
		results, err := search.Search("golang", cfg)
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if results[0].Title != "T" {
			t.Errorf("expected Title='T', got %q", results[0].Title)
		}
		if results[0].URL != "U" {
			t.Errorf("expected URL='U', got %q", results[0].URL)
		}
		if results[0].Snippet != "S" {
			t.Errorf("expected Snippet='S', got %q", results[0].Snippet)
		}
	})

	t.Run("uses_POST_method", func(t *testing.T) {
		var gotMethod string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotMethod = r.Method
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{"results": []map[string]string{}})
		}))
		defer srv.Close()
		cfg := search.Config{Provider: "tavily", TavilyKey: "test-key", TavilyEndpoint: srv.URL}
		_, _ = search.Search("test", cfg)
		if gotMethod != http.MethodPost {
			t.Errorf("expected POST method, got %q", gotMethod)
		}
	})

	t.Run("sends_bearer_auth", func(t *testing.T) {
		var gotAuth string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotAuth = r.Header.Get("Authorization")
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{"results": []map[string]string{}})
		}))
		defer srv.Close()
		cfg := search.Config{Provider: "tavily", TavilyKey: "tvly-abc123", TavilyEndpoint: srv.URL}
		_, _ = search.Search("test", cfg)
		if !strings.HasPrefix(gotAuth, "Bearer ") {
			t.Errorf("expected Authorization header to start with 'Bearer ', got %q", gotAuth)
		}
		if gotAuth != "Bearer tvly-abc123" {
			t.Errorf("expected Authorization 'Bearer tvly-abc123', got %q", gotAuth)
		}
	})

	t.Run("skips_entry_missing_title", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"results": []map[string]string{
					{"title": "", "url": "U", "content": "S"},
				},
			})
		}))
		defer srv.Close()
		cfg := search.Config{Provider: "tavily", TavilyKey: "test-key", TavilyEndpoint: srv.URL}
		results, err := search.Search("test", cfg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(results) != 0 {
			t.Errorf("expected 0 results (empty title filtered), got %d", len(results))
		}
	})

	t.Run("skips_entry_missing_url", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"results": []map[string]string{
					{"title": "T", "url": "", "content": "S"},
				},
			})
		}))
		defer srv.Close()
		cfg := search.Config{Provider: "tavily", TavilyKey: "test-key", TavilyEndpoint: srv.URL}
		results, err := search.Search("test", cfg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(results) != 0 {
			t.Errorf("expected 0 results (empty url filtered), got %d", len(results))
		}
	})

	t.Run("non200_returns_error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
		}))
		defer srv.Close()
		cfg := search.Config{Provider: "tavily", TavilyKey: "bad-key", TavilyEndpoint: srv.URL}
		results, err := search.Search("test", cfg)
		if err == nil {
			t.Fatal("expected error on 401, got nil")
		}
		if results != nil {
			t.Errorf("expected nil results on non-200, got %v", results)
		}
		if !containsString(err.Error(), "401") {
			t.Errorf("expected error to contain '401', got %q", err.Error())
		}
	})
}

func TestSearch_ProviderDispatch(t *testing.T) {
	t.Run("tavily_provider_hits_tavily_endpoint", func(t *testing.T) {
		var tavilyHit bool
		tavSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tavilyHit = true
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{"results": []map[string]string{}})
		}))
		defer tavSrv.Close()

		cfg := search.Config{
			Provider:       "tavily",
			TavilyKey:      "test-key",
			TavilyEndpoint: tavSrv.URL,
		}
		_, _ = search.Search("test", cfg)
		if !tavilyHit {
			t.Error("expected Tavily endpoint to be contacted when Provider==\"tavily\"")
		}
	})

	t.Run("searxng_provider_hits_searxng_endpoint", func(t *testing.T) {
		var searxHit bool
		searxSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			searxHit = true
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{"results": []map[string]string{}})
		}))
		defer searxSrv.Close()

		cfg := search.Config{
			Provider: "searxng",
			Endpoint: searxSrv.URL,
		}
		_, _ = search.Search("test", cfg)
		if !searxHit {
			t.Error("expected SearXNG endpoint to be contacted when Provider==\"searxng\"")
		}
	})
}

func TestLoadConfig_TavilyKeyEnvVar(t *testing.T) {
	t.Run("env_var_sets_tavily_key", func(t *testing.T) {
		t.Chdir(t.TempDir())
		t.Setenv("MYHELPER_TAVILY_KEY", "tvly-testkey")
		t.Setenv("MYHELPER_SEARCH_ENDPOINT", "") // isolate from any real config
		cfg := search.LoadConfig()
		if cfg.TavilyKey != "tvly-testkey" {
			t.Errorf("expected TavilyKey 'tvly-testkey', got %q", cfg.TavilyKey)
		}
	})

	t.Run("auto_selects_tavily_when_key_present", func(t *testing.T) {
		t.Chdir(t.TempDir())
		t.Setenv("MYHELPER_TAVILY_KEY", "tvly-testkey")
		// Do not set MYHELPER_SEARCH_ENDPOINT so Provider remains empty (relying on auto-select)
		cfg := search.LoadConfig()
		// auto-select: TavilyKey non-empty + no explicit Provider → "tavily"
		if cfg.Provider != "tavily" {
			t.Errorf("expected Provider 'tavily' (auto-selected), got %q", cfg.Provider)
		}
	})

	t.Run("no_key_defaults_to_searxng", func(t *testing.T) {
		t.Chdir(t.TempDir())
		t.Setenv("MYHELPER_TAVILY_KEY", "")
		cfg := search.LoadConfig()
		if cfg.Provider != "searxng" {
			t.Errorf("expected Provider 'searxng' when no key, got %q", cfg.Provider)
		}
		if cfg.TavilyKey != "" {
			t.Errorf("expected empty TavilyKey, got %q", cfg.TavilyKey)
		}
	})
}
