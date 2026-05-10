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
	"github.com/chzyer/readline"
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

// joinContinuationLines joins a pre-accumulated slice of input lines with newlines.
// Each line in the slice has already had its trailing backslash stripped by the caller.
// This function is package-level so it can be unit-tested directly without a TTY.
func joinContinuationLines(lines []string) string {
	return strings.Join(lines, "\n")
}

// readMultiLine reads one logical input from rl, supporting \ continuation.
// A line ending in \ is appended (without the backslash) and the prompt
// switches to "... " until a bare-Enter line terminates the input.
// The joined string is returned; callers should TrimSpace before use.
// Errors (io.EOF, readline.ErrInterrupt) are returned immediately.
func readMultiLine(rl *readline.Instance) (string, error) {
	var lines []string
	rl.SetPrompt("> ")
	for {
		line, err := rl.Readline()
		if err != nil {
			return "", err
		}
		if strings.HasSuffix(line, `\`) {
			lines = append(lines, strings.TrimSuffix(line, `\`))
			rl.SetPrompt("... ")
			continue
		}
		lines = append(lines, line)
		rl.SetPrompt("> ")
		break
	}
	return joinContinuationLines(lines), nil
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
// On a real TTY, readline handles arrow-key cursor movement, Home/End, and
// in-session history navigation (up/down arrow). Non-TTY callers (pipes,
// go test) fall through to the unchanged bufio path.
//
// Per D-01: prompt is "> " (readline owns it on TTY; no prompt on bufio path).
// Per D-02: empty input reprints "> " and waits; no model call.
// Per D-03: SIGINT installs os/signal handler for bufio path; readline path
// handles Ctrl+C via readline.ErrInterrupt.
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
	preprocessor func(string) string, // nil = identity (no-op)
) error {
	// Install SIGINT handler (kept for bufio path; readline path handles Ctrl+C
	// via readline.ErrInterrupt at the raw-mode level).
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT)
	defer signal.Stop(sigCh)

	// --- READLINE PATH (TTY only) ---
	// Use os.Stdin.Fd() directly — NOT stdinReader — so that the TTY check
	// reflects the real stdin fd and the test seam (stdinReader) is unaffected.
	if readline.IsTerminal(int(os.Stdin.Fd())) {
		rl, err := readline.NewEx(&readline.Config{
			Prompt:                 "> ",
			HistoryLimit:           100,
			DisableAutoSaveHistory: true, // we call SaveHistory manually after joining
			InterruptPrompt:        "^C",
			EOFPrompt:              "exit",
		})
		if err != nil {
			return fmt.Errorf("readline init: %w", err)
		}
		defer rl.Close()

		for {
			// Summarize history if token threshold exceeded (per D-07, D-08).
			if hist.ExceedsLimit() && len(hist.Messages()) >= 4 {
				fmt.Fprint(os.Stderr, "[Condensing history...]\n")
				if err := summarize(cfg, hist, summarizePrompt, recondensePrompt); err != nil {
					return err
				}
			}

			joined, err := readMultiLine(rl)
			if err == io.EOF || err == readline.ErrInterrupt {
				return nil
			}
			if err != nil {
				return err
			}

			input := strings.TrimSpace(joined)
			if input == "" {
				continue
			}
			if input == "quit" {
				return nil
			}

			// Store the raw joined string as a single history entry so that
			// up-arrow recalls the complete multi-line input (not intermediate lines).
			if saveErr := rl.SaveHistory(input); saveErr != nil {
				_ = saveErr // non-fatal: history failure does not break conversation
			}

			msg := input
			if preprocessor != nil {
				msg = preprocessor(input)
			}
			hist.Add("user", msg)
			response, err := streamFn(cfg, hist.Messages())
			if err != nil {
				return err
			}
			hist.Add("assistant", response)
		}
	}

	// --- BUFIO PATH (non-TTY: pipes, go test, CI) ---
	// Launch a single scanner goroutine outside the loop to avoid per-iteration
	// goroutine allocation (WR-03). The goroutine feeds all lines into resultCh
	// and sends a final zero-value entry on EOF/error.
	// Note: when SIGINT is received and the function returns, this goroutine
	// will remain blocked on the underlying io.Reader until the reader is closed.
	// For os.Stdin this is unavoidable without OS-level plumbing; the goroutine
	// lifetime is bounded by the process lifetime (WR-02).
	type scanResult struct {
		text string
		ok   bool
	}
	scanner := bufio.NewScanner(stdinReader)
	resultCh := make(chan scanResult, 1)
	go func() {
		for scanner.Scan() {
			resultCh <- scanResult{text: scanner.Text(), ok: true}
		}
		resultCh <- scanResult{ok: false}
	}()

	for {
		// Check for SIGINT before blocking on input.
		select {
		case <-sigCh:
			return nil
		default:
		}

		// Summarize history if token threshold exceeded (per D-07, D-08).
		// Guard: summarize requires >= 4 messages; skip if too few to avoid
		// an infinite busy-loop where ExceedsLimit stays true but summarize
		// is a no-op.
		if hist.ExceedsLimit() && len(hist.Messages()) >= 4 {
			fmt.Fprint(os.Stderr, "[Condensing history...]\n")
			if err := summarize(cfg, hist, summarizePrompt, recondensePrompt); err != nil {
				return err
			}
		}

		// No prompt printed here — bufio path is non-interactive (pipes, CI).

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

		msg := input
		if preprocessor != nil {
			msg = preprocessor(input)
		}
		hist.Add("user", msg)
		response, err := streamFn(cfg, hist.Messages())
		if err != nil {
			return err
		}
		hist.Add("assistant", response)
	}
}

// resolveInput returns the first positional arg if non-empty, otherwise prompts
// the user interactively. Used by all query subcommands.
func resolveInput(args []string, interactivePrompt string) (string, error) {
	if len(args) > 0 && strings.TrimSpace(args[0]) != "" {
		return strings.TrimSpace(args[0]), nil
	}
	return readInteractive(interactivePrompt)
}

// readInteractive writes prompt to stderr and reads one line from stdinReader.
func readInteractive(prompt string) (string, error) {
	fmt.Fprint(os.Stderr, prompt)
	sc := bufio.NewScanner(stdinReader)
	if !sc.Scan() {
		if err := sc.Err(); err != nil {
			return "", fmt.Errorf("read input: %w", err)
		}
		return "", fmt.Errorf("no input provided")
	}
	input := strings.TrimSpace(sc.Text())
	if input == "" {
		return "", fmt.Errorf("input cannot be empty")
	}
	return input, nil
}

// validateConfig checks that the required Endpoint and Model fields are set.
// Returns a descriptive error with a "myhelper setup" remediation hint when
// either field is empty. Called after config.Load() + ApplyFlagOverrides()
// and before any Ollama calls.
func validateConfig(cfg config.Config) error {
	if cfg.Endpoint == "" && cfg.Model == "" {
		return fmt.Errorf("endpoint and model are not configured\nRun 'myhelper setup' to configure myhelper")
	}
	if cfg.Endpoint == "" {
		return fmt.Errorf("endpoint is not configured\nRun 'myhelper setup' to configure myhelper")
	}
	if cfg.Model == "" {
		return fmt.Errorf("model is not configured\nRun 'myhelper setup' to configure myhelper")
	}
	return nil
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
