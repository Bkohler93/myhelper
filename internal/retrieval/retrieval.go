// Package retrieval implements the four-stage context retrieval pipeline:
// relevance gate → keyword pre-filter → LLM re-ranking → dependency expansion.
//
// Entry point: BuildContext.
package retrieval

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/bkohler93/myhelper/internal/config"
	"github.com/bkohler93/myhelper/internal/history"
	"github.com/bkohler93/myhelper/internal/scanner"
)

// Budget constants — match the safety factors established in internal/scanner/index.go.
const (
	contextBudgetFactor   = 0.80 // overall context budget as fraction of TokenThreshold
	expansionBudgetFactor = 0.60 // dependency expansion as fraction of REMAINING budget (RET-05)
	smallCorpusThreshold  = 40   // file count at or below which pre-filter acts as additive hint (RET-02)
)

// Strategy configures which pipeline stages run and at what depth.
// Phase 12 will wire per-command strategies; Phase 11 uses a single default.
type Strategy struct {
	Name          string  // "plan" | "starter" | "lookup" | "pattern"
	UseSymbols    bool    // include symbol-level context
	UseFiles      bool    // expand to file content
	MaxTokenRatio float64 // fraction of cfg.TokenThreshold available for context (e.g. 0.80)
}

// DefaultStrategy is used when the caller does not specify a strategy.
var DefaultStrategy = Strategy{
	Name:          "default",
	UseSymbols:    true,
	UseFiles:      true,
	MaxTokenRatio: contextBudgetFactor,
}

// PlanStrategy: project summary only. Commands need token budget for multi-turn.
var PlanStrategy = Strategy{
	Name:          "plan",
	UseSymbols:    false,
	UseFiles:      false,
	MaxTokenRatio: 0.50,
}

// StarterStrategy: symbols + file content when budget allows.
var StarterStrategy = Strategy{
	Name:          "starter",
	UseSymbols:    true,
	UseFiles:      true,
	MaxTokenRatio: contextBudgetFactor, // 0.80
}

// LookupStrategy: symbol signatures only, minimal ratio.
var LookupStrategy = Strategy{
	Name:          "lookup",
	UseSymbols:    true,
	UseFiles:      false,
	MaxTokenRatio: 0.30,
}

// PatternStrategy: near-zero context; idiomatic Go questions rarely need project files.
var PatternStrategy = Strategy{
	Name:          "pattern",
	UseSymbols:    false,
	UseFiles:      false,
	MaxTokenRatio: 0.10,
}

// Context holds the outputs of the retrieval pipeline.
type Context struct {
	Symbols  []scanner.Symbol  // selected symbols from re-ranking or pre-filter
	Files    []string          // selected file paths (relative to root)
	Messages []history.Message // final assembled messages ready for the LLM
}

