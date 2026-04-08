package scanner_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/bkohler93/myhelper/internal/config"
	"github.com/bkohler93/myhelper/internal/history"
	"github.com/bkohler93/myhelper/internal/scanner"
)

func setupTempProject(t *testing.T) (root string, fakeFn scanner.ChatFn, calls *int) {
	t.Helper()
	root = t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".myhelper", "summaries"), 0755); err != nil {
		t.Fatalf("setup: mkdir .myhelper/summaries: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(root, "main.go"),
		[]byte("package myapp\nfunc Hello() string { return \"hi\" }"),
		0644,
	); err != nil {
		t.Fatalf("setup: write main.go: %v", err)
	}

	counter := 0
	fn := func(cfg config.Config, msgs []history.Message) (string, error) {
		counter++
		return "# myapp\nA greeting package.", nil
	}
	return root, fn, &counter
}

// TestScan_IndexJSONCreated verifies that Scan() creates .myhelper/index.json
// as a non-empty valid JSON array.
func TestScan_IndexJSONCreated(t *testing.T) {
	root, fakeFn, _ := setupTempProject(t)
	cfg := config.Config{TokenThreshold: 4100}

	if err := scanner.Scan(root, cfg, fakeFn); err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	indexPath := filepath.Join(root, ".myhelper", "index.json")
	data, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("index.json not found: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("index.json is empty")
	}

	var entries []scanner.FileEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		t.Fatalf("index.json is not valid JSON: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("index.json contains empty array, expected at least one entry")
	}
}

// TestScan_SummaryCreated verifies that Scan() creates .myhelper/summaries/myapp.md.
func TestScan_SummaryCreated(t *testing.T) {
	root, fakeFn, _ := setupTempProject(t)
	cfg := config.Config{TokenThreshold: 4100}

	if err := scanner.Scan(root, cfg, fakeFn); err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	summaryPath := filepath.Join(root, ".myhelper", "summaries", "myapp.md")
	info, err := os.Stat(summaryPath)
	if err != nil {
		t.Fatalf("summaries/myapp.md not found: %v", err)
	}
	if info.Size() == 0 {
		t.Fatal("summaries/myapp.md is empty")
	}
}

// TestScan_EntryFieldsPopulated verifies that index.json entries have path,
// package "myapp", and a non-empty Symbols slice.
func TestScan_EntryFieldsPopulated(t *testing.T) {
	root, fakeFn, _ := setupTempProject(t)
	cfg := config.Config{TokenThreshold: 4100}

	if err := scanner.Scan(root, cfg, fakeFn); err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	indexPath := filepath.Join(root, ".myhelper", "index.json")
	data, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("index.json not found: %v", err)
	}

	var entries []scanner.FileEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}

	found := false
	for _, e := range entries {
		if e.Package == "myapp" {
			found = true
			if e.Path == "" {
				t.Error("entry.Path is empty")
			}
			if len(e.Symbols) == 0 {
				t.Error("entry.Symbols is empty, expected at least Hello")
			}
		}
	}
	if !found {
		t.Error("no entry with package \"myapp\" found in index.json")
	}
}

// TestScan_GitDirExcluded verifies that files inside .git/ are not indexed.
func TestScan_GitDirExcluded(t *testing.T) {
	root, fakeFn, _ := setupTempProject(t)

	// Create a .go file inside a .git subdirectory.
	gitDir := filepath.Join(root, ".git")
	if err := os.MkdirAll(gitDir, 0755); err != nil {
		t.Fatalf("mkdir .git: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(gitDir, "hooks.go"),
		[]byte("package git\nfunc Hook() {}"),
		0644,
	); err != nil {
		t.Fatalf("write .git/hooks.go: %v", err)
	}

	cfg := config.Config{TokenThreshold: 4100}
	if err := scanner.Scan(root, cfg, fakeFn); err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	indexPath := filepath.Join(root, ".myhelper", "index.json")
	data, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("index.json not found: %v", err)
	}

	var entries []scanner.FileEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}

	for _, e := range entries {
		if filepath.HasPrefix(e.Path, ".git") || (len(e.Path) > 4 && e.Path[:4] == ".git") {
			t.Errorf("entry with path %q is under .git/ and should be excluded", e.Path)
		}
	}
}

// TestScan_ChatFnCalled verifies that the injected ChatFn is called at least once.
func TestScan_ChatFnCalled(t *testing.T) {
	root, fakeFn, calls := setupTempProject(t)
	cfg := config.Config{TokenThreshold: 4100}

	if err := scanner.Scan(root, cfg, fakeFn); err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	if *calls == 0 {
		t.Error("ChatFn was never called; expected at least 1 call")
	}
}
