package wizard

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Package-level vars for test injection (set in tests only; never modify in production code).
var ollamaBaseURL = "http://localhost:11434"

// configPathOverride, if non-empty, is returned by homeConfigPath() instead of the real path.
// Set this in tests to redirect config writes to a t.TempDir() path.
var configPathOverride = ""

// ollamaHTTPClient is used for quick reachability checks (short timeout).
// WR-01: use a bounded client instead of the default (no-timeout) http.Get.
var ollamaHTTPClient = &http.Client{Timeout: 5 * time.Second}

// pullHTTPClient is used for model pulls — allow up to 5 minutes for large downloads.
// WR-01: use a bounded client instead of the default (no-timeout) http.Post.
var pullHTTPClient = &http.Client{Timeout: 5 * time.Minute}

// pullRequest is the JSON body sent to the Ollama /api/pull endpoint.
type pullRequest struct {
	Name   string `json:"name"`
	Stream bool   `json:"stream"`
}

// pullProgress is a single NDJSON line from the Ollama /api/pull streaming response.
type pullProgress struct {
	Status    string `json:"status"`
	Digest    string `json:"digest,omitempty"`
	Total     int64  `json:"total,omitempty"`
	Completed int64  `json:"completed,omitempty"`
	Error     string `json:"error,omitempty"`
}

// tagsResponse is the JSON response from the Ollama /api/tags endpoint.
type tagsResponse struct {
	Models []struct {
		Name string `json:"name"`
	} `json:"models"`
}

