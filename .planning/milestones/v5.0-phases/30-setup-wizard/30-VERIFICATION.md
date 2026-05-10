---
phase: 30-setup-wizard
verified: 2026-05-10T00:00:00Z
status: human_needed
score: 12/12 must-haves verified
overrides_applied: 0
human_verification:
  - test: "Run `myhelper setup` on a machine WITHOUT Ollama running (or with port 11434 closed)"
    expected: "Output shows platform-specific install instructions: 'brew install ollama' on macOS, 'curl -fsSL https://ollama.com/install.sh | sh' on Linux/WSL. Should NOT show the model-pull prompt."
    why_human: "checkOllama() is hardcoded to localhost:11434. Visual confirmation that the correct install instructions appear and the wizard exits cleanly requires a live terminal session."
  - test: "Run `myhelper setup` on a machine WITH Ollama running"
    expected: "Shows detected memory in MiB, recommended model name, and the 'Pull [model] now? [Y/n]:' prompt. Accepting the pull streams NDJSON progress to terminal. After pull completes, writes the model field to ~/.config/myhelper/config.json."
    why_human: "Requires a real Ollama instance. The NDJSON streaming progress display (\\r overwrites) needs visual confirmation. Config write side-effect requires inspecting the actual file."
  - test: "Enter a valid Tavily API key at the Tavily prompt"
    expected: "Key is written to ~/.config/myhelper/config.json under 'tavily_key'. Pre-existing keys (e.g. 'endpoint') are preserved. File permissions are 0600."
    why_human: "Requires a real terminal run. The filesystem side-effect and permission bit need human verification on the actual machine."
  - test: "Enter a SearXNG endpoint with https:// prefix, then again with a bare hostname"
    expected: "Valid https:// endpoint is written to config. Bare hostname triggers 'Warning: SearXNG endpoint must begin with http:// or https://; skipping.' and nothing is written."
    why_human: "Input validation path and the warning message need visual confirmation."
---

# Phase 30: Setup Wizard Verification Report

**Phase Goal:** A new user can go from zero to working chat in a single `myhelper setup` run
**Verified:** 2026-05-10
**Status:** human_needed (all automated checks VERIFIED; 4 items require live terminal confirmation)
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | `checkOllama()` returns true when GET returns 200 | VERIFIED | `wizard.go:130` uses `ollamaHTTPClient.Get(ollamaBaseURL + "/")`, returns `resp.StatusCode == http.StatusOK`. `TestCheckOllama` passes with httptest server. |
| 2 | `checkOllama()` returns false on connection refused or non-200 | VERIFIED | `wizard.go:131-132` returns false on any error; `TestCheckOllama` verifies with unused port `127.0.0.1:19999`. |
| 3 | `installInstructions()` returns brew on darwin, curl on linux | VERIFIED | `wizard.go:140-148` switches on `runtime.GOOS`; darwin→"brew install ollama", linux→curl URL. `TestInstallInstructions` asserts non-empty and contains "ollama". |
| 4 | `detectMemoryMiB()` returns GPU VRAM on Linux (nvidia-smi), falls back to /proc/meminfo RAM | VERIFIED | `wizard.go:156-178` runs nvidia-smi with `--query-gpu=memory.total --format=csv,noheader,nounits`, parses first line (WR-02 multi-GPU fix). Falls back to `/proc/meminfo MemTotal`. |
| 5 | `detectMemoryMiB()` returns system RAM from system_profiler on macOS, falls back to sysctl | VERIFIED | `wizard.go:181-213` parses `system_profiler SPHardwareDataType` "Memory: N GB", then falls back to `sysctl -n hw.memsize` in bytes/1024/1024. |
| 6 | `recommendModel()` returns correct model name and required MiB for each memory tier | VERIFIED | `wizard.go:218-228` — 4-tier switch: ≥24576→qwen2.5-coder:14b/10240, ≥12288→qwen2.5-coder:7b/6144, ≥6144→llama3.2:3b/3072, default→llama3.2:1b/1400. `TestRecommendModel` table passes all 4 tiers. |
| 7 | `pullModel()` streams POST /api/pull, prints progress, returns nil on success | VERIFIED | `wizard.go:232-276` posts to `ollamaBaseURL+"/api/pull"`, scans NDJSON with `bufio.Scanner` on resp.Body, prints progress. Returns error if `p.Error != ""` or no success line received (CR-02). `TestPullModel` and `TestPullModel_Error` both pass. |
| 8 | `mergeHomeConfig()` writes tavily_key and search_provider without overwriting unrelated keys | VERIFIED | `wizard.go:282-302` — reads existing file into `map[string]interface{}`, merges only the provided keys, writes back. `TestMergeConfig` (b) confirms pre-existing "endpoint" key is preserved after merge. |
| 9 | `mergeHomeConfig()` writes file with 0600 permissions | VERIFIED | `wizard.go:301` — `os.WriteFile(path, data, 0600)`. `TestMergeConfig` (c) asserts `info.Mode().Perm() != 0600` → test passes. |
| 10 | `mergeHomeConfig()` creates `~/.config/myhelper/` directory if absent | VERIFIED | `wizard.go:298` — `os.MkdirAll(filepath.Dir(path), 0755)` before WriteFile. `TestMergeConfig` (a) writes to a fresh TempDir with no pre-existing file. |
| 11 | `Run()` threads a single `*bufio.Reader` through all prompt steps | VERIFIED | `wizard.go:60` — `br := bufio.NewReader(r)` is the only call; `grep -c "bufio.NewReader" wizard.go` = 1. `pullModel` uses `bufio.NewScanner` on resp.Body only. |
| 12 | Wizard writes model field to config after successful pull | VERIFIED | `wizard.go:84` — `_ = mergeHomeConfig(map[string]interface{}{"model": model})` executed on successful `pullModel` return. |

