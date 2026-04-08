package scanner

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bkohler93/myhelper/internal/config"
	"github.com/bkohler93/myhelper/internal/history"
)

// mkTempProject creates a temp dir with .myhelper/ subdirectory.
// Returns root path. Caller is responsible for cleanup via t.Cleanup.
func mkTempProject(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".myhelper"), 0755); err != nil {
		t.Fatalf("mkTempProject: mkdir .myhelper: %v", err)
	}
	return root
}

// writeGoFile writes content to root/relPath, creating parent directories as needed.
func writeGoFile(t *testing.T, root, relPath, content string) {
	t.Helper()
	abs := filepath.Join(root, relPath)
	if err := os.MkdirAll(filepath.Dir(abs), 0755); err != nil {
		t.Fatalf("writeGoFile: mkdir: %v", err)
	}
	if err := os.WriteFile(abs, []byte(content), 0644); err != nil {
		t.Fatalf("writeGoFile: write %s: %v", relPath, err)
	}
}

// readIndex parses .myhelper/index.json in root and returns the []FileEntry.
func readIndex(t *testing.T, root string) []FileEntry {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, ".myhelper", "index.json"))
	if err != nil {
		t.Fatalf("readIndex: %v", err)
	}
	var entries []FileEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		t.Fatalf("readIndex: unmarshal: %v", err)
	}
	return entries
}

func defaultCfg() config.Config {
	return config.Config{
		Endpoint:       "localhost:11434",
		Model:          "test-model",
		TokenThreshold: 4100,
	}
}

