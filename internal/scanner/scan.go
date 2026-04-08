package scanner

import (
	"fmt"

	"github.com/bkohler93/myhelper/internal/config"
)

// Scan walks root, extracts AST symbols from .go files, reads project metadata,
// writes .myhelper/index.json, and generates per-package LLM summaries in
// .myhelper/summaries/. chatFn is injected for testability; pass ollama.Chat
// in production.
func Scan(root string, cfg config.Config, chatFn ChatFn) error {
	// Step 1: Build index (walk + AST extraction + token budgeting + write index.json)
	entries, err := BuildIndex(root, cfg)
	if err != nil {
		return fmt.Errorf("scan: build index: %w", err)
	}

	// Step 2: Generate per-package summaries
	if err := GenerateSummaries(root, entries, cfg, chatFn); err != nil {
		return fmt.Errorf("scan: generate summaries: %w", err)
	}

	return nil
}