// BuildContext runs the four-stage retrieval pipeline and returns selected symbols,
// selected files, and final assembled messages.
//
// Pipeline: relevance gate → pre-filter → LLM re-ranking → dependency expansion.
//
// If artifacts do not exist (project not initialised), BuildContext returns a bare
// user-query message without error so callers can degrade gracefully.
func BuildContext(
	root string,
	query string,
	strategy Strategy,
	cfg config.Config,
	chatFn scanner.ChatFn,
) (Context, error) {
	proj, _, files, syms, err := loadArtifacts(root)
	if err != nil {
		// No artifacts — return bare user query; caller falls back to unaugmented prompt.
		return Context{Messages: []history.Message{{Role: "user", Content: query}}}, nil
	}

	// Short-circuit for near-zero strategies (e.g. PatternStrategy): skip LLM calls entirely.
	if !strategy.UseSymbols && !strategy.UseFiles && strategy.MaxTokenRatio <= 0.10 {
		msgs := assembleMessages(query, proj, nil, nil, root, strategy, cfg, chatFn)
		return Context{Messages: msgs}, nil
	}

	// Stage 1: Relevance gate (RET-04)
	if !needsContext(query, proj.Summary, cfg, chatFn) {
		return Context{Messages: []history.Message{{Role: "user", Content: query}}}, nil
	}

	// Budget tracking
	totalBudget := int(float64(cfg.TokenThreshold) * contextBudgetFactor)
	usedTokens := 0

	// Stage 2: Deterministic pre-filter (RET-01, RET-02)
	candidates := preFilter(query, syms.Symbols, files.Files)
	candidates = applyTokenCap(candidates, totalBudget, cfg)

	// Stage 3: LLM re-ranking (RET-03)
	selected, reRankErr := llmReRank(query, candidates, cfg, chatFn)
	if reRankErr != nil {
		selected = candidates // fallback: use all candidates on LLM error
	}

	// Count tokens for the final selected set only (drives expansion budget)
	usedTokens = 0
	for _, s := range selected {
		usedTokens += tokenCount(cfg, s.Signature)
	}

	// Stage 4: Dependency expansion (RET-05)
	selectedPaths := uniqueFilePaths(selected)
	expansionBudget := int(float64(totalBudget-usedTokens) * expansionBudgetFactor)
	expandedPaths := expandDeps(selectedPaths, files, expansionBudget, cfg)

	// Assemble messages
	msgs := assembleMessages(query, proj, selected, expandedPaths, root, strategy, cfg, chatFn)

	return Context{
		Symbols:  selected,
		Files:    expandedPaths,
		Messages: msgs,
	}, nil
}

// -----------------------------------------------------------------------------
// Stage 1: Relevance Gate (RET-04)
// -----------------------------------------------------------------------------

const relevanceGatePrompt = `Answer only "yes" or "no". Does answering the following query require looking at the project's source code? Query: `

// needsContext returns true if the query requires repository context.
// Fails open: any response that does not clearly start with "no" is treated as "yes".
func needsContext(query, projectSummary string, cfg config.Config, chatFn scanner.ChatFn) bool {
	msg := relevanceGatePrompt + query
	if projectSummary != "" {
		msg = "Project: " + projectSummary + "\n\n" + msg
	}
	messages := []history.Message{
		{Role: "user", Content: msg},
	}
	response, err := chatFn(cfg, messages)
	if err != nil {
		return true // fail open — context omission is worse than extra tokens
	}
	lower := strings.ToLower(strings.TrimSpace(response))
	return !strings.HasPrefix(lower, "no")
}

// -----------------------------------------------------------------------------
// Stage 2: Deterministic Pre-Filter (RET-01, RET-02)
// -----------------------------------------------------------------------------

// preFilter scores each symbol against query terms and returns scored candidates.
// When the corpus is small (≤ smallCorpusThreshold files), all symbols pass through
// as additive hints (RET-02); the token cap trims them afterwards.
func preFilter(query string, symbols []scanner.Symbol, files []scanner.FileArtifactEntry) []scanner.Symbol {
	terms := strings.Fields(strings.ToLower(query))

	// Small corpus path (RET-02): treat all symbols as candidates when ≤ 40 files.
	if len(files) <= smallCorpusThreshold {
		if len(symbols) == 0 {
			return symbols
		}
		// Still score so that the most relevant rise to the top after cap.
		return scoreAndSort(symbols, terms)
	}

	// Large corpus path (RET-01): only include symbols with at least one term match.
	type scored struct {
		sym   scanner.Symbol
		score int
	}
	var results []scored
	for _, sym := range symbols {
		score := scoreSymbol(sym, terms)
		if score > 0 {
			results = append(results, scored{sym, score})
		}
	}
	if len(results) == 0 {
		// Pitfall 3 guard: never return empty on large corpus — return top-N by name length
		// (a rough proxy for more specific/descriptive names).
		return scoreAndSort(symbols, terms)
	}
	sort.Slice(results, func(i, j int) bool { return results[i].score > results[j].score })
	out := make([]scanner.Symbol, 0, len(results))
	for _, r := range results {
		out = append(out, r.sym)
	}
	return out
}

