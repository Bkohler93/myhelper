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
	"regexp"
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
	scanner := bufio.NewScanner(stdinReader)
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

// summaryPrefix is the prefix used to mark summary messages in the conversation history.
// Used for re-condensation detection and summary message construction.
const summaryPrefix = "Summary of previous conversation:"

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
		if m.Role == "system" && strings.HasPrefix(m.Content, summaryPrefix) {
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
		Content: summaryPrefix + " " + summaryText,
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

	summaryCount := 0
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
		summaryCount++
	}

	if summaryCount == 0 {
		return fmt.Errorf("generateContextMD: no summaries found in %s — run init first", summariesDir)
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

// Pass-1 system prompt constants for two-pass context injection (Phase 7).
const pass1BaseSystemPrompt = "You are a file retrieval assistant. Based on the project index below, identify which files (max 3) are absolutely necessary to read to answer the user's request. Output ONLY a comma-separated list of file paths."
const pass1PlanFocus = "Focus on architectural entry points and interfaces."
const pass1LookupFocus = "Strictly find definitions of the specific term."
const pass1StarterFocus = "Find boilerplate or similar commands/structs to copy."
const pass1PatternFocus = "Find multiple instances of a repeated logic pattern."

// readIndexFile reads and unmarshals .myhelper/index.json from root.
// Returns os.ErrNotExist-wrapped error if the file does not exist.
func readIndexFile(root string) (scanner.Index, error) {
	path := filepath.Join(root, ".myhelper", "index.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return scanner.Index{}, err
	}
	var idx scanner.Index
	if err := json.Unmarshal(data, &idx); err != nil {
		return scanner.Index{}, fmt.Errorf("readIndexFile: unmarshal: %w", err)
	}
	return idx, nil
}

// injectSummaries is the fallback injection path: reads all .md files from
// .myhelper/summaries/ and prepends them to the user query.
// If the summaries directory does not exist or is empty, returns a bare user
// query message with no error.
func injectSummaries(root, query string) ([]history.Message, error) {
	dir := filepath.Join(root, ".myhelper", "summaries")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return []history.Message{{Role: "user", Content: query}}, nil
	}
	var sb strings.Builder
	sb.WriteString("Here are project package summaries for context:\n")
	count := 0
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		sb.WriteString("Package: " + strings.TrimSuffix(e.Name(), ".md") + "\n")
		sb.WriteString(string(data) + "\n")
		count++
	}
	if count == 0 {
		return []history.Message{{Role: "user", Content: query}}, nil
	}
	sb.WriteString(query)
	return []history.Message{{Role: "user", Content: sb.String()}}, nil
}

// microPassRe matches "N-M" line range responses from the micro-pass LLM call.
var microPassRe = regexp.MustCompile(`(\d+)-(\d+)`)

// microPassFile asks the model which line range of an oversized file is needed
// to answer the query, then extracts and returns those lines.
//
// Fallback chain (per D-10):
//  1. Build symbol map via scanner.ExtractSymbolMap.
//  2. Call chatFn with symbol map + query; parse "N-M" response.
//  3. If parse succeeds and extracted range fits budget: return extracted lines.
//  4. Otherwise: truncate raw content at last newline that fits budget.
//  5. If even truncated content doesn't fit: return ("", false).
//
// Never panics. Never surfaces an error to the user (per D-06, D-09).
// root is the project root; path is relative to root.
func microPassFile(root, path, query string, cfg config.Config, chatFn scanner.ChatFn, budget int) (string, bool) {
	if budget <= 0 {
		return "", false
	}

	absPath := filepath.Join(root, path)
	rawBytes, err := os.ReadFile(absPath)
	if err != nil {
		return "", false
	}
	rawContent := string(rawBytes)
	lines := strings.Split(rawContent, "\n")
	totalLines := len(lines)

	// Attempt micro-pass: build symbol map and ask the model for a line range.
	extracted, ok := func() (string, bool) {
		symbols, err := scanner.ExtractSymbolMap(absPath)
		if err != nil {
			return "", false
		}

		// Build symbol map text (D-01, D-02).
		var mapSB strings.Builder
		for _, sym := range symbols {
			fmt.Fprintf(&mapSB, "%s: lines %d-%d\n", sym.Name, sym.Start, sym.End)
		}

		// Compose micro-pass messages (D-03, D-04).
		systemPrompt := "Given this file's symbol map, output ONLY the line range needed to answer the user's request. Format: start-end (e.g., 12-55). Output nothing else."
		userMsg := fmt.Sprintf("Symbols in %s:\n%s\nUser request: %s", path, mapSB.String(), query)
		microMessages := []history.Message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userMsg},
		}

		resp, err := chatFn(cfg, microMessages)
		if err != nil {
			return "", false
		}

		// Parse "N-M" response (D-05).
		m := microPassRe.FindStringSubmatch(strings.TrimSpace(resp))
		if m == nil {
			return "", false
		}
		var start, end int
		fmt.Sscanf(m[1], "%d", &start)
		fmt.Sscanf(m[2], "%d", &end)

		// Clamp out-of-bounds (D-07).
		if start < 1 {
			start = 1
		}
		if end > totalLines {
			end = totalLines
		}
		// On invalid range after clamping, fall through to truncation (D-06).
		if start > end {
			return "", false
		}

		// Extract lines [start, end] inclusive (0-indexed slice, 1-indexed line numbers).
		extracted := strings.Join(lines[start-1:end], "\n")
		tokCount := history.New(cfg.TokenThreshold, []history.Message{{Role: "user", Content: extracted}}).TokenCount()
		if tokCount > budget {
			return "", false
		}
		return extracted, true
	}()

	if ok {
		return extracted, true
	}

	// Truncation fallback (D-08): scan from right for last newline that fits.
	for i := len(rawContent) - 1; i >= 0; i-- {
		if rawContent[i] == '\n' {
			prefix := rawContent[:i+1]
			tokCount := history.New(cfg.TokenThreshold, []history.Message{{Role: "user", Content: prefix}}).TokenCount()
			if tokCount <= budget {
				return prefix, true
			}
		}
	}

	// D-09: even a single line doesn't fit — skip this file.
	return "", false
}

