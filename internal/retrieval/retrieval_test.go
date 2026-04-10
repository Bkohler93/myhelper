package retrieval

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/bkohler93/myhelper/internal/config"
	"github.com/bkohler93/myhelper/internal/history"
	"github.com/bkohler93/myhelper/internal/scanner"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func makeSymbol(name, stableID, signature, filePath string) scanner.Symbol {
	return scanner.Symbol{
		Name:      name,
		Kind:      "func",
		Signature: signature,
		StableID:  stableID,
		FilePath:  filePath,
	}
}

func makeFileEntries(n int) []scanner.FileArtifactEntry {
	entries := make([]scanner.FileArtifactEntry, n)
	for i := range entries {
		entries[i] = scanner.FileArtifactEntry{
			Path:    fmt.Sprintf("internal/pkg%d/file.go", i),
			Package: fmt.Sprintf("pkg%d", i),
		}
	}
	return entries
}

func makeFakeChatFn(response string) scanner.ChatFn {
	return func(cfg config.Config, msgs []history.Message) (string, error) {
		return response, nil
	}
}

var failingChatFn scanner.ChatFn = func(cfg config.Config, msgs []history.Message) (string, error) {
	return "", fmt.Errorf("llm unavailable")
}

var noChatFn scanner.ChatFn = func(cfg config.Config, msgs []history.Message) (string, error) {
	return "no", nil
}

var testCfg = config.Config{TokenThreshold: 4100}

// ---------------------------------------------------------------------------
// TestPreFilter
// ---------------------------------------------------------------------------

func TestPreFilter_LargeCorpus(t *testing.T) {
	// 50 file entries → large corpus path (> smallCorpusThreshold=40)
	files := makeFileEntries(50)
	syms := []scanner.Symbol{
		makeSymbol("BuildContext", "retrieval.BuildContext", "func BuildContext(...) (Context, error)", "internal/retrieval/retrieval.go"),
		makeSymbol("Unrelated", "pkg.Unrelated", "func Unrelated()", "internal/pkg/other.go"),
	}

	result := preFilter("BuildContext", syms, files)

	if len(result) == 0 {
		t.Fatal("expected at least one result, got empty slice")
	}
	if result[0].Name != "BuildContext" {
		t.Errorf("expected top result to be BuildContext, got %q", result[0].Name)
	}
}

func TestPreFilter_SmallCorpus(t *testing.T) {
	// 10 file entries → small corpus path (≤ smallCorpusThreshold=40)
	// None of the symbols match the query — all should be returned (additive hint per RET-02)
	files := makeFileEntries(10)
	syms := []scanner.Symbol{
		makeSymbol("Alpha", "pkg.Alpha", "func Alpha()", "internal/pkg/alpha.go"),
		makeSymbol("Beta", "pkg.Beta", "func Beta()", "internal/pkg/beta.go"),
		makeSymbol("Gamma", "pkg.Gamma", "func Gamma()", "internal/pkg/gamma.go"),
	}

	result := preFilter("xyzzy_no_match", syms, files)

	if len(result) != len(syms) {
		t.Errorf("small corpus: expected all %d symbols returned, got %d", len(syms), len(result))
	}
}

func TestPreFilter_EmptySymbols(t *testing.T) {
	files := makeFileEntries(5)
	result := preFilter("anything", []scanner.Symbol{}, files)
	if result == nil {
		// nil is acceptable for empty input — just must not panic
	}
	if len(result) != 0 {
		t.Errorf("expected empty result for empty input, got %d", len(result))
	}
}

// ---------------------------------------------------------------------------
// TestRelevanceGate
// ---------------------------------------------------------------------------

func TestRelevanceGate_FailsOpen(t *testing.T) {
	result := needsContext("how do I add a feature?", "", testCfg, failingChatFn)
	if !result {
		t.Error("expected needsContext to return true (fail open) when chatFn errors")
	}
}

func TestRelevanceGate_NoResponse(t *testing.T) {
	result := needsContext("what is 2+2?", "", testCfg, noChatFn)
	if result {
		t.Error("expected needsContext to return false when chatFn returns 'no'")
	}
}