func scoreSymbol(sym scanner.Symbol, terms []string) int {
	haystack := strings.ToLower(sym.Name + " " + sym.Signature + " " + sym.StableID)
	score := 0
	for _, term := range terms {
		if strings.Contains(haystack, term) {
			score++
		}
	}
	return score
}

func scoreAndSort(symbols []scanner.Symbol, terms []string) []scanner.Symbol {
	type scored struct {
		sym   scanner.Symbol
		score int
	}
	results := make([]scored, len(symbols))
	for i, sym := range symbols {
		results[i] = scored{sym, scoreSymbol(sym, terms)}
	}
	sort.Slice(results, func(i, j int) bool { return results[i].score > results[j].score })
	out := make([]scanner.Symbol, len(results))
	for i, r := range results {
		out[i] = r.sym
	}
	return out
}

// applyTokenCap trims candidates to fit within the given token budget.
func applyTokenCap(candidates []scanner.Symbol, budget int, cfg config.Config) []scanner.Symbol {
	used := 0
	var result []scanner.Symbol
	for _, c := range candidates {
		cost := tokenCount(cfg, c.Signature)
		if used+cost > budget {
			break
		}
		result = append(result, c)
		used += cost
	}
	return result
}

// -----------------------------------------------------------------------------
// Stage 3: LLM Re-Ranking (RET-03)
// -----------------------------------------------------------------------------

const reRankSystemPrompt = `You are a code search assistant. Given the user's query and a list of candidate symbols with their signatures, output ONLY the stable identifiers (stableID) of symbols that are directly relevant to the query, one per line. Output nothing else.`

// llmReRank asks the LLM to confirm which candidates are relevant.
// Falls back to all candidates on LLM failure or empty response (Pitfall 7 guard).
func llmReRank(
	query string,
	candidates []scanner.Symbol,
	cfg config.Config,
	chatFn scanner.ChatFn,
) ([]scanner.Symbol, error) {
	if len(candidates) == 0 {
		return candidates, nil // Pitfall 7: guard against empty input
	}

	var sb strings.Builder
	for _, sym := range candidates {
		sb.WriteString(sym.StableID + ": " + sym.Signature + "\n")
	}

	messages := []history.Message{
		{Role: "system", Content: reRankSystemPrompt},
		{Role: "user", Content: "Query: " + query + "\n\nCandidates:\n" + sb.String()},
	}
	response, err := chatFn(cfg, messages)
	if err != nil {
		return candidates, nil // fallback: return all candidates on LLM failure
	}

	selected := filterByStableIDs(candidates, response)
	if len(selected) == 0 {
		return candidates, nil // fallback: LLM returned nothing useful
	}
	return selected, nil
}

// filterByStableIDs returns candidates whose StableID appears in the LLM response.
func filterByStableIDs(candidates []scanner.Symbol, response string) []scanner.Symbol {
	responseLines := strings.Fields(response)
	seen := make(map[string]bool, len(responseLines))
	for _, line := range responseLines {
		seen[strings.TrimSpace(line)] = true
	}
	var result []scanner.Symbol
	for _, c := range candidates {
		if seen[c.StableID] {
			result = append(result, c)
		}
	}
	return result
}

// -----------------------------------------------------------------------------
// Stage 4: Dependency Expansion (RET-05)
// -----------------------------------------------------------------------------