// buildInjectedMessages performs two-pass context injection:
//  1. Pass 1: calls chatFn with the project index to select relevant files (max 3).
//  2. Validates each returned path with os.Stat; discards invalid paths.
//  3. Reads raw file content within an 80% token budget; falls back to symbol
//     list when raw content exceeds remaining budget; skips file when both exceed.
//  4. Falls back to injectSummaries when no valid paths survive Pass 1.
//  5. Falls back to bare user query when index.json does not exist.
//
// All returned messages have Role "user" (never system).
func buildInjectedMessages(root, query string, cfg config.Config, chatFn scanner.ChatFn, focus string) ([]history.Message, error) {
	idx, err := readIndexFile(root)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Fprint(os.Stderr, "No index found — run 'mh init' for context-aware answers.\n")
			return []history.Message{{Role: "user", Content: query}}, nil
		}
		return nil, fmt.Errorf("buildInjectedMessages: read index: %w", err)
	}

	indexJSON, err := json.Marshal(idx)
	if err != nil {
		return injectSummaries(root, query)
	}
	systemPrompt := pass1BaseSystemPrompt
	if focus != "" {
		systemPrompt += "\n" + focus
	}
	pass1Messages := []history.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: "Project index:\n" + string(indexJSON) + "\n\nUser request: " + query},
	}

	response, err := chatFn(cfg, pass1Messages)
	if err != nil {
		return injectSummaries(root, query)
	}

	var validPaths []string
	for candidate := range strings.SplitSeq(response, ",") {
		trimmed := strings.TrimSpace(candidate)
		if trimmed == "" {
			continue
		}
		p := filepath.Clean(trimmed)
		// Reject absolute paths and any path that escapes the project root.
		if filepath.IsAbs(p) || strings.HasPrefix(p, "..") {
			continue
		}
		if _, statErr := os.Stat(filepath.Join(root, p)); statErr == nil {
			validPaths = append(validPaths, p)
		}
	}

	if len(validPaths) == 0 {
		return injectSummaries(root, query)
	}

	budget := int(float64(cfg.TokenThreshold) * 0.80)
	usedTokens := 0

	var sb strings.Builder
	sb.WriteString("Here is the relevant source code for context:\n")

	for _, path := range validPaths {
		if usedTokens >= budget {
			break
		}
		rawContent, readErr := os.ReadFile(filepath.Join(root, path))
		if readErr != nil {
			continue
		}
		rawTokens := history.New(cfg.TokenThreshold, []history.Message{{Role: "user", Content: string(rawContent)}}).TokenCount()

		if usedTokens+rawTokens <= budget {
			sb.WriteString("File: " + path + "\n```go\n" + string(rawContent) + "\n```\n")
			usedTokens += rawTokens
			continue
		}

		// Raw content too large — attempt micro-pass, then truncation (D-10, D-11).
		if content, ok := microPassFile(root, path, query, cfg, chatFn, budget-usedTokens); ok {
			sb.WriteString("File: " + path + " (partial)\n```go\n" + content + "\n```\n")
			usedTokens += history.New(cfg.TokenThreshold, []history.Message{{Role: "user", Content: content}}).TokenCount()
		}
		// If microPassFile returns false, skip this file silently (D-09).
	}

	sb.WriteString(query)
	return []history.Message{{Role: "user", Content: sb.String()}}, nil
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