**Score:** 12/12 truths verified

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `internal/wizard/wizard.go` | All wizard logic: Run, checkOllama, installInstructions, detectMemoryMiB, recommendModel, pullModel, mergeHomeConfig, homeConfigPath | VERIFIED | 317 lines, all 8 functions present; `go build ./internal/wizard/...` and `go vet ./internal/wizard/...` exit 0. No `github.com/bkohler93/myhelper/*` imports. |
| `internal/wizard/wizard_test.go` | Unit tests for all wizard functions | VERIFIED | 217 lines; 8 test functions present: TestCheckOllama, TestInstallInstructions, TestRecommendModel, TestMergeConfig, TestPullModel, TestPullModel_Error, TestRun_NoOllama, TestRun_SkipAll. All pass. |
| `cmd/setup.go` | Cobra subcommand wiring for `myhelper setup` | VERIFIED | 23 lines; `setupCmd` registered, `init()` calls `rootCmd.AddCommand(setupCmd)`, `runSetup` delegates to `wizard.Run(os.Stdin, os.Stdout)`. |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| `wizard.go:checkOllama` | `http://localhost:11434/` | `ollamaHTTPClient.Get(ollamaBaseURL + "/")` | WIRED | `wizard.go:130` — uses injected client and URL var. |
| `wizard.go:pullModel` | `http://localhost:11434/api/pull` | `pullHTTPClient.Post(ollamaBaseURL+"/api/pull", ...)` | WIRED | `wizard.go:235` — uses injected client and URL var. |
| `wizard.go:mergeHomeConfig` | `~/.config/myhelper/config.json` | `os.WriteFile(path, data, 0600)` | WIRED | `wizard.go:301` — path from `homeConfigPath()`, 0600 mode confirmed. |
| `cmd/setup.go:runSetup` | `internal/wizard:Run` | `wizard.Run(os.Stdin, os.Stdout)` | WIRED | `cmd/setup.go:22` — direct delegation. |
| `cmd/setup.go:init` | `cmd/root.go:rootCmd` | `rootCmd.AddCommand(setupCmd)` | WIRED | `cmd/setup.go:18` — confirmed. Binary `myhelper --help` shows "setup" entry. |

### Data-Flow Trace (Level 4)

This phase produces a terminal wizard with side effects (config file writes and stdout output) rather than a component rendering dynamic data. Data flow traces are not applicable in the React component sense. The key data paths are verified above through direct grep and test execution:

