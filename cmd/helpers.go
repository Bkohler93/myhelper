package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/bkohler93/myhelper/internal/config"
	"github.com/bkohler93/myhelper/internal/history"
)

// stdinReader is the source of input for the conversation loop.
// In production this is os.Stdin; tests replace it with a pipe.
var stdinReader io.Reader = os.Stdin

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

var emptyHistoryErr = errors.New("cannot initiate conversation with empty history")

func initiateConversation(cfg config.Config, hist *history.History, streamFn func(config.Config, []history.Message) (string, error)) error {
	if len(hist.Messages()) == 0 {
		return emptyHistoryErr
	}
	res, err := streamFn(cfg, hist.Messages())
	if err != nil {
		log.Println(err)
		return err
	}
	hist.Add("assistant", res)
	return nil
}

// runConversationLoop drives the multi-turn conversation after the first
// model response. It reads follow-up input from stdin, calls streamFn for
// each non-empty non-quit turn, and appends messages to hist.
//
// Exit conditions (all return nil):
//   - User types "quit"
//   - EOF on stdin
//   - SIGINT received
//
// Per D-01: prompt is "> " written to os.Stderr.
// Per D-02: empty input reprints "> " and waits; no model call.
// Per D-03: SIGINT installs os/signal handler; exits cleanly.
// Per D-04: "quit" detected before model call; exits cleanly.
// Per D-05: loop lives here; all 4 commands call this.
func runConversationLoop(
	cfg config.Config,
	hist *history.History,
	streamFn func(config.Config, []history.Message) (string, error),
) error {
	// Install SIGINT handler (per D-03).
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT)
	defer signal.Stop(sigCh)

	scanner := bufio.NewScanner(stdinReader)

	for {
		// Check for SIGINT before blocking on input.
		select {
		case <-sigCh:
			return nil
		default:
		}

		fmt.Fprint(os.Stderr, "> ") // per D-01

		// Scan in a goroutine so SIGINT can interrupt the blocking call.
		type scanResult struct {
			text string
			ok   bool
		}
		resultCh := make(chan scanResult, 1)
		go func() {
			ok := scanner.Scan()
			resultCh <- scanResult{text: scanner.Text(), ok: ok}
		}()

		var result scanResult
		select {
		case <-sigCh:
			return nil
		case result = <-resultCh:
		}

		if !result.ok {
			// EOF or scan error — treat as clean exit.
			if err := scanner.Err(); err != nil {
				return fmt.Errorf("read input: %w", err)
			}
			return nil
		}

		input := strings.TrimSpace(result.text)
		if input == "" {
			continue // per D-02: silent reprompt
		}
		if input == "quit" {
			return nil // per D-04
		}

		hist.Add("user", input)
		response, err := streamFn(cfg, hist.Messages())
		if err != nil {
			return err
		}
		hist.Add("assistant", response)
	}
}