// TestBuildIndex_BasicEntry verifies that a single .go file produces one FileEntry
// with correct Path, Package, Symbols, and a positive TokenCount.
func TestBuildIndex_BasicEntry(t *testing.T) {
	root := mkTempProject(t)
	writeGoFile(t, root, "foo.go", `package foo
func Bar(x int) string { return "" }
`)

	cfg := defaultCfg()
	entries, err := BuildIndex(root, cfg)
	if err != nil {
		t.Fatalf("BuildIndex: %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}

	entry := entries[0]
	if !strings.HasSuffix(entry.Path, "foo.go") {
		t.Errorf("Path = %q, want suffix 'foo.go'", entry.Path)
	}
	if entry.Package != "foo" {
		t.Errorf("Package = %q, want 'foo'", entry.Package)
	}
	found := false
	for _, sym := range entry.Symbols {
		if strings.HasPrefix(sym, "func Bar") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Symbols = %v, want an entry starting with 'func Bar'", entry.Symbols)
	}
	if entry.TokenCount <= 0 {
		t.Errorf("TokenCount = %d, want > 0", entry.TokenCount)
	}
}

// TestBuildIndex_TokenCountPerEntry verifies that TokenCount equals the cl100k_base
// token count of the JSON-serialized entry.
func TestBuildIndex_TokenCountPerEntry(t *testing.T) {
	root := mkTempProject(t)
	writeGoFile(t, root, "bar.go", `package bar
func Baz(a string, b int) error { return nil }
`)

	cfg := defaultCfg()
	entries, err := BuildIndex(root, cfg)
	if err != nil {
		t.Fatalf("BuildIndex: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("expected at least 1 entry")
	}

	entry := entries[0]
	// Re-compute expected token count: serialize entry, then count via history.New.
	entryJSON, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("json.Marshal entry: %v", err)
	}
	h := history.New(cfg.TokenThreshold, []history.Message{{Role: "user", Content: string(entryJSON)}})
	expected := h.TokenCount()

	if entry.TokenCount != expected {
		t.Errorf("TokenCount = %d, want %d (tokens of JSON repr)", entry.TokenCount, expected)
	}
}

// TestBuildIndex_BudgetCapDropsEntries verifies that when total tokens would exceed
// the 80% budget, entries are dropped until the total fits within budget.
func TestBuildIndex_BudgetCapDropsEntries(t *testing.T) {
	root := mkTempProject(t)

	// Write enough files that their total tokens will exceed a threshold of 100 (budget = 80).
	// Each file has a unique exported function to ensure non-trivial symbol content.
	for i := 0; i < 20; i++ {
		fname := filepath.Join("src", filepath.Join("pkg", filepath.Join("sub", filepath.Join("deep"))))
		_ = fname
		name := 'A' + rune(i)
		content := "package mypkg\n\nfunc Exported" + string(name) + "(x int, y string, z float64) (string, error) { return \"\", nil }\n"
		writeGoFile(t, root, "file"+string(name)+".go", content)
	}

	cfg := config.Config{
		Endpoint:       "localhost:11434",
		Model:          "test-model",
		TokenThreshold: 100, // budget = 80 tokens
	}

	entries, err := BuildIndex(root, cfg)
	if err != nil {
		t.Fatalf("BuildIndex: %v", err)
	}

	if len(entries) >= 20 {
		t.Errorf("expected entries to be dropped (got %d, want < 20)", len(entries))
	}

	// Verify total fits within budget.
	total := 0
	for _, e := range entries {
		total += e.TokenCount
	}
	budget := int(float64(cfg.TokenThreshold) * 0.80)
	if total > budget {
		t.Errorf("total token count %d exceeds budget %d", total, budget)
	}
}

// TestBuildIndex_TestFilesDroppedFirst verifies that _test.go files are dropped
// before non-test files when the budget is exceeded.
func TestBuildIndex_TestFilesDroppedFirst(t *testing.T) {
	root := mkTempProject(t)

	// Write a normal file and a test file.
	writeGoFile(t, root, "main.go", `package main
func Run(a string, b int, c float64, d bool) (string, error) { return "", nil }
func Setup(x int, y string) error { return nil }
func Teardown(z float64) string { return "" }
`)
	writeGoFile(t, root, "main_test.go", `package main
func TestRun(a string, b int) error { return nil }
func TestSetup(x int) string { return "" }
`)

	// Use a very small threshold so that both files together exceed budget
	// but at least the non-test file alone fits.
	cfg := config.Config{
		Endpoint:       "localhost:11434",
		Model:          "test-model",
		TokenThreshold: 40, // budget = 32 tokens
	}

	entries, err := BuildIndex(root, cfg)
	if err != nil {
		t.Fatalf("BuildIndex: %v", err)
	}

	// If any entries remain, none should be _test.go files (they were dropped first).
	for _, e := range entries {
		if strings.HasSuffix(e.Path, "_test.go") {
			t.Errorf("_test.go file %q was retained when budget exceeded; should be dropped first", e.Path)
		}
	}
}

// TestBuildIndex_OutputFileExists verifies that .myhelper/index.json is created.
func TestBuildIndex_OutputFileExists(t *testing.T) {
	root := mkTempProject(t)
	writeGoFile(t, root, "pkg.go", `package pkg
func Hello() string { return "hello" }
`)

	cfg := defaultCfg()
	if _, err := BuildIndex(root, cfg); err != nil {
		t.Fatalf("BuildIndex: %v", err)
	}

	indexPath := filepath.Join(root, ".myhelper", "index.json")
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		t.Errorf(".myhelper/index.json does not exist after BuildIndex")
	}
}

// TestBuildIndex_ValidJSONArray verifies that index.json contains a valid JSON array.
func TestBuildIndex_ValidJSONArray(t *testing.T) {
	root := mkTempProject(t)
	writeGoFile(t, root, "pkg.go", `package pkg
func Hello() string { return "hello" }
`)

	cfg := defaultCfg()
	if _, err := BuildIndex(root, cfg); err != nil {
		t.Fatalf("BuildIndex: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(root, ".myhelper", "index.json"))
	if err != nil {
		t.Fatalf("read index.json: %v", err)
	}
	var entries []FileEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		t.Errorf("index.json is not a valid JSON array: %v", err)
	}
}

// TestBuildIndex_NoGoFiles verifies that an empty JSON array is written when
// there are no .go files in the project.
func TestBuildIndex_NoGoFiles(t *testing.T) {
	root := mkTempProject(t)
	// No .go files — just the .myhelper dir.

	cfg := defaultCfg()
	entries, err := BuildIndex(root, cfg)
	if err != nil {
		t.Fatalf("BuildIndex: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(entries))
	}

	data, err := os.ReadFile(filepath.Join(root, ".myhelper", "index.json"))
	if err != nil {
		t.Fatalf("read index.json: %v", err)
	}
	// json.MarshalIndent([]FileEntry{}, ...) produces "[]"
	trimmed := strings.TrimSpace(string(data))
	if trimmed != "[]" {
		t.Errorf("index.json = %q, want '[]'", trimmed)
	}
}
