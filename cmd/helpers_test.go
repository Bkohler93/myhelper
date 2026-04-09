package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
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
			Files: []scanner.FileEntry{{Path: goFile, Package: "main", Symbols: []string{"func main"}, TokenCount: 10}},
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

	t.Run("file content exceeds token budget — micro-pass skips file when budget too small", func(t *testing.T) {
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
				Symbols: []string{"func BigFunc"},
				TokenCount:      5,
			}},
		}
		writeIndexFile(t, root, idx)

		// threshold=1 → budget=int(1*0.80)=0. microPassFile receives budget=0
		// and returns false immediately, so the file is skipped silently.
		tinyThreshold := 1
		tinyCfg := config.Config{TokenThreshold: tinyThreshold}

		chatFn := func(cfg config.Config, msgs []history.Message) (string, error) {
			return goFile, nil // Pass-1: select this file (micro-pass never reached)
		}
		msgs, err := buildInjectedMessages(root, "budget query", tinyCfg, chatFn, "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(msgs) != 1 || msgs[0].Role != "user" {
			t.Fatalf("expected one user-role message, got %+v", msgs)
		}
		// File should be skipped — no file content injected (budget exhausted)
		if strings.Contains(msgs[0].Content, "func BigFunc() {}") {
			t.Errorf("did not expect file content when budget=0, got: %q", msgs[0].Content)
		}
		if strings.Contains(msgs[0].Content, "big.go") {
			t.Errorf("did not expect file reference when budget=0, got: %q", msgs[0].Content)
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
				{Path: file1, Package: "main", Symbols: []string{"func Alpha"}, TokenCount: 5},
				{Path: file2, Package: "main", Symbols: []string{"func Beta"}, TokenCount: 5},
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

// writeTempGoFile creates a temp Go file with the given content in a temp dir.
// Returns the dir and the full path to the file.
func writeTempGoFile(t *testing.T, content string) (dir, path string) {
	t.Helper()
	dir = t.TempDir()
	path = filepath.Join(dir, "myfile.go")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return dir, path
}

// TestMicroPassFile covers the full fallback chain of microPassFile.
func TestMicroPassFile(t *testing.T) {
	goContent := "package foo\n\nfunc Alpha() {\n\tx := 1\n\t_ = x\n}\n\nfunc Beta() {}\n"
	// goContent line map (1-indexed):
	// 1: package foo
	// 2: (blank)
	// 3: func Alpha() {
	// 4:     x := 1
	// 5:     _ = x
	// 6: }
	// 7: (blank)
	// 8: func Beta() {}
	// 9: (trailing newline split → empty string)

	defaultCfg := config.Config{TokenThreshold: 4000}

	t.Run("model returns valid range — extracted lines returned", func(t *testing.T) {
		dir, path := writeTempGoFile(t, goContent)
		chatFn := func(cfg config.Config, msgs []history.Message) (string, error) {
			return "3-6", nil
		}
		content, ok := microPassFile(dir, "myfile.go", "tell me about Alpha", defaultCfg, chatFn, 10000)
		if !ok {
			t.Fatal("expected ok=true, got false")
		}
		if !strings.Contains(content, "func Alpha() {") {
			t.Errorf("expected extracted content to contain 'func Alpha() {', got: %q", content)
		}
		if !strings.Contains(content, "_ = x") {
			t.Errorf("expected extracted content to contain '_ = x', got: %q", content)
		}
		if strings.Contains(content, "func Beta() {}") {
			t.Errorf("expected extracted content to NOT contain 'func Beta() {}', got: %q", content)
		}
		_ = path
	})

	t.Run("model returns out-of-bounds range — clamped and returned", func(t *testing.T) {
		dir, _ := writeTempGoFile(t, goContent)
		chatFn := func(cfg config.Config, msgs []history.Message) (string, error) {
			return "0-100", nil // start < 1, end > totalLines
		}
		content, ok := microPassFile(dir, "myfile.go", "tell me everything", defaultCfg, chatFn, 10000)
		if !ok {
			t.Fatal("expected ok=true after clamping, got false")
		}
		if !strings.Contains(content, "package foo") {
			t.Errorf("expected clamped content to contain 'package foo', got: %q", content)
		}
	})

	t.Run("model returns unparseable response — falls back to truncation", func(t *testing.T) {
		dir, _ := writeTempGoFile(t, goContent)
		chatFn := func(cfg config.Config, msgs []history.Message) (string, error) {
			return "not a range", nil
		}
		content, ok := microPassFile(dir, "myfile.go", "some query", defaultCfg, chatFn, 10000)
		if !ok {
			t.Fatal("expected ok=true via truncation fallback, got false")
		}
		if content == "" {
			t.Error("expected non-empty truncated content, got empty string")
		}
	})

	t.Run("model returns start > end — falls back to truncation", func(t *testing.T) {
		dir, _ := writeTempGoFile(t, goContent)
		chatFn := func(cfg config.Config, msgs []history.Message) (string, error) {
			return "6-3", nil // start > end
		}
		content, ok := microPassFile(dir, "myfile.go", "some query", defaultCfg, chatFn, 10000)
		if !ok {
			t.Fatal("expected ok=true via truncation fallback, got false")
		}
		if content == "" {
			t.Error("expected non-empty truncated content, got empty string")
		}
	})

	t.Run("extracted range exceeds budget — falls back to truncation, budget too small → ok=false", func(t *testing.T) {
		dir, _ := writeTempGoFile(t, goContent)
		chatFn := func(cfg config.Config, msgs []history.Message) (string, error) {
			return "1-9", nil // whole file
		}
		// budget=1: nothing fits via micro-pass, truncation also fails
		content, ok := microPassFile(dir, "myfile.go", "some query", defaultCfg, chatFn, 1)
		if ok {
			t.Errorf("expected ok=false when budget=1 too small for any content, got content=%q", content)
		}
	})

	t.Run("budget=0 — returns false immediately", func(t *testing.T) {
		dir, _ := writeTempGoFile(t, goContent)
		chatFn := func(cfg config.Config, msgs []history.Message) (string, error) {
			return "3-6", nil
		}
		content, ok := microPassFile(dir, "myfile.go", "some query", defaultCfg, chatFn, 0)
		if ok || content != "" {
			t.Errorf("expected ok=false and empty content for budget=0, got ok=%v content=%q", ok, content)
		}
	})

	t.Run("chatFn error — falls back to truncation", func(t *testing.T) {
		dir, _ := writeTempGoFile(t, goContent)
		chatFn := func(cfg config.Config, msgs []history.Message) (string, error) {
			return "", errors.New("llm offline")
		}
		content, ok := microPassFile(dir, "myfile.go", "some query", defaultCfg, chatFn, 10000)
		if !ok {
			t.Fatal("expected ok=true via truncation fallback after chatFn error, got false")
		}
		if content == "" {
			t.Error("expected non-empty truncated content, got empty string")
		}
	})

	t.Run("non-Go file — falls back to truncation", func(t *testing.T) {
		dir := t.TempDir()
		txtPath := filepath.Join(dir, "myfile.go") // named .go but invalid Go syntax
		if err := os.WriteFile(txtPath, []byte("This is a plain text file\nwith no Go syntax\n"), 0644); err != nil {
			t.Fatal(err)
		}
		chatFn := func(cfg config.Config, msgs []history.Message) (string, error) {
			return "1-2", nil
		}
		content, ok := microPassFile(dir, "myfile.go", "some query", defaultCfg, chatFn, 10000)
		if !ok {
			t.Fatal("expected ok=true via truncation fallback for non-Go file, got false")
		}
		if content == "" {
			t.Error("expected non-empty truncated content, got empty string")
		}
	})
}

// TestBuildInjectedMessages_NoSymbolBlock verifies that the symbol-block
// fallback (ExportedSymbols/UnexportedSymbols) is gone and micro-pass is used.
func TestBuildInjectedMessages_NoSymbolBlock(t *testing.T) {
	t.Run("large file uses micro-pass not symbol block", func(t *testing.T) {
		root := t.TempDir()

		// Create src/ subdir and big.go with actual Go content
		srcDir := filepath.Join(root, "src")
		if err := os.MkdirAll(srcDir, 0755); err != nil {
			t.Fatal(err)
		}
		bigContent := "package main\n\nfunc BigFn() {\n\tx := 42\n\t_ = x\n}\n"
		bigPath := filepath.Join(srcDir, "big.go")
		if err := os.WriteFile(bigPath, []byte(bigContent), 0644); err != nil {
			t.Fatal(err)
		}

		// Index references the file
		idx := scanner.Index{
			Files: []scanner.FileEntry{{
				Path:            "src/big.go",
				Package:         "main",
				Symbols: []string{"func BigFn"},
				TokenCount:      10,
			}},
		}
		writeIndexFile(t, root, idx)

		// Use a threshold where raw content doesn't fit but micro-pass range (lines 3-6) does.
		// Raw content is ~10 tokens. Budget at 80% of 12 = 9. Micro-pass returns "3-6"
		// which is "func BigFn() {\n\tx := 42\n\t_ = x\n}" (~6 tokens, fits).
		tinyCfg := config.Config{TokenThreshold: 12}

		callCount := 0
		chatFn := func(cfg config.Config, msgs []history.Message) (string, error) {
			callCount++
			if callCount == 1 {
				return "src/big.go", nil // Pass-1
			}
			return "3-6", nil // micro-pass line range
		}

		msgs, err := buildInjectedMessages(root, "what does BigFn do", tinyCfg, chatFn, "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(msgs) != 1 || msgs[0].Role != "user" {
			t.Fatalf("expected one user-role message, got %+v", msgs)
		}
		if !strings.Contains(msgs[0].Content, "src/big.go") {
			t.Errorf("expected content to reference src/big.go, got: %q", msgs[0].Content)
		}
		if strings.Contains(msgs[0].Content, "// Exported:") || strings.Contains(msgs[0].Content, "// Unexported:") {
			t.Errorf("symbol-block strings must not appear after removal, got: %q", msgs[0].Content)
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
