package cmd

import (
	"io"
	"syscall"
	"testing"
	"time"

	"github.com/bkohler93/myhelper/internal/config"
	"github.com/bkohler93/myhelper/internal/history"
)

// fakeStream records calls and returns a canned response.
type fakeStream struct {
	called    int
	responses []string
	err       error
}

func (f *fakeStream) call(cfg config.Config, messages []history.Message) (string, error) {
	f.called++
	if f.err != nil {
		return "", f.err
	}
	if f.called-1 < len(f.responses) {
		return f.responses[f.called-1], nil
	}
	return "response", nil
}

// replaceStdin replaces the package-level stdinReader with a pipe backed by
// the given string and returns a restore function.
func replaceStdin(t *testing.T, input string) func() {
	t.Helper()
	r, w := io.Pipe()
	origStdin := stdinReader
	stdinReader = r
	go func() {
		io.WriteString(w, input)
		w.Close()
	}()
	return func() {
		stdinReader = origStdin
	}
}

// TestRunConversationLoop covers the five required behaviours.
func TestRunConversationLoop(t *testing.T) {

	t.Run("quit exits immediately with nil, no model call", func(t *testing.T) {
		fs := &fakeStream{}
		hist := history.New(4000, nil)
		restore := replaceStdin(t, "quit\n")
		defer restore()

		err := runConversationLoop(config.Config{}, hist, fs.call)
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		if fs.called != 0 {
			t.Fatalf("expected streamFn not called, called %d times", fs.called)
		}
	})

	t.Run("empty input reprints prompt without calling model", func(t *testing.T) {
		fs := &fakeStream{responses: []string{"response"}}
		hist := history.New(4000, nil)
		// First line empty (no model call), second line "quit" (exit)
		restore := replaceStdin(t, "\nquit\n")
		defer restore()

		err := runConversationLoop(config.Config{}, hist, fs.call)
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		if fs.called != 0 {
			t.Fatalf("expected streamFn not called for empty input, called %d times", fs.called)
		}
	})

	t.Run("normal turn: calls streamFn and appends messages to history", func(t *testing.T) {
		fs := &fakeStream{responses: []string{"assistant reply"}}
		hist := history.New(4000, nil)
		restore := replaceStdin(t, "hello\nquit\n")
		defer restore()

		err := runConversationLoop(config.Config{}, hist, fs.call)
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		if fs.called != 1 {
			t.Fatalf("expected streamFn called once, called %d times", fs.called)
		}
		msgs := hist.Messages()
		if len(msgs) != 2 {
			t.Fatalf("expected 2 messages in history, got %d", len(msgs))
		}
		if msgs[0].Role != "user" || msgs[0].Content != "hello" {
			t.Errorf("expected user message 'hello', got %+v", msgs[0])
		}
		if msgs[1].Role != "assistant" || msgs[1].Content != "assistant reply" {
			t.Errorf("expected assistant message 'assistant reply', got %+v", msgs[1])
		}
	})

	t.Run("streamFn error is returned", func(t *testing.T) {
		errExpected := io.EOF
		fs := &fakeStream{err: errExpected}
		hist := history.New(4000, nil)
		restore := replaceStdin(t, "some input\n")
		defer restore()

		err := runConversationLoop(config.Config{}, hist, fs.call)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("SIGINT causes clean return with nil", func(t *testing.T) {
		fs := &fakeStream{}
		hist := history.New(4000, nil)

		// Provide a pipe that never closes so stdin blocks indefinitely.
		r, w := io.Pipe()
		origStdin := stdinReader
		stdinReader = r
		defer func() {
			stdinReader = origStdin
			r.Close()
			w.Close()
		}()

		done := make(chan error, 1)
		go func() {
			done <- runConversationLoop(config.Config{}, hist, fs.call)
		}()

		// Give the goroutine time to block on stdin, then send SIGINT.
		time.Sleep(50 * time.Millisecond)
		syscall.Kill(syscall.Getpid(), syscall.SIGINT)

		select {
		case err := <-done:
			if err != nil {
				t.Fatalf("expected nil on SIGINT, got %v", err)
			}
		case <-time.After(2 * time.Second):
			t.Fatal("runConversationLoop did not return after SIGINT")
		}

		if fs.called != 0 {
			t.Fatalf("expected streamFn not called on SIGINT, called %d times", fs.called)
		}
	})
}
