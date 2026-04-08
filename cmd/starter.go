package cmd

import (
	"fmt"
	"os"

	"github.com/bkohler93/myhelper/internal/config"
	appctx "github.com/bkohler93/myhelper/internal/context"
	"github.com/bkohler93/myhelper/internal/history"
	"github.com/bkohler93/myhelper/internal/ollama"
	"github.com/spf13/cobra"
)

// starterSystemPrompt is the focused system prompt for the starter command.
// The actual prompt text is defined in Plan 1.4.
const starterSystemPrompt = "You are a Go coding assistant. Write the minimal working Go code that accomplishes the following task. Include only what is necessary — no boilerplate, no extra explanation. Output only the code block."

const starterSummarizePrompt = "Summarize the code structures, patterns chosen, and any constraints identified above into a concise technical summary."

const starterRecondensePrompt = "Given the following summary of past events and these new interactions, create an updated, comprehensive summary that preserves all code structure decisions and patterns chosen."

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

	root, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("runStarter: getwd: %w", err)
	}
	injected, err := buildInjectedMessages(root, input, cfg, ollama.Chat, pass1StarterFocus)
	if err != nil {
		return fmt.Errorf("runStarter: buildInjectedMessages: %w", err)
	}

	messages := []history.Message{
		{Role: "system", Content: buildSystemMessage(projectCtx, starterSystemPrompt)},
	}
	messages = append(messages, injected...)
	hist := history.New(cfg.TokenThreshold, messages)
	err = initiateConversation(cfg, hist, ollama.StreamChat)
	if err != nil {
		return err
	}
	err = runConversationLoop(cfg, hist, ollama.StreamChat, starterSummarizePrompt, starterRecondensePrompt)
	return err
}
