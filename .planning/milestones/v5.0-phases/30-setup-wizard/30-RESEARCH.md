# Phase 30: Setup Wizard - Research

**Researched:** 2026-05-10
**Domain:** Go CLI wizard, Ollama API, hardware detection, config file merge
**Confidence:** HIGH

## Summary

Phase 30 implements `myhelper setup` — a linear, interactive terminal wizard that guides a
new user from zero to a working chat session. The wizard is entirely self-contained in a new
`cmd/setup.go` file registered as a cobra subcommand, using only stdlib (bufio.Scanner,
os/exec, encoding/json, net/http, runtime) and the existing internal packages. No new
dependencies are needed.

The wizard flow has five sequential stages: (1) check Ollama reachability, (2) detect
hardware and recommend a model, (3) optionally pull the model via Ollama's streaming pull
API, (4) prompt for Tavily API key, (5) optionally prompt for SearXNG endpoint. Stages 4 and
5 write to `~/.config/myhelper/config.json` using a read-merge-write pattern over a
`map[string]interface{}` to avoid overwriting unrelated keys.

The config merge approach is the most architecturally significant decision: because
`config.Config` (JSON key `endpoint`, `model`, `token_threshold`) and `search.Config` (JSON
keys `search_endpoint`, `search_provider`, `tavily_key`) share a single config file, the
wizard must never unmarshal into a typed struct and re-marshal — doing so would drop unknown
keys. The safe pattern is `json.Unmarshal` into `map[string]interface{}`, update target keys,
then `json.MarshalIndent`.

**Primary recommendation:** Implement as `cmd/setup.go` + `internal/wizard/wizard.go` (pure
logic, no I/O) with a thin `cmd/setup.go` driver. Put testable logic in `internal/wizard/`.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
All implementation choices are at Claude's discretion — discuss phase was skipped per user
setting.

### Claude's Discretion
All implementation choices including:
- File/package structure
- Exact wizard step ordering
- Model recommendation table thresholds
- Whether to write the `model` field to config after pull
- Error handling and graceful degradation

### Deferred Ideas (OUT OF SCOPE)
None — discuss phase skipped.

**Hard constraints from CONTEXT.md:**
- Use simple stdin prompts (bufio.Scanner or fmt.Scan), NOT a TUI library
- Config writes MUST use exact JSON tags: `tavily_key`, `search_provider`, `search_endpoint`
- Config write MUST be non-destructive: merge into existing config, not overwrite whole file
- Minimal dependencies (project philosophy)
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| SETUP-01 | `myhelper setup` checks whether Ollama is installed and reachable on port 11434 | HTTP GET to `http://localhost:11434/` returns "Ollama is running"; `net/http` HEAD/GET is sufficient |
| SETUP-02 | Shows platform-specific Ollama install instructions when Ollama not detected | `runtime.GOOS == "darwin"` → `brew install ollama`; `linux` → `curl -fsSL https://ollama.com/install.sh \| sh` |
| SETUP-03 | Shows recommended model size based on detected GPU VRAM (nvidia-smi) or RAM | macOS: `system_profiler SPHardwareDataType` parses "Memory: 32 GB"; Linux: `nvidia-smi --query-gpu=memory.total --format=csv,noheader,nounits` then `/proc/meminfo` fallback |
| SETUP-04 | User can confirm to have wizard run `ollama pull <model>` without leaving terminal | Ollama POST `/api/pull` streaming endpoint confirmed; progress shown via `completed/total` fields |
| SETUP-05 | User prompted for Tavily API key; key written to `~/.config/myhelper/config.json` | JSON merge pattern verified; key `tavily_key` confirmed from `search.Config` |
| SETUP-06 | User can optionally enter SearXNG endpoint; written to config | Key `search_endpoint` confirmed from `search.Config`; optional prompt (enter to skip) |
</phase_requirements>

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Ollama reachability check | API/Backend | — | TCP/HTTP probe to localhost:11434; no UI concern |
| Hardware detection (VRAM/RAM) | API/Backend | — | `os/exec` subprocess calls; platform-specific shell out |
| Model recommendation | API/Backend | — | Pure logic: memory bytes → model string table |
| Model pull with progress | API/Backend | CLI output | Ollama streaming API; progress printed to stdout |
| Config file read-merge-write | API/Backend | — | `encoding/json` + `os` stdlib; isolated function |
| stdin prompt loop | CLI | — | `bufio.Scanner(os.Stdin)` per project convention |
| Cobra subcommand wiring | CLI | — | `init()` + `rootCmd.AddCommand()` pattern |

