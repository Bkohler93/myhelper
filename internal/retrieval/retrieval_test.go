package retrieval

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

	result, err := llmReRank("query", []scanner.Symbol{}, testCfg, countingChatFn)
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

	result, err := llmReRank("query", syms, testCfg, failingChatFn)
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
	result, err := llmReRank("query", syms, testCfg, makeFakeChatFn("pkg.Alpha"))
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

// ---------------------------------------------------------------------------
// TestAssembleMessages — staged assembly and budget stops (CTX-01, CTX-02)
// ---------------------------------------------------------------------------

func TestAssembleMessages_StageOrder(t *testing.T) {
	proj := scanner.ProjectArtifact{Summary: "A Go project."}
	syms := []scanner.Symbol{makeSymbol("DoThing", "pkg.DoThing", "func DoThing() error", "pkg/file.go")}
	filePaths := []string{"pkg/file.go"}
	strategy := Strategy{Name: "test", UseSymbols: true, UseFiles: false, MaxTokenRatio: 0.80}

	msgs := assembleMessages("my query", proj, syms, filePaths, "/tmp", strategy, testCfg, makeFakeChatFn(""))
	if len(msgs) == 0 {
		t.Fatal("expected non-empty messages")
	}
	content := msgs[0].Content
	projIdx := strings.Index(content, "## Project")
	symIdx := strings.Index(content, "### Relevant Symbols")
	fileIdx := strings.Index(content, "### Selected Files")
	if projIdx == -1 {
		t.Error("expected '## Project' section in output")
	}
	if symIdx == -1 {
		t.Error("expected '### Relevant Symbols' section in output")
	}
	if projIdx != -1 && symIdx != -1 && projIdx > symIdx {
		t.Errorf("expected project summary before symbols: projIdx=%d symIdx=%d", projIdx, symIdx)
	}
	if symIdx != -1 && fileIdx != -1 && symIdx > fileIdx {
		t.Errorf("expected symbols before file list: symIdx=%d fileIdx=%d", symIdx, fileIdx)
	}
}

func TestAssembleMessages_BudgetStop_Symbols(t *testing.T) {
	proj := scanner.ProjectArtifact{} // no summary
	// Each signature is "func SymbolNNNNNNNNNN() error" — ~8 tokens each
	syms := []scanner.Symbol{
		makeSymbol("Alpha", "pkg.Alpha", "func AlphaLongNameToConsumeTokens() error", "pkg/a.go"),
		makeSymbol("Beta", "pkg.Beta", "func BetaLongNameToConsumeTokens() error", "pkg/b.go"),
		makeSymbol("Gamma", "pkg.Gamma", "func GammaLongNameToConsumeTokens() error", "pkg/c.go"),
	}
	// Budget so tiny it cannot fit all 3 symbols
	tinyCfg := config.Config{TokenThreshold: 10}
	strategy := Strategy{Name: "test", UseSymbols: true, UseFiles: false, MaxTokenRatio: 0.30}

	msgs := assembleMessages("query", proj, syms, nil, "/tmp", strategy, tinyCfg, makeFakeChatFn(""))
	content := msgs[0].Content
	// Count how many symbols appear
	alphaIn := strings.Contains(content, "pkg.Alpha")
	betaIn := strings.Contains(content, "pkg.Beta")
	gammaIn := strings.Contains(content, "pkg.Gamma")
	all3 := alphaIn && betaIn && gammaIn
	if all3 {
		t.Error("expected budget stop to exclude at least one symbol; all 3 were included")
	}
}

func TestAssembleMessages_BudgetStop_ProjectSummary(t *testing.T) {
	longSummary := strings.Repeat("word ", 100) // ~100 tokens
	proj := scanner.ProjectArtifact{Summary: longSummary}
	tinyCfg := config.Config{TokenThreshold: 10} // budget = 0.80 * 10 = 8 tokens
	strategy := Strategy{Name: "test", UseSymbols: false, UseFiles: false, MaxTokenRatio: 0.80}

	msgs := assembleMessages("query", proj, nil, nil, "/tmp", strategy, tinyCfg, makeFakeChatFn(""))
	content := msgs[0].Content
	if strings.Contains(content, "word word word") {
		t.Error("expected project summary to be excluded when budget too small")
	}
}