// expandDeps adds depth-1 import neighbors of selectedPaths that are project-internal files.
// The expansion is bounded by budgetCap tokens (RET-05: ≤ 60% of remaining budget).
func expandDeps(
	selectedPaths []string,
	filesArtifact scanner.FilesArtifact,
	budgetCap int,
	cfg config.Config,
) []string {
	// Build import-path → []relFilePath map from files.json.
	// FileArtifactEntry.Imports contains full module-qualified import paths.
	// We match against the derived import path for each file entry (Pitfall 6 guard).
	importToFiles := make(map[string][]string)
	for _, fe := range filesArtifact.Files {
		// Derive this file's import path: it is stored in the FileArtifactEntry itself
		// as the Package field (short name), but we need the full path.
		// Use the file path to find it: strip filename, join with module path prefix.
		// Since we stored imports as full paths and FileArtifactEntry.Path is relative,
		// we index by Path for lookup.
		for _, imp := range fe.Imports {
			importToFiles[imp] = append(importToFiles[imp], fe.Path)
		}
	}

	// Build reverse: relPath → importPath for the selected files.
	// We need to find which entries import the selected files' packages.
	// Strategy: find FileArtifactEntry.Imports that resolve to selected file paths.
	selectedSet := make(map[string]bool, len(selectedPaths))
	for _, p := range selectedPaths {
		selectedSet[p] = true
	}

	// Find all files whose Imports reference the packages of selected files.
	// We build a map: file path → set of import paths declared in that file.
	// Then for each project file, if it imports a package that contains a selected file, include it.
	//
	// Simpler depth-1 approach: for each selected file, find the files.json entries
	// whose Imports list contains an import path that matches this file's package.
	// Since FileArtifactEntry does not store the file's own import path directly,
	// we derive it from the file's own Imports field's package relationship.
	// Per Pitfall 6: import path ≠ short Package name.
	// Best approach: collect ALL imports of selected files, then find files.json entries
	// whose Path matches one of those imports' file paths.

	// Collect imports of all selected files.
	selectedFileImports := make(map[string]bool)
	for _, fe := range filesArtifact.Files {
		if selectedSet[fe.Path] {
			for _, imp := range fe.Imports {
				selectedFileImports[imp] = true
			}
		}
	}

	// Build import path → file path mapping:
	// For each file entry, find which import paths it "is" by matching against
	// importToFiles built above. Use the reverse: file entry path appears as the
	// target of some imports.
	// We need to identify which import paths correspond to which file paths.
	// Since BuildArtifacts derives importPath = moduleName + "/" + dir (artifacts.go line 119-121),
	// files in the same directory share an import path.
	// Build: importPath → []filePath by grouping FileArtifactEntry by their directory.

	// Derive each file's own import path using its Path field.
	// Path is relative (e.g., "internal/scanner/ast.go") → importPath = module + "/" + dir.
	// We don't have the module path here, but we can match on the directory suffix.
	// Alternative: match based on what other files import them.
	// For depth-1, we add files that are themselves in the packages that selected files import.

	// Build: importPath → files with that import path (by grouping by directory).
	dirToFiles := make(map[string][]string)
	for _, fe := range filesArtifact.Files {
		dir := filepath.ToSlash(filepath.Dir(fe.Path))
		dirToFiles[dir] = append(dirToFiles[dir], fe.Path)
	}

	// Match selected file imports to project directories.
	// An import like "github.com/bkohler93/myhelper/internal/scanner" maps to dir "internal/scanner".
	addedSet := make(map[string]bool)
	for _, p := range selectedPaths {
		addedSet[p] = true
	}
	result := make([]string, len(selectedPaths))
	copy(result, selectedPaths)

	usedBudget := 0

	for imp := range selectedFileImports {
		// Extract the package directory suffix from the import path.
		// Import paths are like "github.com/bkohler93/myhelper/internal/scanner".
		// Find project-internal packages by looking for a dir match.
		parts := strings.Split(imp, "/")
		// Try progressively shorter suffixes to find a matching directory.
		for suffixLen := len(parts); suffixLen >= 1; suffixLen-- {
			suffix := strings.Join(parts[len(parts)-suffixLen:], "/")
			if neighborFiles, ok := dirToFiles[suffix]; ok {
				for _, nf := range neighborFiles {
					if addedSet[nf] {
						continue
					}
					cost := tokenCount(cfg, nf)
					if usedBudget+cost > budgetCap {
						goto expansionDone
					}
					result = append(result, nf)
					addedSet[nf] = true
					usedBudget += cost
				}
				break
			}
		}
	}

expansionDone:
	return result
}

