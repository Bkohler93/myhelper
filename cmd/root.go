package cmd

import (
	"fmt"
	"os"

	"github.com/bkohler93/myhelper/internal/config"
	"github.com/bkohler93/myhelper/internal/history"
	"github.com/bkohler93/myhelper/internal/ollama"
	"github.com/bkohler93/myhelper/internal/search"
	"github.com/spf13/cobra"
)

var (
	searchForce    bool
	searchSuppress bool
)

func init() {
	rootCmd.PersistentFlags().BoolVar(&searchForce, "search", false, "Force web search regardless of gate result")
	rootCmd.PersistentFlags().BoolVar(&searchSuppress, "no-search", false, "Suppress web search entirely")
}

var rootCmd = &cobra.Command{
	Use:               "myhelper [question]",
	Short:             "A focused chat assistant backed by a local Ollama model",
	Long:              "myhelper sends messages to a local Ollama server and streams responses to stdout.\n\nRun with no arguments to start an interactive REPL, or pass a question to get a one-shot response.",
	CompletionOptions: cobra.CompletionOptions{HiddenDefaultCmd: true},
	Args:              cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := config.Load()
		searchCfg := search.LoadConfig() // load once, capture in closure (Pitfall 6)
		hist := history.New(cfg.TokenThreshold, nil)

		if len(args) == 1 {
			// One-shot mode: augment query before adding to history (GATE-03, GATE-04)
			augmented := buildUserMessage(args[0], cfg, searchCfg, searchForce, searchSuppress)
			hist.Add("user", augmented)
			return initiateConversation(cfg, hist, ollama.StreamChat)
		}

		// REPL mode: build preprocessor closure capturing search state (Pitfall 6)
		preprocessor := func(input string) string {
			return buildUserMessage(input, cfg, searchCfg, searchForce, searchSuppress)
		}
		return runConversationLoop(cfg, hist, ollama.StreamChat, summarizePrompt, recondensePrompt, preprocessor)
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