// ---------------------------------------------------------------------------
// TestStrategy_* — per-command Strategy variable values (CTX-04)
// ---------------------------------------------------------------------------

func TestStrategy_Plan(t *testing.T) {
	if PlanStrategy.Name != "plan" {
		t.Errorf("PlanStrategy.Name: want %q got %q", "plan", PlanStrategy.Name)
	}
	if PlanStrategy.UseSymbols != false {
		t.Error("PlanStrategy.UseSymbols should be false (summaries only per CTX-04)")
	}
	if PlanStrategy.UseFiles != false {
		t.Error("PlanStrategy.UseFiles should be false")
	}
	if PlanStrategy.MaxTokenRatio != 0.50 {
		t.Errorf("PlanStrategy.MaxTokenRatio: want 0.50 got %v", PlanStrategy.MaxTokenRatio)
	}
}

func TestStrategy_Starter(t *testing.T) {
	if StarterStrategy.Name != "starter" {
		t.Errorf("StarterStrategy.Name: want %q got %q", "starter", StarterStrategy.Name)
	}
	if StarterStrategy.UseSymbols != true {
		t.Error("StarterStrategy.UseSymbols should be true")
	}
	if StarterStrategy.UseFiles != true {
		t.Error("StarterStrategy.UseFiles should be true (expands to file content per CTX-04)")
	}
	if StarterStrategy.MaxTokenRatio != 0.80 {
		t.Errorf("StarterStrategy.MaxTokenRatio: want 0.80 got %v", StarterStrategy.MaxTokenRatio)
	}
}

func TestStrategy_Lookup(t *testing.T) {
	if LookupStrategy.Name != "lookup" {
		t.Errorf("LookupStrategy.Name: want %q got %q", "lookup", LookupStrategy.Name)
	}
	if LookupStrategy.UseSymbols != true {
		t.Error("LookupStrategy.UseSymbols should be true (minimal context per CTX-04)")
	}
	if LookupStrategy.UseFiles != false {
		t.Error("LookupStrategy.UseFiles should be false")
	}
	if LookupStrategy.MaxTokenRatio != 0.30 {
		t.Errorf("LookupStrategy.MaxTokenRatio: want 0.30 got %v", LookupStrategy.MaxTokenRatio)
	}
}

func TestStrategy_Pattern(t *testing.T) {
	if PatternStrategy.Name != "pattern" {
		t.Errorf("PatternStrategy.Name: want %q got %q", "pattern", PatternStrategy.Name)
	}
	if PatternStrategy.UseSymbols != false {
		t.Error("PatternStrategy.UseSymbols should be false (near-zero context per CTX-04)")
	}
	if PatternStrategy.UseFiles != false {
		t.Error("PatternStrategy.UseFiles should be false")
	}
	if PatternStrategy.MaxTokenRatio != 0.10 {
		t.Errorf("PatternStrategy.MaxTokenRatio: want 0.10 got %v", PatternStrategy.MaxTokenRatio)
	}
}

// ---------------------------------------------------------------------------
// writeArtifacts — test helper for BuildInspectContext tests
// ---------------------------------------------------------------------------

// writeArtifacts writes minimal four-artifact files to root/.myhelper/ for tests.
func writeArtifacts(t *testing.T, root string, syms []scanner.Symbol) {
	t.Helper()
	dir := filepath.Join(root, ".myhelper")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("writeArtifacts: mkdir: %v", err)
	}
	proj := scanner.ProjectArtifact{SchemaVersion: "1.0", Summary: "test project"}
	pkgs := scanner.PackagesArtifact{SchemaVersion: "1.0"}
	files := scanner.FilesArtifact{SchemaVersion: "1.0", Files: []scanner.FileArtifactEntry{
		{Path: "internal/pkg/file.go", Package: "pkg"},
	}}
	symArtifact := scanner.SymbolsArtifact{SchemaVersion: "1.0", Symbols: syms}
	writeJSON := func(name string, v interface{}) {
		data, err := json.Marshal(v)
		if err != nil {
			t.Fatalf("writeArtifacts: marshal %s: %v", name, err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), data, 0644); err != nil {
			t.Fatalf("writeArtifacts: write %s: %v", name, err)
		}
	}
	writeJSON("project.json", proj)
	writeJSON("packages.json", pkgs)
	writeJSON("files.json", files)
	writeJSON("symbols.json", symArtifact)
}

