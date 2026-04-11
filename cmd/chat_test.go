package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bkohler93/myhelper/internal/config"
	"github.com/bkohler93/myhelper/internal/history"
)

// TestRootCmd_OneShot verifies that one-shot mode (single positional arg)
// calls the stream function exactly once and produces user+assistant history.
func TestRootCmd_OneShot(t *testing.T) {
	fs := &fakeStream{responses: []string{"mutex is a synchronization primitive"}}

	cfg := config.Config{}
	hist := history.New(4000, nil)
	hist.Add("user", "what is a mutex?")

	err := initiateConversation(cfg, hist, fs.call)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if fs.called != 1 {
		t.Fatalf("expected streamFn called once, got %d", fs.called)
	}
	msgs := hist.Messages()
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages (user+assistant), got %d", len(msgs))
	}
	if msgs[0].Role != "user" {
		t.Errorf("expected first message role 'user', got %q", msgs[0].Role)
	}
	if msgs[1].Role != "assistant" {
		t.Errorf("expected second message role 'assistant', got %q", msgs[1].Role)
	}
}

// TestRootCmd_REPL verifies that REPL mode calls streamFn once before
// the user types "quit" to exit.
func TestRootCmd_REPL(t *testing.T) {
	fs := &fakeStream{responses: []string{"hello reply"}}
	hist := history.New(4000, nil)
	restore := replaceStdin(t, "hello\nquit\n")
	defer restore()

	err := runConversationLoop(config.Config{}, hist, fs.call, "", "", nil)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if fs.called != 1 {
		t.Fatalf("expected streamFn called once (one turn before quit), got %d", fs.called)
	}
}

// TestRootCmd_NoSystemPrompt verifies that no system-role message is ever
// passed to the stream function in one-shot mode.
func TestRootCmd_NoSystemPrompt(t *testing.T) {
	var capturedMessages []history.Message
	captureStream := func(cfg config.Config, msgs []history.Message) (string, error) {
		capturedMessages = make([]history.Message, len(msgs))
		copy(capturedMessages, msgs)
		return "response", nil
	}

	cfg := config.Config{}
	hist := history.New(4000, nil)
	hist.Add("user", "what is a mutex?")

	err := initiateConversation(cfg, hist, captureStream)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	for _, m := range capturedMessages {
		if m.Role == "system" {
			t.Errorf("expected no system messages, but found one: %+v", m)
		}
	}
}

// TestSummarize_NoSystemPrompt verifies the new summarize behavior:
// history with 4 messages [user, assistant, user, assistant] (NO system prompt at index 0)
// compresses to 3 messages [system(summary), last_user, last_assistant].
func TestSummarize_NoSystemPrompt(t *testing.T) {
	// Spin up an httptest server that returns a well-formed ollama non-streaming response.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp := map[string]interface{}{
			"message": map[string]string{
				"role":    "assistant",
				"content": "condensed summary",
			},
			"done": true,
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	cfg := config.Config{Endpoint: ts.URL}

	hist := history.New(4000, []history.Message{
		{Role: "user", Content: "first question"},
		{Role: "assistant", Content: "first answer"},
		{Role: "user", Content: "second question"},
		{Role: "assistant", Content: "second answer"},
	})

	err := summarize(cfg, hist, summarizePrompt, recondensePrompt)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	msgs := hist.Messages()
	if len(msgs) != 3 {
		t.Fatalf("expected 3 messages after summarization, got %d: %+v", len(msgs), msgs)
	}

	if msgs[0].Role != "system" {
		t.Errorf("expected first message role 'system' (summary), got %q", msgs[0].Role)
	}
	if msgs[1].Role != "user" {
		t.Errorf("expected second message role 'user', got %q", msgs[1].Role)
	}
	if msgs[2].Role != "assistant" {
		t.Errorf("expected third message role 'assistant', got %q", msgs[2].Role)
	}
}