## Standard Stack

### Core (all stdlib — no new dependencies)
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `encoding/json` | stdlib | Config file read-merge-write | Already used in project |
| `net/http` | stdlib | Ollama reachability check, pull API | Already used in `internal/ollama` |
| `os/exec` | stdlib | Spawn `nvidia-smi`, `system_profiler` | No external dep needed for subprocess |
| `runtime` | stdlib | `runtime.GOOS` for platform detection | Zero-cost platform branching |
| `bufio` | stdlib | `bufio.NewScanner(os.Stdin)` for prompts | Project convention (used in conversation loop) |
| `os` | stdlib | `os.MkdirAll`, `os.WriteFile`, `os.ReadFile` | File I/O |
| `path/filepath` | stdlib | Construct `~/.config/myhelper/config.json` path | Already used in `config.go` and `search.go` |

**No new imports to go.mod.** [VERIFIED: go.mod read; all listed packages are Go stdlib]

### Existing Internal Packages Available
| Package | What Wizard Uses |
|---------|-----------------|
| (none required) | Wizard is self-contained; does NOT import internal/search or internal/config at write time — it does its own config merge to avoid re-introducing the MarshalJSON redaction issue |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `os/exec` for hardware detection | Ollama `/api/ps` or GPU endpoint | Ollama doesn't expose hardware info via API; `os/exec` is the only reliable option |
| `bufio.Scanner` for input | `fmt.Scan` | Scanner handles empty lines and EOF more cleanly; project already uses it in `cmd/helpers.go` |
| `map[string]interface{}` merge | Typed struct merge | Typed struct drops unknown keys on marshal; `map[string]interface{}` preserves them |

## Architecture Patterns

### System Architecture Diagram

```
myhelper setup
     │
     ▼
[Stage 1: Ollama Check]
  GET http://localhost:11434/
  ├─ 200 / "Ollama is running" ──────────────────────► [Stage 2: Hardware Detection]
  └─ error / non-200 ──► print platform install cmd ──► exit (non-fatal)
                          (runtime.GOOS: darwin→brew, linux→curl)

[Stage 2: Hardware Detection]
  ├─ linux: exec nvidia-smi → VRAM MiB
  │         └─ fail → read /proc/meminfo → RAM kB
  └─ darwin: exec system_profiler SPHardwareDataType → "Memory: N GB"
             └─ fail → sysctl hw.memsize → bytes
                        └─ fail → unknown (0)
  → memoryMiB → recommendModel() → modelName string
     │
     ▼
[Stage 3: Model Pull Prompt]
  print "Recommended model: <name> (requires ~<N>GB)"
  prompt "Pull now? [Y/n]: "
  ├─ yes → POST /api/pull stream → print progress bar → "done"
  └─ no  → skip
     │
     ▼
[Stage 4: Tavily Key Prompt]
  prompt "Tavily API key (enter to skip): "
  ├─ non-empty → mergeConfig({"tavily_key": key, "search_provider": "tavily"})
  └─ empty     → skip
     │
     ▼
[Stage 5: SearXNG Endpoint Prompt]
  prompt "SearXNG endpoint (enter to skip): "
  ├─ non-empty → mergeConfig({"search_endpoint": endpoint})
  └─ empty     → skip
     │
     ▼
  print "Setup complete. Run: myhelper <question>"
```

### Recommended Project Structure
```
cmd/
└── setup.go          # cobra subcommand, init(), thin RunE calling wizard.Run()
internal/
└── wizard/
    ├── wizard.go      # Run(stdin io.Reader, stdout io.Writer) — all wizard logic
    └── wizard_test.go # unit tests with fake stdin/stdout
```