// -----------------------------------------------------------------------------
// Message Assembly
// -----------------------------------------------------------------------------

// assembleMessages builds the final message list for the LLM using four ordered stages.
// Stage 1: project summary; Stage 2: symbol matches; Stage 3: file list; Stage 4: file content.
// Each stage checks remaining budget before appending; a stage that would overflow is skipped.
// Query is appended after all stages and is NOT counted against the budget.
func assembleMessages(
	query string,
	proj scanner.ProjectArtifact,
	symbols []scanner.Symbol,
	filePaths []string,
	root string,
	strategy Strategy,
	cfg config.Config,
	chatFn scanner.ChatFn,
) []history.Message {
	budget := int(float64(cfg.TokenThreshold) * strategy.MaxTokenRatio)
	usedTokens := 0
	var sb strings.Builder
	hasContext := false

	// Stage 1: Project summary
	if proj.Summary != "" {
		cost := tokenCount(cfg, proj.Summary)
		if usedTokens+cost <= budget {
			sb.WriteString("## Project\n\n")
			sb.WriteString(proj.Summary)
			sb.WriteString("\n\n")
			usedTokens += cost
			hasContext = true
		}
	}

	// Stage 2: Symbol matches
	if strategy.UseSymbols && len(symbols) > 0 {
		var symSB strings.Builder
		added := 0
		for _, sym := range symbols {
			line := fmt.Sprintf("- `%s` (%s): %s\n", sym.StableID, sym.Kind, sym.Signature)
			cost := tokenCount(cfg, line)
			if usedTokens+cost > budget {
				break
			}
			symSB.WriteString(line)
			usedTokens += cost
			added++
		}
		if added > 0 {
			sb.WriteString("### Relevant Symbols\n\n")
			sb.WriteString(symSB.String())
			sb.WriteString("\n")
			hasContext = true
		}
	}

	// Stage 3: File list (metadata only — cheap)
	if strategy.UseFiles && len(filePaths) > 0 {
		var fileSB strings.Builder
		added := 0
		for _, fp := range filePaths {
			line := fmt.Sprintf("- %s\n", fp)
			cost := tokenCount(cfg, line)
			if usedTokens+cost > budget {
				break
			}
			fileSB.WriteString(line)
			usedTokens += cost
			added++
		}
		if added > 0 {
			sb.WriteString("### Selected Files\n\n")
			sb.WriteString(fileSB.String())
			sb.WriteString("\n")
			hasContext = true
		}
	}

	// Stage 4: Conditional file content expansion
	if strategy.UseFiles {
		for _, fp := range filePaths {
			remaining := budget - usedTokens
			if remaining <= 0 {
				break
			}
			rawBytes, err := os.ReadFile(filepath.Join(root, fp))
			if err != nil {
				continue
			}
			rawContent := string(rawBytes)
			rawCost := tokenCount(cfg, rawContent)
			if rawCost <= remaining {
				sb.WriteString(fmt.Sprintf("#### %s\n\n```go\n%s\n```\n\n", fp, rawContent))
				usedTokens += rawCost
				hasContext = true
			} else {
				if content, ok := microPassFile(root, fp, query, cfg, chatFn, remaining); ok {
					sb.WriteString(fmt.Sprintf("#### %s (partial)\n\n```go\n%s\n```\n\n", fp, content))
					usedTokens += tokenCount(cfg, content)
					hasContext = true
				}
			}
		}
	}

	// If no context was added, return bare user query.
	if !hasContext {
		return []history.Message{{Role: "user", Content: query}}
	}

	sb.WriteString("## Query\n\n")
	sb.WriteString(query)

	return []history.Message{{Role: "user", Content: sb.String()}}
}

