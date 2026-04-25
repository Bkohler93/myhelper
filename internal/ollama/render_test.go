package ollama

import (
	"strings"
	"testing"
)

func TestRenderMarkdown(t *testing.T) {
	t.Run("non-empty passes through glamour unchanged text content", func(t *testing.T) {
		// In non-TTY test environments, glamour uses ASCII style which preserves
		// markdown syntax characters. We verify the text content is present and
		// glamour processed the input (output ends with \n). The visual rendering
		// of ** as bold is TTY-only and verified by the human-verify checkpoint.
		out := renderMarkdown("**bold** text")
		if !strings.Contains(out, "bold") {
			t.Errorf("expected 'bold' to appear in output, got %q", out)
		}
		if !strings.HasSuffix(out, "\n") {
			t.Errorf("expected output to end with \\n, got %q", out)
		}
	})

	t.Run("empty-guard returns empty unchanged", func(t *testing.T) {
		out := renderMarkdown("")
		if out != "" {
			t.Errorf("expected empty string back for empty input, got %q", out)
		}
	})

	t.Run("whitespace-guard returns whitespace unchanged", func(t *testing.T) {
		in := "   "
		out := renderMarkdown(in)
		if out != in {
			t.Errorf("expected whitespace-only input unchanged, got %q", out)
		}
	})

	t.Run("trailing newline present", func(t *testing.T) {
		out := renderMarkdown("hello")
		if !strings.HasSuffix(out, "\n") {
			t.Errorf("expected output to end with \\n, got %q", out)
		}
	})

	t.Run("code-block replaces raw backtick fences", func(t *testing.T) {
		out := renderMarkdown("```go\nfmt.Println()\n```")
		if strings.Contains(out, "```") {
			t.Errorf("expected no raw backtick fences in output, got %q", out)
		}
	})
}
