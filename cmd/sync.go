package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/bkohler93/myhelper/internal/config"
	"github.com/bkohler93/myhelper/internal/history"
	"github.com/bkohler93/myhelper/internal/ollama"
	"github.com/bkohler93/myhelper/internal/scanner"
	"github.com/spf13/cobra"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Refresh .myhelper/ index and context for changed files",
	Long:  "Detects .go files changed since the last init or sync (via mtime), re-indexes and re-summarizes only changed files, and regenerates context.md. Run init first if .myhelper/ does not exist.",
	RunE:  runSync,
}

func init() {
	rootCmd.AddCommand(syncCmd)
}

func runSync(cmd *cobra.Command, args []string) error {
	root, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getwd: %w", err)
	}

	cfg := config.Load()
	ApplyFlagOverrides(&cfg)

	// Guard: require init to have run first.
	metaPath := filepath.Join(root, ".myhelper", "meta.json")
	if _, statErr := os.Stat(metaPath); os.IsNotExist(statErr) {
		return fmt.Errorf("no .myhelper/meta.json found — run 'myhelper init' first to generate the project index")
	}

	since, err := readLastSync(root)
	if err != nil {
		return fmt.Errorf("read last_sync: %w", err)
	}

	return RunWithSpinner(func(progress func(string)) error {
		// Step 1: Find changed .go files since last sync.
		progress("Detecting changed files...")
		changed, err := changedFilesSince(root, since)
		if err != nil {
			return fmt.Errorf("detect changes: %w", err)
		}

		if len(changed) > 0 {
			// Step 2: Re-index changed files and merge into existing index.json.
			progress("Updating index...")
			if err := deltaIndex(root, cfg, changed); err != nil {
				return fmt.Errorf("delta index: %w", err)
			}

			// Step 3: Regenerate summaries for packages containing changed files.
			progress("Re-summarizing changed packages...")
			if err := deltaSummaries(root, cfg, ollama.Chat, changed); err != nil {
				return fmt.Errorf("delta summaries: %w", err)
			}
		}

		// Step 4: Always regenerate context.md (per D-07).
		progress("Generating context.md...")
		if err := generateContextMD(root, cfg, ollama.Chat); err != nil {
			return fmt.Errorf("context.md: %w", err)
		}

		// Step 5: Update last_sync timestamp (per D-05).
		progress("Finalizing...")
		return writeLastSync(root, time.Now())
	})
}

// changedFilesSince returns relative paths of .go files under root whose mtime
// is after since. Excludes .git/, vendor/, testdata/, .myhelper/ directories.
// An empty since (zero time.Time) causes all files to be returned.
func changedFilesSince(root string, since time.Time) ([]string, error) {
	var changed []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			name := d.Name()
			if name == ".git" || name == "vendor" || name == "testdata" || name == ".myhelper" {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".go" {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if since.IsZero() || info.ModTime().After(since) {
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			changed = append(changed, rel)
		}
		return nil
	})
	return changed, err
}

// deltaIndex reads the existing .myhelper/index.json, re-extracts symbols for
// each changed file, merges updated entries into the existing map (replacing
// stale entries, adding new ones, removing entries for deleted files), and
// re-serializes the result to .myhelper/index.json.
//
// Token budget is NOT re-applied on delta (existing entries already fit; changed
// entries replace existing ones at similar size). This avoids re-sorting and
// dropping entries unexpectedly during sync.
func deltaIndex(root string, cfg config.Config, changedPaths []string) error {
	indexPath := filepath.Join(root, ".myhelper", "index.json")

	// Read existing index.json.
	existing := scanner.Index{}
	if data, err := os.ReadFile(indexPath); err == nil {
		_ = json.Unmarshal(data, &existing) // best-effort; empty Index if corrupt
	}

	// Build lookup map from existing entries.
	byPath := make(map[string]scanner.FileEntry, len(existing.Files))
	for _, e := range existing.Files {
		byPath[e.Path] = e
	}

	// Build set of all current .go files (for deleted-file detection).
	currentPaths := make(map[string]bool)
	if err := filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			name := d.Name()
			if name == ".git" || name == "vendor" || name == "testdata" || name == ".myhelper" {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".go" {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err == nil {
			currentPaths[rel] = true
		}
		return nil
	}); err != nil {
		return fmt.Errorf("deltaIndex: walk for deleted-file detection: %w", err)
	}

	// Remove deleted entries.
	for path := range byPath {
		if !currentPaths[path] {
			delete(byPath, path)
		}
	}

	// Re-extract symbols for changed files and upsert into map.
	for _, relPath := range changedPaths {
		absPath := filepath.Join(root, relPath)
		pkg, symbols, err := scanner.ExtractSymbols(absPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "sync: skip %s: %v\n", relPath, err)
			continue
		}
		entry := scanner.FileEntry{
			Path:    relPath,
			Package: pkg,
			Symbols: symbols,
		}
		// Compute token count.
		entryJSON, err := json.Marshal(entry)
		if err != nil {
			fmt.Fprintf(os.Stderr, "sync: marshal entry %s: %v\n", relPath, err)
			continue
		}
		h := history.New(cfg.TokenThreshold, []history.Message{{Role: "user", Content: string(entryJSON)}})
		entry.TokenCount = h.TokenCount()
		byPath[relPath] = entry
	}

	// Flatten map back to slice.
	merged := make([]scanner.FileEntry, 0, len(byPath))
	for _, e := range byPath {
		merged = append(merged, e)
	}

	// Re-serialize using existing meta.
	idx := scanner.Index{Meta: existing.Meta, Files: merged}
	out, err := json.MarshalIndent(idx, "", "  ")
	if err != nil {
		return fmt.Errorf("deltaIndex: marshal: %w", err)
	}
	return os.WriteFile(indexPath, out, 0644)
}

// deltaSummaries identifies which packages are affected by changedPaths, then
// calls scanner.GenerateSummaries with only those packages' FileEntry objects.
// Reads the merged index.json to get the current FileEntry list.
func deltaSummaries(root string, cfg config.Config, chatFn scanner.ChatFn, changedPaths []string) error {
	// Build set of affected packages from changed paths.
	changedSet := make(map[string]bool, len(changedPaths))
	for _, p := range changedPaths {
		changedSet[p] = true
	}

	// Read current index.json to get FileEntry objects (with package names).
	indexPath := filepath.Join(root, ".myhelper", "index.json")
	data, err := os.ReadFile(indexPath)
	if err != nil {
		return fmt.Errorf("deltaSummaries: read index.json: %w", err)
	}
	var idx scanner.Index
	if err := json.Unmarshal(data, &idx); err != nil {
		return fmt.Errorf("deltaSummaries: unmarshal index: %w", err)
	}

	// Identify packages affected by changed files.
	affectedPkgs := make(map[string]bool)
	for _, e := range idx.Files {
		if changedSet[e.Path] {
			affectedPkgs[e.Package] = true
		}
	}

	// Collect all entries for affected packages (not just the changed files).
	var affected []scanner.FileEntry
	for _, e := range idx.Files {
		if affectedPkgs[e.Package] {
			affected = append(affected, e)
		}
	}

	if len(affected) == 0 {
		return nil
	}

	return scanner.GenerateSummaries(root, affected, cfg, chatFn)
}
