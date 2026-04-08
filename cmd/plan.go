package cmd

import (
	"github.com/bkohler93/myhelper/internal/config"
	appctx "github.com/bkohler93/myhelper/internal/context"
	"github.com/bkohler93/myhelper/internal/history"
	"github.com/bkohler93/myhelper/internal/ollama"
	"github.com/spf13/cobra"
)

// planSystemPrompt is the focused system prompt for the plan command.
// The actual prompt text is defined in Plan 1.4.
const planSystemPrompt = "You are a Go software planning assistant. Break the following feature or task into a short, ordered list of concrete subtasks. Each subtask should be one clear action. Output only the numbered list, no preamble."

const planSummarizePrompt = "Summarize the key decisions, ordered subtasks, and any blockers discussed above into a concise technical summary."

const planRecondensePrompt = "Given the following summary of past events and these new interactions, create an updated, comprehensive summary that preserves all subtasks, ordered decisions, and blockers relevant to the plan."

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
	messages := []history.Message{
		{Role: "system", Content: buildSystemMessage(projectCtx, planSystemPrompt)},
		{Role: "user", Content: input},
	}
	h := history.New(cfg.TokenThreshold, messages)

	err = initiateConversation(cfg, h, ollama.StreamChat)
	if err != nil {
		return err
	}
	err = runConversationLoop(cfg, h, ollama.StreamChat)
	return err
}
