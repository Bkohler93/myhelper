package wizard

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestCheckOllama verifies the Ollama reachability check using a fake HTTP server.
func TestCheckOllama(t *testing.T) {
	// Passing case: server returns 200 with "Ollama is running".
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "Ollama is running")
	}))
	defer srv.Close()
	ollamaBaseURL = srv.URL
	t.Cleanup(func() { ollamaBaseURL = "http://localhost:11434" })

	if !checkOllama() {
		t.Error("expected checkOllama() true with 200 server")
	}

	// Failing case: point at an unused port (connection refused).
	ollamaBaseURL = "http://127.0.0.1:19999"
	t.Cleanup(func() { ollamaBaseURL = "http://localhost:11434" })
	if checkOllama() {
		t.Error("expected checkOllama() false on connection refused")
	}
}

// TestInstallInstructions verifies that install instructions are non-empty and mention ollama.
func TestInstallInstructions(t *testing.T) {
	result := installInstructions()
	if result == "" {
		t.Error("expected non-empty installInstructions()")
	}
	if !strings.Contains(strings.ToLower(result), "ollama") {
		t.Errorf("expected installInstructions() to contain 'ollama', got: %q", result)
	}
}

// TestRecommendModel verifies the 4-tier model recommendation table.
func TestRecommendModel(t *testing.T) {
	cases := []struct {
		memMiB    int64
		wantModel string
	}{
		{30 * 1024, "qwen2.5-coder:14b"},
		{14 * 1024, "qwen2.5-coder:7b"},
		{7 * 1024, "llama3.2:3b"},
		{2 * 1024, "llama3.2:1b"},
	}
	for _, tc := range cases {
		name, _ := recommendModel(tc.memMiB)
		if name != tc.wantModel {
			t.Errorf("memMiB=%d: got %q, want %q", tc.memMiB, name, tc.wantModel)
		}
	}
}

// TestMergeConfig verifies non-destructive config merge with correct permissions.
func TestMergeConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	configPathOverride = path
	t.Cleanup(func() { configPathOverride = "" })

	// (a) Creates file when missing.
	if err := mergeHomeConfig(map[string]interface{}{"tavily_key": "key1"}); err != nil {
		t.Fatalf("mergeHomeConfig (create): %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile after create: %v", err)
	}
	if !strings.Contains(string(data), "key1") {
		t.Errorf("expected key1 in config after create, got: %s", data)
	}

	// (b) Preserves pre-existing key when merging a new key.
	if err := os.WriteFile(path, []byte(`{"endpoint":"myendpoint"}`), 0600); err != nil {
		t.Fatalf("WriteFile for pre-existing key test: %v", err)
	}
	if err := mergeHomeConfig(map[string]interface{}{"tavily_key": "key2"}); err != nil {
		t.Fatalf("mergeHomeConfig (preserve): %v", err)
	}
	data, err = os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile after merge: %v", err)
	}
	if !strings.Contains(string(data), "myendpoint") {
		t.Errorf("existing endpoint key was lost after merge; got: %s", data)
	}
	if !strings.Contains(string(data), "key2") {
		t.Errorf("tavily_key not written after merge; got: %s", data)
	}

	// (c) File permissions are 0600.
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("config perms: got %o, want 0600", info.Mode().Perm())
	}

	// (d) Existing tavily_key is overwritten when merging a new value.
	if err := mergeHomeConfig(map[string]interface{}{"tavily_key": "key3"}); err != nil {
		t.Fatalf("mergeHomeConfig (overwrite): %v", err)
	}
	data, err = os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile after overwrite: %v", err)
	}
	if !strings.Contains(string(data), "key3") {
		t.Errorf("expected overwritten tavily_key=key3 in config, got: %s", data)
	}
	if strings.Contains(string(data), "key2") {
		t.Errorf("old tavily_key key2 should have been replaced by key3, got: %s", data)
	}
}

// TestPullModel verifies NDJSON streaming progress output using a fake HTTP server.
func TestPullModel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-ndjson")
		fmt.Fprintln(w, `{"status":"pulling","total":1000,"completed":500}`)
		fmt.Fprintln(w, `{"status":"success"}`)
	}))
	defer srv.Close()
	ollamaBaseURL = srv.URL
	t.Cleanup(func() { ollamaBaseURL = "http://localhost:11434" })

	var out bytes.Buffer
	if err := pullModel("llama3.2:1b", &out); err != nil {
		t.Errorf("pullModel: %v", err)
	}
	if !strings.Contains(out.String(), "pulling") {
		t.Errorf("expected 'pulling' in output, got: %q", out.String())
	}
}

