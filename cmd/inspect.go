package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var inspectCmd = &cobra.Command{
	Use:   "inspect <query>",
	Short: "Inspect the web search pipeline for a query (rewrite in progress)",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runInspect,
}

func init() {
	rootCmd.AddCommand(inspectCmd)
}

func runInspect(cmd *cobra.Command, args []string) error {
	fmt.Println("inspect: rewrite in progress (Phase 27)")
	return nil
}
