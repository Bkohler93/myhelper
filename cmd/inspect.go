package cmd

import (
	"fmt"
	"strings"

	"github.com/bkohler93/myhelper/internal/config"
	"github.com/bkohler93/myhelper/internal/history"
	"github.com/bkohler93/myhelper/internal/ollama"
	"github.com/bkohler93/myhelper/internal/search"
	"github.com/spf13/cobra"
)

var inspectCmd = &cobra.Command{
	Use:   "inspect <query>",
	Short: "Inspect the web search pipeline for a query (diagnostic dry-run)",
	Args:  cobra.ExactArgs(1),
	RunE:  runInspect,
}

func init() {
	rootCmd.AddCommand(inspectCmd)
}

func runInspect(cmd *cobra.Command, args []string) error {
	query := args[0]
	cfg := config.Load()
	ApplyFlagOverrides(&cfg)
	if err := validateConfig(cfg); err != nil {
		return err
	}
	searchCfg := search.LoadConfig()

	if searchSuppress {
		fmt.Println("search suppressed")
		return nil
	}

	fmt.Println("---")
	fmt.Println("[Gate Decision]")

	var doSearch bool
	if searchForce {
		fmt.Println("Gate: BYPASSED (--search flag)")
		doSearch = true
	} else {
		fmt.Print("Calling gate LLM...")
		messages := []history.Message{{Role: "user", Content: searchGatePrompt + query}}
		response, err := ollama.Chat(cfg, messages)
		fmt.Println()
		if err != nil {
			fmt.Printf("Gate: NO (error: %v)\n", err)
			fmt.Println("search not needed")
			return nil
		}
		trimmed := strings.TrimSpace(response)
		doSearch = strings.HasPrefix(strings.ToLower(trimmed), "yes")
		if doSearch {
			fmt.Println("Gate: YES")
		} else {
			fmt.Println("Gate: NO")
		}
		fmt.Println(trimmed)
		if !doSearch {
			fmt.Println("search not needed")
			return nil
		}
	}

	fmt.Println("---")
	fmt.Println("[Fetched Results]")
	fmt.Println("Fetching results...")
	fetched, err := search.Search(query, searchCfg)
	if err != nil {
		fmt.Printf("search error: %v\n", err)
		return nil
	}
	if len(fetched) == 0 {
		fmt.Println("(no results returned)")
		return nil
	}
	for i, r := range fetched {
		fmt.Printf("[%d] %s\n%s\n%s\n\n", i+1, r.Title, r.URL, r.Snippet)
	}

	fmt.Println("---")
	fmt.Println("[Re-rank]")
	fmt.Print("Calling re-rank LLM...")
	survivors, _ := reRankResults(query, fetched, cfg)
	fmt.Println()

	survivorURLs := make(map[string]bool)
	for _, r := range survivors {
		survivorURLs[r.URL] = true
	}
	var dropped []search.Result
	for _, r := range fetched {
		if !survivorURLs[r.URL] {
			dropped = append(dropped, r)
		}
	}

	if survivors == nil {
		fmt.Printf("Survivors (0): none\n")
		fmt.Printf("Dropped (%d): all dropped\n", len(fetched))
		fmt.Println("injection skipped")
		return nil
	}

	fmt.Printf("Survivors (%d):\n", len(survivors))
	for i, r := range survivors {
		fmt.Printf("[%d] %s\n%s\n%s\n\n", i+1, r.Title, r.URL, r.Snippet)
	}
	fmt.Printf("Dropped (%d):\n", len(dropped))
	for i, r := range dropped {
		fmt.Printf("[%d] %s\n%s\n%s\n\n", i+1, r.Title, r.URL, r.Snippet)
	}

	fmt.Println("---")
	fmt.Println("[Injected Block]")
	budget := cfg.TokenThreshold / 4
	block := buildWebBlock(survivors, budget, cfg)
	if block == "" {
		fmt.Println("(no results fit in token budget)")
		return nil
	}
	fmt.Print(block)
	tokenCost := history.New(cfg.TokenThreshold, []history.Message{{Role: "user", Content: block}}).TokenCount()
	fmt.Printf("Token cost: %d tokens\n", tokenCost)

	return nil
}
