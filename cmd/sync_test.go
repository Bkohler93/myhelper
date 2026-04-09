package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bkohler93/myhelper/internal/config"
	"github.com/bkohler93/myhelper/internal/history"
	"github.com/bkohler93/myhelper/internal/scanner"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// writeMetaJSON writes a syncMeta JSON file to root/.myhelper/meta.json.
func writeMetaJSON(t *testing.T, root string, m syncMeta) {
	t.Helper()
	dir := filepath.Join(root, ".myhelper")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("writeMetaJSON: mkdir: %v", err)
	}
	data, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("writeMetaJSON: marshal: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "meta.json"), data, 0644); err != nil {
		t.Fatalf("writeMetaJSON: write: %v", err)
	}
}

// mkMyhelperDir ensures root/.myhelper exists.
func mkMyhelperDir(t *testing.T, root string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(root, ".myhelper"), 0755); err != nil {
		t.Fatalf("mkMyhelperDir: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Gap 1: readLastSync
// ---------------------------------------------------------------------------

func TestReadLastSync(t *testing.T) {
	t.Run("missing meta.json returns zero time and nil error", func(t *testing.T) {
		root := t.TempDir()
		// Do NOT create .myhelper/meta.json
		got, err := readLastSync(root)
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		if !got.IsZero() {
			t.Fatalf("expected zero time, got %v", got)
		}
	})

	t.Run("valid meta.json returns correct timestamp", func(t *testing.T) {
		root := t.TempDir()
		want := time.Date(2025, 6, 15, 12, 30, 0, 0, time.UTC)
		writeMetaJSON(t, root, syncMeta{LastSync: want})

		got, err := readLastSync(root)
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		if !got.Equal(want) {
			t.Fatalf("expected %v, got %v", want, got)
		}
	})

	t.Run("corrupt meta.json returns error", func(t *testing.T) {
		root := t.TempDir()
		mkMyhelperDir(t, root)
		path := filepath.Join(root, ".myhelper", "meta.json")
		if err := os.WriteFile(path, []byte("{not valid json}"), 0644); err != nil {
			t.Fatalf("setup: %v", err)
		}

		_, err := readLastSync(root)
		if err == nil {
			t.Fatal("expected error for corrupt JSON, got nil")
		}
	})
}

// ---------------------------------------------------------------------------
// Gap 2: writeLastSync
// ---------------------------------------------------------------------------

func TestWriteLastSync(t *testing.T) {
	t.Run("writes meta.json with correct timestamp", func(t *testing.T) {
		root := t.TempDir()
		mkMyhelperDir(t, root)
		want := time.Date(2025, 9, 1, 8, 0, 0, 0, time.UTC)

		if err := writeLastSync(root, want); err != nil {
			t.Fatalf("writeLastSync: %v", err)
		}

		data, err := os.ReadFile(filepath.Join(root, ".myhelper", "meta.json"))
		if err != nil {
			t.Fatalf("read meta.json: %v", err)
		}
		var m syncMeta
		if err := json.Unmarshal(data, &m); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if !m.LastSync.Equal(want) {
			t.Fatalf("expected %v, got %v", want, m.LastSync)
		}
	})

	t.Run("round-trip write then read returns same time truncated to second", func(t *testing.T) {
		root := t.TempDir()
		mkMyhelperDir(t, root)
		// JSON time encoding truncates sub-second precision; use a second-boundary time.
		want := time.Date(2025, 11, 20, 14, 0, 0, 0, time.UTC)

		if err := writeLastSync(root, want); err != nil {
			t.Fatalf("writeLastSync: %v", err)
		}
		got, err := readLastSync(root)
		if err != nil {
			t.Fatalf("readLastSync: %v", err)
		}
		// Truncate both to second precision for comparison.
		if !got.Truncate(time.Second).Equal(want.Truncate(time.Second)) {
			t.Fatalf("round-trip mismatch: want %v, got %v", want, got)
		}
	})
}

// ---------------------------------------------------------------------------
// Gap 3: generateContextMD
// ---------------------------------------------------------------------------

func TestGenerateContextMD(t *testing.T) {
	defaultCfg := config.Config{TokenThreshold: 4000}

	t.Run("reads md files calls chatFn once and writes context.md", func(t *testing.T) {
		root := t.TempDir()
		mkMyhelperDir(t, root)
		// Write two summary files.
		writeSummaryFile(t, root, "pkgA", "# pkgA summary content")
		writeSummaryFile(t, root, "pkgB", "# pkgB summary content")

		callCount := 0
		chatFn := func(cfg config.Config, msgs []history.Message) (string, error) {
			callCount++
			return "generated context overview", nil
		}

		if err := generateContextMD(root, defaultCfg, chatFn); err != nil {
			t.Fatalf("generateContextMD: %v", err)
		}
		if callCount != 1 {
			t.Fatalf("expected chatFn called once, got %d", callCount)
		}
		data, err := os.ReadFile(filepath.Join(root, ".myhelper", "context.md"))
		if err != nil {
			t.Fatalf("read context.md: %v", err)
		}
		if string(data) != "generated context overview" {
			t.Fatalf("unexpected context.md content: %q", string(data))
		}
	})

	t.Run("chatFn receives user-role message containing package headings", func(t *testing.T) {
		root := t.TempDir()
		mkMyhelperDir(t, root)
		writeSummaryFile(t, root, "mypkg", "the summary text")

		var capturedMsgs []history.Message
		chatFn := func(cfg config.Config, msgs []history.Message) (string, error) {
			capturedMsgs = msgs
			return "ok", nil
		}

		if err := generateContextMD(root, defaultCfg, chatFn); err != nil {
			t.Fatalf("generateContextMD: %v", err)
		}
		if len(capturedMsgs) != 1 {
			t.Fatalf("expected 1 message, got %d", len(capturedMsgs))
		}
		if capturedMsgs[0].Role != "user" {
			t.Fatalf("expected user role, got %q", capturedMsgs[0].Role)
		}
		if !strings.Contains(capturedMsgs[0].Content, "### mypkg") {
			t.Errorf("expected '### mypkg' heading in message, got: %q", capturedMsgs[0].Content)
		}
	})

	t.Run("empty summaries dir returns error containing 'no summaries found'", func(t *testing.T) {
		root := t.TempDir()
		mkMyhelperDir(t, root)
		// Create summaries dir but leave it empty.
		if err := os.MkdirAll(filepath.Join(root, ".myhelper", "summaries"), 0755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}

		chatFn := func(cfg config.Config, msgs []history.Message) (string, error) {
			return "", nil
		}

		err := generateContextMD(root, defaultCfg, chatFn)
		if err == nil {
			t.Fatal("expected error for empty summaries dir, got nil")
		}
		if !strings.Contains(err.Error(), "no summaries found") {
			t.Errorf("expected error to contain 'no summaries found', got: %v", err)
		}
	})

	t.Run("chatFn error propagates", func(t *testing.T) {
		root := t.TempDir()
		mkMyhelperDir(t, root)
		writeSummaryFile(t, root, "pkg", "some content")

		wantErr := errors.New("llm offline")
		chatFn := func(cfg config.Config, msgs []history.Message) (string, error) {
			return "", wantErr
		}

		err := generateContextMD(root, defaultCfg, chatFn)
		if err == nil {
			t.Fatal("expected chatFn error to propagate, got nil")
		}
		if !errors.Is(err, wantErr) {
			t.Errorf("expected wrapped wantErr, got: %v", err)
		}
	})
}

// ---------------------------------------------------------------------------
// Gap 4: changedFilesSince
// ---------------------------------------------------------------------------

func TestChangedFilesSince(t *testing.T) {
	t.Run("zero since time returns all go files", func(t *testing.T) {
		root := t.TempDir()
		// Create two Go files.
		if err := os.WriteFile(filepath.Join(root, "a.go"), []byte("package main\n"), 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, "b.go"), []byte("package main\n"), 0644); err != nil {
			t.Fatal(err)
		}

		got, err := changedFilesSince(root, time.Time{})
		if err != nil {
			t.Fatalf("changedFilesSince: %v", err)
		}
		if len(got) != 2 {
			t.Fatalf("expected 2 files, got %d: %v", len(got), got)
		}
	})

	t.Run("non-zero since after file mtime — file NOT returned", func(t *testing.T) {
		root := t.TempDir()
		goPath := filepath.Join(root, "old.go")
		if err := os.WriteFile(goPath, []byte("package main\n"), 0644); err != nil {
			t.Fatal(err)
		}
		// Set mtime to a known past time.
		past := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
		if err := os.Chtimes(goPath, past, past); err != nil {
			t.Fatalf("chtimes: %v", err)
		}
		// since is after that past mtime.
		since := time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)

		got, err := changedFilesSince(root, since)
		if err != nil {
			t.Fatalf("changedFilesSince: %v", err)
		}
		if len(got) != 0 {
			t.Fatalf("expected 0 files (file older than since), got %d: %v", len(got), got)
		}
	})

	t.Run("non-zero since before file mtime — file IS returned", func(t *testing.T) {
		root := t.TempDir()
		goPath := filepath.Join(root, "new.go")
		if err := os.WriteFile(goPath, []byte("package main\n"), 0644); err != nil {
			t.Fatal(err)
		}
		// Set mtime to a known future-ish time.
		future := time.Date(2030, 6, 1, 0, 0, 0, 0, time.UTC)
		if err := os.Chtimes(goPath, future, future); err != nil {
			t.Fatalf("chtimes: %v", err)
		}
		// since is before that mtime.
		since := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

		got, err := changedFilesSince(root, since)
		if err != nil {
			t.Fatalf("changedFilesSince: %v", err)
		}
		if len(got) != 1 {
			t.Fatalf("expected 1 file, got %d: %v", len(got), got)
		}
		if got[0] != "new.go" {
			t.Errorf("expected 'new.go', got %q", got[0])
		}
	})

	t.Run("excludes .git vendor testdata .myhelper directories", func(t *testing.T) {
		root := t.TempDir()

		// Create excluded directories with Go files inside.
		for _, dir := range []string{".git", "vendor", "testdata", ".myhelper"} {
			subDir := filepath.Join(root, dir)
			if err := os.MkdirAll(subDir, 0755); err != nil {
				t.Fatalf("mkdir %s: %v", dir, err)
			}
			goFile := filepath.Join(subDir, "file.go")
			if err := os.WriteFile(goFile, []byte("package excluded\n"), 0644); err != nil {
				t.Fatal(err)
			}
		}
		// Create one included Go file at root.
		if err := os.WriteFile(filepath.Join(root, "included.go"), []byte("package main\n"), 0644); err != nil {
			t.Fatal(err)
		}

		got, err := changedFilesSince(root, time.Time{})
		if err != nil {
			t.Fatalf("changedFilesSince: %v", err)
		}
		if len(got) != 1 {
			t.Fatalf("expected 1 file (excluded dirs skipped), got %d: %v", len(got), got)
		}
		if got[0] != "included.go" {
			t.Errorf("expected 'included.go', got %q", got[0])
		}
	})
}