Placing logic in `internal/wizard/` follows the project's internal package pattern
(`internal/config`, `internal/search`, `internal/ollama`) and makes the logic testable
without cobra dependency.

### Pattern 1: Cobra Subcommand Registration

**What:** New file in `cmd/` registers the command via `init()`.
**When to use:** Every new subcommand in this project.
**Example:**
```go
// Source: cmd/inspect.go (verified in codebase)
var setupCmd = &cobra.Command{
    Use:   "setup",
    Short: "Interactive first-run wizard",
    Args:  cobra.NoArgs,
    RunE:  runSetup,
}

func init() {
    rootCmd.AddCommand(setupCmd)
}

func runSetup(cmd *cobra.Command, args []string) error {
    return wizard.Run(os.Stdin, os.Stdout)
}
```

### Pattern 2: Config File Read-Merge-Write

**What:** Read existing JSON into `map[string]interface{}`, update specific keys, write back.
**When to use:** Any time the wizard writes config — preserves keys owned by other packages.
**Example:**
```go
// Source: [VERIFIED: stdlib encoding/json; pattern derived from config_test.go]
func mergeConfig(path string, updates map[string]interface{}) error {
    existing := map[string]interface{}{}
    if data, err := os.ReadFile(path); err == nil {
        _ = json.Unmarshal(data, &existing) // ignore error: empty/corrupt = start fresh
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
    return os.WriteFile(path, data, 0600) // 0600: config contains API key
}
```

**Critical:** Use `0600` permissions for the config file — it holds the Tavily API key.

### Pattern 3: Ollama Pull Streaming Progress

**What:** POST to `/api/pull` with `"stream": true`, parse NDJSON lines for progress.
**When to use:** SETUP-04 — pull model in-wizard.
**Verified pull response fields:**
```json
{"status":"pulling 74701a8c35f6","digest":"sha256:...","total":1321082688,"completed":866218560}
{"status":"verifying sha256 digest"}
{"status":"success"}
```
**Example:**
```go
// Source: [VERIFIED: live Ollama API at localhost:11434/api/pull]
type pullProgress struct {
    Status    string `json:"status"`
    Total     int64  `json:"total"`
    Completed int64  `json:"completed"`
}
// Stream lines, print "\r<status> <completed>/<total>" until status=="success"
```

### Pattern 4: Hardware Detection — macOS (Apple Silicon + Intel)

**What:** Parse `system_profiler SPHardwareDataType` output.
**Verified output format:**
```
      Memory: 32 GB
```
**Example:**
```go
// Source: [VERIFIED: exec on darwin/arm64 machine in this session]
out, err := exec.Command("system_profiler", "SPHardwareDataType").Output()
// parse lines for "Memory:" → "32 GB" → 32768 MiB
```
Note: On Apple Silicon, RAM is unified memory — use RAM as the VRAM equivalent.

### Pattern 5: Hardware Detection — Linux/WSL (Nvidia GPU)

**What:** `nvidia-smi --query-gpu=memory.total --format=csv,noheader,nounits` returns MiB.
**When to use:** Linux with Nvidia GPU.
**Fallback:** `/proc/meminfo` → `MemTotal: NNNNNN kB` → divide by 1024 to get MiB.
**Example:**
```go
// Source: [ASSUMED: nvidia-smi not present on this macOS dev machine; format is documented standard]
out, err := exec.Command("nvidia-smi",
    "--query-gpu=memory.total",
    "--format=csv,noheader,nounits").Output()
// returns "8192\n" for 8GB VRAM
```

### Pattern 6: bufio.Scanner stdin Prompts

**What:** Project uses `bufio.NewScanner(os.Stdin)` / `bufio.NewReader(os.Stdin)`.
**Verified in:** `cmd/helpers.go` (`stdinReader` package var, `readLine`, `readMultiLine`).
**Example:**
```go
// Source: [VERIFIED: cmd/helpers_test.go replaceStdin pattern confirms io.Reader interface]
func prompt(w io.Writer, r io.Reader, msg string) string {
    fmt.Fprint(w, msg)
    scanner := bufio.NewScanner(r)
    if scanner.Scan() {
        return strings.TrimSpace(scanner.Text())
    }
    return ""
}
```
Accepting `io.Reader` / `io.Writer` makes wizard logic testable without redirecting `os.Stdin`.

