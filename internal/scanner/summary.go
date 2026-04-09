package scanner

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bkohler93/myhelper/internal/config"
	"github.com/bkohler93/myhelper/internal/history"
)

const summarySystemPrompt = "You are a Go code analyst. Given a list of exported symbols from a Go package, write a concise design and pattern summary in markdown. Focus on what the package does, its key abstractions, and usage patterns. Be brief — under 200 words."

// GenerateSummaries groups FileEntry objects by package, calls chatFn once per
// package with a user-role message listing all exported symbols, and writes
// the resulting summary to .myhelper/summaries/{pkg}.md under root.
//
// ChatFn is injected as a parameter — no direct ollama.Chat call is made here.
// In tests: pass a stub func. In production: pass ollama.Chat directly.
func GenerateSummaries(root string, entries []FileEntry, cfg config.Config, chatFn ChatFn) error {
	if len(entries) == 0 {
		return nil
	}

	// Group entries by package.
	byPkg := make(map[string][]FileEntry)
	for _, e := range entries {
		byPkg[e.Package] = append(byPkg[e.Package], e)
	}

	// Ensure output directory exists.
	summariesDir := filepath.Join(root, ".myhelper", "summaries")
	if err := os.MkdirAll(summariesDir, 0755); err != nil {
		return fmt.Errorf("create summaries dir: %w", err)
	}

	// For each package, build a prompt and call ChatFn once.
	for pkg, pkgEntries := range byPkg {
		// Collect all symbols from all files in the package.
		var allSymbols []string
		for _, e := range pkgEntries {
			allSymbols = append(allSymbols, e.Symbols...)
		}

		// Build the prompt content string.
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("Package: %s\n\nExported symbols:\n", pkg))
		for _, sym := range allSymbols {
			sb.WriteString("- " + sym + "\n")
		}

		// Build messages slice: single user-role message (per v1.2 constraint:
		// file content injected in user-role messages only, never system message).
		messages := []history.Message{
			{Role: "user", Content: sb.String()},
		}

		// Call ChatFn — return error immediately if it fails (no partial file).
		content, err := chatFn(cfg, messages)
		if err != nil {
			return fmt.Errorf("chatFn for package %q: %w", pkg, err)
		}

		// Write result to .myhelper/summaries/{pkg}.md.
		outPath := filepath.Join(summariesDir, pkg+".md")
		if err := os.WriteFile(outPath, []byte(content), 0644); err != nil {
			return fmt.Errorf("write summary %s: %w", outPath, err)
		}
	}

	return nil
}
