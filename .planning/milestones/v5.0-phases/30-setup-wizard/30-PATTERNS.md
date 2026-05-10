# Phase 30: Setup Wizard - Pattern Map

**Mapped:** 2026-05-10
**Files analyzed:** 3 new/modified files
**Analogs found:** 3 / 3

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `cmd/setup.go` | command | request-response | `cmd/inspect.go` | exact |
| `internal/wizard/wizard.go` | service | request-response + file-I/O | `cmd/helpers.go` + `internal/search/search.go` | role-match |
| `internal/wizard/wizard_test.go` | test | — | `cmd/helpers_test.go` | exact |

## Pattern Assignments

### `cmd/setup.go` (command, request-response)

**Analog:** `cmd/inspect.go`

**Imports pattern** (`cmd/inspect.go` lines 1-12):
```go
package cmd

import (
	"fmt"
	"strings"

	"github.com/bkohler93/myhelper/internal/config"
	// ... other internal imports ...
	"github.com/spf13/cobra"
)
```

For `cmd/setup.go`, imports will be minimal — only `os` and the new `internal/wizard` package:
```go
package cmd

import (
	"os"

	"github.com/bkohler93/myhelper/internal/wizard"
	"github.com/spf13/cobra"
)
```

**Cobra registration pattern** (`cmd/inspect.go` lines 14-23):
```go
var inspectCmd = &cobra.Command{
	Use:   "inspect <query>",
	Short: "Inspect the web search pipeline for a query (diagnostic dry-run)",
	Args:  cobra.ExactArgs(1),
	RunE:  runInspect,
}

func init() {
	rootCmd.AddCommand(inspectCmd)
}
```

Apply the same pattern for setup — `cobra.NoArgs` instead of `ExactArgs(1)`:
```go
var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Interactive first-run wizard: check Ollama, detect hardware, configure keys",
	Args:  cobra.NoArgs,
	RunE:  runSetup,
}

func init() {
	rootCmd.AddCommand(setupCmd)
}
```

**RunE pattern** (`cmd/inspect.go` lines 25-34):
```go
func runInspect(cmd *cobra.Command, args []string) error {
	query := args[0]
	cfg := config.Load()
	// ... call internal packages ...
	return nil
}
```

For setup, the `RunE` is a thin driver that delegates all logic to `wizard.Run`:
```go
func runSetup(cmd *cobra.Command, args []string) error {
	return wizard.Run(os.Stdin, os.Stdout)
}
```

---

### `internal/wizard/wizard.go` (service, request-response + file-I/O)

**Analogs:**
- `cmd/helpers.go` — bufio.Scanner stdin prompt pattern, io.Reader/io.Writer injection
- `internal/search/search.go` — HTTP client, JSON decode, config file path helpers
- `internal/ollama/client.go` — NDJSON streaming response parse, net/http POST with body

**Imports pattern** (`cmd/helpers.go` lines 1-18 and `internal/search/search.go` lines 1-14):
```go
package wizard

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)
```
No internal package imports — the wizard is self-contained to avoid the `search.Config.MarshalJSON` redaction issue.

**io.Reader/io.Writer injection pattern** (`cmd/helpers.go` lines 261-275):
```go
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
	// ...
}
```

The wizard's `Run` function signature injects `io.Reader` and `io.Writer` for testability. Use a single `*bufio.Reader` (not `bufio.Scanner`) threaded through all steps to avoid the multi-Scanner buffering pitfall documented in RESEARCH.md:
```go
// Run executes the setup wizard, reading from r and writing to w.
// Pass os.Stdin / os.Stdout in production; use bytes.Buffer / strings.Reader in tests.
func Run(r io.Reader, w io.Writer) error {
	br := bufio.NewReader(r) // single reader threaded through all steps — never create a second one
	// stage 1, 2, 3, 4, 5 all receive br
}
```

**HTTP GET reachability check pattern** (`internal/search/search.go` lines 162-199):
```go
resp, err := httpClient.Get(reqURL)
if err != nil {
	return nil, fmt.Errorf("GET %s: %w", reqURL, err)
}
defer resp.Body.Close()

if resp.StatusCode != http.StatusOK {
	body, _ := io.ReadAll(resp.Body)
	return nil, fmt.Errorf("searxng returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
}
```

For the Ollama reachability check, apply the same pattern:
```go
func checkOllama() bool {
	resp, err := http.Get("http://localhost:11434/")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}
```

