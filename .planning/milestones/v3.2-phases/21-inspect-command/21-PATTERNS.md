# Phase 21: inspect Command - Pattern Map

**Mapped:** 2026-04-24
**Files analyzed:** 2 (1 new, 1 modified)
**Analogs found:** 2 / 2

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `cmd/inspect.go` | command (cobra subcommand) | request-response | `cmd/inspect.go` @ git `715d55d^` | exact (previously existed, deleted in phase 16) |
| `internal/retrieval/retrieval.go` | service (struct extension) | transform | `internal/retrieval/retrieval.go` lines 758–766 (current `InspectResult`) | exact (same file, additive field) |

## Pattern Assignments

### `cmd/inspect.go` (NEW — cobra subcommand, request-response)

**Primary analog:** `cmd/inspect.go` from git commit `715d55d^` (deleted in phase 16, to be recreated)
**Secondary analog:** `cmd/starter.go` from git commit `5f822db^` (same command structure)

**Import pattern** (from deleted `cmd/inspect.go`):
```go
import (
	"fmt"
	"os"

	"github.com/bkohler93/myhelper/internal/config"
	"github.com/bkohler93/myhelper/internal/ollama"
	"github.com/bkohler93/myhelper/internal/retrieval"
	"github.com/spf13/cobra"
)
```

**cobra command declaration pattern** (from deleted `cmd/inspect.go`):
```go
var inspectCmd = &cobra.Command{
	Use:   "inspect <query>",
	Short: "Dry-run retrieval and print per-stage context selection details",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runInspect,
}
```

**init() registration pattern** (from deleted `cmd/inspect.go`):
```go
func init() {
	rootCmd.AddCommand(inspectCmd)
}
```

**RunE body pattern with noContextFlag, resolveInput, ApplyFlagOverrides** (from deleted `cmd/inspect.go` and `cmd/starter.go`):
```go
func runInspect(cmd *cobra.Command, args []string) error {
	input, err := resolveInput(args, "Query to inspect: ")
	if err != nil {
		return err
	}

	cfg := config.Load()
	ApplyFlagOverrides(&cfg)

	root, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("runInspect: getwd: %w", err)
	}

	if noContextFlag {
		fmt.Println("Context bypassed (--no-context)")
		return nil
	}

	result, err := retrieval.BuildInspectContext(root, input, retrieval.StarterStrategy, cfg, ollama.Chat)
	if err != nil {
		return fmt.Errorf("runInspect: BuildInspectContext: %w", err)
	}

	printInspectResult(result, input)
	return nil
}
```

Note: The deleted version used `retrieval.DefaultStrategy`. The new version should use `retrieval.StarterStrategy` (UseSymbols: true, UseFiles: true, MaxTokenRatio: 0.80) as `DefaultStrategy` is equivalent and `StarterStrategy` is the explicit named constant per CLAUDE.md table.

**printInspectResult pattern** (from deleted `cmd/inspect.go`). This function needs to be extended to show pre-filter candidates (INSP-03):
```go
func printInspectResult(result retrieval.InspectResult, query string) {
	fmt.Printf("--- Retrieval Inspect: %q ---\n\n", query)

	if result.GatePassed {
		fmt.Println("Gate: context required (passed)")
	} else {
		fmt.Println("Gate: context not required (skipped)")
		fmt.Println("\nNo symbols or files selected.")
		return
	}

	fmt.Println()
	for _, sm := range result.StageMetrics {
		fmt.Printf("Stage: %-14s tokens: %d\n", sm.Name, sm.TokensUsed)
	}

	// INSP-03: print pre-filter candidates (new field)
	fmt.Printf("\nPre-filter Candidates (%d):\n", len(result.PreFilterCandidates))
	for _, sym := range result.PreFilterCandidates {
		fmt.Printf("  %-40s (%s)\n", sym.StableID, sym.Kind)
	}

	fmt.Printf("\nSelected Symbols (%d):\n", len(result.Symbols))
	for _, sr := range result.Symbols {
		fmt.Printf("  %-40s (%s)\t[%s]\n", sr.Symbol.StableID, sr.Symbol.Kind, sr.Source.String())
	}

	fmt.Printf("\nSelected Files (%d):\n", len(result.Files))
	for _, fr := range result.Files {
		fmt.Printf("  %-50s [%s]\n", fr.Path, fr.Source.String())
	}

	fmt.Printf("\nFinal context size: %d tokens\n", result.FinalTokens)
}
```