// TestPullModel_Error verifies that an error NDJSON line is returned as an error.
func TestPullModel_Error(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-ndjson")
		fmt.Fprintln(w, `{"error":"model not found"}`)
	}))
	defer srv.Close()
	ollamaBaseURL = srv.URL
	t.Cleanup(func() { ollamaBaseURL = "http://localhost:11434" })

	var out bytes.Buffer
	err := pullModel("nonexistent:model", &out)
	if err == nil {
		t.Error("expected error from pullModel on error NDJSON line, got nil")
	}
	if !strings.Contains(err.Error(), "model not found") {
		t.Errorf("expected 'model not found' in error, got: %v", err)
	}
}

// TestRun_NoOllama verifies that when Ollama is not reachable the wizard prints
// install instructions and does not reach the model-pull prompt.
func TestRun_NoOllama(t *testing.T) {
	// Point at an unused port so checkOllama() returns false.
	ollamaBaseURL = "http://127.0.0.1:19999"
	t.Cleanup(func() { ollamaBaseURL = "http://localhost:11434" })

	input := strings.NewReader("")
	var out bytes.Buffer
	if err := Run(input, &out); err != nil {
		t.Fatalf("Run: %v", err)
	}
	output := out.String()
	if !strings.Contains(strings.ToLower(output), "install ollama") &&
		!strings.Contains(strings.ToLower(output), "install") {
		t.Errorf("expected install instructions in output, got: %q", output)
	}
	// Must not reach the model-pull prompt.
	if strings.Contains(output, "Pull") && strings.Contains(output, "now?") {
		t.Errorf("wizard should not reach pull prompt when Ollama is not running; got: %q", output)
	}
}

// TestRun_SkipAll verifies that when Ollama is running and the user skips all prompts
// the wizard completes with "Setup complete" in output.
func TestRun_SkipAll(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "Ollama is running")
	}))
	defer srv.Close()
	ollamaBaseURL = srv.URL
	t.Cleanup(func() { ollamaBaseURL = "http://localhost:11434" })

	// Redirect config writes to temp dir.
	dir := t.TempDir()
	configPathOverride = filepath.Join(dir, "config.json")
	t.Cleanup(func() { configPathOverride = "" })

	// Stage 1.5: "" accepts default endpoint; "n" skips pull; "mymodel" satisfies fallback;
	// "" skips Tavily; "" skips SearXNG.
	input := strings.NewReader("\nn\nmymodel\n\n\n")
	var out bytes.Buffer
	if err := Run(input, &out); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !strings.Contains(out.String(), "Setup complete") {
		t.Errorf("expected 'Setup complete' in output, got: %q", out.String())
	}
}

func TestRun_EndpointPrompt_AcceptDefault(t *testing.T) {
	// Fake server returns 200 for all requests.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	ollamaBaseURL = srv.URL
	t.Cleanup(func() { ollamaBaseURL = "http://localhost:11434" })

	dir := t.TempDir()
	configPathOverride = filepath.Join(dir, "config.json")
	t.Cleanup(func() { configPathOverride = "" })

	// Stage 1.5: empty = accept default; Stage 3: "n"; fallback: "llama3.2:1b"; Tavily: ""; SearXNG: ""
	input := strings.NewReader("\nn\nllama3.2:1b\n\n\n")
	var out bytes.Buffer
	if err := Run(input, &out); err != nil {
		t.Fatalf("Run: %v", err)
	}
	data, _ := os.ReadFile(configPathOverride)
	if !strings.Contains(string(data), `"endpoint"`) {
		t.Errorf("expected endpoint in config, got: %s", data)
	}
	if !strings.Contains(string(data), srv.URL) {
		t.Errorf("expected endpoint value %q in config, got: %s", srv.URL, data)
	}
}

func TestRun_EndpointPrompt_CustomValue(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	ollamaBaseURL = srv.URL
	t.Cleanup(func() { ollamaBaseURL = "http://localhost:11434" })

	dir := t.TempDir()
	configPathOverride = filepath.Join(dir, "config.json")
	t.Cleanup(func() { configPathOverride = "" })

	// Stage 1.5: provide custom endpoint; Stage 3: "n"; fallback model; skip rest
	input := strings.NewReader("http://192.168.0.9:11434\nn\nllama3.2:1b\n\n\n")
	var out bytes.Buffer
	if err := Run(input, &out); err != nil {
		t.Fatalf("Run: %v", err)
	}
	data, _ := os.ReadFile(configPathOverride)
	if !strings.Contains(string(data), "192.168.0.9:11434") {
		t.Errorf("expected custom endpoint in config, got: %s", data)
	}
}