### Anti-Patterns to Avoid
- **Unmarshal into typed struct for config write:** Drops unknown JSON keys on re-marshal. Use `map[string]interface{}` instead.
- **Single `bufio.Scanner` across multiple prompt calls:** A `bufio.Scanner` may buffer ahead; create a new `bufio.Reader` per read or use a single `bufio.Reader` passed through all steps.
- **`exec.Command("ollama", "pull", ...)` instead of HTTP API:** `ollama` binary may not be in PATH even if the Ollama service is running. Use the HTTP API directly (`POST /api/pull`).
- **Parsing hardware detection stderr:** `nvidia-smi` errors go to stderr; only parse stdout.
- **Blocking stdin after wizard exits:** Do not leave an unconsumed `bufio.Scanner` that would swallow the first chat input after setup.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Terminal spinner during pull | Custom animation | Reuse `startSpinner` from `cmd/search.go` | Already implemented; pull progress line overrides work without spinner complexity |
| Config file locking | Custom flock/advisory lock | None needed | Single-user CLI; concurrent writes to config are not a realistic concern |
| JSON merge library | Third-party lib | `map[string]interface{}` + stdlib | Zero deps; pattern is 8 lines of code |

**Key insight:** Every non-trivial capability (HTTP streaming, subprocess execution, JSON I/O)
is covered by Go stdlib. Adding a dependency for a 200-line wizard would violate the project's
explicit minimal-dependency philosophy.

## Common Pitfalls

### Pitfall 1: Config Overwrite Destroys Existing Settings
**What goes wrong:** Wizard unmarshals into `search.Config`, sets `TavilyKey`, re-marshals — silently drops `endpoint`, `model`, `token_threshold` fields owned by `config.Config`.
**Why it happens:** `search.Config.MarshalJSON` even redacts `TavilyKey` to `[REDACTED]` — do not serialize through any typed struct.
**How to avoid:** Always use the `map[string]interface{}` merge pattern.
**Warning signs:** Config file contains only the fields written by the wizard after setup.

### Pitfall 2: Ollama Check Wrongly Reports "Not Installed"
**What goes wrong:** Checking for `ollama` binary in PATH via `exec.LookPath` fails even when Ollama is running as a service (e.g., started via macOS app, systemd, or launchd).
**Why it happens:** The Ollama service runs independently of the CLI binary location.
**How to avoid:** Check reachability via HTTP `GET http://localhost:11434/` — returns `200 OK` with body `"Ollama is running"` when the service is up. This is the authoritative check.
**Warning signs:** Test on a machine where `ollama` is not in PATH but the service IS running.

### Pitfall 3: bufio.Scanner Buffering Consumes Input Across Steps
**What goes wrong:** A `bufio.Scanner` wrapping `os.Stdin` is created for step 1 and GC'd; the next step creates a new one but the first scanner already buffered ahead lines.
**Why it happens:** `bufio.Scanner` reads in chunks from the underlying reader.
**How to avoid:** Use a single `*bufio.Reader` (not `Scanner`) threaded through all wizard steps, OR use `bufio.NewReader` consistently and call `.ReadString('\n')`. Do NOT create multiple independent `bufio.Scanner` instances over the same `io.Reader`.
**Warning signs:** Wizard skips a step in tests where stdin has multiple lines pre-loaded.

### Pitfall 4: nvidia-smi Not Present on Linux Silently Returns Wrong Memory
**What goes wrong:** `exec.Command("nvidia-smi", ...)` returns exit code 1; code checks only `err != nil` but also needs to check if output is empty.
**Why it happens:** Some Linux machines lack nvidia-smi but have AMD or integrated graphics.
**How to avoid:** After `nvidia-smi` fails, fall through to `/proc/meminfo` RAM detection without returning an error.
**Warning signs:** Model recommendation shows "< 4GB" on a 16GB RAM machine without Nvidia GPU.

