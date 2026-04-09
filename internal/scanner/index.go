package scanner

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bkohler93/myhelper/internal/config"
	"github.com/bkohler93/myhelper/internal/history"
)

const tokenBudgetSafetyFactor = 0.80

// BuildIndex walks root, extracts AST symbols from each Go file, computes
// per-entry token counts, applies the 80% token budget (dropping test files
// first, then files by path order), and writes the result to
// .myhelper/index.json as a JSON array.
//
// Returns the retained []FileEntry slice. Returns an error if Walk fails or
// if writing index.json fails.
func BuildIndex(root string, cfg config.Config) ([]FileEntry, error) {
	paths, err := Walk(root)
	if err != nil {
		return nil, fmt.Errorf("BuildIndex: walk: %w", err)
	}

	var entries []FileEntry
	for _, relPath := range paths {
		absPath := filepath.Join(root, relPath)
		pkg, symbols, err := ExtractSymbols(absPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "scanner: skip %s: %v\n", relPath, err)
			continue
		}

		entry := FileEntry{
			Path:    relPath,
			Package: pkg,
			Symbols: symbols,
		}

		// Compute token count from the JSON representation of this entry.
		entryJSON, err := json.Marshal(entry)
		if err != nil {
			fmt.Fprintf(os.Stderr, "scanner: marshal entry %s: %v\n", relPath, err)
			continue
		}
		h := history.New(cfg.TokenThreshold, []history.Message{{Role: "user", Content: string(entryJSON)}})
		entry.TokenCount = h.TokenCount()

		entries = append(entries, entry)
	}

	// Apply token budget: sort so test files come last, then truncate.
	budget := int(float64(cfg.TokenThreshold) * tokenBudgetSafetyFactor)

	sort.SliceStable(entries, func(i, j int) bool {
		return !isTestFile(entries[i].Path) && isTestFile(entries[j].Path)
	})

	total := 0
	cutoff := len(entries)
	for i, e := range entries {
		total += e.TokenCount
		if total > budget {
			cutoff = i
			break
		}
	}
	entries = entries[:cutoff]

	// Serialize to JSON object and write to .myhelper/index.json.
	// json.MarshalIndent on a nil slice produces "null"; use empty slice to get "[]".
	if entries == nil {
		entries = []FileEntry{}
	}

	meta, metaErr := ReadMeta(root)
	if metaErr != nil {
		fmt.Fprintf(os.Stderr, "scanner: ReadMeta: %v\n", metaErr)
	}

	idx := Index{Meta: meta, Files: entries}
	out, err := json.MarshalIndent(idx, "", "  ")
	if err != nil {
		return entries, fmt.Errorf("BuildIndex: marshal: %w", err)
	}

	outPath := filepath.Join(root, ".myhelper", "index.json")
	if err := os.WriteFile(outPath, out, 0644); err != nil {
		return entries, fmt.Errorf("BuildIndex: write index.json: %w", err)
	}

	return entries, nil
}

// isTestFile reports whether path ends with "_test.go".
func isTestFile(path string) bool {
	return strings.HasSuffix(path, "_test.go")
}