// listModels returns the names of locally available Ollama models, sorted alphabetically.
// Returns an empty slice (not an error) when Ollama is reachable but has no models.
func listModels(endpoint string) ([]string, error) {
	resp, err := ollamaHTTPClient.Get(endpoint + "/api/tags")
	if err != nil {
		return nil, fmt.Errorf("list models: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ollama /api/tags returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var tags tagsResponse
	if err := json.NewDecoder(resp.Body).Decode(&tags); err != nil {
		return nil, fmt.Errorf("decode /api/tags: %w", err)
	}
	names := make([]string, 0, len(tags.Models))
	for _, m := range tags.Models {
		names = append(names, m.Name)
	}
	sort.Strings(names)
	return names, nil
}

// Run executes the interactive setup wizard. It reads from r and writes to w.
// Pass os.Stdin / os.Stdout in production; use *strings.Reader / *bytes.Buffer in tests.
//
// Wizard stages:
//  1. Prompt for Ollama endpoint (so the reachability check uses the correct URL).
//  2. Check Ollama reachability at the confirmed endpoint; print install instructions and return if not running.
//  3. List locally available Ollama models; let user pick one or pull a new one by name.
//  4. Prompt for Tavily API key (optional).
//  5. Prompt for SearXNG endpoint (optional).
func Run(r io.Reader, w io.Writer) error {
	// Single bufio.Reader threaded through all steps — never create a second one over r.
	br := bufio.NewReader(r)

	// Declare line at function scope — used across all prompt stages.
	var line string

	// Stage 1: Ollama endpoint prompt — required field, loop until valid.
	// Must run BEFORE the reachability check so checkOllama() uses the user's endpoint.
	var endpointValue string
	for {
		fmt.Fprintf(w, "Ollama endpoint [%s]: ", ollamaBaseURL)
		line, _ = br.ReadString('\n')
		line = strings.TrimSpace(line)
		if line == "" {
			line = ollamaBaseURL // accept the pre-filled default
		}
		u, parseErr := url.Parse(line)
		if parseErr != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
			fmt.Fprintf(w, "Endpoint must be a valid http:// or https:// URL with a host. Try again.\n")
			continue
		}
		endpointValue = line
		break
	}
	// Update the package-level URL so checkOllama() targets the user's endpoint.
	ollamaBaseURL = endpointValue
	if err := mergeHomeConfig(map[string]interface{}{"endpoint": endpointValue}); err != nil {
		fmt.Fprintf(w, "Warning: could not save endpoint: %v\n", err)
	}
	fmt.Fprintln(w)

	// Stage 2: Ollama reachability check against the confirmed endpoint.
	if !checkOllama() {
		fmt.Fprintf(w, "Ollama is not running at %s.\n\nInstall Ollama:\n  %s\n\nAfter installing, start Ollama and run `myhelper setup` again.\n", endpointValue, installInstructions())
		return nil
	}
	fmt.Fprintf(w, "Ollama is running.\n\n")

	// Stage 3: Model selection — list available models or pull a new one.
	models, listErr := listModels(endpointValue)
	if listErr != nil {
		fmt.Fprintf(w, "Warning: could not list models: %v\n", listErr)
	}

	chosenModel := ""
	if len(models) > 0 {
		fmt.Fprintf(w, "Available models:\n")
		for i, m := range models {
			fmt.Fprintf(w, "  [%d] %s\n", i+1, m)
		}
		fmt.Fprintf(w, "  [0] Pull a new model\n\n")
		fmt.Fprintf(w, "Select a model [1]: ")
		line, _ = br.ReadString('\n')
		line = strings.TrimSpace(line)
		if line == "" {
			line = "1"
		}
		if idx, err := strconv.Atoi(line); err == nil && idx >= 1 && idx <= len(models) {
			chosenModel = models[idx-1]
			if err := mergeHomeConfig(map[string]interface{}{"model": chosenModel}); err != nil {
				fmt.Fprintf(w, "Warning: could not save model: %v\n", err)
			} else {
				fmt.Fprintf(w, "Default model set to: %s\n", chosenModel)
			}
		}
		// idx == 0 or invalid → fall through to pull flow below
	}

	if chosenModel == "" {
		// No models available or user chose to pull a new one.
		fmt.Fprintf(w, "Enter model name to pull (e.g. llama3.2:3b): ")
		line, _ = br.ReadString('\n')
		modelName := strings.TrimSpace(line)
		if modelName == "" {
			fmt.Fprintf(w, "Model name cannot be empty. Enter a model name: ")
			line, _ = br.ReadString('\n')
			modelName = strings.TrimSpace(line)
		}
		if modelName == "" {
			return fmt.Errorf("no model name provided — setup incomplete")
		}
		fmt.Fprintf(w, "Pulling %s...\n", modelName)
		if err := pullModel(modelName, endpointValue, w); err != nil {
			fmt.Fprintf(w, "Pull failed: %v\n", err)
			fmt.Fprintf(w, "Saving model name anyway. If the pull failed, run 'ollama pull %s' manually.\n", modelName)
		} else {
			fmt.Fprintf(w, "Model ready.\n")
		}
		if err := mergeHomeConfig(map[string]interface{}{"model": modelName}); err != nil {
			fmt.Fprintf(w, "Warning: could not save model: %v\n", err)
		} else {
			fmt.Fprintf(w, "Default model set to: %s\n", modelName)
		}
	}
	fmt.Fprintln(w)

	// Stage 4: Tavily API key (optional).
	fmt.Fprintf(w, "Tavily API key (enter to skip): ")
	line, _ = br.ReadString('\n')
	line = strings.TrimSpace(line)
	if line != "" {
		if err := mergeHomeConfig(map[string]interface{}{"tavily_key": line, "search_provider": "tavily"}); err != nil {
			fmt.Fprintf(w, "Warning: could not save Tavily key: %v\n", err)
		} else {
			fmt.Fprintf(w, "Tavily key saved.\n")
		}
	}
	fmt.Fprintln(w)

	// Stage 5: SearXNG endpoint (optional).
	fmt.Fprintf(w, "SearXNG endpoint (enter to skip): ")
	line, _ = br.ReadString('\n')
	line = strings.TrimSpace(line)
	if line != "" {
		// T-30-03: validate that endpoint is a well-formed http/https URL with a non-empty host.
		su, parseErr := url.Parse(line)
		if parseErr != nil || su.Host == "" || (su.Scheme != "http" && su.Scheme != "https") {
			fmt.Fprintf(w, "Warning: SearXNG endpoint must be a valid http:// or https:// URL with a host; skipping.\n")
		} else {
			if err := mergeHomeConfig(map[string]interface{}{"search_endpoint": line}); err != nil {
				fmt.Fprintf(w, "Warning: could not save SearXNG endpoint: %v\n", err)
			} else {
				fmt.Fprintf(w, "SearXNG endpoint saved.\n")
			}
		}
	}
	fmt.Fprintln(w)

	fmt.Fprintf(w, "Setup complete. Run: myhelper <question>\n")
	return nil
}

// checkOllama reports whether the Ollama service is reachable.
// Uses ollamaBaseURL (overridable in tests); defaults to http://localhost:11434.
// Returns true on HTTP 200; returns false on any error or non-200 status.
func checkOllama() bool {
	// WR-01: use ollamaHTTPClient (5s timeout) instead of the default no-timeout http.Get.
	resp, err := ollamaHTTPClient.Get(ollamaBaseURL + "/")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// installInstructions returns the platform-specific command to install Ollama.
func installInstructions() string {
	switch runtime.GOOS {
	case "darwin":
		return "brew install ollama"
	case "linux":
		return "curl -fsSL https://ollama.com/install.sh | sh"
	default:
		return "Visit https://ollama.com/download"
	}
}

// detectMemoryMiB returns the available GPU VRAM (Linux/Nvidia) or system RAM in MiB.
// Falls back gracefully through multiple detection strategies; returns 0 if undetectable.
func detectMemoryMiB() int64 {
	switch runtime.GOOS {
	case "linux":
		// Try nvidia-smi for GPU VRAM first.
		out, err := exec.Command("nvidia-smi", "--query-gpu=memory.total", "--format=csv,noheader,nounits").Output()
		if err == nil && len(strings.TrimSpace(string(out))) > 0 {
			// WR-02: multi-GPU systems emit one line per GPU; use the first line only.
			lines := strings.Split(strings.TrimSpace(string(out)), "\n")
			if n, err := strconv.ParseInt(strings.TrimSpace(lines[0]), 10, 64); err == nil {
				return n
			}
		}
		// Fallback: /proc/meminfo RAM.
		data, err := os.ReadFile("/proc/meminfo")
		if err == nil {
			for _, line := range strings.Split(string(data), "\n") {
				if strings.HasPrefix(line, "MemTotal:") {
					fields := strings.Fields(line)
					if len(fields) >= 2 {
						if kb, err := strconv.ParseInt(fields[1], 10, 64); err == nil {
							return kb / 1024
						}
					}
				}
			}
		}
		return 0

	case "darwin":
		// system_profiler: "      Memory: 32 GB"
		out, err := exec.Command("system_profiler", "SPHardwareDataType").Output()
		if err == nil {
			for _, line := range strings.Split(string(out), "\n") {
				if strings.Contains(line, "Memory:") {
					parts := strings.Fields(strings.TrimSpace(line))
					// parts: ["Memory:", "32", "GB"]
					if len(parts) >= 3 {
						if n, err := strconv.ParseInt(parts[1], 10, 64); err == nil {
							unit := strings.ToUpper(parts[2])
							if unit == "GB" {
								return n * 1024
							}
							if unit == "MB" {
								return n
							}
						}
					}
				}
			}
		}
		// Fallback: sysctl hw.memsize (returns bytes).
		out, err = exec.Command("sysctl", "-n", "hw.memsize").Output()
		if err == nil {
			if n, err := strconv.ParseInt(strings.TrimSpace(string(out)), 10, 64); err == nil {
				return n / 1024 / 1024
			}
		}
		return 0

	default:
		return 0
	}
}

// recommendModel returns the name and required MiB for the model best suited to memMiB.
func recommendModel(memMiB int64) (name string, requiredMiB int64) {
	switch {
	case memMiB >= 24*1024:
		return "qwen2.5-coder:14b", 10 * 1024
	case memMiB >= 12*1024:
		return "qwen2.5-coder:7b", 6 * 1024
	case memMiB >= 6*1024:
		return "llama3.2:3b", 3 * 1024
	default:
		return "llama3.2:1b", 1400
	}
}

// pullModel posts to the Ollama /api/pull endpoint and streams progress to w.
// It uses a bufio.Scanner over resp.Body (a separate io.Reader from the wizard's stdin).
// endpoint is the base URL of the Ollama server (e.g. "http://localhost:11434").
func pullModel(name string, endpoint string, w io.Writer) error {
	body, _ := json.Marshal(pullRequest{Name: name, Stream: true})
	// WR-01: use pullHTTPClient (5m timeout) instead of the default no-timeout http.Post.
	resp, err := pullHTTPClient.Post(endpoint+"/api/pull", "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("pull request: %w", err)
	}
	defer resp.Body.Close()

	// CR-01: check HTTP status before consuming the NDJSON body.
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("ollama pull returned %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}

	// CR-02: track whether a "success" status line was received.
	var succeeded bool
	sc := bufio.NewScanner(resp.Body)
	for sc.Scan() {
		var p pullProgress
		if err := json.Unmarshal(sc.Bytes(), &p); err != nil {
			continue
		}
		if p.Error != "" {
			return fmt.Errorf("pull error: %s", p.Error)
		}
		if p.Total > 0 {
			fmt.Fprintf(w, "\r%s %.0f%%  ", p.Status, float64(p.Completed)/float64(p.Total)*100)
		} else {
			fmt.Fprintf(w, "\r%s       ", p.Status)
		}
		if p.Status == "success" {
			fmt.Fprintln(w)
			succeeded = true
			break
		}
	}
	if err := sc.Err(); err != nil {
		return err
	}
	// CR-02: if the stream ended without a "success" line, the download may be incomplete.
	if !succeeded {
		return fmt.Errorf("model pull ended without confirmation — download may be incomplete")
	}
	return nil
}

// mergeHomeConfig reads the existing home config file, merges updates into it (preserving
// all unrelated keys), then writes it back with 0600 permissions.
// Tolerates a missing or corrupt config file (treated as empty).
func mergeHomeConfig(updates map[string]interface{}) error {
	path := homeConfigPath()
	if path == "" {
		return fmt.Errorf("could not resolve home directory")
	}
	existing := map[string]interface{}{}
	if data, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(data, &existing)
	}
	for k, v := range updates {
		existing[k] = v
	}
	data, err := json.MarshalIndent(existing, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

// homeConfigPath returns the canonical path to the global myhelper config file.
// If configPathOverride is non-empty, returns that path instead (used in tests).
// Mirrors the same function in internal/config/config.go and internal/search/search.go.
func homeConfigPath() string {
	if configPathOverride != "" {
		return configPathOverride
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "myhelper", "config.json")
}
