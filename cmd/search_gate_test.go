package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bkohler93/myhelper/internal/config"
	"github.com/bkohler93/myhelper/internal/search"
)

// ollamaResponse builds a minimal non-streaming ollama /api/chat response body.
func ollamaResponse(content string) map[string]interface{} {
	return map[string]interface{}{
		"message": map[string]string{
			"role":    "assistant",
			"content": content,
		},
		"done": true,
	}
}

// TestSearchGate covers GATE-01 and GATE-02.
func TestSearchGate(t *testing.T) {
	t.Run("yes response returns true", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(ollamaResponse("yes"))
		}))
		defer ts.Close()

		cfg := config.Config{Endpoint: ts.URL}
		got := searchGate("what is the latest Go release?", cfg)
		if !got {
			t.Error("expected searchGate to return true for 'yes' response, got false")
		}
	})

	t.Run("error returns false", func(t *testing.T) {
		// Use a closed server so the HTTP call will fail.
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
		ts.Close() // close immediately to force an error

		cfg := config.Config{Endpoint: ts.URL}
		got := searchGate("what is the latest Go release?", cfg)
		if got {
			t.Error("expected searchGate to return false on error (fail-closed), got true")
		}
	})
}

// TestSearchGate_FailClosed covers GATE-02 edge: non-200 response also returns false.
func TestSearchGate_FailClosed(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	cfg := config.Config{Endpoint: ts.URL}
	got := searchGate("what is the weather today?", cfg)
	if got {
		t.Error("expected searchGate to return false on non-200 response (fail-closed), got true")
	}
}

// TestReRankResults covers RANK-01.
func TestReRankResults(t *testing.T) {
	t.Run("returns filtered subset on valid response", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(ollamaResponse("1\n3"))
		}))
		defer ts.Close()

		cfg := config.Config{Endpoint: ts.URL}
		results := []search.Result{
			{Title: "First", URL: "http://a.com", Snippet: "snippet a"},
			{Title: "Second", URL: "http://b.com", Snippet: "snippet b"},
			{Title: "Third", URL: "http://c.com", Snippet: "snippet c"},
		}
		got, err := reRankResults("go release", results, cfg)
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		if len(got) != 2 {
			t.Fatalf("expected 2 results, got %d", len(got))
		}
		if got[0].Title != "First" {
			t.Errorf("expected first result title 'First', got %q", got[0].Title)
		}
		if got[1].Title != "Third" {
			t.Errorf("expected second result title 'Third', got %q", got[1].Title)
		}
	})
}

// TestReRankResults_ErrorFallback covers RANK-02.
func TestReRankResults_ErrorFallback(t *testing.T) {
	// Closed server forces network error.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	ts.Close()

	cfg := config.Config{Endpoint: ts.URL}
	results := []search.Result{
		{Title: "A", URL: "http://a.com", Snippet: "snip a"},
		{Title: "B", URL: "http://b.com", Snippet: "snip b"},
	}
	got, err := reRankResults("query", results, cfg)
	if err != nil {
		t.Fatalf("expected nil error on LLM failure (RANK-02), got %v", err)
	}
	if len(got) != len(results) {
		t.Errorf("expected all %d results on error fallback, got %d", len(results), len(got))
	}
}

// TestReRankResults_ZeroRelevant covers RANK-03.
func TestReRankResults_ZeroRelevant(t *testing.T) {
	// LLM returns out-of-range indices — no valid matches.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ollamaResponse("0\n99"))
	}))
	defer ts.Close()

	cfg := config.Config{Endpoint: ts.URL}
	results := []search.Result{
		{Title: "A", URL: "http://a.com", Snippet: "snip a"},
		{Title: "B", URL: "http://b.com", Snippet: "snip b"},
	}
	got, err := reRankResults("query", results, cfg)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if got != nil {
		t.Errorf("expected nil result slice when zero relevant (RANK-03), got %v", got)
	}
}

// TestBuildWebBlock covers INJ-01 and INJ-03.
func TestBuildWebBlock(t *testing.T) {
	results := []search.Result{
		{Title: "Go 1.22 Released", URL: "https://go.dev/blog/go1.22", Snippet: "The Go team announces Go 1.22."},
		{Title: "Go 1.21 Notes", URL: "https://go.dev/doc/go1.21", Snippet: "Release notes for Go 1.21."},
	}
	cfg := config.Config{TokenThreshold: 10000}

	t.Run("block starts with [WEB RESULTS]", func(t *testing.T) {
		block := buildWebBlock(results, 10000, cfg)
		if block == "" {
			t.Fatal("expected non-empty block")
		}
		prefix := "[WEB RESULTS]"
		if len(block) < len(prefix) || block[:len(prefix)] != prefix {
			t.Errorf("expected block to start with %q, got %q", prefix, block[:min(len(block), 30)])
		}
	})

	t.Run("each entry contains title URL snippet", func(t *testing.T) {
		block := buildWebBlock(results, 10000, cfg)
		for _, r := range results {
			if !contains(block, r.Title) {
				t.Errorf("expected block to contain title %q", r.Title)
			}
			if !contains(block, r.URL) {
				t.Errorf("expected block to contain URL %q", r.URL)
			}
			if !contains(block, r.Snippet) {
				t.Errorf("expected block to contain snippet %q", r.Snippet)
			}
		}
	})
}

// TestBuildWebBlock_BudgetTrim covers INJ-02.
func TestBuildWebBlock_BudgetTrim(t *testing.T) {
	// 3 results, tight budget — only the first (or none) should fit.
	results := []search.Result{
		{Title: "Result One", URL: "http://one.example.com", Snippet: "Snippet for result one."},
		{Title: "Result Two", URL: "http://two.example.com", Snippet: "Snippet for result two."},
		{Title: "Result Three", URL: "http://three.example.com", Snippet: "Snippet for result three."},
	}
	cfg := config.Config{TokenThreshold: 4100}

	// Small budget: 20 tokens — likely only header overhead fits or nothing.
	block := buildWebBlock(results, 20, cfg)
	// Either block is empty (nothing fit) or it doesn't contain all results.
	if block != "" {
		if contains(block, "Result Two") && contains(block, "Result Three") {
			t.Error("expected budget trim to drop at least some results, but all 3 appear in block")
		}
	}
}

func TestBuildUserMessage_NoSearch(t *testing.T) {
	// noSearch=true must return query unchanged regardless of gate response
	result := buildUserMessage("what is the latest Go release?", config.Config{}, search.Config{}, false, true)
	if result != "what is the latest Go release?" {
		t.Errorf("expected query unchanged with noSearch=true, got %q", result)
	}
}

func TestBuildUserMessage_ForceSearch(t *testing.T) {
	// forceSearch=true bypasses gate; if search.Search fails (no server), returns query unchanged
	result := buildUserMessage("what is a goroutine?", config.Config{}, search.Config{Endpoint: "http://127.0.0.1:1"}, true, false)
	// search.Search will fail (no server at port 1) → graceful degrade → returns original query
	if result != "what is a goroutine?" {
		t.Errorf("expected query unchanged when search.Search errors, got %q", result)
	}
}

// contains checks whether substr appears in s.
func contains(s, substr string) bool {
	return len(substr) > 0 && len(s) >= len(substr) && indexOf(s, substr) >= 0
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
