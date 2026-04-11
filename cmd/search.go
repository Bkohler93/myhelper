package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/bkohler93/myhelper/internal/config"
	"github.com/bkohler93/myhelper/internal/history"
	"github.com/bkohler93/myhelper/internal/ollama"
	"github.com/bkohler93/myhelper/internal/search"
)

const searchGatePrompt = `Answer only "yes" or "no". Does answering the following query require current or real-time information from the internet? Query: `

// searchGate returns true if the query needs web search.
// Fails CLOSED: returns false on any LLM error (GATE-02).
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
	var selected []search.Result
	for _, field := range strings.Fields(response) {
		cleaned := strings.Trim(field, "[]().,")
		if idx, err := strconv.Atoi(cleaned); err == nil {
			if idx >= 1 && idx <= len(results) {
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
