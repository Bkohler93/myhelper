package ollama

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/glamour"
	"github.com/chzyer/readline"

	"github.com/bkohler93/myhelper/internal/config"
	"github.com/bkohler93/myhelper/internal/history"
)

// renderMarkdown renders a markdown string using glamour with auto style detection.
// Returns the raw string unchanged on any error or if the input is blank.
func renderMarkdown(s string) string {
	if strings.TrimSpace(s) == "" {
		return s
	}
	r, err := glamour.NewTermRenderer(glamour.WithAutoStyle())
	if err != nil {
		return s
	}
	out, err := r.Render(s)
	if err != nil {
		return s
	}
	return out
}

// startSpinner starts an animated stderr spinner with the given label.
// The returned done func blocks until the spinner goroutine has cleared the line.
func startSpinner(label string) func() {
	stop := make(chan struct{})
	cleaned := make(chan struct{})
	go func() {
		frames := []rune{'|', '/', '-', '\\'}
		i := 0
		t := time.NewTicker(100 * time.Millisecond)
		defer t.Stop()
		for {
			fmt.Fprintf(os.Stderr, "\r%c %s", frames[i], label)
			i = (i + 1) % len(frames)
			select {
			case <-stop:
				fmt.Fprintf(os.Stderr, "\r%s\r", strings.Repeat(" ", len(label)+3))
				close(cleaned)
				return
			case <-t.C:
			}
		}
	}()
	return func() {
		close(stop)
		<-cleaned
	}
}

func startRenderSpinner() func() { return startSpinner("Rendering...") }

type chatRequest struct {
	Model    string            `json:"model"`
	Messages []history.Message `json:"messages"`
	Stream   bool              `json:"stream"`
	Format   json.RawMessage   `json:"format,omitempty"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatResponse struct {
	Message chatMessage `json:"message"`
	Done    bool        `json:"done"`
}

// StreamChat sends a messages array to the Ollama /api/chat endpoint,
// streams each response token to stdout, and returns the full accumulated
// response text. The caller is responsible for appending the returned text
// to history as an assistant message.
func StreamChat(cfg config.Config, messages []history.Message) (string, error) {
	url := chatURL(cfg.Endpoint)

	reqBody := chatRequest{
		Model:    cfg.Model,
		Messages: messages,
		Stream:   true,
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	resp, err := http.Post(url, "application/json", bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("POST %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("ollama returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	isTTY := readline.IsTerminal(int(os.Stdout.Fd()))

	var sb strings.Builder
	scanner := bufio.NewScanner(resp.Body)

	var stopGen func()
	firstToken := true
	if isTTY {
		fmt.Fprint(os.Stdout, "\033[s") // save cursor position for erase-and-replace after stream
		stopGen = startSpinner("Generating...")
	}

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var chunk chatResponse
		if err := json.Unmarshal(line, &chunk); err != nil {
			if isTTY && firstToken {
				stopGen()
			}
			return "", fmt.Errorf("unmarshal chunk: %w", err)
		}
		if isTTY && firstToken {
			stopGen() // clear "Generating..." before first token appears on screen
			firstToken = false
		}
		fmt.Fprint(os.Stdout, chunk.Message.Content)
		sb.WriteString(chunk.Message.Content)
		if chunk.Done {
			break
		}
	}

	if isTTY {
		if firstToken {
			stopGen() // no tokens received — clear spinner anyway
		}
		fmt.Fprint(os.Stdout, "\033[u\033[J") // restore saved cursor, erase streamed content
		done := startRenderSpinner()
		rendered := renderMarkdown(sb.String())
		done()
		fmt.Fprint(os.Stdout, rendered) // glamour output already ends with \n
	} else {
		fmt.Fprintln(os.Stdout) // non-TTY trailing newline — unchanged
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("reading stream: %w", err)
	}

	return sb.String(), nil
}

// Chat sends a messages array to the Ollama /api/chat endpoint in
// non-streaming mode and returns the full response content as a string.
// Unlike StreamChat, nothing is written to stdout — the response text is
// returned to the caller only. Used for internal calls like summarization.
func Chat(cfg config.Config, messages []history.Message) (string, error) {
	url := chatURL(cfg.Endpoint)

	reqBody := chatRequest{
		Model:    cfg.Model,
		Messages: messages,
		Stream:   false,
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	resp, err := http.Post(url, "application/json", bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("POST %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("ollama returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var res chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}
	return res.Message.Content, nil
}

// ChatWithFormat sends a messages array to the Ollama /api/chat endpoint in
// non-streaming mode with a JSON schema constraint. The schema is passed as
// json.RawMessage and serialized into the "format" field of the request body.
// Returns the model's response content as a string. Used for internal pipeline
// calls requiring structured JSON output. Nothing is written to stdout.
func ChatWithFormat(cfg config.Config, messages []history.Message, schema json.RawMessage) (string, error) {
	url := chatURL(cfg.Endpoint)

	reqBody := chatRequest{
		Model:    cfg.Model,
		Messages: messages,
		Stream:   false,
		Format:   schema,
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	resp, err := http.Post(url, "application/json", bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("POST %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("ollama returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var res chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}
	return res.Message.Content, nil
}

// chatURL constructs the full /api/chat endpoint URL.
// Accepts endpoint with or without http:// prefix.
func chatURL(endpoint string) string {
	if strings.HasPrefix(endpoint, "http://") || strings.HasPrefix(endpoint, "https://") {
		return endpoint + "/api/chat"
	}
	return "http://" + endpoint + "/api/chat"
}
