package cmd

import (
	"github.com/bkohler93/my-helper/internal/config"
	appctx "github.com/bkohler93/my-helper/internal/context"
	"github.com/bkohler93/my-helper/internal/ollama"
	"github.com/spf13/cobra"
)

// lookupSystemPrompt is the focused system prompt for the lookup command.
// The actual prompt text is defined in Plan 1.4.
const lookupSystemPrompt = "TODO: replace in Plan 1.4"

var lookupCmd = &cobra.Command{
	Use:   "lookup <question>",
	Short: "Recommend the right Go API or library for a task",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runLookup,
}

func init() {
	rootCmd.AddCommand(lookupCmd)
}

func runLookup(cmd *cobra.Command, args []string) error {
	input, err := resolveInput(args, "What do you need to do? ")
	if err != nil {
		return err
	}

	projectCtx, err := appctx.LoadContext()
	if err != nil {
		return err
	}

	cfg := config.Load()
	prompt := buildPrompt(projectCtx, lookupSystemPrompt, input)
	return ollama.StreamPrompt(cfg, prompt)
}
