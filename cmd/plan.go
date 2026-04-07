package cmd

import (
	"github.com/bkohler93/my-helper/internal/config"
	appctx "github.com/bkohler93/my-helper/internal/context"
	"github.com/bkohler93/my-helper/internal/ollama"
	"github.com/spf13/cobra"
)

// planSystemPrompt is the focused system prompt for the plan command.
// The actual prompt text is defined in Plan 1.4.
const planSystemPrompt = "You are a Go software planning assistant. Break the following feature or task into a short, ordered list of concrete subtasks. Each subtask should be one clear action. Output only the numbered list, no preamble."

var planCmd = &cobra.Command{
	Use:   "plan <description>",
	Short: "Break a feature or task into ordered subtasks",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runPlan,
}

func init() {
	rootCmd.AddCommand(planCmd)
}

func runPlan(cmd *cobra.Command, args []string) error {
	input, err := resolveInput(args, "Describe the feature or task: ")
	if err != nil {
		return err
	}

	projectCtx, err := appctx.LoadContext()
	if err != nil {
		return err
	}

	cfg := config.Load()
	prompt := buildPrompt(projectCtx, planSystemPrompt, input)
	return ollama.StreamPrompt(cfg, prompt)
}
