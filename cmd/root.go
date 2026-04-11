package cmd

import (
	"fmt"
	"os"

	"github.com/bkohler93/myhelper/internal/config"
	"github.com/bkohler93/myhelper/internal/history"
	"github.com/bkohler93/myhelper/internal/ollama"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:               "myhelper [question]",
	Short:             "A focused chat assistant backed by a local Ollama model",
	Long:              "myhelper sends messages to a local Ollama server and streams responses to stdout.\n\nRun with no arguments to start an interactive REPL, or pass a question to get a one-shot response.",
	CompletionOptions: cobra.CompletionOptions{HiddenDefaultCmd: true},
	Args:              cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := config.Load()
		hist := history.New(cfg.TokenThreshold, nil)

		if len(args) == 1 {
			// One-shot mode (CHAT-02): add user message, stream response, exit.
			hist.Add("user", args[0])
			return initiateConversation(cfg, hist, ollama.StreamChat)
		}

		// REPL mode (CHAT-01): loop handles first prompt itself; do NOT call initiateConversation first.
		return runConversationLoop(cfg, hist, ollama.StreamChat, summarizePrompt, recondensePrompt)
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
