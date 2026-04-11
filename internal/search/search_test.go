package search_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
		// Ensure env var is unset
		t.Setenv("MYHELPER_SEARCH_ENDPOINT", "")
		cfg := search.LoadConfig()
		if cfg.Endpoint != search.DefaultSearchEndpoint {
			t.Errorf("expected default endpoint %q, got %q", search.DefaultSearchEndpoint, cfg.Endpoint)
		}
	})

	t.Run("env_var_overrides_default", func(t *testing.T) {
		t.Setenv("MYHELPER_SEARCH_ENDPOINT", "http://custom:9999")
		cfg := search.LoadConfig()
		if cfg.Endpoint != "http://custom:9999" {
			t.Errorf("expected endpoint 'http://custom:9999', got %q", cfg.Endpoint)
		}
	})
}

// containsString is a local helper to avoid importing strings in the test file.
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
