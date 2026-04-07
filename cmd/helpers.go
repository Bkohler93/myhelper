package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// readInteractive prints a prompt to stderr and reads a line from stdin.
// Used when the user omits the positional argument.
func readInteractive(prompt string) (string, error) {
	fmt.Fprint(os.Stderr, prompt)
	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return "", fmt.Errorf("read input: %w", err)
		}
		return "", fmt.Errorf("no input provided")
	}
	input := strings.TrimSpace(scanner.Text())
	if input == "" {
		return "", fmt.Errorf("input cannot be empty")
	}
	return input, nil
}

// buildSystemMessage composes the system message content from optional project
// context and the command-specific system prompt. In the /api/chat model,
// this becomes the Content field of the system message at messages[0].
func buildSystemMessage(projectContext, systemPrompt string) string {
	var sb strings.Builder
	if projectContext != "" {
		sb.WriteString("Project context:\n")
		sb.WriteString(projectContext)
		sb.WriteString("\n\n")
	}
	sb.WriteString(systemPrompt)
	return sb.String()
}

// resolveInput returns the first positional arg if provided, otherwise
// prompts the user interactively on stderr.
func resolveInput(args []string, interactivePrompt string) (string, error) {
	if len(args) > 0 && strings.TrimSpace(args[0]) != "" {
		return strings.TrimSpace(args[0]), nil
	}
	return readInteractive(interactivePrompt)
}
