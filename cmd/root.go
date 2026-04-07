package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:                   "myhelper",
	Short:                 "A focused Go coding assistant backed by a local Ollama model",
	Long:                  "myhelper sends focused prompts to a local Ollama server and streams responses to stdout.",
	CompletionOptions:     cobra.CompletionOptions{DisableDefaultCmd: true},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