func TestRun_EndpointPrompt_InvalidThenValid(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	ollamaBaseURL = srv.URL
	t.Cleanup(func() { ollamaBaseURL = "http://localhost:11434" })

	dir := t.TempDir()
	configPathOverride = filepath.Join(dir, "config.json")
	t.Cleanup(func() { configPathOverride = "" })

	// Stage 1.5 attempt 1: invalid; attempt 2: valid; then skip model pull with name; skip rest
	input := strings.NewReader("not-a-url\nhttp://192.168.0.9:11434\nn\nllama3.2:1b\n\n\n")
	var out bytes.Buffer
	if err := Run(input, &out); err != nil {
		t.Fatalf("Run: %v", err)
	}
	data, _ := os.ReadFile(configPathOverride)
	if !strings.Contains(string(data), "192.168.0.9") {
		t.Errorf("expected valid endpoint written after retry, got: %s", data)
	}
	if strings.Contains(string(data), "not-a-url") {
		t.Errorf("invalid endpoint should not be written to config, got: %s", data)
	}
	// Output should contain the re-prompt hint about http://
	if !strings.Contains(out.String(), "http://") {
		t.Errorf("expected re-prompt hint in output, got: %q", out.String())
	}
}

func TestRun_SkipModel_FallbackWritesModel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	ollamaBaseURL = srv.URL
	t.Cleanup(func() { ollamaBaseURL = "http://localhost:11434" })

	dir := t.TempDir()
	configPathOverride = filepath.Join(dir, "config.json")
	t.Cleanup(func() { configPathOverride = "" })

	// Accept default endpoint, skip pull, provide model name, skip optional fields
	input := strings.NewReader("\nn\nllama3.2:1b\n\n\n")
	var out bytes.Buffer
	if err := Run(input, &out); err != nil {
		t.Fatalf("Run: %v", err)
	}
	data, _ := os.ReadFile(configPathOverride)
	if !strings.Contains(string(data), `"model"`) {
		t.Errorf("expected model in config, got: %s", data)
	}
	if !strings.Contains(string(data), "llama3.2:1b") {
		t.Errorf("expected llama3.2:1b as model value, got: %s", data)
	}
}

func TestRun_SkipModel_EmptyThenProvided(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	ollamaBaseURL = srv.URL
	t.Cleanup(func() { ollamaBaseURL = "http://localhost:11434" })

	dir := t.TempDir()
	configPathOverride = filepath.Join(dir, "config.json")
	t.Cleanup(func() { configPathOverride = "" })

	// Accept endpoint default, skip pull, first fallback empty, second fallback valid
	input := strings.NewReader("\nn\n\ngemma3:4b\n\n\n")
	var out bytes.Buffer
	if err := Run(input, &out); err != nil {
		t.Fatalf("Run: %v", err)
	}
	data, _ := os.ReadFile(configPathOverride)
	if !strings.Contains(string(data), "gemma3:4b") {
		t.Errorf("expected gemma3:4b in config after retry, got: %s", data)
	}
}

func TestRun_SkipModel_EmptyTwice(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	ollamaBaseURL = srv.URL
	t.Cleanup(func() { ollamaBaseURL = "http://localhost:11434" })

	dir := t.TempDir()
	configPathOverride = filepath.Join(dir, "config.json")
	t.Cleanup(func() { configPathOverride = "" })

	// Accept endpoint default, skip pull, both fallback answers empty
	input := strings.NewReader("\nn\n\n\n")
	var out bytes.Buffer
	err := Run(input, &out)
	if err == nil {
		t.Error("expected error when model name entered twice as empty, got nil")
	}
	// Config must not contain a model key
	data, readErr := os.ReadFile(configPathOverride)
	if readErr == nil && strings.Contains(string(data), `"model"`) {
		t.Errorf("model key must not be written when no name provided, got: %s", data)
	}
}

func TestRun_PullFail_FallbackWritesModel(t *testing.T) {
	// Server: 200 on /, 500 on /api/pull.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/pull" {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, "internal error")
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	ollamaBaseURL = srv.URL
	t.Cleanup(func() { ollamaBaseURL = "http://localhost:11434" })

	dir := t.TempDir()
	configPathOverride = filepath.Join(dir, "config.json")
	t.Cleanup(func() { configPathOverride = "" })

	// Accept endpoint default, accept pull prompt (y), pull fails, enter fallback model name
	input := strings.NewReader("\n\nllama3.2:1b\n\n\n")
	var out bytes.Buffer
	if err := Run(input, &out); err != nil {
		t.Fatalf("Run: %v", err)
	}
	data, _ := os.ReadFile(configPathOverride)
	if !strings.Contains(string(data), "llama3.2:1b") {
		t.Errorf("expected fallback model in config after pull failure, got: %s", data)
	}
}
