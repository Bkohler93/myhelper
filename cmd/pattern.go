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

// patternSystemPrompt is the focused system prompt for the pattern command.
// The actual prompt text is defined in Plan 1.4.
const patternSystemPrompt = "You are a Go expert. Describe the idiomatic Go way to implement or structure the following. Be concise: focus on the key pattern, naming conventions, and any common pitfalls. No code unless a short snippet is essential."

const patternSummarizePrompt = "Summarize the idiomatic Go pattern identified, its constraints, and any pitfalls discussed above into a concise technical summary."

const patternRecondensePrompt = "Given the following summary of past events and these new interactions, create an updated, comprehensive summary that preserves the idiomatic pattern identified and all its constraints."

var patternCmd = &cobra.Command{
	Use:   "pattern <topic>",
	Short: "Describe the idiomatic Go way to structure or write something",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runPattern,
}

func init() {
	rootCmd.AddCommand(patternCmd)
}

func runPattern(cmd *cobra.Command, args []string) error {
	input, err := resolveInput(args, "What pattern or structure do you want? ")
	if err != nil {
		return err
	}

	projectCtx, err := appctx.LoadContext()
	if err != nil {
		return err
	}

	cfg := config.Load()
	ApplyFlagOverrides(&cfg)

	root, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("runPattern: getwd: %w", err)
	}
	var rctx retrieval.Context
	if noContextFlag {
		rctx = retrieval.Context{
			Messages: []history.Message{{Role: "user", Content: input}},
		}
	} else {
		rctx, err = retrieval.BuildContext(root, input, retrieval.PatternStrategy, cfg, ollama.Chat)
		if err != nil {
			return fmt.Errorf("runPattern: BuildContext: %w", err)
		}
	}

	messages := []history.Message{
		{Role: "system", Content: buildSystemMessage(projectCtx, patternSystemPrompt)},
	}
	messages = append(messages, rctx.Messages...)
	hist := history.New(cfg.TokenThreshold, messages)
	err = initiateConversation(cfg, hist, ollama.StreamChat)
	if err != nil {
		return err
	}
	err = runConversationLoop(cfg, hist, ollama.StreamChat, patternSummarizePrompt, patternRecondensePrompt)
	return err
}
