package cmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/bkohler93/myhelper/internal/config"
	"github.com/bkohler93/myhelper/internal/history"
	"github.com/bkohler93/myhelper/internal/ollama"
	"github.com/bkohler93/myhelper/internal/search"
)

type spinner struct{ stop chan struct{} }

func startSpinner(label string) *spinner {
	s := &spinner{stop: make(chan struct{})}
	go func() {
		frames := []rune{'|', '/', '-', '\\'}
		i := 0
		t := time.NewTicker(100 * time.Millisecond)
		defer t.Stop()
		for {
			fmt.Fprintf(os.Stderr, "\r%c %s", frames[i], label)
			i = (i + 1) % len(frames)
			select {
			case <-s.stop:
				fmt.Fprintf(os.Stderr, "\r%s\r", strings.Repeat(" ", len(label)+3))
				return
			case <-t.C:
			}
		}
	}()
	return s
}

func (s *spinner) done() { close(s.stop) }

const searchGatePrompt = `Answer only "yes" or "no". Would the following query benefit from retrieving current or real-time information from the internet? Query: `

// searchGate returns true if the query needs web search.
// fails open (search skipped on error) — GATE-02.
func searchGate(query string, cfg config.Config) bool {
	messages := []history.Message{
		{Role: "user", Content: searchGatePrompt + query},
	}
	response, err := ollama.Chat(cfg, messages)
	if err != nil {
		return false
	}
	lower := strings.ToLower(strings.TrimSpace(response))
	return strings.HasPrefix(lower, "yes")
}

const reRankPrompt = `You are a search result filter. Given the user's query and a numbered list of search results, output ONLY the numbers (1-based) of results that are genuinely relevant, one per line. Output nothing else.`

// reRankResults filters results using a second LLM call.
// err != nil → return allResults, nil  (RANK-02: graceful fallback)
// success but zero valid indices → return nil, nil  (RANK-03: skip injection)
func reRankResults(query string, results []search.Result, cfg config.Config) ([]search.Result, error) {
	if len(results) == 0 {
		return nil, nil
	}
	var sb strings.Builder
	for i, r := range results {
		fmt.Fprintf(&sb, "[%d] %s\n%s\n\n", i+1, r.Title, r.Snippet)
	}
	messages := []history.Message{
		{Role: "system", Content: reRankPrompt},
		{Role: "user", Content: "Query: " + query + "\n\nResults:\n" + sb.String()},
	}
	response, err := ollama.Chat(cfg, messages)
	if err != nil {
		return results, nil // RANK-02: error → all results
	}
	selected := filterByIndices(results, response)
	if len(selected) == 0 {
		return nil, nil // RANK-03: re-rank succeeded, nothing relevant → skip
	}
	return selected, nil
}

// filterByIndices parses integer indices from the LLM response and returns the matching results.
func filterByIndices(results []search.Result, response string) []search.Result {
	seen := make(map[int]bool)
	var selected []search.Result
	for _, field := range strings.Fields(response) {
		cleaned := strings.Trim(field, "[]().,")
		if idx, err := strconv.Atoi(cleaned); err == nil {
			if idx >= 1 && idx <= len(results) && !seen[idx] {
				seen[idx] = true
				selected = append(selected, results[idx-1])
			}
		}
	}
	return selected
}

// buildWebBlock builds a [WEB RESULTS] ... [/WEB RESULTS] block.
// Drops trailing results to stay within budgetTokens.
// Returns "" if no results fit.
func buildWebBlock(results []search.Result, budgetTokens int, cfg config.Config) string {
	if len(results) == 0 || budgetTokens <= 0 {
		return ""
	}
	header := "[WEB RESULTS]\n"
	footer := "[/WEB RESULTS]\n\n"
	overhead := countTokens(header+footer, cfg)
	if overhead >= budgetTokens {
		return ""
	}
	var sb strings.Builder
	used := overhead
	for i, r := range results {
		entry := fmt.Sprintf("[%d] %s\n%s\n%s\n\n", i+1, r.Title, r.URL, r.Snippet)
		cost := countTokens(entry, cfg)
		if used+cost > budgetTokens {
			break
		}
		sb.WriteString(entry)
		used += cost
	}
	if sb.Len() == 0 {
		return ""
	}
	return header + sb.String() + footer
}

// countTokens mirrors the retrieval.tokenCount helper.
func countTokens(s string, cfg config.Config) int {
	return history.New(cfg.TokenThreshold, []history.Message{{Role: "user", Content: s}}).TokenCount()
}

// buildUserMessage augments the user query with a [WEB RESULTS] block if appropriate.
// Returns the original query unchanged when search is skipped.
// noSearch (GATE-04) takes priority over forceSearch (GATE-03).
func buildUserMessage(query string, cfg config.Config, searchCfg search.Config, forceSearch, noSearch bool) string {
	if noSearch {
		return query // GATE-04: --no-search suppresses all search
	}

	doSearch := forceSearch // GATE-03: --search forces gate=true
	if !doSearch {
		sp1 := startSpinner("Checking if web search is needed...")
		doSearch = searchGate(query, cfg) // GATE-01/GATE-02
		sp1.done()
	}
	if !doSearch {
		return query
	}

	sp2 := startSpinner("Fetching web results...")
	results, err := search.Search(query, searchCfg)
	sp2.done()
	if err != nil || len(results) == 0 {
		return query // network/empty: degrade gracefully
	}

	sp3 := startSpinner("Filtering results...")
	ranked, _ := reRankResults(query, results, cfg)
	sp3.done()
	if ranked == nil {
		return query // RANK-03: zero relevant results → skip injection
	}

	budget := cfg.TokenThreshold / 4 // 25% reserved for web context
	block := buildWebBlock(ranked, budget, cfg)
	if block == "" {
		return query
	}
	return block + query // block BEFORE query (context before question, per INJ-01)
}