// ---------------------------------------------------------------------------
// TestSelectionSource_String (CMD-02)
// ---------------------------------------------------------------------------

func TestSelectionSource_String(t *testing.T) {
	cases := []struct {
		src      SelectionSource
		expected string
	}{
		{SourcePreFilter, "pre-filter"},
		{SourceReRank, "re-rank"},
		{SourceExpansion, "expansion"},
	}
	for _, tc := range cases {
		got := tc.src.String()
		if got != tc.expected {
			t.Errorf("SelectionSource(%d).String() = %q, want %q", tc.src, got, tc.expected)
		}
	}
}

// ---------------------------------------------------------------------------
// TestBuildInspectContext (CMD-02)
// ---------------------------------------------------------------------------

func TestBuildInspectContext_NoArtifacts(t *testing.T) {
	tmpDir := t.TempDir()
	// No .myhelper/ directory — artifacts missing.
	result, err := BuildInspectContext(tmpDir, "test query", DefaultStrategy, testCfg, failingChatFn)
	if err != nil {
		t.Fatalf("expected nil error on missing artifacts, got %v", err)
	}
	if result.GatePassed {
		t.Error("expected GatePassed == false when no artifacts present")
	}
	if len(result.Symbols) != 0 {
		t.Errorf("expected no symbols, got %d", len(result.Symbols))
	}
	if len(result.StageMetrics) != 0 {
		t.Errorf("expected no stage metrics, got %d", len(result.StageMetrics))
	}
}

func TestBuildInspectContext_GateBlocks(t *testing.T) {
	tmpDir := t.TempDir()
	syms := []scanner.Symbol{
		makeSymbol("BuildContext", "retrieval.BuildContext", "func BuildContext(...) (Context, error)", "internal/retrieval/retrieval.go"),
	}
	writeArtifacts(t, tmpDir, syms)
	// noChatFn returns "no" — relevance gate should block all subsequent stages.
	result, err := BuildInspectContext(tmpDir, "what is 2+2", DefaultStrategy, testCfg, noChatFn)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.GatePassed {
		t.Error("expected GatePassed == false when gate returns 'no'")
	}
	if len(result.StageMetrics) != 0 {
		t.Errorf("expected no stage metrics when gate blocks, got %d", len(result.StageMetrics))
	}
}

func TestBuildInspectContext_WithSymbols(t *testing.T) {
	tmpDir := t.TempDir()
	sym := makeSymbol("BuildContext", "retrieval.BuildContext", "func BuildContext(...) (Context, error)", "internal/retrieval/retrieval.go")
	writeArtifacts(t, tmpDir, []scanner.Symbol{sym})

	// chatFn returns "yes" for the gate call (first call), then the stableID for re-rank (second call).
	callCount := 0
	chatFn := func(cfg config.Config, msgs []history.Message) (string, error) {
		callCount++
		if callCount == 1 {
			return "yes", nil // relevance gate passes
		}
		return "retrieval.BuildContext", nil // re-rank selects the symbol
	}

	result, err := BuildInspectContext(tmpDir, "how does BuildContext work", DefaultStrategy, testCfg, chatFn)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.GatePassed {
		t.Error("expected GatePassed == true when gate returns 'yes'")
	}
	if len(result.Symbols) == 0 {
		t.Error("expected at least one symbol in result")
	}
	if result.Symbols[0].Symbol.Name != "BuildContext" {
		t.Errorf("expected symbol BuildContext, got %q", result.Symbols[0].Symbol.Name)
	}
	if result.Symbols[0].Source != SourceReRank {
		t.Errorf("expected source SourceReRank, got %v", result.Symbols[0].Source)
	}
	foundPreFilter := false
	foundReRank := false
	for _, sm := range result.StageMetrics {
		if sm.Name == "pre-filter" {
			foundPreFilter = true
		}
		if sm.Name == "re-rank" {
			foundReRank = true
		}
	}
	if !foundPreFilter {
		t.Error("expected 'pre-filter' stage metric")
	}
	if !foundReRank {
		t.Error("expected 're-rank' stage metric")
	}
	if result.FinalTokens <= 0 {
		t.Error("expected FinalTokens > 0 when symbols are selected")
	}
}
