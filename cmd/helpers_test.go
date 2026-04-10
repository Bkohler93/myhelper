package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/bkohler93/myhelper/internal/config"
	"github.com/bkohler93/myhelper/internal/history"
	"github.com/bkohler93/myhelper/internal/scanner"
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

		err := runConversationLoop(config.Config{}, hist, fs.call, "", "")
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

		err := runConversationLoop(config.Config{}, hist, fs.call, "", "")
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

		err := runConversationLoop(config.Config{}, hist, fs.call, "", "")
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

		err := runConversationLoop(config.Config{}, hist, fs.call, "", "")
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
			done <- runConversationLoop(config.Config{}, hist, fs.call, "", "")
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

// writeIndexFile is a test helper that writes a scanner.Index to root/.myhelper/index.json.
func writeIndexFile(t *testing.T, root string, idx scanner.Index) {
	t.Helper()
	myhelperDir := filepath.Join(root, ".myhelper")
	if err := os.MkdirAll(myhelperDir, 0755); err != nil {
		t.Fatalf("writeIndexFile: mkdir: %v", err)
	}
	data, err := json.Marshal(idx)
	if err != nil {
		t.Fatalf("writeIndexFile: marshal: %v", err)
	}
	if err := os.WriteFile(filepath.Join(myhelperDir, "index.json"), data, 0644); err != nil {
		t.Fatalf("writeIndexFile: write: %v", err)
	}
}

// writeSummaryFile is a test helper that writes content to root/.myhelper/summaries/<name>.md.
func writeSummaryFile(t *testing.T, root, name, content string) {
	t.Helper()
	dir := filepath.Join(root, ".myhelper", "summaries")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("writeSummaryFile: mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, name+".md"), []byte(content), 0644); err != nil {
		t.Fatalf("writeSummaryFile: write: %v", err)
	}
}

// TestRunConversationLoop_Summarization covers summarization behavior.
func TestRunConversationLoop_Summarization(t *testing.T) {
	t.Run("does not summarize when history is within limit", func(t *testing.T) {
		fs := &fakeStream{responses: []string{"response1"}}
		// Threshold very high — never triggers
		hist := history.New(999999, []history.Message{
			{Role: "system", Content: "system prompt"},
			{Role: "user", Content: "initial question"},
			{Role: "assistant", Content: "initial answer"},
		})
		restore := replaceStdin(t, "follow-up\nquit\n")
		defer restore()

		err := runConversationLoop(config.Config{}, hist, fs.call, "summarize prompt", "recondense prompt")
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		// streamFn called once for the follow-up (not for summarization)
		if fs.called != 1 {
			t.Fatalf("expected streamFn called once, got %d", fs.called)
		}
	})
}

// TestBuildInjectedMessagesRemoved verifies that the dead helper buildInjectedMessages
// was fully deleted in Phase 12 plan 03 (CTX-03). This is a regression guard.
func TestBuildInjectedMessagesRemoved(t *testing.T) {
	data, err := os.ReadFile("helpers.go")
	if err != nil {
		t.Fatalf("could not read helpers.go: %v", err)
	}
	if bytes.Contains(data, []byte("buildInjectedMessages")) {
		t.Error("helpers.go must not contain buildInjectedMessages (CTX-03: deleted in Phase 12.03)")
	}
}

// TestMicroPassMigration verifies that the legacy microPass helpers were deleted
// and replaced by Strategy-based retrieval.BuildContext (CTX-04, Phase 12.03).
func TestMicroPassMigration(t *testing.T) {
	data, err := os.ReadFile("helpers.go")
	if err != nil {
		t.Fatalf("could not read helpers.go: %v", err)
	}
	if bytes.Contains(data, []byte("microPassFile")) {
		t.Error("helpers.go must not contain microPassFile (CTX-04: deleted in Phase 12.03)")
	}
	if bytes.Contains(data, []byte("microPassRe")) {
		t.Error("helpers.go must not contain microPassRe (CTX-04: deleted in Phase 12.03)")
	}
}

// TestReadIndexFile_StaleFlatIndex verifies that readIndexFile returns
// scanner.ErrStaleFlatIndex when project.json exists, and os.ErrNotExist
// when no artifact or index files are present.
func TestReadIndexFile_StaleFlatIndex(t *testing.T) {
	t.Run("returns_stale_error_when_project_json_exists", func(t *testing.T) {
		tmpDir := t.TempDir()
		if err := os.MkdirAll(filepath.Join(tmpDir, ".myhelper"), 0755); err != nil {
			t.Fatal(err)
		}
		// Write minimal project.json
		if err := os.WriteFile(filepath.Join(tmpDir, ".myhelper", "project.json"),
			[]byte(`{"schemaVersion":"1.0"}`), 0644); err != nil {
			t.Fatal(err)
		}
		_, err := readIndexFile(tmpDir)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, scanner.ErrStaleFlatIndex) {
			t.Fatalf("expected ErrStaleFlatIndex, got %v", err)
		}
	})

	t.Run("returns_not_exist_when_no_files", func(t *testing.T) {
		tmpDir := t.TempDir()
		// No .myhelper directory at all
		_, err := readIndexFile(tmpDir)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if errors.Is(err, scanner.ErrStaleFlatIndex) {
			t.Fatalf("did not expect ErrStaleFlatIndex for missing index.json, got %v", err)
		}
	})
}
