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

	result, err := retrieval.BuildInspectContext(root, input, retrieval.DefaultStrategy, cfg, ollama.Chat)
	if err != nil {
		return fmt.Errorf("runInspect: BuildInspectContext: %w", err)
	}

	printInspectResult(result, input)
	return nil
}

// printInspectResult formats and prints an InspectResult to stdout.
// Output format (plain text, no color):
//
//	--- Retrieval Inspect: "<query>" ---
//
//	Gate: context required (passed)         OR   Gate: context not required (skipped)
//
//	Stage: pre-filter    tokens: N
//	Stage: re-rank       tokens: N
//	Stage: expansion     tokens: N
//
//	Selected Symbols (N):
//	  <stableID>  (<kind>)  [<source>]
//	  ...
//
//	Selected Files (N):
//	  <path>  [<source>]
//	  ...
//
//	Final context size: N tokens
func printInspectResult(result retrieval.InspectResult, query string) {
	fmt.Printf("--- Retrieval Inspect: %q ---\n\n", query)

	if result.GatePassed {
		fmt.Println("Gate: context required (passed)")
	} else {
		fmt.Println("Gate: context not required (skipped)")
		fmt.Println("\nNo symbols or files selected.")
		return
	}

	fmt.Println()
	for _, sm := range result.StageMetrics {
		fmt.Printf("Stage: %-14s tokens: %d\n", sm.Name, sm.TokensUsed)
	}

	fmt.Printf("\nSelected Symbols (%d):\n", len(result.Symbols))
	for _, sr := range result.Symbols {
		fmt.Printf("  %-40s (%s)\t[%s]\n", sr.Symbol.StableID, sr.Symbol.Kind, sr.Source.String())
	}

	fmt.Printf("\nSelected Files (%d):\n", len(result.Files))
	for _, fr := range result.Files {
		fmt.Printf("  %-50s [%s]\n", fr.Path, fr.Source.String())
	}

	fmt.Printf("\nFinal context size: %d tokens\n", result.FinalTokens)
}