// ---------------------------------------------------------------------------
// Gap 5: deltaIndex
// ---------------------------------------------------------------------------

func TestDeltaIndex(t *testing.T) {
	defaultCfg := config.Config{TokenThreshold: 4000}

	t.Run("new file added to changedPaths appears in output index.json", func(t *testing.T) {
		root := t.TempDir()
		mkMyhelperDir(t, root)

		// Write a real Go file so ExtractSymbols can parse it.
		goContent := "package mypkg\n\nfunc NewFunc() {}\n"
		if err := os.WriteFile(filepath.Join(root, "new.go"), []byte(goContent), 0644); err != nil {
			t.Fatal(err)
		}
		// Start with an empty index.json.
		writeIndexFile(t, root, scanner.Index{})

		if err := deltaIndex(root, defaultCfg, []string{"new.go"}); err != nil {
			t.Fatalf("deltaIndex: %v", err)
		}

		idx, err := readIndexFile(root)
		if err != nil {
			t.Fatalf("readIndexFile: %v", err)
		}
		found := false
		for _, e := range idx.Files {
			if e.Path == "new.go" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected 'new.go' in index, got: %+v", idx.Files)
		}
	})

	t.Run("existing file in changedPaths gets entry updated", func(t *testing.T) {
		root := t.TempDir()
		mkMyhelperDir(t, root)

		// Write a Go file with a new function.
		goContent := "package mypkg\n\nfunc UpdatedFunc() {}\n"
		if err := os.WriteFile(filepath.Join(root, "existing.go"), []byte(goContent), 0644); err != nil {
			t.Fatal(err)
		}
		// Seed index with stale entry for same file.
		staleEntry := scanner.FileEntry{
			Path:    "existing.go",
			Package: "mypkg",
			Symbols: []string{"func OldFunc"},
		}
		writeIndexFile(t, root, scanner.Index{Files: []scanner.FileEntry{staleEntry}})

		if err := deltaIndex(root, defaultCfg, []string{"existing.go"}); err != nil {
			t.Fatalf("deltaIndex: %v", err)
		}

		idx, err := readIndexFile(root)
		if err != nil {
			t.Fatalf("readIndexFile: %v", err)
		}
		var updated scanner.FileEntry
		for _, e := range idx.Files {
			if e.Path == "existing.go" {
				updated = e
				break
			}
		}
		if updated.Path == "" {
			t.Fatal("expected 'existing.go' in updated index")
		}
		// The updated entry should reflect the new symbol.
		found := false
		for _, sym := range updated.Symbols {
			if strings.Contains(sym, "UpdatedFunc") {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected 'UpdatedFunc' in updated symbols, got: %v", updated.Symbols)
		}
	})

	t.Run("file deleted from filesystem is removed from index.json", func(t *testing.T) {
		root := t.TempDir()
		mkMyhelperDir(t, root)

		// Seed index with entry for a file that does NOT exist on disk.
		deletedEntry := scanner.FileEntry{
			Path:    "deleted.go",
			Package: "mypkg",
			Symbols: []string{"func Ghost"},
		}
		// Also keep a real file so the index is non-empty after merge.
		goContent := "package mypkg\n\nfunc RealFunc() {}\n"
		if err := os.WriteFile(filepath.Join(root, "real.go"), []byte(goContent), 0644); err != nil {
			t.Fatal(err)
		}
		realEntry := scanner.FileEntry{
			Path:    "real.go",
			Package: "mypkg",
			Symbols: []string{"func RealFunc"},
		}
		writeIndexFile(t, root, scanner.Index{Files: []scanner.FileEntry{deletedEntry, realEntry}})

		// changedPaths only includes the real file (deleted.go no longer exists).
		if err := deltaIndex(root, defaultCfg, []string{"real.go"}); err != nil {
			t.Fatalf("deltaIndex: %v", err)
		}

		idx, err := readIndexFile(root)
		if err != nil {
			t.Fatalf("readIndexFile: %v", err)
		}
		for _, e := range idx.Files {
			if e.Path == "deleted.go" {
				t.Errorf("expected 'deleted.go' to be removed from index, but it remains")
			}
		}
	})

	t.Run("corrupt index.json treated as empty — no crash", func(t *testing.T) {
		root := t.TempDir()
		mkMyhelperDir(t, root)

		// Write corrupt index.json.
		corruptPath := filepath.Join(root, ".myhelper", "index.json")
		if err := os.WriteFile(corruptPath, []byte("{bad json!!}"), 0644); err != nil {
			t.Fatal(err)
		}
		// Write a valid Go file to process.
		goContent := "package mypkg\n\nfunc NewFunc() {}\n"
		if err := os.WriteFile(filepath.Join(root, "new.go"), []byte(goContent), 0644); err != nil {
			t.Fatal(err)
		}

		// Should not crash — corrupt index treated as empty.
		if err := deltaIndex(root, defaultCfg, []string{"new.go"}); err != nil {
			t.Fatalf("deltaIndex should not error on corrupt index, got: %v", err)
		}

		// Result should still have the new file.
		idx, err := readIndexFile(root)
		if err != nil {
			t.Fatalf("readIndexFile: %v", err)
		}
		found := false
		for _, e := range idx.Files {
			if e.Path == "new.go" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected 'new.go' after corrupt-index recovery, got: %+v", idx.Files)
		}
	})
}

// ---------------------------------------------------------------------------
// Gap 6: deltaSummaries
// ---------------------------------------------------------------------------

func TestDeltaSummaries(t *testing.T) {
	defaultCfg := config.Config{TokenThreshold: 4000}

	t.Run("changed file in package causes chatFn to be called", func(t *testing.T) {
		root := t.TempDir()
		mkMyhelperDir(t, root)

		// Write index.json with one file entry belonging to "mypkg".
		idx := scanner.Index{
			Files: []scanner.FileEntry{
				{Path: "mypkg/foo.go", Package: "mypkg", Symbols: []string{"func Foo"}},
			},
		}
		writeIndexFile(t, root, idx)

		// Ensure summaries dir exists (GenerateSummaries will write there).
		if err := os.MkdirAll(filepath.Join(root, ".myhelper", "summaries"), 0755); err != nil {
			t.Fatal(err)
		}

		chatCalled := false
		chatFn := func(cfg config.Config, msgs []history.Message) (string, error) {
			chatCalled = true
			return "summary for mypkg", nil
		}

		if err := deltaSummaries(root, defaultCfg, chatFn, []string{"mypkg/foo.go"}, func(s string) {}); err != nil {
			t.Fatalf("deltaSummaries: %v", err)
		}
		if !chatCalled {
			t.Error("expected chatFn to be called for affected package, but it was not")
		}
	})

	t.Run("empty changedPaths means chatFn is NOT called", func(t *testing.T) {
		root := t.TempDir()
		mkMyhelperDir(t, root)

		idx := scanner.Index{
			Files: []scanner.FileEntry{
				{Path: "pkg/bar.go", Package: "pkg", Symbols: []string{"func Bar"}},
			},
		}
		writeIndexFile(t, root, idx)

		chatCalled := false
		chatFn := func(cfg config.Config, msgs []history.Message) (string, error) {
			chatCalled = true
			return "should not be called", nil
		}

		if err := deltaSummaries(root, defaultCfg, chatFn, []string{}, func(s string) {}); err != nil {
			t.Fatalf("deltaSummaries: %v", err)
		}
		if chatCalled {
			t.Error("expected chatFn NOT to be called for empty changedPaths, but it was")
		}
	})

	t.Run("injectable scanner.ChatFn used for testability", func(t *testing.T) {
		root := t.TempDir()
		mkMyhelperDir(t, root)

		idx := scanner.Index{
			Files: []scanner.FileEntry{
				{Path: "util/helper.go", Package: "util", Symbols: []string{"func Helper"}},
			},
		}
		writeIndexFile(t, root, idx)

		if err := os.MkdirAll(filepath.Join(root, ".myhelper", "summaries"), 0755); err != nil {
			t.Fatal(err)
		}

		var recordedPkg string
		var stubFn scanner.ChatFn = func(cfg config.Config, msgs []history.Message) (string, error) {
			fmt.Println("messages are", msgs)
			// GenerateSummaries passes a user message with "Package: <name>"
			if len(msgs) > 0 && strings.Contains(msgs[0].Content, "PACKAGE: util") {
				recordedPkg = "util"
			}
			return "util summary", nil
		}

		if err := deltaSummaries(root, defaultCfg, stubFn, []string{"util/helper.go"}, func(s string) {}); err != nil {
			t.Fatalf("deltaSummaries: %v", err)
		}
		if recordedPkg != "util" {
			t.Errorf("expected injectable chatFn to receive package 'util', got: %q", recordedPkg)
		}
	})
}