### Pitfall 5: Pull Progress Display Garbles Terminal on Non-TTY
**What goes wrong:** `\r` carriage-return progress overwrites work on TTY but produces junk in piped/non-TTY output.
**Why it happens:** `\r` without TTY detection appends literal CR characters.
**How to avoid:** Check `readline.IsTerminal(int(os.Stdout.Fd()))` (already imported via `chzyer/readline`) before using `\r` progress display. Fall back to line-by-line status on non-TTY.
**Warning signs:** `myhelper setup | tee setup.log` produces garbled output.

### Pitfall 6: API Key Written with 0644 Permissions
**What goes wrong:** Config file containing Tavily API key is world-readable.
**Why it happens:** Default `os.WriteFile` permission example uses `0644`.
**How to avoid:** Use `0600` for the config file. If the file already exists with wider permissions, consider `os.Chmod` after write.
**Warning signs:** `ls -la ~/.config/myhelper/config.json` shows `-rw-r--r--`.

## Code Examples

Verified patterns from official sources and codebase inspection:

### Ollama Reachability Check
```go
// Source: [VERIFIED: GET http://localhost:11434/ returns 200 + "Ollama is running"]
func checkOllama(endpoint string) bool {
    resp, err := http.Get("http://" + endpoint + "/")
    if err != nil {
        return false
    }
    defer resp.Body.Close()
    return resp.StatusCode == http.StatusOK
}
```

### Pull API Streaming (complete response types)
```go
// Source: [VERIFIED: live /api/pull streaming response in this session]
type pullRequest struct {
    Name   string `json:"name"`
    Stream bool   `json:"stream"`
}
type pullProgress struct {
    Status    string `json:"status"`
    Digest    string `json:"digest,omitempty"`
    Total     int64  `json:"total,omitempty"`
    Completed int64  `json:"completed,omitempty"`
}
// Final line: {"status":"success"}
// Error line:  {"error":"pull model manifest: file does not exist"}
```

### macOS RAM Detection
```go
// Source: [VERIFIED: system_profiler SPHardwareDataType on darwin/arm64 in this session]
// Output line: "      Memory: 32 GB"
func parseMacRAMMiB(output string) int64 {
    for _, line := range strings.Split(output, "\n") {
        if strings.Contains(line, "Memory:") {
            // Extract "32 GB" → 32768 MiB
            parts := strings.Fields(strings.TrimPrefix(strings.TrimSpace(line), "Memory:"))
            // parts[0] = "32", parts[1] = "GB"
        }
    }
}
```

### Config Merge (complete, safe)
```go
// Source: [VERIFIED: derived from config_test.go pattern + stdlib encoding/json]
// homeConfigPath() returns filepath.Join(home, ".config", "myhelper", "config.json")
func mergeHomeConfig(updates map[string]interface{}) error {
    home, err := os.UserHomeDir()
    if err != nil {
        return err
    }
    path := filepath.Join(home, ".config", "myhelper", "config.json")
    existing := map[string]interface{}{}
    if data, readErr := os.ReadFile(path); readErr == nil {
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
```