func TestRelevanceGate_YesResponse(t *testing.T) {
	result := needsContext("how does BuildContext work?", "", testCfg, makeFakeChatFn("yes"))
	if !result {
		t.Error("expected needsContext to return true when chatFn returns 'yes'")
	}
}

// ---------------------------------------------------------------------------
// TestRerank
// ---------------------------------------------------------------------------

func TestRerank_EmptyInput(t *testing.T) {
	callCount := 0
	countingChatFn := func(cfg config.Config, msgs []history.Message) (string, error) {
		callCount++
		return "some.StableID", nil
	}

	result, err := llmReRank("query", []scanner.Symbol{}, nil, testCfg, countingChatFn)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty result, got %d symbols", len(result))
	}
	if callCount != 0 {
		t.Errorf("chatFn should not be called for empty input, was called %d times", callCount)
	}
}

func TestRerank_Fallback(t *testing.T) {
	syms := []scanner.Symbol{
		makeSymbol("Alpha", "pkg.Alpha", "func Alpha()", "internal/pkg/alpha.go"),
		makeSymbol("Beta", "pkg.Beta", "func Beta()", "internal/pkg/beta.go"),
	}

	result, err := llmReRank("query", syms, nil, testCfg, failingChatFn)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != len(syms) {
		t.Errorf("expected all %d candidates returned on LLM failure, got %d", len(syms), len(result))
	}
}

func TestRerank_FiltersByStableID(t *testing.T) {
	syms := []scanner.Symbol{
		makeSymbol("Alpha", "pkg.Alpha", "func Alpha()", "internal/pkg/alpha.go"),
		makeSymbol("Beta", "pkg.Beta", "func Beta()", "internal/pkg/beta.go"),
		makeSymbol("Gamma", "pkg.Gamma", "func Gamma()", "internal/pkg/gamma.go"),
	}

	// chatFn returns only "pkg.Alpha" — only that symbol should be selected
	result, err := llmReRank("query", syms, nil, testCfg, makeFakeChatFn("pkg.Alpha"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Errorf("expected 1 symbol filtered by stableID, got %d", len(result))
	}
	if result[0].StableID != "pkg.Alpha" {
		t.Errorf("expected stableID pkg.Alpha, got %q", result[0].StableID)
	}
}

// ---------------------------------------------------------------------------
// TestDependencyExpansion
// ---------------------------------------------------------------------------

func TestDependencyExpansion_BudgetCap(t *testing.T) {
	// Create a FilesArtifact where neighbor files are expensive (large names that
	// consume tokens), set a very tight budget, and verify no overflow.
	selected := []string{"internal/pkg/main.go"}

	// A neighbor file whose path is referenced via imports of the selected file
	neighbor := scanner.FileArtifactEntry{
		Path:    "internal/neighbor/file.go",
		Package: "neighbor",
		Imports: []string{},
	}
	// The selected file imports the neighbor package
	selectedFile := scanner.FileArtifactEntry{
		Path:    "internal/pkg/main.go",
		Package: "pkg",
		Imports: []string{"github.com/bkohler93/myhelper/internal/neighbor"},
	}

	fa := scanner.FilesArtifact{
		SchemaVersion: "1.0",
		Files:         []scanner.FileArtifactEntry{selectedFile, neighbor},
	}

	// Budget of 0 — no expansion should happen
	result := expandDeps(selected, fa, 0, testCfg)
	if len(result) != 1 {
		t.Errorf("budget cap of 0 should prevent expansion; expected 1 path, got %d", len(result))
	}
	if result[0] != "internal/pkg/main.go" {
		t.Errorf("expected original selected path, got %q", result[0])
	}
}

func TestDependencyExpansion_NoOverlap(t *testing.T) {
	// expandDeps should not re-add files already in the selected set
	selectedPaths := []string{"internal/pkg/main.go", "internal/neighbor/file.go"}

	selectedFile := scanner.FileArtifactEntry{
		Path:    "internal/pkg/main.go",
		Package: "pkg",
		Imports: []string{"github.com/bkohler93/myhelper/internal/neighbor"},
	}
	neighbor := scanner.FileArtifactEntry{
		Path:    "internal/neighbor/file.go",
		Package: "neighbor",
		Imports: []string{},
	}

	fa := scanner.FilesArtifact{
		SchemaVersion: "1.0",
		Files:         []scanner.FileArtifactEntry{selectedFile, neighbor},
	}

	result := expandDeps(selectedPaths, fa, 10000, testCfg)

	// Verify no duplicates
	seen := make(map[string]int)
	for _, p := range result {
		seen[p]++
	}
	for path, count := range seen {
		if count > 1 {
			t.Errorf("file %q appears %d times in result (expected once)", path, count)
		}
	}
}

// ---------------------------------------------------------------------------
// TestBuildContext
// ---------------------------------------------------------------------------

func TestBuildContext_NoArtifacts(t *testing.T) {
	root := "/tmp/retrieval_test_nonexistent_dir_xyzzy"
	ctx, err := BuildContext(root, "how does the scanner work?", DefaultStrategy, testCfg, failingChatFn)

	if err != nil {
		t.Fatalf("expected no error when artifacts missing, got: %v", err)
	}
	if len(ctx.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(ctx.Messages))
	}
	if ctx.Messages[0].Content != "how does the scanner work?" {
		t.Errorf("expected bare user query content, got %q", ctx.Messages[0].Content)
	}
}

