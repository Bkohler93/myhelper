package cmd

import (
	"github.com/bkohler93/myhelper/internal/config"
	appctx "github.com/bkohler93/myhelper/internal/context"
	"github.com/bkohler93/myhelper/internal/ollama"
	"github.com/spf13/cobra"
)

// starterSystemPrompt is the focused system prompt for the starter command.
// The actual prompt text is defined in Plan 1.4.
const starterSystemPrompt = "You are a Go coding assistant. Write the minimal working Go code that accomplishes the following task. Include only what is necessary — no boilerplate, no extra explanation. Output only the code block."

var starterCmd = &cobra.Command{
	Use:   "starter <task>",
	Short: "Print minimal working Go code for a given task",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runStarter,
}

func init() {
	rootCmd.AddCommand(starterCmd)
}

func runStarter(cmd *cobra.Command, args []string) error {
	input, err := resolveInput(args, "Describe the task: ")
	if err != nil {
		return err
	}

	projectCtx, err := appctx.LoadContext()
	if err != nil {
		return err
	}

	cfg := config.Load()
	prompt := buildPrompt(projectCtx, starterSystemPrompt, input)
	return ollama.StreamPrompt(cfg, prompt)
}