### Model Recommendation Table
```go
// Source: [ASSUMED: thresholds derived from Ollama model size conventions]
// memMiB = detected memory in MiB (VRAM preferred, RAM fallback)
func recommendModel(memMiB int64) (name string, requiredMiB int64) {
    switch {
    case memMiB >= 24*1024:
        return "qwen2.5-coder:14b", 10 * 1024  // ~10GB
    case memMiB >= 12*1024:
        return "qwen2.5-coder:7b", 6 * 1024    // ~6GB (project default model)
    case memMiB >= 6*1024:
        return "llama3.2:3b", 3 * 1024         // ~3GB
    default:
        return "llama3.2:1b", 1400             // ~1.4GB
    }
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Check `ollama` binary in PATH | HTTP GET `/` endpoint | — | Works when service runs as daemon without CLI in PATH |
| Write entire typed struct to config | Merge `map[string]interface{}` | — | Safe for multi-package shared config file |

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | nvidia-smi on Linux returns VRAM MiB via `--query-gpu=memory.total --format=csv,noheader,nounits` | Hardware Detection pattern | Output format differs → VRAM parse fails → fallback to RAM (graceful) |
| A2 | Model recommendation thresholds (6GB → 7B, 12GB → 14B, etc.) | Code Examples | Recommendations suboptimal for specific hardware, but not blocking |
| A3 | Wizard should also write `model` field to config when user confirms pull | Wizard flow | If omitted, user gets recommended model pulled but still uses default qwen2.5-coder:7b; fixable post-review |
| A4 | Ollama service endpoint for wizard check is `localhost:11434` (hardcoded in wizard, not read from config) | Ollama check | If user has custom Ollama port, wizard check may miss it; negligible for first-run wizard |

## Open Questions

1. **Should wizard write `model` field to config after successful pull?**
   - What we know: SETUP-04 says "pull the recommended model" but doesn't mention setting it as active model
   - What's unclear: Whether setup should also set `model` in config so subsequent `myhelper` runs use the pulled model
   - Recommendation: Write `model` field — pulling without activating would confuse users

2. **Should wizard show a progress percentage during pull, or just status string?**
   - What we know: Pull response includes `completed` and `total` bytes
   - What's unclear: Whether `total` is always present (manifest pull line has no total)
   - Recommendation: Show percentage only when `total > 0`; show raw status otherwise

3. **What if Ollama is not installed AND platform is not darwin/linux (e.g., Windows)?**
   - What we know: SETUP-02 only specifies macOS (brew) and Linux/WSL (curl)
   - What's unclear: How to handle `runtime.GOOS == "windows"` (non-WSL)
   - Recommendation: Windows native is explicitly out of scope (REQUIREMENTS.md); print a generic message directing to ollama.com/download

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Ollama service | SETUP-01/03/04 | ✓ | 0.21.1 at localhost:11434 | N/A (wizard detects absence and shows install instructions) |
| `system_profiler` | SETUP-03 (macOS) | ✓ | macOS built-in | `sysctl hw.memsize` fallback |
| `nvidia-smi` | SETUP-03 (Linux) | ✗ (macOS dev machine) | — | `/proc/meminfo` RAM fallback |
| `/proc/meminfo` | SETUP-03 (Linux no GPU) | ✗ (macOS) | — | 0 MiB → recommend smallest model |

**Missing dependencies with no fallback:** None — all missing tools have graceful fallbacks.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing (stdlib `testing` package) |
| Config file | none — Go test runner uses `go test ./...` |
| Quick run command | `go test ./internal/wizard/...` |
| Full suite command | `go test ./...` |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| SETUP-01 | checkOllama returns true when GET / returns 200 | unit | `go test ./internal/wizard/... -run TestCheckOllama` | ❌ Wave 0 |
| SETUP-01 | checkOllama returns false on connection refused | unit | `go test ./internal/wizard/... -run TestCheckOllama` | ❌ Wave 0 |
| SETUP-02 | installInstructions returns brew command on darwin | unit | `go test ./internal/wizard/... -run TestInstallInstructions` | ❌ Wave 0 |
| SETUP-02 | installInstructions returns curl command on linux | unit | `go test ./internal/wizard/... -run TestInstallInstructions` | ❌ Wave 0 |
| SETUP-03 | recommendModel returns correct model for each memory tier | unit | `go test ./internal/wizard/... -run TestRecommendModel` | ❌ Wave 0 |
| SETUP-04 | pull progress parsing: status/total/completed fields decoded | unit | `go test ./internal/wizard/... -run TestPullProgress` | ❌ Wave 0 |
| SETUP-05 | mergeHomeConfig writes tavily_key without overwriting other fields | unit | `go test ./internal/wizard/... -run TestMergeConfig` | ❌ Wave 0 |
| SETUP-05 | mergeHomeConfig creates directory if absent | unit | `go test ./internal/wizard/... -run TestMergeConfig` | ❌ Wave 0 |
| SETUP-05 | mergeHomeConfig sets file permissions to 0600 | unit | `go test ./internal/wizard/... -run TestMergeConfig` | ❌ Wave 0 |
| SETUP-06 | mergeHomeConfig writes search_endpoint without overwriting tavily_key | unit | `go test ./internal/wizard/... -run TestMergeConfig` | ❌ Wave 0 |
| SETUP-01–06 | Full wizard Run() with no-Ollama path and fake stdin | integration | `go test ./internal/wizard/... -run TestRun` | ❌ Wave 0 |

### Sampling Rate
- **Per task commit:** `go test ./internal/wizard/...`
- **Per wave merge:** `go test ./...`
- **Phase gate:** `go test ./...` green before `/gsd-verify-work`

### Wave 0 Gaps
- [ ] `internal/wizard/wizard.go` — wizard logic (new package)
- [ ] `internal/wizard/wizard_test.go` — unit tests for all testable functions
- [ ] `cmd/setup.go` — cobra subcommand wiring

*(No existing test infrastructure covers these — all new files)*

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | — |
| V3 Session Management | no | — |
| V4 Access Control | no | — |
| V5 Input Validation | yes | Validate Tavily key is non-empty before writing; validate SearXNG endpoint has http/https prefix |
| V6 Cryptography | no | Key stored at rest in config file (not encrypted — consistent with existing config pattern) |

### Known Threat Patterns for Setup Wizard

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| API key written with wrong permissions | Information Disclosure | `os.WriteFile(path, data, 0600)` — owner read/write only |
| API key logged to stdout/stderr | Information Disclosure | Never print the key; print only `"Tavily key saved."` confirmation |
| Path traversal in SearXNG endpoint input | Tampering | Endpoint is written to config as a string, not used as a filesystem path; validate `http://` or `https://` prefix |