func TestBuildContext_Integration(t *testing.T) {
	// Create a temp dir with .myhelper/ subdirectory containing minimal valid artifacts
	tmpDir := t.TempDir()
	myhelperDir := filepath.Join(tmpDir, ".myhelper")
	if err := os.MkdirAll(myhelperDir, 0755); err != nil {
		t.Fatalf("failed to create .myhelper dir: %v", err)
	}

	// Write minimal artifact files
	projectArtifact := scanner.ProjectArtifact{
		SchemaVersion: "1.0",
		ModulePath:    "github.com/test/project",
		GoVersion:     "1.24",
		FileCount:     1,
		SymbolCount:   1,
		Summary:       "A test project.",
	}
	writeTestJSON(t, filepath.Join(myhelperDir, "project.json"), projectArtifact)

	pkgsArtifact := scanner.PackagesArtifact{
		SchemaVersion: "1.0",
		Packages: []scanner.PackageEntry{
			{ImportPath: "github.com/test/project/pkg", Files: []string{"pkg/file.go"}, Responsibility: "test package"},
		},
	}
	writeTestJSON(t, filepath.Join(myhelperDir, "packages.json"), pkgsArtifact)

	filesArtifact := scanner.FilesArtifact{
		SchemaVersion: "1.0",
		Files: []scanner.FileArtifactEntry{
			{Path: "pkg/file.go", Package: "pkg", ExportedNames: []string{"DoThing"}, Imports: []string{}},
		},
	}
	writeTestJSON(t, filepath.Join(myhelperDir, "files.json"), filesArtifact)

	symbolsArtifact := scanner.SymbolsArtifact{
		SchemaVersion: "1.0",
		Symbols: []scanner.Symbol{
			makeSymbol("DoThing", "pkg.DoThing", "func DoThing() error", "pkg/file.go"),
		},
	}
	writeTestJSON(t, filepath.Join(myhelperDir, "symbols.json"), symbolsArtifact)

	// chatFn: "yes" for relevance gate; empty string for re-rank (triggers empty-selection fallback).
	// Re-rank calls include a system message as msgs[0]; relevance gate uses a single user message.
	noopChatFn := func(cfg config.Config, msgs []history.Message) (string, error) {
		if len(msgs) >= 2 && msgs[0].Role == "system" {
			return "", nil // re-rank: return empty → fallback to all candidates
		}
		return "yes", nil // relevance gate
	}

	ctx, err := BuildContext(tmpDir, "how do I use DoThing?", DefaultStrategy, testCfg, noopChatFn)
	if err != nil {
		t.Fatalf("unexpected error from BuildContext: %v", err)
	}
	if len(ctx.Messages) == 0 {
		t.Fatal("expected non-empty Messages from BuildContext with valid artifacts")
	}
}

// ---------------------------------------------------------------------------
// Test helper
// ---------------------------------------------------------------------------

func writeTestJSON(t *testing.T, path string, v interface{}) {
	t.Helper()
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		t.Fatalf("marshal %s: %v", path, err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