// -----------------------------------------------------------------------------
// Helpers
// -----------------------------------------------------------------------------

// tokenCount returns the token cost of a string using the project's standard counter.
func tokenCount(cfg config.Config, content string) int {
	return history.New(cfg.TokenThreshold, []history.Message{{Role: "user", Content: content}}).TokenCount()
}

// uniqueFilePaths returns the deduplicated set of file paths from the given symbols.
func uniqueFilePaths(symbols []scanner.Symbol) []string {
	seen := make(map[string]bool)
	var paths []string
	for _, sym := range symbols {
		if sym.FilePath != "" && !seen[sym.FilePath] {
			paths = append(paths, sym.FilePath)
			seen[sym.FilePath] = true
		}
	}
	return paths
}

// loadArtifacts reads all four artifact files from root/.myhelper/.
func loadArtifacts(root string) (
	proj scanner.ProjectArtifact,
	pkgs scanner.PackagesArtifact,
	files scanner.FilesArtifact,
	syms scanner.SymbolsArtifact,
	err error,
) {
	myhelperDir := filepath.Join(root, ".myhelper")
	if err = readJSON(filepath.Join(myhelperDir, "project.json"), &proj); err != nil {
		return
	}
	if err = readJSON(filepath.Join(myhelperDir, "packages.json"), &pkgs); err != nil {
		return
	}
	if err = readJSON(filepath.Join(myhelperDir, "files.json"), &files); err != nil {
		return
	}
	if err = readJSON(filepath.Join(myhelperDir, "symbols.json"), &syms); err != nil {
		return
	}
	return
}

func readJSON(path string, v interface{}) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("readJSON %s: %w", path, err)
	}
	return json.Unmarshal(data, v)
}

// -----------------------------------------------------------------------------
// Micro-Pass File Extraction (moved from cmd/helpers.go — Phase 12)
// -----------------------------------------------------------------------------

// microPassRe matches "N-M" line range responses from the micro-pass LLM call.
var microPassRe = regexp.MustCompile(`(\d+)-(\d+)`)

// microPassFile asks the model which line range of an oversized file is needed
// to answer the query, then extracts and returns those lines.
//
// Fallback chain (per D-10):
//  1. Build symbol map via scanner.ExtractSymbolMap.
//  2. Call chatFn with symbol map + query; parse "N-M" response.
//  3. If parse succeeds and extracted range fits budget: return extracted lines.
//  4. Otherwise: truncate raw content at last newline that fits budget.
//  5. If even truncated content doesn't fit: return ("", false).
//
// Never panics. Never surfaces an error to the user (per D-06, D-09).
// root is the project root; path is relative to root.
func microPassFile(root, path, query string, cfg config.Config, chatFn scanner.ChatFn, budget int) (string, bool) {
	if budget <= 0 {
		return "", false
	}

	absPath := filepath.Join(root, path)
	rawBytes, err := os.ReadFile(absPath)
	if err != nil {
		return "", false
	}
	rawContent := string(rawBytes)
	lines := strings.Split(rawContent, "\n")
	totalLines := len(lines)

	// Attempt micro-pass: build symbol map and ask the model for a line range.
	extracted, ok := func() (string, bool) {
		symbols, err := scanner.ExtractSymbolMap(absPath)
		if err != nil {
			return "", false
		}

		// Build symbol map text (D-01, D-02).
		var mapSB strings.Builder
		for _, sym := range symbols {
			fmt.Fprintf(&mapSB, "%s: lines %d-%d\n", sym.Name, sym.Start, sym.End)
		}

		// Compose micro-pass messages (D-03, D-04).
		systemPrompt := "Given this file's symbol map, output ONLY the line range needed to answer the user's request. Format: start-end (e.g., 12-55). Output nothing else."
		userMsg := fmt.Sprintf("Symbols in %s:\n%s\nUser request: %s", path, mapSB.String(), query)
		microMessages := []history.Message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userMsg},
		}

		resp, err := chatFn(cfg, microMessages)
		if err != nil {
			return "", false
		}

		// Parse "N-M" response (D-05).
		m := microPassRe.FindStringSubmatch(strings.TrimSpace(resp))
		if m == nil {
			return "", false
		}
		var start, end int
		fmt.Sscanf(m[1], "%d", &start)
		fmt.Sscanf(m[2], "%d", &end)

		// Clamp out-of-bounds (D-07).
		if start < 1 {
			start = 1
		}
		if end > totalLines {
			end = totalLines
		}
		// On invalid range after clamping, fall through to truncation (D-06).
		if start > end {
			return "", false
		}

		// Extract lines [start, end] inclusive (0-indexed slice, 1-indexed line numbers).
		extracted := strings.Join(lines[start-1:end], "\n")
		tokCount := history.New(cfg.TokenThreshold, []history.Message{{Role: "user", Content: extracted}}).TokenCount()
		if tokCount > budget {
			return "", false
		}
		return extracted, true
	}()

	if ok {
		return extracted, true
	}

	// Truncation fallback (D-08): scan from right for last newline that fits.
	for i := len(rawContent) - 1; i >= 0; i-- {
		if rawContent[i] == '\n' {
			prefix := rawContent[:i+1]
			tokCount := history.New(cfg.TokenThreshold, []history.Message{{Role: "user", Content: prefix}}).TokenCount()
			if tokCount <= budget {
				return prefix, true
			}
		}
	}

	// D-09: even a single line doesn't fit — skip this file.
	return "", false
}

