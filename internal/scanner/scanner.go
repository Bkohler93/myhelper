package scanner

import (
	"github.com/bkohler93/myhelper/internal/config"
	"github.com/bkohler93/myhelper/internal/history"
)

// ChatFn is a function type for calling the LLM in non-streaming mode.
// It matches the signature of ollama.Chat and allows injection for tests.
type ChatFn func(cfg config.Config, messages []history.Message) (string, error)

// FileEntry represents a single Go source file in the project index.
type FileEntry struct {
	Path       string   `json:"path"`
	Package    string   `json:"package"`
	Symbols    []string `json:"symbols"`
	TokenCount int      `json:"token_count"`
}

// Index is the top-level structure written to .myhelper/index.json.
// Meta holds project-level metadata; Files holds per-file entries.
type Index struct {
	Meta  ProjectMeta `json:"meta"`
	Files []FileEntry `json:"files"`
}