---

### `internal/retrieval/retrieval.go` (MODIFY — add PreFilterCandidates to InspectResult)

**Analog:** Current `InspectResult` struct at lines 760–766 in the same file.

**Current InspectResult struct** (lines 760–766):
```go
type InspectResult struct {
	Symbols      []SymbolResult
	Files        []FileResult
	StageMetrics []StageMetrics
	FinalTokens  int
	GatePassed   bool
}
```

**Modified InspectResult struct** — add `PreFilterCandidates` field:
```go
type InspectResult struct {
	Symbols             []SymbolResult
	Files               []FileResult
	StageMetrics        []StageMetrics
	FinalTokens         int
	GatePassed          bool
	PreFilterCandidates []scanner.Symbol // symbols surviving keyword pre-filter (INSP-03)
}
```

**Capture site in BuildInspectContext** — after `candidates` is assigned (currently lines 801–806, after `applyTokenCap`):
```go
// Stage 2: Pre-filter (RET-01, RET-02)
candidates := preFilter(query, syms.Symbols, files.Files)
candidates = applyTokenCap(candidates, totalBudget, cfg)
result.PreFilterCandidates = candidates  // ADD THIS LINE (INSP-03)
preFilterTokens := 0
for _, c := range candidates {
    preFilterTokens += tokenCount(cfg, c.Signature)
}
```

---

## Shared Patterns

### resolveInput / readInteractive / ApplyFlagOverrides / noContextFlag

These helpers existed in `cmd/helpers.go` and `cmd/root.go` in older versions but were deleted in phase 16 along with the subcommands. They must be restored.

**resolveInput and readInteractive** — source: `cmd/helpers.go` @ git `715d55d^`:
```go
func resolveInput(args []string, interactivePrompt string) (string, error) {
	if len(args) > 0 && strings.TrimSpace(args[0]) != "" {
		return strings.TrimSpace(args[0]), nil
	}
	return readInteractive(interactivePrompt)
}

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
```

**noContextFlag, tokenLimitFlag, ApplyFlagOverrides** — source: `cmd/root.go` @ git `715d55d^`:
```go
var tokenLimitFlag int
var noContextFlag bool

func init() {
	rootCmd.PersistentFlags().IntVar(&tokenLimitFlag, "token-limit", 0, "override token threshold for conversation history (default 4100)")
	rootCmd.PersistentFlags().BoolVar(&noContextFlag, "no-context", false, "bypass retrieval and inject no project context")
}

func ApplyFlagOverrides(cfg *config.Config) {
	if tokenLimitFlag != 0 {
		cfg.TokenThreshold = tokenLimitFlag
	}
}
```

**Current root.go state:** The current `cmd/root.go` has a different `init()` that registers `--search` and `--no-search` persistent flags (for web search). The `--no-context` flag and `ApplyFlagOverrides` must be added back alongside these, not replacing them.

### cobra subcommand registration (init pattern)

**Apply to:** `cmd/inspect.go`

Every subcommand registers itself in its own `init()`:
```go
func init() {
	rootCmd.AddCommand(inspectCmd)
}
```

This is the sole place `rootCmd.AddCommand` is called; there is no central registration file.

### Error wrapping

**Apply to:** `cmd/inspect.go` RunE function

All RunE functions wrap errors with the function name prefix:
```go
return fmt.Errorf("runInspect: getwd: %w", err)
return fmt.Errorf("runInspect: BuildInspectContext: %w", err)
```

## No Analog Found

All files have clear analogs. No gaps requiring RESEARCH.md fallback.

## Metadata

**Analog search scope:** `/Users/brettkohler/dev/apps/myhelper/cmd/`, `/Users/brettkohler/dev/apps/myhelper/internal/retrieval/`, git history (`715d55d^`, `5f822db`)
**Files scanned:** 6 current source files + 3 historical versions
**Pattern extraction date:** 2026-04-24
