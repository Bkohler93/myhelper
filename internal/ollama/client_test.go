package ollama_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/bkohler93/myhelper/internal/config"
	"github.com/bkohler93/myhelper/internal/history"
	"github.com/bkohler93/myhelper/internal/ollama"
)

func TestChat(t *testing.T) {
	t.Run("200 response returns message content", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"message": map[string]string{"role": "assistant", "content": "the summary"},
				"done":    true,
			})
		}))
		defer srv.Close()
		cfg := config.Config{Endpoint: srv.URL, Model: "testmodel"}
		result, err := ollama.Chat(cfg, []history.Message{{Role: "user", Content: "summarize"}})
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		if result != "the summary" {
			t.Errorf("expected 'the summary', got %q", result)
		}
	})

	t.Run("non-200 response returns error with status code", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "model not found", http.StatusNotFound)
		}))
		defer srv.Close()
		cfg := config.Config{Endpoint: srv.URL, Model: "testmodel"}
		_, err := ollama.Chat(cfg, []history.Message{{Role: "user", Content: "hi"}})
		if err == nil {
			t.Fatal("expected error on non-200 response")
		}
		if !strings.Contains(err.Error(), "404") {
			t.Errorf("expected error to contain '404', got %q", err.Error())
		}
	})

	t.Run("POST failure returns error", func(t *testing.T) {
		cfg := config.Config{Endpoint: "http://127.0.0.1:1", Model: "testmodel"}
		_, err := ollama.Chat(cfg, []history.Message{{Role: "user", Content: "hi"}})
		if err == nil {
			t.Fatal("expected error on unreachable endpoint")
		}
	})

	t.Run("nothing written to stdout during Chat", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(map[string]any{
				"message": map[string]string{"role": "assistant", "content": "silent"},
				"done":    true,
			})
		}))
		defer srv.Close()

		// Redirect stdout to /dev/null and verify no panic / no output.
		old := os.Stdout
		devNull, _ := os.Open(os.DevNull)
		os.Stdout = devNull
		defer func() {
			os.Stdout = old
			devNull.Close()
		}()

		cfg := config.Config{Endpoint: srv.URL, Model: "testmodel"}
		result, err := ollama.Chat(cfg, []history.Message{{Role: "user", Content: "hi"}})
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		if result != "silent" {
			t.Errorf("expected 'silent', got %q", result)
		}
	})
}