// -----------------------------------------------------------------------------
// Inspect Pipeline (CMD-02)
// -----------------------------------------------------------------------------

// SelectionSource identifies which pipeline stage selected a symbol or file.
type SelectionSource int

const (
	SourcePreFilter SelectionSource = iota // symbol passed keyword pre-filter
	SourceReRank                           // symbol confirmed by LLM re-ranking
	SourceExpansion                        // file added by dependency expansion
)

// String returns a human-readable label for a SelectionSource.
func (s SelectionSource) String() string {
	switch s {
	case SourcePreFilter:
		return "pre-filter"
	case SourceReRank:
		return "re-rank"
	case SourceExpansion:
		return "expansion"
	default:
		return "unknown"
	}
}

// SymbolResult pairs a selected symbol with the stage that selected it.
type SymbolResult struct {
	Symbol scanner.Symbol
	Source SelectionSource
}

// FileResult pairs a selected file path with the stage that selected it.
type FileResult struct {
	Path   string
	Source SelectionSource
}

// StageMetrics records the token cost at a pipeline stage boundary.
type StageMetrics struct {
	Name       string
	TokensUsed int
}

// PreFilterCandidate pairs a symbol with its keyword relevance score from the
// pre-filter stage. Used by the inspect command to display scores (INSP-03).
type PreFilterCandidate struct {
	Symbol scanner.Symbol
	Score  int
}

// InspectResult holds per-stage diagnostic output from a dry-run retrieval.
// It does not contain assembled messages — BuildInspectContext never calls assembleMessages.
type InspectResult struct {
	Symbols             []SymbolResult
	Files               []FileResult
	StageMetrics        []StageMetrics
	FinalTokens         int
	GatePassed          bool
	GateAnswer          string               // raw LLM string from relevance gate (INSP-02)
	PreFilterCandidates []PreFilterCandidate // symbols surviving pre-filter with scores (INSP-03)
}

