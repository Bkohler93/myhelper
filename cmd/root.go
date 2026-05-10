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
	tokenLimitFlag int
)

// Version info — set by goreleaser ldflags at build time:
//
//	-X github.com/bkohler93/myhelper/cmd.Version={{.Version}}
//	-X github.com/bkohler93/myhelper/cmd.Commit={{.Commit}}
//	-X github.com/bkohler93/myhelper/cmd.Date={{.Date}}
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

func init() {
	rootCmd.PersistentFlags().BoolVar(&searchForce, "search", false, "Force web search regardless of gate result")
	rootCmd.PersistentFlags().BoolVar(&searchSuppress, "no-search", false, "Suppress web search entirely")
	rootCmd.PersistentFlags().IntVar(&tokenLimitFlag, "token-limit", 0, "override token threshold for conversation history (default 4100)")
}

var rootCmd = &cobra.Command{
	Use:               "myhelper [question]",
	Short:             "A focused chat assistant backed by a local Ollama model",
	Long:              "myhelper sends messages to a local Ollama server and streams responses to stdout.\n\nRun with no arguments to start an interactive REPL, or pass a question to get a one-shot response.",
	Version:           fmt.Sprintf("%s (commit %s, built %s)", Version, Commit, Date),
	CompletionOptions: cobra.CompletionOptions{HiddenDefaultCmd: true},
	Args:              cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := config.Load()
		ApplyFlagOverrides(&cfg)
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

// ApplyFlagOverrides applies CLI flag values to cfg, overriding config-file values.
func ApplyFlagOverrides(cfg *config.Config) {
	if tokenLimitFlag != 0 {
		cfg.TokenThreshold = tokenLimitFlag
	}
}
