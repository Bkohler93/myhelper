package cmd

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/bkohler93/myhelper/internal/config"
	"github.com/bkohler93/myhelper/internal/history"
	"github.com/bkohler93/myhelper/internal/ollama"
	"github.com/bkohler93/myhelper/internal/scanner"
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
// Per D-07: ExceedsLimit() checked at top of loop; summarize() called when true.
// Per D-08: "[Condensing history...]" printed to stderr before summarization.
func runConversationLoop(
	cfg config.Config,
	hist *history.History,
	streamFn func(config.Config, []history.Message) (string, error),
	summarizePrompt string,
	recondensePrompt string,
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

		// Summarize history if token threshold exceeded (per D-07, D-08).
		if hist.ExceedsLimit() {
			fmt.Fprint(os.Stderr, "[Condensing history...]\n")
			if err := summarize(cfg, hist, summarizePrompt, recondensePrompt); err != nil {
				return err
			}
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

// summarize compresses history when the token threshold is exceeded.
// It detects re-condensation by checking for an existing summary message.
// The system message at index [0] is always preserved.
//
// Per D-06: re-condensation detected by "Summary of previous conversation:" prefix.
// Per D-09: if len(msgs) < 5, nothing meaningful to compress; returns nil.
func summarize(cfg config.Config, hist *history.History, summarizePrompt, recondensePrompt string) error {
	msgs := hist.Messages()
	// msgs[0] is the system prompt — never summarized.
	// The last two messages are the most recent user+assistant exchange — kept verbatim.
	// Everything between [1] and [len-3] (inclusive) is the candidate for summarization.

	// Detect re-condensation: is there already a summary message in the slice?
	prompt := summarizePrompt
	for _, m := range msgs[1:] {
		if m.Role == "system" && strings.HasPrefix(m.Content, "Summary of previous conversation:") {
			prompt = recondensePrompt
			break
		}
	}

	// Safe to summarize only if there are at least 5 messages (system + ≥1 exchange before final pair).
	// If only [system, user, assistant] exist (3 messages), there is nothing to compress separately
	// from the final exchange — return nil.
	if len(msgs) < 5 {
		return nil
	}

	finalPair := msgs[len(msgs)-2:] // last user + last assistant
	candidates := msgs[1 : len(msgs)-2] // everything between system and final pair

	summarizeMessages := make([]history.Message, 0, len(candidates)+1)
	summarizeMessages = append(summarizeMessages, candidates...)
	summarizeMessages = append(summarizeMessages, history.Message{Role: "user", Content: prompt})

	summaryText, err := ollama.Chat(cfg, summarizeMessages)
	if err != nil {
		return fmt.Errorf("summarize: %w", err)
	}

	newMessages := make([]history.Message, 0, 4)
	newMessages = append(newMessages, msgs[0]) // original system message
	newMessages = append(newMessages, history.Message{
		Role:    "system",
		Content: "Summary of previous conversation: " + summaryText,
	})
	newMessages = append(newMessages, finalPair...)
	hist.Replace(newMessages)
	return nil
}

// syncMeta is written to .myhelper/meta.json after every successful init or sync.
type syncMeta struct {
	LastSync time.Time `json:"last_sync"`
}

// generateContextMD reads per-package summaries from .myhelper/summaries/,
// synthesizes them into a human-readable project overview via chatFn, and
// writes the result to .myhelper/context.md.
// ChatFn is injected for testability (per D-09).
func generateContextMD(root string, cfg config.Config, chatFn scanner.ChatFn) error {
	summariesDir := filepath.Join(root, ".myhelper", "summaries")
	entries, err := os.ReadDir(summariesDir)
	if err != nil {
		return fmt.Errorf("generateContextMD: read summaries dir: %w", err)
	}

	var sb strings.Builder
	sb.WriteString("Below are per-package summaries for a Go project. ")
	sb.WriteString("Write a concise, human-readable project overview in markdown. ")
	sb.WriteString("Describe what the project does, its key packages, and how they relate. ")
	sb.WriteString("Be brief — under 300 words. Format as clean markdown prose, not a symbol list.\n\n")

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(summariesDir, e.Name()))
		if err != nil {
			return fmt.Errorf("generateContextMD: read %s: %w", e.Name(), err)
		}
		sb.WriteString("### " + strings.TrimSuffix(e.Name(), ".md") + "\n")
		sb.WriteString(string(data) + "\n\n")
	}

	messages := []history.Message{
		{Role: "user", Content: sb.String()},
	}
	content, err := chatFn(cfg, messages)
	if err != nil {
		return fmt.Errorf("generateContextMD: chatFn: %w", err)
	}

	outPath := filepath.Join(root, ".myhelper", "context.md")
	return os.WriteFile(outPath, []byte(content), 0644)
}

// readLastSync reads the stored last_sync timestamp from .myhelper/meta.json.
// Returns time.Time{} (zero value) if the file does not exist — callers treat
// zero as "never synced" (all files are considered changed).
func readLastSync(root string) (time.Time, error) {
	path := filepath.Join(root, ".myhelper", "meta.json")
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return time.Time{}, nil
	}
	if err != nil {
		return time.Time{}, fmt.Errorf("readLastSync: %w", err)
	}
	var m syncMeta
	if err := json.Unmarshal(data, &m); err != nil {
		return time.Time{}, fmt.Errorf("readLastSync: unmarshal: %w", err)
	}
	return m.LastSync, nil
}

// writeLastSync writes t as the last_sync timestamp to .myhelper/meta.json.
// Called after every successful init or sync (per D-05).
func writeLastSync(root string, t time.Time) error {
	path := filepath.Join(root, ".myhelper", "meta.json")
	data, err := json.Marshal(syncMeta{LastSync: t})
	if err != nil {
		return fmt.Errorf("writeLastSync: marshal: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}
