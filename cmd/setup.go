package cmd

import (
	"os"

	"github.com/bkohler93/myhelper/internal/wizard"
	"github.com/spf13/cobra"
)

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Interactive first-run wizard: check Ollama, detect hardware, configure search keys",
	Args:  cobra.NoArgs,
	RunE:  runSetup,
}

func init() {
	rootCmd.AddCommand(setupCmd)
}

func runSetup(cmd *cobra.Command, args []string) error {
	return wizard.Run(os.Stdin, os.Stdout)
}
