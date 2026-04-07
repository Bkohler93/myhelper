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

	"github.com/bkohler93/my-helper/internal/config"
)

type generateRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

type generateResponse struct {
	Response string `json:"response"`
	Done     bool   `json:"done"`
}

// StreamPrompt sends a single-shot prompt to the Ollama /api/generate endpoint
// and streams each response token to stdout as it arrives.
func StreamPrompt(cfg config.Config, prompt string) error {
	url := buildURL(cfg.Endpoint)

	reqBody := generateRequest{
		Model:  cfg.Model,
		Prompt: prompt,
		Stream: true,
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	resp, err := http.Post(url, "application/json", bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("POST %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("ollama returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var chunk generateResponse
		if err := json.Unmarshal(line, &chunk); err != nil {
			return fmt.Errorf("unmarshal chunk: %w", err)
		}
		fmt.Fprint(os.Stdout, chunk.Response)
		if chunk.Done {
			break
		}
	}
	fmt.Fprintln(os.Stdout) // trailing newline after stream completes

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("reading stream: %w", err)
	}

	return nil
}

// buildURL constructs the full generate endpoint URL.
// Accepts endpoint with or without http:// prefix.
func buildURL(endpoint string) string {
	if strings.HasPrefix(endpoint, "http://") || strings.HasPrefix(endpoint, "https://") {
		return endpoint + "/api/generate"
	}
	return "http://" + endpoint + "/api/generate"
}