**HTTP POST with JSON body + NDJSON streaming** (`internal/ollama/client.go` lines 1-19, and the NDJSON streaming logic):
The wizard's pull function posts to `/api/pull` and reads NDJSON lines. Follow the same `bufio.NewScanner` + `json.Unmarshal` line-by-line pattern used in `ollama.StreamChat`. Each NDJSON line is unmarshalled into a typed struct:
```go
type pullRequest struct {
	Name   string `json:"name"`
	Stream bool   `json:"stream"`
}

type pullProgress struct {
	Status    string `json:"status"`
	Digest    string `json:"digest,omitempty"`
	Total     int64  `json:"total,omitempty"`
	Completed int64  `json:"completed,omitempty"`
	Error     string `json:"error,omitempty"`
}

func pullModel(name string, w io.Writer) error {
	body, _ := json.Marshal(pullRequest{Name: name, Stream: true})
	resp, err := http.Post("http://localhost:11434/api/pull", "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("pull request: %w", err)
	}
	defer resp.Body.Close()

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
			fmt.Fprintf(w, "\r%s %.0f%%", p.Status, float64(p.Completed)/float64(p.Total)*100)
		} else {
			fmt.Fprintf(w, "\r%s", p.Status)
		}
		if p.Status == "success" {
			fmt.Fprintln(w)
			break
		}
	}
	return sc.Err()
}
```

**Config file path helpers** (`internal/config/config.go` lines 76-86 and `internal/search/search.go` lines 136-144):
```go
// Both packages define homeConfigPath() identically — copy this pattern:
func homeConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "myhelper", "config.json")
}
```

**Config merge-write pattern** (`internal/config/config.go` lines 88-101 for the read pattern; merge-write is new):
The existing `loadFile` in `internal/config` unmarshals into a typed struct — the wizard MUST NOT use that. Instead use `map[string]interface{}` to avoid dropping unknown keys. Pattern derived from the research's verified example:
```go
func mergeHomeConfig(updates map[string]interface{}) error {
	path := homeConfigPath()
	if path == "" {
		return fmt.Errorf("could not resolve home directory")
	}
	existing := map[string]interface{}{}
	if data, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(data, &existing) // corrupt/empty: start fresh
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
	return os.WriteFile(path, data, 0600) // 0600: config holds API key
}
```

**os/exec subprocess pattern** (no direct codebase analog — stdlib only):
```go
// Hardware detection — macOS
out, err := exec.Command("system_profiler", "SPHardwareDataType").Output()
// Only use stdout (out); err indicates command not found or non-zero exit

// Hardware detection — Linux Nvidia
out, err := exec.Command("nvidia-smi",
	"--query-gpu=memory.total",
	"--format=csv,noheader,nounits").Output()
// On failure, fall through to /proc/meminfo

// /proc/meminfo fallback
data, err := os.ReadFile("/proc/meminfo")
```

**Error handling pattern** (`cmd/inspect.go` lines 47-50 and throughout):
```go
// Project convention: return errors from RunE, use fmt.Fprintf(w, ...) for user-facing output.
// Wizard stages should return error only for fatal failures; recoverable stages print a message
// and continue (graceful degradation per RESEARCH.md architecture).
if err != nil {
	fmt.Fprintf(w, "Warning: could not detect hardware: %v\n", err)
	// continue with fallback, do not return err
}
```

---

### `internal/wizard/wizard_test.go` (test)

**Analog:** `cmd/helpers_test.go`

**Test file structure** (`cmd/helpers_test.go` lines 1-47):
```go
package wizard

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)
```

**io.Reader injection for stdin simulation** (`cmd/helpers_test.go` lines 35-47):
```go
// In helpers_test.go, stdinReader is a package-level var replaced per test.
// In wizard_test.go, pass a *strings.Reader or *bytes.Buffer to Run() directly:
func TestRun_NoOllama(t *testing.T) {
	input := strings.NewReader("n\n\n\n") // no pull, skip keys
	var out bytes.Buffer
	// inject a fake HTTP server so checkOllama returns false
	// ...
	err := Run(input, &out)
	// assert output contains install instructions
}
```

**Table-driven unit tests** (pattern from `cmd/helpers_test.go` lines 50-156):
```go
func TestRecommendModel(t *testing.T) {
	cases := []struct {
		memMiB int64
		want   string
	}{
		{memMiB: 30 * 1024, want: "qwen2.5-coder:14b"},
		{memMiB: 14 * 1024, want: "qwen2.5-coder:7b"},
		{memMiB: 7 * 1024,  want: "llama3.2:3b"},
		{memMiB: 2 * 1024,  want: "llama3.2:1b"},
	}
	for _, tc := range cases {
		got, _ := recommendModel(tc.memMiB)
		if got != tc.want {
			t.Errorf("memMiB=%d: got %q, want %q", tc.memMiB, got, tc.want)
		}
	}
}
```

**Config file temp-dir pattern** (Go stdlib test convention; no direct codebase analog):
```go
func TestMergeHomeConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	// write initial config, call mergeHomeConfig variant with injected path,
	// read back and assert keys preserved + new keys added
}
```

