package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

const contextTemplate = `# Project Context
# Language: Go
# Key libraries: (e.g. Ebitengine, gorilla/websocket)
# Architecture: (e.g. multiplayer game server, WebSocket-based)
# Patterns: (e.g. avoid global state, prefer interfaces)
# Constraints: (e.g. 8k context limit, keep examples minimal)
`

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Write a context.md template to the current directory",
	Long:  "Creates context.md with a commented template. Edit it to describe your project so myhelper can give focused answers.",
	RunE:  runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	const filename = "context.md"

	if _, err := os.Stat(filename); err == nil {
		fmt.Fprintf(cmd.OutOrStdout(), "%s already exists — not overwriting. Edit it directly.\n", filename)
		return nil
	}

	if err := os.WriteFile(filename, []byte(contextTemplate), 0644); err != nil {
		return fmt.Errorf("write %s: %w", filename, err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Created %s — edit it to describe your project.\n", filename)
	return nil
}