// BuildInspectContext runs the retrieval pipeline in dry-run mode and returns
// per-stage diagnostics. It does NOT call assembleMessages or make any streaming
// calls beyond the relevance gate and re-ranking LLM calls.
//
// If artifacts do not exist, it returns InspectResult{GatePassed: false} with nil
// error — same graceful fallback as BuildContext.
// If the relevance gate returns false, it returns InspectResult{GatePassed: false}
// with nil error and no stage metrics.
func BuildInspectContext(
	root string,
	query string,
	strategy Strategy,
	cfg config.Config,
	chatFn scanner.ChatFn,
) (InspectResult, error) {
	proj, _, files, syms, err := loadArtifacts(root)
	if err != nil {
		// No artifacts — not an error; caller (inspect command) handles the display.
		return InspectResult{GatePassed: false}, nil
	}

	var result InspectResult

	// Stage 1: Relevance gate (RET-04) — inline to capture raw LLM answer (INSP-02)
	{
		msg := relevanceGatePrompt + query
		if proj.Summary != "" {
			msg = "Project: " + proj.Summary + "\n\n" + msg
		}
		messages := []history.Message{
			{Role: "user", Content: msg},
		}
		rawAnswer, gateErr := chatFn(cfg, messages)
		if gateErr != nil {
			rawAnswer = "[gate LLM error: " + gateErr.Error() + "]"
		}
		result.GateAnswer = strings.TrimSpace(rawAnswer)
		lower := strings.ToLower(result.GateAnswer)
		result.GatePassed = gateErr == nil && !strings.HasPrefix(lower, "no")
		if !result.GatePassed {
			return result, nil
		}
	}

	// Budget setup (mirrors BuildContext)
	totalBudget := int(float64(cfg.TokenThreshold) * contextBudgetFactor)

	// Stage 2: Pre-filter (RET-01, RET-02)
	candidates := preFilter(query, syms.Symbols, files.Files)
	candidates = applyTokenCap(candidates, totalBudget, cfg)
	queryTerms := strings.Fields(strings.ToLower(query))
	preFilterTokens := 0
	for _, c := range candidates {
		preFilterTokens += tokenCount(cfg, c.Signature)
		result.PreFilterCandidates = append(result.PreFilterCandidates, PreFilterCandidate{
			Symbol: c,
			Score:  scoreSymbol(c, queryTerms),
		})
	}
	result.StageMetrics = append(result.StageMetrics, StageMetrics{
		Name:       "pre-filter",
		TokensUsed: preFilterTokens,
	})

	// Stage 3: LLM re-ranking (RET-03)
	// Only symbols that survive re-ranking appear in output (per A4 assumption in RESEARCH.md).
	selected, reRankErr := llmReRank(query, candidates, cfg, chatFn)
	if reRankErr != nil {
		selected = candidates // fallback
	}
	reRankTokens := 0
	for _, s := range selected {
		reRankTokens += tokenCount(cfg, s.Signature)
		result.Symbols = append(result.Symbols, SymbolResult{
			Symbol: s,
			Source: SourceReRank,
		})
	}
	result.StageMetrics = append(result.StageMetrics, StageMetrics{
		Name:       "re-rank",
		TokensUsed: reRankTokens,
	})

	// Stage 4: Dependency expansion (RET-05)
	// Track which paths come from re-ranking vs expansion (Pitfall 4 from RESEARCH.md).
	selectedPaths := uniqueFilePaths(selected)
	selectedSet := make(map[string]bool, len(selectedPaths))
	for _, p := range selectedPaths {
		selectedSet[p] = true
		result.Files = append(result.Files, FileResult{Path: p, Source: SourceReRank})
	}

	usedTokens := reRankTokens
	expansionBudget := int(float64(totalBudget-usedTokens) * expansionBudgetFactor)
	expandedPaths := expandDeps(selectedPaths, files, expansionBudget, cfg)

	expansionTokens := 0
	for _, p := range expandedPaths {
		if !selectedSet[p] {
			cost := tokenCount(cfg, p)
			expansionTokens += cost
			result.Files = append(result.Files, FileResult{Path: p, Source: SourceExpansion})
		}
	}
	result.StageMetrics = append(result.StageMetrics, StageMetrics{
		Name:       "expansion",
		TokensUsed: expansionTokens,
	})

	result.FinalTokens = reRankTokens + expansionTokens
	return result, nil
}
