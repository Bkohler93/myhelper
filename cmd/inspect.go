package cmd

import (
	"fmt"
	"os"

	"github.com/bkohler93/myhelper/internal/config"
	"github.com/bkohler93/myhelper/internal/ollama"
	"github.com/bkohler93/myhelper/internal/retrieval"
	"github.com/spf13/cobra"
)

var inspectCmd = &cobra.Command{
	Use:   "inspect <query>",
	Short: "Dry-run retrieval and print per-stage context selection details",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runInspect,
}

func init() {
	rootCmd.AddCommand(inspectCmd)
}

func runInspect(cmd *cobra.Command, args []string) error {
	input, err := resolveInput(args, "Query to inspect: ")
	if err != nil {
		return err
	}

	cfg := config.Load()
	ApplyFlagOverrides(&cfg)

	root, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("runInspect: getwd: %w", err)
	}

	if noContextFlag {
		fmt.Println("Context bypassed (--no-context flag set)")
		return nil
	}

	result, err := retrieval.BuildInspectContext(root, input, retrieval.StarterStrategy, cfg, ollama.Chat)
	if err != nil {
		return fmt.Errorf("runInspect: BuildInspectContext: %w", err)
	}

	// Detect missing artifacts: BuildInspectContext returns GatePassed:false with empty
	// GateAnswer when .myhelper/ artifacts do not exist (no gate call was made).
	if !result.GatePassed && result.GateAnswer == "" {
		fmt.Fprintln(os.Stderr, "No .myhelper/ artifacts found. Run `myhelper init` first.")
		os.Exit(1)
	}

	printInspectResult(result, input)
	return nil
}

func printInspectResult(result retrieval.InspectResult, query string) {
	fmt.Printf("--- Retrieval Inspect: %q ---\n\n", query)

	// Gate decision with raw LLM answer (INSP-02)
	if !result.GatePassed {
		fmt.Printf("Gate: FAIL (raw: %q)\n", result.GateAnswer)
		fmt.Println("Context not needed for this query — retrieval stopped at gate.")
		return
	}
	fmt.Printf("Gate: PASS (raw: %q)\n", result.GateAnswer)
	fmt.Println()

	// Pre-filter candidates with scores (INSP-03)
	fmt.Printf("Pre-filter: %d candidates\n", len(result.PreFilterCandidates))
	for _, c := range result.PreFilterCandidates {
		fmt.Printf("  - (%s) %s [%s] score:%d\n", c.Symbol.StableID, c.Symbol.Name, c.Symbol.Kind, c.Score)
	}
	fmt.Println()

	// Re-rank: survivors vs dropped (INSP-04)
	survivorCount := len(result.Symbols)
	droppedCount := len(result.PreFilterCandidates) - survivorCount
	if droppedCount < 0 {
		droppedCount = 0
	}
	fmt.Printf("Re-rank: %d survivors / %d dropped\n", survivorCount, droppedCount)
	if survivorCount > 0 {
		fmt.Println("Survivors:")
		for _, sr := range result.Symbols {
			fmt.Printf("  - (%s) %s [%s]\n", sr.Symbol.StableID, sr.Symbol.Name, sr.Symbol.Kind)
		}
	}
	fmt.Println()

	// Stage metrics (INSP-02)
	fmt.Println("Stage metrics:")
	total := 0
	for _, sm := range result.StageMetrics {
		fmt.Printf("  %s: %d tokens\n", sm.Name, sm.TokensUsed)
		total += sm.TokensUsed
	}
	fmt.Printf("  total: %d tokens\n", total)
	fmt.Println()

	// Selected files
	if len(result.Files) > 0 {
		fmt.Printf("Selected files (%d):\n", len(result.Files))
		for _, fr := range result.Files {
			fmt.Printf("  - %s [%s]\n", fr.Path, fr.Source.String())
		}
		fmt.Println()
	}

	fmt.Printf("Final context size: %d tokens\n", result.FinalTokens)
}
