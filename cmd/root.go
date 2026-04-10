package cmd

import (
	"fmt"
	"os"

	"github.com/bkohler93/myhelper/internal/config"
	"github.com/spf13/cobra"
)

var tokenLimitFlag int
var noContextFlag bool

var rootCmd = &cobra.Command{
	Use:                   "myhelper",
	Short:                 "A focused Go coding assistant backed by a local Ollama model",
	Long:                  "myhelper sends focused prompts to a local Ollama server and streams responses to stdout.",
	CompletionOptions:     cobra.CompletionOptions{DisableDefaultCmd: true},
}

func init() {
	rootCmd.PersistentFlags().IntVar(&tokenLimitFlag, "token-limit", 0, "override token threshold for conversation history (default 4100)")
	rootCmd.PersistentFlags().BoolVar(&noContextFlag, "no-context", false, "bypass retrieval and inject no project context")
}

// ApplyFlagOverrides applies any CLI flag overrides to a resolved Config.
// Call this after config.Load() in each command's RunE.
func ApplyFlagOverrides(cfg *config.Config) {
	if tokenLimitFlag != 0 {
		cfg.TokenThreshold = tokenLimitFlag
	}
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
