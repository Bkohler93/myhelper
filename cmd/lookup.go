package cmd

import (
	"github.com/bkohler93/myhelper/internal/config"
	appctx "github.com/bkohler93/myhelper/internal/context"
	"github.com/bkohler93/myhelper/internal/history"
	"github.com/bkohler93/myhelper/internal/ollama"
	"github.com/spf13/cobra"
)

// lookupSystemPrompt is the focused system prompt for the lookup command.
// The actual prompt text is defined in Plan 1.4.
const lookupSystemPrompt = "You are a Go expert. Recommend the best standard library package, or well-known third-party library, for the following task. State the package name, import path, and one sentence on why it fits. Be direct and concise."

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
	messages := []history.Message{
		{Role: "system", Content: buildSystemMessage(projectCtx, lookupSystemPrompt)},
		{Role: "user", Content: input},
	}
	_, err = ollama.StreamChat(cfg, messages)
	return err
}