| Data Path | Source | Sink | Produces Real Data | Status |
|-----------|--------|------|--------------------|--------|
| `detectMemoryMiB()` → model recommendation | `nvidia-smi` / `/proc/meminfo` / `system_profiler` / `sysctl` | `recommendModel()` → `fmt.Fprintf(w, ...)` | Yes — reads real system hardware | FLOWING |
| User input → `mergeHomeConfig()` | `br.ReadString('\n')` | `os.WriteFile(homeConfigPath(), ...)` | Yes — writes live user input to real config | FLOWING |
| `pullModel()` progress | Ollama HTTP stream | `fmt.Fprintf(w, "\r%s ...")` | Yes — streams NDJSON from Ollama API | FLOWING (verified by test with fake server) |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| All wizard tests pass | `go test ./internal/wizard/... -v` | 8/8 PASS | PASS |
| All packages compile | `go build ./...` | exit 0 | PASS |
| Full test suite passes | `go test ./...` | all packages ok | PASS |
| `myhelper setup --help` shows command | `go build -o /tmp/myhelper-verify . && /tmp/myhelper-verify setup --help` | "Interactive first-run wizard: check Ollama, detect hardware, configure search keys" | PASS |
| Single `bufio.NewReader` in wizard | `grep -c "bufio.NewReader" internal/wizard/wizard.go` | 1 | PASS |
| No internal/* imports | `grep "bkohler93/myhelper" internal/wizard/wizard.go` | (empty — no matches) | PASS |
| 0600 perms written | `grep "0600" internal/wizard/wizard.go` | `os.WriteFile(path, data, 0600)` | PASS |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|-------------|-------------|--------|----------|
| SETUP-01 | 30-01, 30-02 | User can run `myhelper setup` to check Ollama on port 11434 | SATISFIED | `cmd/setup.go` registers cobra command; `wizard.go:checkOllama()` hits localhost:11434. |
| SETUP-02 | 30-01, 30-02 | Platform-specific install instructions when Ollama not detected | SATISFIED | `wizard.go:64` prints `installInstructions()` result when `!checkOllama()`; darwin→brew, linux→curl. |
| SETUP-03 | 30-01, 30-02 | Recommended model based on detected GPU VRAM or RAM | SATISFIED | `wizard.go:70-72` calls `detectMemoryMiB()` then `recommendModel()`, prints result. |
| SETUP-04 | 30-01, 30-02 | User can confirm to pull recommended model in-wizard | SATISFIED | `wizard.go:75-87` prompts [Y/n], calls `pullModel()` with NDJSON streaming on accept. |
| SETUP-05 | 30-01, 30-02 | Tavily API key prompted; written to `~/.config/myhelper/config.json` | SATISFIED | `wizard.go:91-100` prompts, calls `mergeHomeConfig({"tavily_key": line, "search_provider": "tavily"})`. |
| SETUP-06 | 30-01, 30-02 | SearXNG endpoint optionally entered; written to config | SATISFIED | `wizard.go:103-118` prompts, validates http(s) prefix (T-30-03), calls `mergeHomeConfig({"search_endpoint": line})`. |

### Anti-Patterns Found

No blockers or warnings found. Scan of `internal/wizard/wizard.go` and `cmd/setup.go`:

- No TODO/FIXME/PLACEHOLDER comments
- No `return null` / `return {}` / `return []` stubs
- No hardcoded empty state passed to rendering
- The `time` import in wizard.go is used for `ollamaHTTPClient` and `pullHTTPClient` timeout values — not dead code

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| (none) | — | — | — | — |

### Human Verification Required

**Items needing live terminal testing:**

#### 1. Wizard on machine without Ollama

**Test:** Stop or block Ollama (or run on a machine where it was never installed). Run `myhelper setup`.
**Expected:** Output shows "Ollama is not running." followed by the platform-correct install command ("brew install ollama" on macOS, the curl command on Linux/WSL). Wizard exits without showing the model-pull prompt.
**Why human:** `checkOllama()` probes real `localhost:11434`. Confirming the correct branch fires in a real terminal with the right OS-specific string requires a live run.

#### 2. Wizard on machine with Ollama running

**Test:** Start Ollama (`ollama serve`). Run `myhelper setup`. Accept the pull prompt.
**Expected:** Shows detected memory in MiB, recommended model name (matching tier table), then streams NDJSON progress ending in "success". After completion, `~/.config/myhelper/config.json` contains a `"model"` key set to the pulled model name.
**Why human:** Requires a live Ollama instance. The `\r` progress overwrite display and the final config file state need human inspection.

#### 3. Tavily key write to real config

**Test:** Run `myhelper setup` with Ollama running. Skip pull. Enter a test Tavily key (e.g. `tvly-test123`). Then: `cat ~/.config/myhelper/config.json && ls -la ~/.config/myhelper/config.json`.
**Expected:** File contains `"tavily_key": "tvly-test123"` and `"search_provider": "tavily"`. Any pre-existing keys (endpoint, model, token_threshold) are preserved. File permissions show `-rw-------` (0600).
**Why human:** Filesystem side-effect and permission bit require real machine verification.

#### 4. SearXNG endpoint validation

**Test:** Run `myhelper setup`. At the SearXNG prompt, enter `mysearx.local:8080` (no protocol prefix). Then run again and enter `https://mysearx.local:8080`.
**Expected:** Bare hostname → "Warning: SearXNG endpoint must begin with http:// or https://; skipping." and nothing written to config. Valid URL → endpoint written to config under `"search_endpoint"`.
**Why human:** Input validation branch and warning message need visual confirmation against a live terminal.

### Gaps Summary

No gaps. All 12 must-have truths verified. All 3 artifacts are substantive and wired. All 6 requirements (SETUP-01 through SETUP-06) are satisfied. The 4 human verification items are standard live-terminal confirmations for a wizard that inherently requires OS I/O, filesystem writes, and a real Ollama service — they are not blockers to shipping.

Note: `30-02-SUMMARY.md` was not created (the file was absent from the phase directory). This is a documentation omission only — the underlying code and tests from Plan 02 (`internal/wizard/wizard_test.go` and `cmd/setup.go`) exist and are verified by git log (commits `e054513` and `e7b1c47`).

---

_Verified: 2026-05-10_
_Verifier: Claude (gsd-verifier)_
