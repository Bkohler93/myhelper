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
	"github.com/bkohler93/myhelper/internal/ollama"
)

// stdinReader is the source of input for the conversation loop.
// In production this is os.Stdin; tests replace it with a pipe.
var stdinReader io.Reader = os.Stdin

var emptyHistoryErr = errors.New("cannot initiate conversation with empty history")

// summaryPrefix is the prefix used to mark summary messages in the conversation history.
// Used for re-condensation detection and summary message construction.
const summaryPrefix = "Summary of previous conversation:"

const summarizePrompt = "Summarize the conversation above as a brief paragraph, preserving key facts and questions."
const recondensePrompt = "The conversation already contains a summary. Produce an updated, condensed summary that incorporates the new exchanges."

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
// It operates on no-system-prompt history: msgs[0] is always a user message.
//
// Per CHAT-04: no system prompt at index 0.
// Per CHAT-05: minimum 4 messages required ([u,a,u,a]) — last pair kept verbatim.
// Per D-06: re-condensation detected by summaryPrefix in any system-role message.
func summarize(cfg config.Config, hist *history.History, summarizePrompt, recondensePrompt string) error {
	msgs := hist.Messages()

	// Need at least [user, assistant, user, assistant] = 4 messages.
	// With only 3 or fewer there is nothing to compress separately from the final pair.
	if len(msgs) < 4 {
		return nil
	}

	finalPair := msgs[len(msgs)-2:]     // last user + last assistant kept verbatim
	candidates := msgs[0 : len(msgs)-2] // everything before final pair (no system msg to skip)

	// Detect re-condensation: scan ALL messages for an existing summary.
	prompt := summarizePrompt
	for _, m := range msgs {
		if m.Role == "system" && strings.HasPrefix(m.Content, summaryPrefix) {
			prompt = recondensePrompt
			break
		}
	}

	summarizeMessages := make([]history.Message, 0, len(candidates)+1)
	summarizeMessages = append(summarizeMessages, candidates...)
	summarizeMessages = append(summarizeMessages, history.Message{Role: "user", Content: prompt})

	summaryText, err := ollama.Chat(cfg, summarizeMessages)
	if err != nil {
		return fmt.Errorf("summarize: %w", err)
	}

	newMessages := make([]history.Message, 0, 3)
	newMessages = append(newMessages, history.Message{
		Role:    "system",
		Content: summaryPrefix + " " + summaryText,
	})
	newMessages = append(newMessages, finalPair...)
	hist.Replace(newMessages)
	return nil
}