**Note:** `search.Config.MarshalJSON` redacts `TavilyKey` to `[REDACTED]` in any JSON serialization through that type — but the wizard must NOT use that type for config writing (see Pitfall 1). The wizard writes raw JSON string values directly.

## Sources

### Primary (HIGH confidence)
- [VERIFIED: codebase — `internal/search/search.go`] — exact JSON tags for config fields (`tavily_key`, `search_provider`, `search_endpoint`, `tavily_endpoint`)
- [VERIFIED: codebase — `internal/config/config.go`] — `homeConfigPath()` returns `~/.config/myhelper/config.json`
- [VERIFIED: codebase — `cmd/inspect.go`] — cobra subcommand registration pattern via `init()` + `rootCmd.AddCommand()`
- [VERIFIED: codebase — `cmd/helpers_test.go`] — `replaceStdin` / `stdinReader` pattern confirms `io.Reader` injection for testability
- [VERIFIED: live Ollama API at localhost:11434] — `GET /` returns `"Ollama is running"` with 200; `POST /api/pull stream:true` NDJSON response format confirmed with `status`, `total`, `completed`, `digest` fields; final `{"status":"success"}` confirmed
- [VERIFIED: exec on this machine] — `system_profiler SPHardwareDataType` output format: `"      Memory: 32 GB"`
- [VERIFIED: codebase — `go.mod`] — no new dependencies required; all needed packages are Go stdlib

### Secondary (MEDIUM confidence)
- [CITED: https://ollama.com/download] — Linux install command: `curl -fsSL https://ollama.com/install.sh | sh`
- [CITED: CLAUDE.md] — macOS install: `brew install ollama`

### Tertiary (LOW confidence)
- [ASSUMED] — nvidia-smi `--query-gpu=memory.total --format=csv,noheader,nounits` output format (not verifiable on macOS dev machine)
- [ASSUMED] — Model recommendation memory thresholds

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — all stdlib, zero new deps, verified against go.mod
- Architecture: HIGH — Ollama API verified live; config merge pattern derived from existing codebase
- Hardware detection: MEDIUM — macOS path verified live; Linux/nvidia-smi path assumed from documented API
- Pitfalls: HIGH — most derived from direct codebase reading (MarshalJSON redaction, bufio Scanner behavior)

**Research date:** 2026-05-10
**Valid until:** 2026-06-10 (Ollama API is stable; stdlib is stable)
