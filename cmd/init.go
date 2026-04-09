package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/bkohler93/myhelper/internal/config"
	"github.com/bkohler93/myhelper/internal/ollama"
	"github.com/bkohler93/myhelper/internal/scanner"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Scan project and generate .myhelper/ index, summaries, and context",
	Long:  "Scans the current Go project directory, generates .myhelper/index.json with per-file symbols, creates per-package summaries in .myhelper/summaries/, and writes an LLM-generated .myhelper/context.md. Safe to re-run — overwrites existing data.",
	RunE:  runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	root, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getwd: %w", err)
	}

	cfg := config.Load()
	ApplyFlagOverrides(&cfg)

	// Ensure .myhelper/summaries/ directory exists before scanner.Scan writes to it.
	if err := os.MkdirAll(filepath.Join(root, ".myhelper", "summaries"), 0755); err != nil {
		return fmt.Errorf("mkdir .myhelper/summaries: %w", err)
	}

	return RunWithSpinner(func(progress updateLabelFn) error {
		// Step 1: Full scan — BuildIndex + GenerateSummaries via scanner.Scan.
		progress("Building index...")
		if err := scanner.Scan(root, cfg, ollama.Chat, progress); err != nil {
			return fmt.Errorf("scan: %w", err)
		}

		// Step 1b: Build hierarchical artifact files.
		progress("Building artifact index...")
		if err := scanner.BuildArtifacts(root, cfg, ollama.Chat); err != nil {
			return fmt.Errorf("artifacts: %w", err)
		}

		// Step 2: Generate context.md from per-package summaries.
		progress("Generating context.md...")
		if err := generateContextMD(root, cfg, ollama.Chat); err != nil {
			return fmt.Errorf("context.md: %w", err)
		}

		// Step 3: Write last_sync timestamp (per D-05).
		progress("Finalizing...")
		return writeLastSync(root, time.Now())
	})
}
