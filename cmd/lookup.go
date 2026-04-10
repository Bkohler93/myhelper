package cmd

import (
	"fmt"
	"os"

	"github.com/bkohler93/myhelper/internal/config"
	appctx "github.com/bkohler93/myhelper/internal/context"
	"github.com/bkohler93/myhelper/internal/history"
	"github.com/bkohler93/myhelper/internal/ollama"
	"github.com/bkohler93/myhelper/internal/retrieval"
	"github.com/spf13/cobra"
)

// lookupSystemPrompt is the focused system prompt for the lookup command.
// The actual prompt text is defined in Plan 1.4.
const lookupSystemPrompt = "You are a Go expert. Recommend the best standard library package, or well-known third-party library, for the following task. State the package name, import path, and one sentence on why it fits. Be direct and concise."

const lookupSummarizePrompt = "Summarize the API and library choices, their import paths, and the rationale for each selection discussed above into a concise technical summary."

const lookupRecondensePrompt = "Given the following summary of past events and these new interactions, create an updated, comprehensive summary that preserves all API/library choices and the rationale behind each selection."

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

	root, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("runLookup: getwd: %w", err)
	}
	rctx, err := retrieval.BuildContext(root, input, retrieval.LookupStrategy, cfg, ollama.Chat)
	if err != nil {
		return fmt.Errorf("runLookup: BuildContext: %w", err)
	}

	messages := []history.Message{
		{Role: "system", Content: buildSystemMessage(projectCtx, lookupSystemPrompt)},
	}
	messages = append(messages, rctx.Messages...)
	hist := history.New(cfg.TokenThreshold, messages)
	err = initiateConversation(cfg, hist, ollama.StreamChat)
	if err != nil {
		return err
	}
	err = runConversationLoop(cfg, hist, ollama.StreamChat, lookupSummarizePrompt, lookupRecondensePrompt)
	return err
}