**httptest.Server for Ollama check** (stdlib pattern used for HTTP-dependent unit tests):
```go
func TestCheckOllama(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "Ollama is running")
	}))
	defer srv.Close()
	// pass srv.URL to checkOllama or temporarily override the endpoint constant
}
```

---

## Shared Patterns

### Cobra subcommand registration
**Source:** `cmd/inspect.go` lines 14-23
**Apply to:** `cmd/setup.go`
```go
var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "...",
	Args:  cobra.NoArgs,
	RunE:  runSetup,
}

func init() {
	rootCmd.AddCommand(setupCmd)
}
```

### Config file path resolution
**Source:** `internal/config/config.go` lines 80-86 (duplicated verbatim in `internal/search/search.go` lines 139-144)
**Apply to:** `internal/wizard/wizard.go`
```go
func homeConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "myhelper", "config.json")
}
```

### Non-destructive config merge (map-based, not typed struct)
**Source:** Pattern derived from `internal/config/config.go` read pattern + `internal/search/search.go` (RESEARCH.md verified)
**Apply to:** `internal/wizard/wizard.go` — `mergeHomeConfig` function
**Critical:** Do NOT unmarshal into `config.Config` or `search.Config` — use `map[string]interface{}` to preserve all keys. Do NOT use `search.Config.MarshalJSON` — it redacts `TavilyKey`.

### Single bufio.Reader threaded through all prompt steps
**Source:** `cmd/helpers.go` lines 260-275 (uses `stdinReader` package var; wizard uses injected `io.Reader` instead)
**Apply to:** `internal/wizard/wizard.go` — `Run` function
**Critical:** Create exactly ONE `*bufio.Reader` wrapping the passed `io.Reader` at the top of `Run`; pass it to all sub-steps. Never create a second `bufio.Scanner` or `bufio.Reader` over the same reader.

### NDJSON line streaming
**Source:** `internal/ollama/client.go` (StreamChat function — NDJSON decode loop)
**Apply to:** `internal/wizard/wizard.go` — `pullModel` function
```go
sc := bufio.NewScanner(resp.Body)
for sc.Scan() {
	var p pullProgress
	_ = json.Unmarshal(sc.Bytes(), &p)
	// process p
}
```

### File permissions for API key storage
**Source:** Security requirement from RESEARCH.md; no codebase analog (existing config uses default permissions)
**Apply to:** `internal/wizard/wizard.go` — `mergeHomeConfig` function
```go
os.WriteFile(path, data, 0600) // owner read/write only — never 0644
```

### Error return from RunE
**Source:** `cmd/inspect.go` lines 25-34 (all RunE functions return `error`)
**Apply to:** `cmd/setup.go` — `runSetup` function
```go
func runSetup(cmd *cobra.Command, args []string) error {
	return wizard.Run(os.Stdin, os.Stdout)
}
```

## Exact JSON Tag Reference

**Source:** `internal/search/search.go` lines 22-28 (VERIFIED)

The wizard writes these exact keys to `~/.config/myhelper/config.json`:

| JSON key | Go struct field | Written when |
|----------|----------------|--------------|
| `tavily_key` | `search.Config.TavilyKey` | Tavily key is non-empty |
| `search_provider` | `search.Config.Provider` | Tavily key is non-empty (set to `"tavily"`) |
| `search_endpoint` | `search.Config.Endpoint` | SearXNG endpoint is non-empty |
| `model` | `config.Config.Model` | Model pull is confirmed (Assumption A3 — recommended per RESEARCH.md) |

```go
// search.Config JSON tags (from internal/search/search.go lines 23-27):
Endpoint       string `json:"search_endpoint"`
Provider       string `json:"search_provider"`
TavilyKey      string `json:"tavily_key"`
TavilyEndpoint string `json:"tavily_endpoint"`
```

## No Analog Found

| File / Pattern | Role | Data Flow | Reason |
|----------------|------|-----------|--------|
| Hardware detection (`os/exec` + `/proc/meminfo`) | utility | file-I/O | No subprocess-based hardware detection exists in the codebase |
| Model recommendation table | utility | transform | Pure logic function; no analog in codebase |
| `httptest.Server` in wizard tests | test | request-response | Existing tests mock via package-level vars, not HTTP servers; wizard needs httptest for Ollama check |

For these, use the patterns from RESEARCH.md Code Examples section directly.

## Metadata

**Analog search scope:** `cmd/`, `internal/config/`, `internal/search/`, `internal/ollama/`
**Files scanned:** 8 (inspect.go, helpers.go, helpers_test.go, root.go, search.go, search.go(internal), config.go, client.go)
**Pattern extraction date:** 2026-05-10
