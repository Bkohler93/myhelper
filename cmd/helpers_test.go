package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
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

// TestBuildInjectedMessages covers all 7 behavioral branches of buildInjectedMessages.
func TestBuildInjectedMessages(t *testing.T) {
	defaultCfg := config.Config{TokenThreshold: 4000}

	t.Run("no index.json returns bare user query and prints warning", func(t *testing.T) {
		root := t.TempDir()
		// Capture stderr
		oldStderr := os.Stderr
		r, w, _ := os.Pipe()
		os.Stderr = w

		chatFn := func(cfg config.Config, msgs []history.Message) (string, error) {
			return "", fmt.Errorf("should not be called")
		}
		msgs, err := buildInjectedMessages(root, "my query", defaultCfg, chatFn, "")

		w.Close()
		var buf bytes.Buffer
		buf.ReadFrom(r)
		os.Stderr = oldStderr

		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		if len(msgs) != 1 || msgs[0].Role != "user" || msgs[0].Content != "my query" {
			t.Fatalf("expected bare user query message, got %+v", msgs)
		}
		if !strings.Contains(buf.String(), "No index found") {
			t.Errorf("expected warning containing 'No index found', got: %q", buf.String())
		}
	})

	t.Run("valid path returned by Pass-1 — message contains header and query", func(t *testing.T) {
		root := t.TempDir()
		// Create index.json with one entry
		goFile := "main.go"
		goContent := "package main\n\nfunc main() {}\n"
		if err := os.WriteFile(filepath.Join(root, goFile), []byte(goContent), 0644); err != nil {
			t.Fatal(err)
		}
		idx := scanner.Index{
			Files: []scanner.FileEntry{{Path: goFile, Package: "main", ExportedSymbols: []string{"main"}, TokenCount: 10}},
		}
		writeIndexFile(t, root, idx)

		chatFn := func(cfg config.Config, msgs []history.Message) (string, error) {
			return goFile, nil
		}
		msgs, err := buildInjectedMessages(root, "original query", defaultCfg, chatFn, "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(msgs) != 1 || msgs[0].Role != "user" {
			t.Fatalf("expected one user-role message, got %+v", msgs)
		}
		if !strings.Contains(msgs[0].Content, "Here is the relevant source code for context:") {
			t.Errorf("expected header in content, got: %q", msgs[0].Content)
		}
		if !strings.Contains(msgs[0].Content, "original query") {
			t.Errorf("expected original query in content, got: %q", msgs[0].Content)
		}
	})

	t.Run("invalid path from Pass-1 falls back to summaries injection", func(t *testing.T) {
		root := t.TempDir()
		idx := scanner.Index{
			Files: []scanner.FileEntry{{Path: "real.go", Package: "main", TokenCount: 5}},
		}
		writeIndexFile(t, root, idx)
		writeSummaryFile(t, root, "main", "# Main package summary")

		chatFn := func(cfg config.Config, msgs []history.Message) (string, error) {
			return "nonexistent/path.go", nil
		}
		msgs, err := buildInjectedMessages(root, "fallback query", defaultCfg, chatFn, "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(msgs) != 1 || msgs[0].Role != "user" {
			t.Fatalf("expected one user-role message, got %+v", msgs)
		}
		if !strings.Contains(msgs[0].Content, "Here are project package summaries for context:") {
			t.Errorf("expected summaries header in content, got: %q", msgs[0].Content)
		}
	})

	t.Run("zero parseable paths from Pass-1 falls back to summaries injection", func(t *testing.T) {
		root := t.TempDir()
		idx := scanner.Index{Files: []scanner.FileEntry{}}
		writeIndexFile(t, root, idx)
		writeSummaryFile(t, root, "pkg", "# pkg summary content")

		chatFn := func(cfg config.Config, msgs []history.Message) (string, error) {
			return "   ,  ,  ", nil // no parseable non-empty paths
		}
		msgs, err := buildInjectedMessages(root, "zero paths query", defaultCfg, chatFn, "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(msgs) != 1 || msgs[0].Role != "user" {
			t.Fatalf("expected one user-role message, got %+v", msgs)
		}
		if !strings.Contains(msgs[0].Content, "Here are project package summaries for context:") {
			t.Errorf("expected summaries fallback, got: %q", msgs[0].Content)
		}
	})

	t.Run("file content exceeds token budget uses symbol fallback", func(t *testing.T) {
		root := t.TempDir()
		// Write a small Go file
		goFile := "big.go"
		// Any content; budget will be set tiny so raw content always exceeds it
		fileContent := "package main\n\nfunc BigFunc() {}\n"
		if err := os.WriteFile(filepath.Join(root, goFile), []byte(fileContent), 0644); err != nil {
			t.Fatal(err)
		}
		idx := scanner.Index{
			Files: []scanner.FileEntry{{
				Path:            goFile,
				Package:         "main",
				ExportedSymbols: []string{"BigFunc"},
				TokenCount:      5,
			}},
		}
		writeIndexFile(t, root, idx)

		// threshold=9 → budget=int(9*0.80)=7. raw content is 8 tokens (exceeds budget),
		// sig content is 7 tokens (fits). This forces the symbol fallback path.
		tinyThreshold := 9
		tinyCfg := config.Config{TokenThreshold: tinyThreshold}

		chatFn := func(cfg config.Config, msgs []history.Message) (string, error) {
			return goFile, nil
		}
		msgs, err := buildInjectedMessages(root, "budget query", tinyCfg, chatFn, "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(msgs) != 1 || msgs[0].Role != "user" {
			t.Fatalf("expected one user-role message, got %+v", msgs)
		}
		// Content should contain signature fallback text (symbols only), not the raw source
		if strings.Contains(msgs[0].Content, "func BigFunc() {}") {
			t.Errorf("did not expect raw file content in budget-exceeded case, got: %q", msgs[0].Content)
		}
		if !strings.Contains(msgs[0].Content, "BigFunc") {
			t.Errorf("expected symbol name in fallback content, got: %q", msgs[0].Content)
		}
	})

	t.Run("budget exhausted after first file — second file not included", func(t *testing.T) {
		root := t.TempDir()
		file1 := "a.go"
		file2 := "b.go"
		content1 := "package main\n\nfunc Alpha() {}\n"
		content2 := "package main\n\nfunc Beta() {}\n"
		if err := os.WriteFile(filepath.Join(root, file1), []byte(content1), 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, file2), []byte(content2), 0644); err != nil {
			t.Fatal(err)
		}
		idx := scanner.Index{
			Files: []scanner.FileEntry{
				{Path: file1, Package: "main", ExportedSymbols: []string{"Alpha"}, TokenCount: 5},
				{Path: file2, Package: "main", ExportedSymbols: []string{"Beta"}, TokenCount: 5},
			},
		}
		writeIndexFile(t, root, idx)

		// Threshold small enough that first file content fits but second exhausts budget.
		// content1 is ~8 tokens. Budget at 80% of 12 = 9.6 → 9, so first file fits,
		// second file would push over.
		smallCfg := config.Config{TokenThreshold: 12}

		chatFn := func(cfg config.Config, msgs []history.Message) (string, error) {
			return file1 + "," + file2, nil
		}
		msgs, err := buildInjectedMessages(root, "two files query", smallCfg, chatFn, "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(msgs) != 1 {
			t.Fatalf("expected one message, got %d", len(msgs))
		}
		// Second file should not appear
		if strings.Contains(msgs[0].Content, "b.go") {
			t.Errorf("second file should be excluded when budget exhausted, got: %q", msgs[0].Content)
		}
	})

	t.Run("no summaries directory on fallback returns bare user query", func(t *testing.T) {
		root := t.TempDir()
		// Create index but no summaries dir
		idx := scanner.Index{Files: []scanner.FileEntry{}}
		writeIndexFile(t, root, idx)
		// No summaries dir created

		chatFn := func(cfg config.Config, msgs []history.Message) (string, error) {
			return "", nil // empty response → no valid paths
		}
		msgs, err := buildInjectedMessages(root, "no summaries query", defaultCfg, chatFn, "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(msgs) != 1 || msgs[0].Role != "user" || msgs[0].Content != "no summaries query" {
			t.Fatalf("expected bare user query message, got %+v", msgs)
		}
	})
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
