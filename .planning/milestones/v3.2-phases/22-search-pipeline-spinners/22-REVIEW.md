---
phase: 22-search-pipeline-spinners
reviewed: 2026-04-24T19:51:10Z
depth: standard
files_reviewed: 1
files_reviewed_list:
  - cmd/search.go
findings:
  critical: 0
  warning: 1
  info: 2
  total: 3
status: issues_found
---

# Phase 22: Code Review Report

**Reviewed:** 2026-04-24T19:51:10Z
**Depth:** standard
**Files Reviewed:** 1
**Status:** issues_found

## Summary

`cmd/search.go` introduces a goroutine-based terminal spinner (`type spinner`, `startSpinner`, `done`) and wires it at three call sites in `buildUserMessage`. The goroutine safety story is sound: no double-close of the stop channel is possible, no goroutine leaks exist, and all early returns in `buildUserMessage` are preceded by a `done()` call on the active spinner. One warning-level issue exists in `filterByIndices` (duplicate indices from the LLM produce duplicate entries in the result set). Two info-level issues are noted: a first-frame flash on the spinner before the stop channel is checked, and the clear-width calculation using `len(label)` (byte count) rather than `utf8.RuneCountInString(label)`.

## Warnings

### WR-01: `filterByIndices` does not deduplicate LLM-returned indices

**File:** `cmd/search.go:86`
**Issue:** If the LLM response contains the same 1-based index more than once (e.g., `"1\n1\n2"`), `filterByIndices` appends the corresponding `search.Result` twice. The duplicated entry is then passed through `buildWebBlock` and injected into the user message as two identical numbered result blocks. While the model will likely ignore the duplicate, it wastes token budget and produces malformed-looking web context.

**Fix:**
```go
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
```

## Info

### IN-01: Spinner prints one frame before checking the stop channel (first-frame flash)

**File:** `cmd/search.go:26`
**Issue:** The goroutine unconditionally prints the current frame at line 26 before entering the `select` at line 28. If `done()` is called before the goroutine is scheduled (extremely fast LLM response, or goroutine scheduler delay), the goroutine will still print one frame to stderr before clearing it. This leaves a brief visible flash even for sub-millisecond operations. It is cosmetically minor but noticeable on fast local Ollama responses.

**Fix:** Check the stop channel before printing the first frame:
```go
go func() {
    frames := []rune{'|', '/', '-', '\\'}
    i := 0
    t := time.NewTicker(100 * time.Millisecond)
    defer t.Stop()
    for {
        select {
        case <-s.stop:
            fmt.Fprintf(os.Stderr, "\r%s\r", strings.Repeat(" ", len(label)+3))
            return
        default:
        }
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
```

### IN-02: Clear-width uses `len(label)` (byte count), not rune count

**File:** `cmd/search.go:30`
**Issue:** The clear line uses `strings.Repeat(" ", len(label)+3)`. `len(label)` returns the byte length of the string, not the number of terminal columns it occupies. For the current labels ("Checking if web search is needed...", "Fetching web results...", "Filtering results...") this is harmless because all characters are single-byte ASCII. However, if a label ever contains multibyte UTF-8 characters, the clear string would be shorter than the printed line, leaving a trailing artifact on the terminal.

**Fix:**
```go
fmt.Fprintf(os.Stderr, "\r%s\r", strings.Repeat(" ", utf8.RuneCountInString(label)+3))
```
Add `"unicode/utf8"` to imports. For the current codebase this is a defensive improvement rather than an active bug.

---

_Reviewed: 2026-04-24T19:51:10Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
