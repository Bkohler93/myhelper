package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:               "myhelper",
	Short:             "A focused chat assistant backed by a local Ollama model",
	Long:              "myhelper sends messages to a local Ollama server and streams responses to stdout.",
	CompletionOptions: cobra.CompletionOptions{HiddenDefaultCmd: true},
	Args:              cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return nil
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
